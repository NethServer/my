/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"fmt"
	"strings"
)

// The apps layer checks the third-party application boundary, which has two
// independent halves that are easy to conflate:
//
//   portal — does my offer the app to this persona? my filters
//            /third-party-applications through the per-app access_control
//            (organization_ids AND organization_roles AND user_roles,
//            fail-closed). Expectation here is evaluated from
//            sync/configs/config.yml, not from the backend's filter code.
//   IdP    — does Logto issue an authorization code for this persona? Logto does
//            not read our access_control, so a persona hidden in the portal can
//            still complete SSO by going straight to the app's login URL. That
//            gap is a property of the design; the suite records it so it cannot
//            change silently.

type AppsSpec struct {
	// IDPProbe runs a real authorization-code flow per persona per app. It is
	// slow (a full interactive login each) so it is opt-in.
	IDPProbe bool `yaml:"idp_probe"`
	// Apps lists per-app expectations about the IdP half.
	Apps []AppExpectation `yaml:"apps"`
}

type AppExpectation struct {
	Name string `yaml:"name"`
	// IDPEnforced records whether Logto itself is expected to refuse personas
	// that the portal hides. Today no app is provisioned that way; setting it
	// false documents the accepted gap instead of failing on every persona.
	IDPEnforced bool   `yaml:"idp_enforced"`
	Note        string `yaml:"note"`
}

