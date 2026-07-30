/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// The special layer covers the authorization rules that are not "role X may call
// endpoint Y": the ones enforced by dedicated middleware or by a credential type.
//
//   self_modification — nobody may act on their own account with dangerous verbs
//   api_key           — a myk_ token is capped by BOTH its read/write mode and
//                       the live permissions of the user who owns it, and is
//                       refused outright on session-bound routes
//   impersonate       — needs impersonate:users AND the target's consent

// SpecialSpec is one hand-written case. Kind selects the check; the remaining
// fields are its parameters.
type SpecialSpec struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
	As   string `yaml:"as"`
	// Mode is read|write for kind=api_key.
	Mode         string `yaml:"mode"`
	Method       string `yaml:"method"`
	Path         string `yaml:"path"`
	Body         string `yaml:"body"`
	Expect       string `yaml:"expect"`
	ExpectStatus []int  `yaml:"expect_status"`
	// Setup runs before the case, as other personas, to establish the state the
	// case needs — enabling a target's impersonation consent, for instance. A
	// failing setup step makes the case REVIEW rather than a silent pass, because
	// a case whose precondition never happened proves nothing.
	Setup []SpecialStep `yaml:"setup"`
	// Teardown always runs, so consent flags and sessions do not leak into the
	// next case.
	Teardown []SpecialStep `yaml:"teardown"`
	// MustContain / MustNotContain assert on the response body. Used to pin down
	// WHY something was refused: a cross-hierarchy impersonation that comes back
	// with the consent message was not stopped by the hierarchy check.
	MustContain    []string `yaml:"must_contain"`
	MustNotContain []string `yaml:"must_not_contain"`
	Note           string   `yaml:"note"`
}

// SpecialStep is a prerequisite or cleanup request run as a given persona.
type SpecialStep struct {
	As     string `yaml:"as"`
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
	Body   string `yaml:"body"`
}

// apiKeyToken lazily mints one API key per (persona, mode) and remembers it so a
// run creates at most two keys per persona. All of them are revoked at the end.
type apiKeyRef struct {
	id    string
	token string
}

