/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"fmt"
	"regexp"
	"strings"
)

// The scope layer is where cross-organization isolation is proved. The gate layer
// only shows whether a persona may call an endpoint at all; here we point real
// personas at real objects that belong to somebody else and assert refusal, and
// we read list endpoints to assert that out-of-scope objects are absent.
//
// Unlike the gate layer these scenarios are hand-written one by one, because
// "who may see whose data" cannot be derived from a permission name.

// ScenarioSpec is one hand-written object-level expectation.
type ScenarioSpec struct {
	Name string `yaml:"name"`
	// As is the fixture persona key making the call.
	As     string `yaml:"as"`
	Method string `yaml:"method"`
	// Path, Body and the assertion lists may contain placeholders resolved
	// against the fixture:
	//   {org:KEY}          Logto id of a fixture organization
	//   {org_name:KEY}     its name, for searching a response body
	//   {user:KEY}         Logto id of a fixture user
	//   {user_email:KEY}   its email, for searching a response body
	//   {system:KEY}       internal UUID of a fixture system
	//   {system_name:KEY}  its name, for searching a response body
	//   {my_org}           the caller's own organization id
	//   {my_user}          the caller's own user id
	//   {bogus}            an id that matches nothing
	Path string `yaml:"path"`
	Body string `yaml:"body"`
	// Expect is denied|allowed, or use ExpectStatus for an exact set.
	//   denied  — 403 or 404 (hiding existence is a legitimate refusal)
	//   allowed — 2xx
	Expect       string `yaml:"expect"`
	ExpectStatus []int  `yaml:"expect_status"`
	// MustNotContain flags data leaking through a list/aggregate endpoint. Each
	// entry is a placeholder or literal that must be absent from the response.
	MustNotContain []string `yaml:"must_not_contain"`
	MustContain    []string `yaml:"must_contain"`
	Note           string   `yaml:"note"`
}

var placeholderRe = regexp.MustCompile(`\{(\w+)(?::([\w -]+))?\}`)

// roleIDByName resolves a technical role name ("Super Admin") to the id of the
// current Logto tenant, reading the catalogue as the owner — the only caller that
// sees every role. Cached for the run.
func (a *authzRunner) roleIDByName(name string) (string, error) {
	if a.roleIDs == nil {
		client, err := loginAs(a.reg, "owner")
		if err != nil {
			return "", err
		}
		roles, err := client.GetRoles()
		if err != nil {
			return "", err
		}
		a.roleIDs = roles
	}
	id, ok := a.roleIDs[name]
	if !ok {
		return "", fmt.Errorf("role %q not found in the tenant catalogue", name)
	}
	return id, nil
}

// resolve substitutes fixture placeholders in a path or an assertion string.
func (a *authzRunner) resolve(s string, p *persona) (string, error) {
	var bad []string
	out := placeholderRe.ReplaceAllStringFunc(s, func(m string) string {
		parts := placeholderRe.FindStringSubmatch(m)
		kind, key := parts[1], parts[2]
		prefix := a.spec.Fixture.Prefix
		switch kind {
		case "my_org":
			return p.orgID
		case "my_user":
			if u, ok := a.reg.Users[p.regKey]; ok {
				return u.LogtoID
			}
			return p.regKey
		case "org":
			if id, ok := a.orgLogtoID[key]; ok {
				return id
			}
		case "org_name":
			if n, ok := a.orgName[key]; ok {
				return n
			}
		case "user":
			if u, ok := a.reg.Users[fixtureRegKey(prefix, key)]; ok {
				return u.LogtoID
			}
		case "user_email":
			if u, ok := a.reg.Users[fixtureRegKey(prefix, key)]; ok {
				return u.Email
			}
		case "system":
			if s, ok := a.reg.Systems[fixtureRegKey(prefix, key)]; ok {
				return s.ID
			}
		case "system_name":
			if s, ok := a.reg.Systems[fixtureRegKey(prefix, key)]; ok {
				return s.Name
			}
		case "bogus":
			return bogusID
		case "role":
			// Technical role ids are minted per Logto tenant, so they cannot be
			// written into a spec. Resolved by name through the owner.
			if id, err := a.roleIDByName(key); err == nil {
				return id
			}
		}
		bad = append(bad, m)
		return m
	})
	if len(bad) > 0 {
		return out, fmt.Errorf("unresolved placeholder(s) %s — is the fixture provisioned?", strings.Join(bad, " "))
	}
	return out, nil
}