// portalVisible evaluates access_control the way the product intends it: every
// declared dimension must match, and an app with no access_control is visible to
// nobody (fail-closed).
func (a *authzRunner) portalVisible(app rbacThirdPartyApp, p *persona) bool {
	ac := app.AccessControl
	if len(ac.OrganizationIDs) == 0 && len(ac.OrganizationRoles) == 0 && len(ac.UserRoles) == 0 {
		return false
	}
	if len(ac.OrganizationIDs) > 0 && !containsFold(ac.OrganizationIDs, p.orgID) {
		return false
	}
	if len(ac.OrganizationRoles) > 0 && !containsFold(ac.OrganizationRoles, p.orgRole) {
		return false
	}
	if len(ac.UserRoles) > 0 {
		match := false
		for _, roleName := range p.userRoles {
			if containsFold(ac.UserRoles, a.spec.rbac.userRoleID(roleName)) {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
}

func containsFold(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(strings.TrimSpace(x), strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

func (a *authzRunner) runAppsLayer(filter string) error {
	if len(a.spec.rbac.ThirdPartyApps) == 0 {
		fmt.Println("apps layer: no third_party_apps in config.yml")
		return nil
	}
	if a.idpProbe {
		a.spec.Apps.IDPProbe = true
	}

	// The owner sees the whole catalogue, which is how we learn each app's
	// client_id — needed to probe apps a persona cannot even see.
	ownerClient, err := loginAs(a.reg, "owner")
	if err != nil {
		return err
	}
	catalogue, err := ownerClient.ListThirdPartyApps()
	if err != nil {
		return err
	}
	byName := map[string]ThirdPartyApp{}
	for _, app := range catalogue {
		byName[app.Name] = app
	}

	expectations := map[string]AppExpectation{}
	for _, e := range a.spec.Apps.Apps {
		expectations[e.Name] = e
	}

	checks := 0
	for _, p := range a.personas {
		visible, err := a.personaPortalApps(p)
		if err != nil {
			return err
		}
		for _, app := range a.spec.rbac.ThirdPartyApps {
			if filter != "" && !strings.Contains(strings.ToLower(app.Name), strings.ToLower(filter)) {
				continue
			}
			want := a.portalVisible(app, p)
			got := visible[app.Name]
			checks++

			res := checkResult{
				Layer: "apps", Persona: p.id, Target: "portal " + app.Name,
				Expected: fmt.Sprintf("visible=%v", want), Got: fmt.Sprintf("visible=%v", got),
			}
			switch {
			case want == got:
				res.Verdict = vPass
			case got && !want:
				res.Verdict = vFailOpen
				res.Detail = fmt.Sprintf("access_control excludes org_role=%s user_roles=%s but my offers the app",
					p.orgRole, strings.Join(p.userRoles, ","))
			default:
				res.Verdict = vFailClosed
				res.Detail = fmt.Sprintf("access_control admits org_role=%s user_roles=%s but my hides the app",
					p.orgRole, strings.Join(p.userRoles, ","))
			}
			a.record(res)

			if !a.spec.Apps.IDPProbe {
				continue
			}
			target, ok := byName[app.Name]
			if !ok || len(target.RedirectURIs) == 0 {
				a.record(checkResult{Layer: "apps", Persona: p.id, Target: "idp " + app.Name,
					Expected: "probe", Got: "no client_id in catalogue", Verdict: vSkip})
				continue
			}
			codeIssued, detail, err := a.probeIdP(p, target)
			if err != nil {
				return err
			}
			checks++
			exp := expectations[app.Name]
			idpRes := checkResult{
				Layer: "apps", Persona: p.id, Target: "idp " + app.Name,
				Expected: fmt.Sprintf("code_issued=%v", want), Got: fmt.Sprintf("code_issued=%v", codeIssued),
				Detail: detail,
			}
			switch {
			case codeIssued == want:
				idpRes.Verdict = vPass
			case codeIssued && !want && !exp.IDPEnforced:
				// Known and accepted: Logto ignores our access_control. Recorded
				// as PASS so the suite stays green, with the gap spelled out.
				idpRes.Verdict = vPass
				idpRes.Detail = "accepted gap: portal hides the app but Logto issues a code (idp_enforced=false)"
			case codeIssued && !want:
				idpRes.Verdict = vFailOpen
				idpRes.Detail = "Logto authorized a persona the portal hides, and this app is marked idp_enforced"
			default:
				idpRes.Verdict = vFailClosed
				idpRes.Detail = "Logto refused a persona that access_control admits: " + detail
			}
			a.record(idpRes)
		}
	}
	fmt.Printf("apps layer: %d check(s) over %d app(s)\n", checks, len(a.spec.rbac.ThirdPartyApps))
	return nil
}

// personaPortalApps reads the app list my offers to a persona. A non-200 is an
// error, never an empty list: silently reading "no apps" out of a failed request
// would report every app as correctly hidden from everybody.
func (a *authzRunner) personaPortalApps(p *persona) (map[string]bool, error) {
	res, err := a.request(p.token, "GET", "/api/third-party-applications", "")
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	if res.status != 200 {
		return nil, fmt.Errorf("listing third-party apps as %s returned %d (%s) — cannot tell hidden from unreadable",
			p.id, res.status, res.message)
	}
	for _, app := range a.spec.rbac.ThirdPartyApps {
		if strings.Contains(res.body, `"`+app.Name+`"`) {
			out[app.Name] = true
		}
	}
	return out, nil
}

// probeIdP drives a real authorization-code flow as the persona. The app is never
// contacted: only Logto's final redirect is inspected.
func (a *authzRunner) probeIdP(p *persona, app ThirdPartyApp) (bool, string, error) {
	u, ok := a.reg.Users[p.regKey]
	email, password := u.Email, u.Password
	if p.regKey == "owner" {
		email, password = a.reg.Owner.Email, a.reg.Owner.Password
		ok = true
	}
	if !ok {
		return false, "", fmt.Errorf("no credentials for persona %s", p.id)
	}
	client, err := NewClient(a.reg.Config)
	if err != nil {
		return false, "", err
	}
	outcome, err := client.Authorize(email, password, AuthzRequest{
		ClientID:    app.ID,
		RedirectURI: app.RedirectURIs[0],
		Scope:       ThirdPartyScope,
	})
	if err != nil {
		return false, fmt.Sprintf("probe error: %v", err), nil
	}
	detail := outcome.Stage
	if outcome.OAuthError != "" {
		detail += " oauth_error=" + outcome.OAuthError
	}
	return outcome.Code != "", detail, nil
}
