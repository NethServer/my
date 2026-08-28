/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// The gate layer answers one question for every endpoint and every persona:
// does the route-level authorization gate let this persona in?
//
// It is non-destructive by construction. Path parameters are filled with an id
// that matches nothing, and write verbs get a body that cannot validate, so an
// authorized call lands on 404/400 *after* the gate ran. Nothing real is ever
// modified, which is what makes the matrix safe to re-run on every change.
//
// Interpretation of the observed status:
//   403 "insufficient permissions" → the RBAC middleware refused        (gate)
//   403 anything else              → a handler refused                  (deeper)
//   404 / 400                      → the gate passed, the object/body did not
//   2xx                            → the gate passed
// A persona that should be blocked but gets 404 means the endpoint has no
// route-level gate for it: the request reached the handler. Whether data
// actually leaks is then a question for the scope layer.

var pathParamRe = regexp.MustCompile(`:[a-zA-Z_]+`)

// gateProbe is one concrete request built from a RouteSpec.
type gateProbe struct {
	label string
	path  string
	body  string
	// ownOrg marks the probe that targets the caller's own organization, which
	// is the case the "may always read its own org" intent is about.
	ownOrg bool
}

func (a *authzRunner) buildProbes(rt RouteSpec, p *persona) []gateProbe {
	// Placeholders such as {my_org} let a probe carry the caller's own
	// organization, so it reaches the permission decision instead of stopping on
	// an object-level hierarchy check with somebody else's id.
	subst := func(s string) string {
		if !strings.Contains(s, "{") {
			return s
		}
		out, err := a.resolve(s, p)
		if err != nil {
			return s
		}
		return out
	}
	fill := func(param string) string {
		name := strings.TrimPrefix(param, ":")
		if v, ok := rt.IDs[name]; ok {
			return subst(v)
		}
		return bogusID
	}
	base := pathParamRe.ReplaceAllStringFunc(rt.Path, fill)
	if rt.Query != "" {
		base += "?" + subst(rt.Query)
	}
	body := subst(rt.Body)
	if body == "" && (rt.Method == "POST" || rt.Method == "PUT" || rt.Method == "PATCH") {
		// An empty object passes JSON binding and fails field validation, so an
		// authorized caller gets 400 without creating anything.
		body = "{}"
	}
	probes := []gateProbe{{label: "bogus-id", path: base, body: body}}

	// For org-scoped GETs the caller's own organization is a distinct and
	// meaningful case: intent says a user may always read its own org.
	if rt.SelfOrgGet && rt.Method == "GET" && strings.Contains(rt.Path, ":") && p.orgID != "" {
		own := pathParamRe.ReplaceAllStringFunc(rt.Path, func(param string) string {
			name := strings.TrimPrefix(param, ":")
			if v, ok := rt.IDs[name]; ok {
				return subst(v)
			}
			if name == "id" || name == "org_id" {
				return p.orgID
			}
			return bogusID
		})
		if rt.Query != "" {
			own += "?" + subst(rt.Query)
		}
		probes = append(probes, gateProbe{label: "own-org", path: own, body: body, ownOrg: true})
	}
	return probes
}

// expectAllowed decides, from the hand-authored intent alone, whether this
// persona is supposed to get past the gate.
func expectAllowed(rt RouteSpec, p *persona, probe gateProbe) bool {
	if rt.OrgRole != "" && !strings.EqualFold(p.orgRole, rt.OrgRole) && !p.hasUserRole(rt.OrUserRole) {
		return false
	}
	switch rt.Intent {
	case "public", "authenticated":
		return true
	case "owner_only":
		return strings.EqualFold(p.orgRole, "owner")
	}
	if probe.ownOrg && rt.SelfOrgGet {
		return true
	}
	if rt.Intent != "" && p.has(rt.Intent) {
		return true
	}
	return p.hasAny(rt.IntentAny)
}

func intentLabel(rt RouteSpec) string {
	if rt.OrgRole != "" {
		who := rt.OrgRole + " org"
		if rt.OrUserRole != "" {
			who += "/" + rt.OrUserRole
		}
		return who + " + " + intentPermissionLabel(rt)
	}
	return intentPermissionLabel(rt)
}