// containsToken looks for needle as a whole token rather than as a substring.
// Plain substring matching is wrong here: the fixture names nest by design
// ("authz-d1r1" is a prefix of "authz-d1r1c1"), so a customer's own name in the
// response would read as its parent reseller leaking. Requiring a non-identifier
// character on both sides works across JSON and CSV alike.
func containsToken(body, needle string) bool {
	if needle == "" {
		return false
	}
	re, err := regexp.Compile(`(^|[^A-Za-z0-9_-])` + regexp.QuoteMeta(needle) + `([^A-Za-z0-9_-]|$)`)
	if err != nil {
		return strings.Contains(body, needle)
	}
	return re.MatchString(body)
}

func (a *authzRunner) runScopeLayer(filter string) error {
	count := 0
	for _, sc := range a.spec.Scenarios {
		if filter != "" && !strings.Contains(strings.ToLower(sc.Name), strings.ToLower(filter)) {
			continue
		}
		p, ok := a.byKey[sc.As]
		if !ok {
			return fmt.Errorf("scenario %q: unknown persona %q", sc.Name, sc.As)
		}
		path, err := a.resolve(sc.Path, p)
		if err != nil {
			return fmt.Errorf("scenario %q: %w", sc.Name, err)
		}
		body := sc.Body
		if body == "" && (sc.Method == "POST" || sc.Method == "PUT" || sc.Method == "PATCH") {
			body = "{}"
		}
		if body != "" {
			if body, err = a.resolve(body, p); err != nil {
				return fmt.Errorf("scenario %q body: %w", sc.Name, err)
			}
		}
		res, err := a.request(p.token, sc.Method, path, body)
		if err != nil {
			return fmt.Errorf("scenario %q: %w", sc.Name, err)
		}
		count++
		a.record(a.scopeVerdict(sc, p, res))
	}
	fmt.Printf("scope layer: %d scenario(s)\n", count)
	return nil
}

func (a *authzRunner) scopeVerdict(sc ScenarioSpec, p *persona, res *probeResult) checkResult {
	out := checkResult{
		Layer: "scope", Persona: p.id, Target: sc.Name,
		Got: res.class, Status: res.status,
	}

	if len(sc.ExpectStatus) > 0 {
		out.Expected = fmt.Sprintf("status %v", sc.ExpectStatus)
		if containsInt(sc.ExpectStatus, res.status) {
			out.Verdict = vPass
		} else {
			out.Verdict = vFailOpen
			out.Detail = fmt.Sprintf("got %d (%s)", res.status, res.message)
		}
	} else {
		switch sc.Expect {
		case "denied":
			out.Expected = "denied"
			switch res.class {
			case outGateDenied, outHandlerDenied, outNotFound:
				out.Verdict = vPass
			case outAllowed:
				out.Verdict = vFailOpen
				out.Detail = "returned data: " + truncBody(res.body)
			case outServerError:
				out.Verdict = vReview
				out.Detail = "5xx: " + truncBody(res.body)
			default:
				out.Verdict = vReview
				out.Detail = fmt.Sprintf("%d %s", res.status, res.message)
			}
		case "allowed":
			out.Expected = "allowed"
			switch res.class {
			case outAllowed:
				out.Verdict = vPass
			case outGateDenied, outHandlerDenied:
				out.Verdict = vFailClosed
				out.Detail = res.message
			case outNotFound:
				out.Verdict = vFailClosed
				out.Detail = "not found, but this object is in the caller's scope"
			default:
				out.Verdict = vReview
				out.Detail = fmt.Sprintf("%d %s", res.status, res.message)
			}
		case "":
			// Only content assertions.
			out.Expected = "content assertions"
			out.Verdict = vPass
			if res.class != outAllowed {
				out.Verdict = vReview
				out.Detail = fmt.Sprintf("cannot check content: %d %s", res.status, res.message)
			}
		default:
			out.Verdict = vReview
			out.Detail = "unknown expect value " + sc.Expect
		}
	}

	// Content assertions run whenever a body came back, so a scenario can both
	// allow the call and assert that somebody else's rows are not in it.
	for _, needle := range sc.MustNotContain {
		resolved, err := a.resolve(needle, p)
		if err != nil {
			out.Verdict = vReview
			out.Detail = err.Error()
			continue
		}
		if containsToken(res.body, resolved) {
			out.Verdict = vFailOpen
			out.Expected = strings.TrimSpace(out.Expected + " + must not contain " + resolved)
			out.Detail = "response leaks out-of-scope object " + resolved
		}
	}
	for _, needle := range sc.MustContain {
		resolved, err := a.resolve(needle, p)
		if err != nil {
			out.Verdict = vReview
			out.Detail = err.Error()
			continue
		}
		if !containsToken(res.body, resolved) {
			if out.Verdict == vPass {
				out.Verdict = vFailClosed
			}
			out.Expected = strings.TrimSpace(out.Expected + " + must contain " + resolved)
			out.Detail = "response is missing in-scope object " + resolved
		}
	}
	return out
}