func (a *authzRunner) apiKeyFor(p *persona, mode string) (*apiKeyRef, error) {
	if a.apiKeys == nil {
		a.apiKeys = map[string]*apiKeyRef{}
	}
	cacheKey := p.regKey + "|" + mode
	if ref, ok := a.apiKeys[cacheKey]; ok {
		return ref, nil
	}
	password := a.reg.Owner.Password
	if p.regKey != "owner" {
		u, ok := a.reg.Users[p.regKey]
		if !ok {
			return nil, fmt.Errorf("no credentials for persona %s", p.id)
		}
		password = u.Password
	}
	payload := map[string]interface{}{
		"name":            "authz-suite-" + mode,
		"mode":            mode,
		"expires_in_days": 1,
		"password":        password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	res, err := a.request(p.token, "POST", "/api/me/api-keys", string(body))
	if err != nil {
		return nil, err
	}
	if res.status >= 300 {
		return nil, fmt.Errorf("minting %s api key for %s failed (%d): %s", mode, p.id, res.status, truncBody(res.body))
	}
	var parsed struct {
		Data struct {
			ID    string `json:"id"`
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(res.body), &parsed); err != nil {
		return nil, err
	}
	if parsed.Data.Token == "" {
		return nil, fmt.Errorf("no token in api key response: %s", truncBody(res.body))
	}
	ref := &apiKeyRef{id: parsed.Data.ID, token: parsed.Data.Token}
	a.apiKeys[cacheKey] = ref
	return ref, nil
}

// revokeAPIKeys cleans up every key the run minted. Called even on failure so a
// crashed run does not leave live credentials behind.
func (a *authzRunner) revokeAPIKeys() {
	for cacheKey, ref := range a.apiKeys {
		regKey := strings.SplitN(cacheKey, "|", 2)[0]
		p, ok := a.personaByRegKey(regKey)
		if !ok || ref.id == "" {
			continue
		}
		if _, err := a.request(p.token, "DELETE", "/api/me/api-keys/"+ref.id, ""); err != nil {
			fmt.Printf("  WARNING: could not revoke api key %s of %s: %v\n", ref.id, p.id, err)
		}
	}
	a.apiKeys = nil
}

func (a *authzRunner) personaByRegKey(regKey string) (*persona, bool) {
	for _, p := range a.personas {
		if p.regKey == regKey {
			return p, true
		}
	}
	return nil, false
}

func (a *authzRunner) runSpecialLayer(filter string) error {
	defer a.revokeAPIKeys()

	count := 0
	for _, sc := range a.spec.Special {
		if filter != "" && !strings.Contains(strings.ToLower(sc.Name), strings.ToLower(filter)) {
			continue
		}
		p, ok := a.byKey[sc.As]
		if !ok {
			return fmt.Errorf("special case %q: unknown persona %q", sc.Name, sc.As)
		}
		path, err := a.resolve(sc.Path, p)
		if err != nil {
			return fmt.Errorf("special case %q: %w", sc.Name, err)
		}
		body := sc.Body
		if body != "" {
			if body, err = a.resolve(body, p); err != nil {
				return fmt.Errorf("special case %q body: %w", sc.Name, err)
			}
		} else if sc.Method == "POST" || sc.Method == "PUT" || sc.Method == "PATCH" {
			body = "{}"
		}

		// Preconditions first: without them the case may pass for the wrong
		// reason.
		if setupErr := a.runSteps(sc.Setup); setupErr != nil {
			a.record(checkResult{Layer: "special", Persona: p.id, Target: sc.Name,
				Expected: sc.Expect, Got: "setup failed", Verdict: vReview, Detail: setupErr.Error()})
			_ = a.runSteps(sc.Teardown)
			continue
		}

		token := p.token
		switch sc.Kind {
		case "self_modification", "impersonate":
			// caller's own session token
		case "api_key":
			mode := sc.Mode
			if mode == "" {
				mode = "read"
			}
			ref, err := a.apiKeyFor(p, mode)
			if err != nil {
				a.record(checkResult{Layer: "special", Persona: p.id, Target: sc.Name,
					Expected: sc.Expect, Got: "setup failed", Verdict: vReview, Detail: err.Error()})
				continue
			}
			token = ref.token
		default:
			return fmt.Errorf("special case %q: unknown kind %q", sc.Name, sc.Kind)
		}

		res, err := a.request(token, sc.Method, path, body)
		if err != nil {
			_ = a.runSteps(sc.Teardown)
			return fmt.Errorf("special case %q: %w", sc.Name, err)
		}
		count++
		a.record(a.specialVerdict(sc, p, res))

		// An impersonation that succeeded when it should not have leaves a live
		// session behind, and the original session cannot close it (DELETE
		// /impersonate answers "not currently impersonating" on the caller's own
		// token). Left alone it blocks every later attempt by that persona with a
		// 409, so the very finding would hide the ones after it. Close it here
		// with the token the escalation just handed us.
		if sc.Kind == "impersonate" && res.status < 300 {
			a.exitImpersonation(p, res.body)
		}

		if err := a.runSteps(sc.Teardown); err != nil {
			fmt.Printf("  WARNING: teardown of %q failed: %v\n", sc.Name, err)
		}
	}
	fmt.Printf("special layer: %d case(s)\n", count)
	return nil
}

// exitImpersonation closes a session the suite just opened, using the token the
// impersonation response returned.
func (a *authzRunner) exitImpersonation(p *persona, responseBody string) {
	var parsed struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(responseBody), &parsed); err != nil || parsed.Data.Token == "" {
		fmt.Printf("  WARNING: %s opened an impersonation session but no token came back to close it; "+
			"clear impersonation_session:* in Redis before the next run\n", p.id)
		return
	}
	if _, err := a.request(parsed.Data.Token, "DELETE", "/api/impersonate", ""); err != nil {
		fmt.Printf("  WARNING: could not close the impersonation session of %s: %v\n", p.id, err)
	}
}

// runSteps executes setup/teardown requests as their own personas.
func (a *authzRunner) runSteps(steps []SpecialStep) error {
	for i, st := range steps {
		p, ok := a.byKey[st.As]
		if !ok {
			return fmt.Errorf("step %d: unknown persona %q", i+1, st.As)
		}
		path, err := a.resolve(st.Path, p)
		if err != nil {
			return fmt.Errorf("step %d: %w", i+1, err)
		}
		body := st.Body
		if body != "" {
			if body, err = a.resolve(body, p); err != nil {
				return fmt.Errorf("step %d body: %w", i+1, err)
			}
		}
		res, err := a.request(p.token, st.Method, path, body)
		if err != nil {
			return fmt.Errorf("step %d (%s %s as %s): %w", i+1, st.Method, path, p.id, err)
		}
		if res.status >= 300 {
			return fmt.Errorf("step %d (%s %s as %s) returned %d: %s",
				i+1, st.Method, path, p.id, res.status, res.message)
		}
	}
	return nil
}

func (a *authzRunner) specialVerdict(sc SpecialSpec, p *persona, res *probeResult) checkResult {
	out := specialStatusVerdict(sc, p, res)

	// Content assertions pin down the REASON, which matters when several checks
	// can refuse the same call: a cross-hierarchy impersonation refused with the
	// consent message means the hierarchy check never ran.
	for _, needle := range sc.MustNotContain {
		resolved, err := a.resolve(needle, p)
		if err != nil {
			out.Verdict = vReview
			out.Detail = err.Error()
			continue
		}
		if containsToken(res.body, resolved) || strings.Contains(strings.ToLower(res.body), strings.ToLower(resolved)) {
			out.Verdict = vReview
			out.Expected = strings.TrimSpace(out.Expected + " + response must not mention " + resolved)
			out.Detail = "refused for the wrong reason: " + truncBody(res.body)
		}
	}
	for _, needle := range sc.MustContain {
		resolved, err := a.resolve(needle, p)
		if err != nil {
			out.Verdict = vReview
			out.Detail = err.Error()
			continue
		}
		if !strings.Contains(strings.ToLower(res.body), strings.ToLower(resolved)) {
			if out.Verdict == vPass {
				out.Verdict = vReview
			}
			out.Expected = strings.TrimSpace(out.Expected + " + response must mention " + resolved)
			out.Detail = "unexpected reason: " + truncBody(res.body)
		}
	}
	return out
}

func specialStatusVerdict(sc SpecialSpec, p *persona, res *probeResult) checkResult {
	out := checkResult{
		Layer: "special", Persona: p.id, Target: sc.Name,
		Got: res.class, Status: res.status,
	}
	if len(sc.ExpectStatus) > 0 {
		out.Expected = fmt.Sprintf("status %v", sc.ExpectStatus)
		if containsInt(sc.ExpectStatus, res.status) {
			out.Verdict = vPass
		} else {
			out.Verdict = vFailOpen
			out.Detail = fmt.Sprintf("got %d %s", res.status, res.message)
		}
		return out
	}
	switch sc.Expect {
	case "denied":
		out.Expected = "denied"
		switch res.class {
		case outGateDenied, outHandlerDenied, outUnauth, outNotFound:
			out.Verdict = vPass
			out.Detail = res.message
		case outAllowed:
			out.Verdict = vFailOpen
			out.Detail = "allowed: " + truncBody(res.body)
		case outBadRequest:
			// The request never reached an authorization decision, so this case
			// proves nothing — the body needs fixing in special.yml.
			out.Verdict = vReview
			out.Detail = "400 before any authorization decision: " + res.message
		default:
			out.Verdict = vReview
			out.Detail = fmt.Sprintf("%d %s", res.status, res.message)
		}
	case "allowed":
		out.Expected = "allowed"
		switch res.class {
		case outAllowed, outNotFound, outBadRequest:
			out.Verdict = vPass
		case outGateDenied, outHandlerDenied, outUnauth:
			out.Verdict = vFailClosed
			out.Detail = res.message
		default:
			out.Verdict = vReview
			out.Detail = fmt.Sprintf("%d %s", res.status, res.message)
		}
	default:
		out.Verdict = vReview
		out.Detail = "unknown expect value " + sc.Expect
	}
	return out
}