func intentPermissionLabel(rt RouteSpec) string {
	switch {
	case rt.Intent != "" && len(rt.IntentAny) > 0:
		return rt.Intent + "|" + strings.Join(rt.IntentAny, "|")
	case rt.Intent != "":
		return rt.Intent
	default:
		return strings.Join(rt.IntentAny, "|")
	}
}

func (a *authzRunner) runGateLayer(filter string) error {
	routes := a.spec.Routes
	total := 0
	for _, rt := range routes {
		if filter != "" && !strings.Contains(strings.ToLower(rt.id()), strings.ToLower(filter)) {
			continue
		}
		if rt.Skip != "" {
			a.record(checkResult{Layer: "gate", Persona: "-", Target: rt.id(),
				Expected: "-", Got: "-", Verdict: vSkip, Detail: rt.Skip})
			continue
		}

		// No token at all: everything that is not declared public must 401.
		if err := a.checkAnonymous(rt); err != nil {
			return err
		}

		for _, p := range a.personas {
			if !p.matrix {
				continue
			}
			for _, probe := range a.buildProbes(rt, p) {
				res, err := a.request(p.token, rt.Method, probe.path, probe.body)
				if err != nil {
					return fmt.Errorf("%s as %s: %w", rt.id(), p.id, err)
				}
				total++
				a.record(gateVerdict(rt, p, probe, res))
			}
		}
	}
	fmt.Printf("gate layer: %d request(s) over %d route(s)\n", total, len(routes))
	return nil
}

// checkAnonymous asserts the authentication boundary itself: an unauthenticated
// request must be rejected on every route that is not declared public.
func (a *authzRunner) checkAnonymous(rt RouteSpec) error {
	probe := a.buildProbes(rt, &persona{})[0]
	res, err := a.request("", rt.Method, probe.path, probe.body)
	if err != nil {
		return err
	}
	target := rt.id()

	if len(rt.ExpectAnonymous) > 0 {
		verdict := vPass
		if !containsInt(rt.ExpectAnonymous, res.status) {
			verdict = vFailOpen
		}
		a.record(checkResult{Layer: "gate", Persona: "anonymous", Target: target,
			Expected: fmt.Sprintf("status %v", rt.ExpectAnonymous), Got: res.class,
			Status: res.status, Verdict: verdict, Detail: res.message})
		return nil
	}
	if rt.Intent == "public" {
		verdict := vPass
		if res.class == outUnauth {
			verdict = vFailClosed
		}
		a.record(checkResult{Layer: "gate", Persona: "anonymous", Target: target,
			Expected: "reachable (public)", Got: res.class, Status: res.status, Verdict: verdict,
			Detail: res.message})
		return nil
	}
	verdict := vPass
	if res.class != outUnauth {
		verdict = vFailOpen
	}
	a.record(checkResult{Layer: "gate", Persona: "anonymous", Target: target,
		Expected: outUnauth, Got: res.class, Status: res.status, Verdict: verdict, Detail: res.message})
	return nil
}

func gateVerdict(rt RouteSpec, p *persona, probe gateProbe, res *probeResult) checkResult {
	allowed := expectAllowed(rt, p, probe)
	target := rt.id()
	if probe.ownOrg {
		target += " [own-org]"
	}
	out := checkResult{
		Layer: "gate", Persona: p.id, Target: target,
		Got: res.class, Status: res.status,
	}
	if res.class == outNoRoute {
		out.Expected = "a routable endpoint"
		out.Verdict = vReview
		out.Detail = "no endpoint matched this path — fix the path in routes.yml, or the route was removed"
		return out
	}
	if allowed {
		out.Expected = "allowed (" + intentLabel(rt) + ")"
		switch res.class {
		case outAllowed, outNotFound, outBadRequest, outRateLimited:
			out.Verdict = vPass
			if len(rt.ExpectAuthorized) > 0 && !containsInt(rt.ExpectAuthorized, res.status) {
				out.Verdict = vReview
				out.Detail = fmt.Sprintf("status %d not in expect_authorized %v", res.status, rt.ExpectAuthorized)
			}
		case outGateDenied:
			out.Verdict = vFailClosed
			out.Detail = "persona holds " + intentLabel(rt) + " per config.yml but the gate refused"
		case outHandlerDenied:
			if rt.ObjectScoped {
				// The endpoint is object-scoped by design: for an object that
				// does not belong to the caller, a handler refusal IS the
				// expected answer.
				out.Verdict = vPass
				out.Detail = "object-scoped refusal: " + res.message
				return out
			}
			out.Verdict = vReview
			out.Detail = "handler refused: " + res.message
		case outUnauth:
			out.Verdict = vFailClosed
			out.Detail = "token rejected: " + res.message
		default:
			out.Verdict = vReview
			out.Detail = fmt.Sprintf("unexpected %d: %s", res.status, truncBody(res.body))
		}
		return out
	}

	out.Expected = "denied (lacks " + intentLabel(rt) + ")"
	switch res.class {
	case outGateDenied:
		out.Verdict = vPass
	case outHandlerDenied:
		if rt.EnforcedBy == "handler" || rt.ObjectScoped {
			// The check lives in the handler on purpose. The denial is still
			// asserted: were it removed, this probe would return 404 and land in
			// the fail-open branch below.
			out.Verdict = vPass
			out.Detail = "denied by the handler as declared: " + res.message
			return out
		}
		out.Verdict = vReview
		out.Detail = "denied by the handler, not by a route gate: " + res.message
	case outNotFound, outBadRequest, outAllowed:
		out.Verdict = vFailOpen
		out.Detail = fmt.Sprintf("no route-level gate: request reached the handler (%d %s)", res.status, res.message)
	case outRateLimited:
		out.Verdict = vReview
		out.Detail = "throttled before the authorization answer was observable"
	case outServerError:
		out.Verdict = vReview
		out.Detail = "5xx hides the authorization answer: " + truncBody(res.body)
	default:
		out.Verdict = vReview
		out.Detail = fmt.Sprintf("unexpected %d: %s", res.status, truncBody(res.body))
	}
	return out
}

func containsInt(list []int, v int) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func truncBody(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > 160 {
		return s[:160] + "…"
	}
	return s
}

// ---------------------------------------------------------------------------
// Reporting
// ---------------------------------------------------------------------------

type layerCount struct {
	pass, failOpen, failClosed, review, skip int
}

func (a *authzRunner) summary() map[string]*layerCount {
	out := map[string]*layerCount{}
	for _, r := range a.results {
		c, ok := out[r.Layer]
		if !ok {
			c = &layerCount{}
			out[r.Layer] = c
		}
		switch r.Verdict {
		case vPass:
			c.pass++
		case vFailOpen:
			c.failOpen++
		case vFailClosed:
			c.failClosed++
		case vReview:
			c.review++
		case vSkip:
			c.skip++
		}
	}
	return out
}

// printFindings groups by target so one missing gate on one route reads as a
// single finding listing the affected personas, not as N unrelated failures.
func (a *authzRunner) printFindings(verdict, heading string) int {
	type group struct {
		detail   string
		personas []string
	}
	groups := map[string]*group{}
	var order []string
	for _, r := range a.results {
		if r.Verdict != verdict {
			continue
		}
		key := r.Layer + " | " + r.Target
		g, ok := groups[key]
		if !ok {
			g = &group{detail: r.Detail}
			groups[key] = g
			order = append(order, key)
		}
		g.personas = append(g.personas, fmt.Sprintf("%s→%d", r.Persona, r.Status))
	}
	if len(order) == 0 {
		return 0
	}
	sort.Strings(order)
	fmt.Printf("\n%s (%d)\n%s\n", heading, len(order), strings.Repeat("─", len(heading)))
	for _, key := range order {
		g := groups[key]
		fmt.Printf("\n  %s\n", key)
		if g.detail != "" {
			fmt.Printf("    %s\n", g.detail)
		}
		fmt.Printf("    personas: %s\n", strings.Join(g.personas, " "))
	}
	return len(order)
}
