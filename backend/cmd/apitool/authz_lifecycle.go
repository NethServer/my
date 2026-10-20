/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// The lifecycle layer follows objects through state changes instead of probing
// them at rest: a system created, moved, deleted and restored, an organization
// deleted and restored, a reseller promoted to distributor. At every step it
// asserts who may make the change, that a refused change left nothing behind,
// and who can see the object afterwards. Alert routing is part of what moves:
// a recipient scoped to a system or an organization must follow it into the
// right Mimir tenant and never into a foreign one, so each step also reads the
// alertmanager config back from Mimir.
//
// Unlike the other layers it creates real objects. Everything it creates is
// throwaway (prefix lc-) and destroyed at the end; the fixture organizations
// are only ever read, and fixture systems are never moved.

const lcMimirWait = 20 * time.Second

type lcRun struct {
	a        *authzRunner
	mimirURL string
	mimirOK  bool
	// created objects, destroyed in reverse order by cleanup
	systems []string
	orgs    []lcOrg
	users   []string // registry keys added in memory only
	// moved is the registered system the move steps leave in d1r1c1
	moved string
}

type lcOrg struct {
	kind string // distributors|resellers|customers, for the API path
	id   string
}

func (a *authzRunner) runLifecycleLayer(mimirURL string) error {
	lc := &lcRun{a: a, mimirURL: strings.TrimSuffix(mimirURL, "/")}
	lc.mimirOK = lc.probeMimir()
	if !lc.mimirOK {
		fmt.Printf("  Mimir not reachable at %s: routing checks are SKIP\n", lc.mimirURL)
	}
	before := len(a.results)
	defer lc.cleanup()

	steps := []func() error{
		lc.systemCreation,
		lc.systemMoves,
		lc.systemDeleteRestore,
		lc.customerDeleteRestoreAndPromotion,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	fmt.Printf("lifecycle layer: %d check(s)\n", len(a.results)-before)
	return nil
}

// ---------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------

func (lc *lcRun) systemCreation() error {
	create := func(name, as, org, expect string) error {
		res, err := lc.expect(name, as, "POST", "/api/systems",
			map[string]string{"name": "authz-lc-" + as, "organization_id": lc.org(org)}, expect)
		if err != nil {
			return err
		}
		if res.status < 300 {
			lc.systems = append(lc.systems, dataField(res.body, "id"))
		}
		return nil
	}
	cases := []struct{ name, as, org, expect string }{
		{"reseller admin creates a system in its own customer", "d1r1-admin", "d1r1c1", "allowed"},
		{"reseller support creates a system in its own customer", "d1r1-support", "d1r1c2", "allowed"},
		{"customer admin creates a system in its own organization", "d1r1c1-admin", "d1r1c1", "allowed"},
		{"distributor admin creates a system two levels down", "d1-admin", "d1r2c1", "allowed"},
		{"reseller backoffice cannot create a system", "d1r1-backoffice", "d1r1c1", "denied"},
		{"reseller reader cannot create a system", "d1r1-reader", "d1r1c1", "denied"},
		{"customer admin cannot create a system in the sibling customer", "d1r1c1-admin", "d1r1c2", "denied"},
		{"customer admin cannot create a system in its reseller", "d1r1c1-admin", "d1r1", "denied"},
		{"reseller admin cannot create a system in its distributor", "d1r1-admin", "d1", "denied"},
		{"reseller admin cannot create a system in the sibling reseller's customer", "d1r1-admin", "d1r2c1", "denied"},
		{"distributor admin cannot create a system in the other branch", "d1-admin", "d2r1c1", "denied"},
	}
	for _, c := range cases {
		if err := create(c.name, c.as, c.org, c.expect); err != nil {
			return err
		}
	}
	return nil
}

func (lc *lcRun) systemMoves() error {
	s, err := lc.createRegisteredSystem("d1r1-admin", lc.org("d1r1c1"), "authz-lc-move")
	if err != nil {
		return err
	}
	if err := lc.saveLayer("d1-admin", []map[string]interface{}{
		lc.recipient("lc-sys", nil, []string{s}),
		lc.recipient("lc-org", []string{lc.org("d1r2c1")}, nil),
	}); err != nil {
		return err
	}
	lc.routed("system-scoped recipient lands in the tenant holding the system", "d1r1", "lc-sys@", true)
	lc.routed("customer-scoped recipient lands in that customer's tenant", "d1r2", "lc-org@", true)
	lc.routed("customer-scoped recipient stays out of other tenants", "d1r1", "lc-org@", false)

	move := func(org string) map[string]string {
		return map[string]string{"name": "authz-lc-move", "organization_id": lc.org(org)}
	}
	denied := []struct{ name, as, to string }{
		{"customer admin cannot move its system to the sibling customer", "d1r1c1-admin", "d1r1c2"},
		{"customer admin cannot move its system up to its reseller", "d1r1c1-admin", "d1r1"},
		{"reseller backoffice cannot move a system", "d1r1-backoffice", "d1r1c2"},
		{"reseller reader cannot move a system", "d1r1-reader", "d1r1c2"},
		{"reseller admin cannot move a system to the sibling reseller's customer", "d1r1-admin", "d1r2c1"},
		{"reseller admin cannot move a system up to its distributor", "d1r1-admin", "d1"},
		{"reseller admin cannot move a system to the other branch", "d1r1-admin", "d2r1c1"},
		{"sibling reseller cannot move a foreign system", "d1r2-admin", "d1r2c1"},
		{"other distributor cannot move a foreign system", "d2-admin", "d2r1c1"},
	}
	for _, c := range denied {
		if _, err := lc.expect(c.name, c.as, "PUT", "/api/systems/"+s, move(c.to), "denied"); err != nil {
			return err
		}
	}
	lc.state("refused moves leave the system where it was", lc.systemOrg(s) == lc.org("d1r1c1"), vFailOpen,
		"system now in "+lc.systemOrg(s))

	steps := []struct{ name, as, to string }{
		{"reseller support moves a system between its customers", "d1r1-support", "d1r1c2"},
		{"reseller admin moves a system into its own organization", "d1r1-admin", "d1r1"},
		{"distributor admin moves a system to the other reseller's customer", "d1-admin", "d1r2c1"},
	}
	for _, c := range steps {
		if _, err := lc.expect(c.name, c.as, "PUT", "/api/systems/"+s, move(c.to), "allowed"); err != nil {
			return err
		}
	}
	lc.routed("the scoped route follows the system into the new tenant", "d1r2", "lc-sys@", true)
	lc.routed("the scoped route leaves the old tenant", "d1r1", "lc-sys@", false)
	if _, err := lc.expect("the reseller that lost the system can no longer read it", "d1r1-admin",
		"GET", "/api/systems/"+s, nil, "denied"); err != nil {
		return err
	}
	if _, err := lc.expect("distributor admin cannot move a system to the other branch", "d1-admin",
		"PUT", "/api/systems/"+s, move("d2r1c1"), "denied"); err != nil {
		return err
	}

	if _, err := lc.expect("owner moves a system to the other branch", "owner",
		"PUT", "/api/systems/"+s, move("d2r1c1"), "allowed"); err != nil {
		return err
	}
	lc.routed("the scoped route leaves d1 when the system does", "d1r2", "lc-sys@", false)
	lc.routed("d1's recipient never enters the other branch's tenant", "d2r1", "lc-sys@", false)
	if _, err := lc.expect("the distributor can no longer read the moved system", "d1-admin",
		"GET", "/api/systems/"+s, nil, "denied"); err != nil {
		return err
	}
	res, err := lc.call("d1-admin", "GET", "/api/alerts/config", nil)
	if err != nil {
		return err
	}
	lc.state("scope names no longer resolve a system that left the hierarchy",
		!strings.Contains(scopeNames(res.body), s), vFailOpen, "scope_names: "+truncBody(scopeNames(res.body)))
	res, err = lc.saveLayerRaw("d1-admin", []map[string]interface{}{
		lc.recipient("lc-sys", nil, []string{s}),
		lc.recipient("lc-org", []string{lc.org("d1r2c1")}, nil),
	})
	if err != nil {
		return err
	}
	lc.state("a layer still holding a moved-away system can be saved again", res.status == 200, vFailClosed,
		fmt.Sprintf("%d %s", res.status, res.message))
	if _, err := lc.expect("a system outside the hierarchy cannot be newly scoped", "d1-admin",
		"POST", "/api/alerts/config", layerBody([]map[string]interface{}{
			lc.recipient("lc-new", nil, []string{lc.system("s-d2r1c1")}),
			lc.recipient("lc-sys", nil, []string{s}),
		}), "out_of_scope"); err != nil {
		return err
	}

	if _, err := lc.expect("owner moves the system back", "owner",
		"PUT", "/api/systems/"+s, move("d1r1c1"), "allowed"); err != nil {
		return err
	}
	lc.routed("the scoped route comes back with the system", "d1r1", "lc-sys@", true)
	lc.moved = s
	return nil
}

func (lc *lcRun) systemDeleteRestore() error {
	s := lc.moved
	for _, c := range []struct{ name, as string }{
		{"sibling reseller cannot delete a foreign system", "d1r2-admin"},
		{"reseller backoffice cannot delete a system", "d1r1-backoffice"},
		{"customer of another reseller cannot delete a foreign system", "d1r2c1-admin"},
	} {
		if _, err := lc.expect(c.name, c.as, "DELETE", "/api/systems/"+s, nil, "denied"); err != nil {
			return err
		}
	}
	if _, err := lc.expect("reseller admin deletes a system of its customer", "d1r1-admin",
		"DELETE", "/api/systems/"+s, nil, "allowed"); err != nil {
		return err
	}
	lc.routed("a deleted system leaves the scoped routes", "d1r1", "lc-sys@", false)
	if _, err := lc.expect("sibling reseller cannot restore a foreign system", "d1r2-admin",
		"PATCH", "/api/systems/"+s+"/restore", nil, "denied"); err != nil {
		return err
	}
	if _, err := lc.expect("customer admin restores its own system", "d1r1c1-admin",
		"PATCH", "/api/systems/"+s+"/restore", nil, "allowed"); err != nil {
		return err
	}
	lc.routed("a restored system is routed again", "d1r1", "lc-sys@", true)
	for _, c := range []struct{ name, as string }{
		{"distributor admin cannot destroy a system (destroy:systems is Owner-only)", "d1-admin"},
		{"staff cannot destroy a system", "owner-staff"},
	} {
		if _, err := lc.expect(c.name, c.as, "DELETE", "/api/systems/"+s+"/destroy", nil, "denied"); err != nil {
			return err
		}
	}
	return nil
}

// customerDeleteRestoreAndPromotion works on a throwaway branch under d1:
// lc-r (reseller, with its own Admin) → lc-c (customer) → one system.
// Deleting a fixture customer would revoke its users' tokens mid-run.
func (lc *lcRun) customerDeleteRestoreAndPromotion() error {
	stamp := time.Now().UTC().Format("150405")
	lcr, err := lc.createOrg("d1-admin", "resellers", "authz-lc-r-"+stamp)
	if err != nil {
		return err
	}
	adminKey, err := lc.createAdmin(lcr, "lc-r-admin-"+stamp)
	if err != nil {
		return err
	}
	lcc, err := lc.createOrg(adminKey, "customers", "authz-lc-c-"+stamp)
	if err != nil {
		return err
	}
	ls, err := lc.createRegisteredSystem(adminKey, lcc, "authz-lc-promo")
	if err != nil {
		return err
	}

	// Customer delete / restore.
	if err := lc.saveLayer("d1-admin", []map[string]interface{}{
		lc.recipient("lc-c", nil, []string{ls}),
	}); err != nil {
		return err
	}
	lc.routed("customer: scoped route on its system", lcr, "lc-c@", true)
	for _, c := range []struct{ name, as string }{
		{"a customer cannot delete another reseller's customer", "d1r1c1-admin"},
		{"another reseller cannot delete the customer", "d1r1-admin"},
		{"a distributor reader cannot delete the customer", "d1-reader"},
	} {
		if _, err := lc.expect(c.name, c.as, "DELETE", "/api/customers/"+lcc, nil, "denied"); err != nil {
			return err
		}
	}
	if _, err := lc.expect("reseller admin deletes its customer", adminKey,
		"DELETE", "/api/customers/"+lcc, nil, "allowed"); err != nil {
		return err
	}
	if _, err := lc.expect("another reseller cannot restore the customer", "d1r1-admin",
		"PATCH", "/api/customers/"+lcc+"/restore", nil, "denied"); err != nil {
		return err
	}
	// Re-render while the customer is deleted, so only the restore hook can
	// bring the route back.
	if err := lc.saveLayer("d1-admin", []map[string]interface{}{
		lc.recipient("lc-c", nil, []string{ls}),
	}); err != nil {
		return err
	}
	lc.routed("customer: no route while deleted", lcr, "lc-c@", false)
	if _, err := lc.expect("reseller admin restores its customer", adminKey,
		"PATCH", "/api/customers/"+lcc+"/restore", nil, "allowed"); err != nil {
		return err
	}
	lc.routed("customer: the restore re-renders the tenant", lcr, "lc-c@", true)

	// Promotion.
	if err := lc.saveLayer("d1-admin", []map[string]interface{}{
		lc.recipient("lc-promo-sys", nil, []string{ls}),
		lc.recipient("lc-promo-org", []string{lcr}, nil),
	}); err != nil {
		return err
	}
	lc.routed("promotion: before, the system-scoped route is in the reseller tenant", lcr, "lc-promo-sys@", true)
	lc.routed("promotion: before, the org-scoped route is in the reseller tenant", lcr, "lc-promo-org@", true)
	for _, c := range []struct{ name, as string }{
		{"a distributor cannot promote its reseller", "d1-admin"},
		{"a reseller cannot promote itself", adminKey},
	} {
		if _, err := lc.expect(c.name, c.as, "PATCH", "/api/resellers/"+lcr+"/promote", nil, "denied"); err != nil {
			return err
		}
	}
	if _, err := lc.expect("staff promotes a reseller", "owner-staff",
		"PATCH", "/api/resellers/"+lcr+"/promote", nil, "allowed"); err != nil {
		return err
	}
	for i := range lc.orgs {
		if lc.orgs[i].id == lcr {
			lc.orgs[i].kind = "distributors"
		}
	}
	if err := lc.relogin(adminKey); err != nil {
		return err
	}
	for _, c := range []struct{ name, path string }{
		{"the former parent loses the promoted organization", "/api/distributors/" + lcr},
		{"the former parent loses it as a reseller too", "/api/resellers/" + lcr},
		{"the former parent loses the customer below it", "/api/customers/" + lcc},
		{"the former parent loses the system below it", "/api/systems/" + ls},
	} {
		if _, err := lc.expect(c.name, "d1-admin", "GET", c.path, nil, "denied"); err != nil {
			return err
		}
	}
	for _, c := range []struct{ name, path, expect string }{
		{"the promoted admin still reads its customer", "/api/customers/" + lcc, "allowed"},
		{"the promoted admin still reads its system", "/api/systems/" + ls, "allowed"},
		{"the promoted admin cannot read its former distributor", "/api/distributors/" + lc.org("d1"), "denied"},
	} {
		if _, err := lc.expect(c.name, adminKey, "GET", c.path, nil, c.expect); err != nil {
			return err
		}
	}
	lc.routed("promotion: the former parent's system-scoped route is gone", lcr, "lc-promo-sys@", false)
	lc.routed("promotion: the former parent's org-scoped route is gone", lcr, "lc-promo-org@", false)
	res, err := lc.call("d1-admin", "GET", "/api/alerts/config", nil)
	if err != nil {
		return err
	}
	names := scopeNames(res.body)
	lc.state("promotion: the former parent's scope names hide the promoted branch",
		!strings.Contains(names, lcr) && !strings.Contains(names, ls), vFailOpen, "scope_names: "+truncBody(names))
	if _, err := lc.expect("promotion: the former parent cannot newly scope the lost customer", "d1-admin",
		"POST", "/api/alerts/config", layerBody([]map[string]interface{}{
			lc.recipient("lc-x", []string{lcc}, nil),
		}), "out_of_scope"); err != nil {
		return err
	}
	if _, err := lc.expect("promotion: the promoted distributor cannot scope its former parent's customers", adminKey,
		"POST", "/api/alerts/config", layerBody([]map[string]interface{}{
			lc.recipient("lc-r", []string{lc.org("d1r1c1")}, nil),
		}), "out_of_scope"); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Checks
// ---------------------------------------------------------------------------

func (lc *lcRun) persona(key string) (*persona, error) {
	p, ok := lc.a.byKey[key]
	if !ok {
		return nil, fmt.Errorf("lifecycle: unknown persona %q", key)
	}
	return p, nil
}

func (lc *lcRun) call(as, method, path string, body interface{}) (*probeResult, error) {
	p, err := lc.persona(as)
	if err != nil {
		return nil, err
	}
	raw := ""
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		raw = string(b)
	}
	return lc.a.request(p.token, method, path, raw)
}

// expect fires one call and records its verdict. expect is allowed, denied, or
// out_of_scope (a 400 naming the scope violation).
func (lc *lcRun) expect(name, as, method, path string, body interface{}, expect string) (*probeResult, error) {
	res, err := lc.call(as, method, path, body)
	if err != nil {
		return nil, fmt.Errorf("lifecycle %q: %w", name, err)
	}
	p, _ := lc.persona(as)
	out := checkResult{Layer: "lifecycle", Persona: p.id, Target: name, Expected: expect,
		Got: res.class, Status: res.status}
	switch expect {
	case "allowed":
		switch {
		case res.status >= 200 && res.status < 300:
			out.Verdict = vPass
		case res.status == 403 || res.status == 404 || res.status == 401:
			out.Verdict = vFailClosed
		default:
			out.Verdict = vReview
		}
	case "denied":
		switch {
		case res.status == 403 || res.status == 404:
			out.Verdict = vPass
		case res.status >= 200 && res.status < 300:
			out.Verdict = vFailOpen
		default:
			out.Verdict = vReview
		}
	case "out_of_scope":
		switch {
		case res.status == 400 && strings.Contains(res.body, "out_of_scope"):
			out.Verdict = vPass
		case res.status >= 200 && res.status < 300:
			out.Verdict = vFailOpen
		default:
			out.Verdict = vReview
		}
	}
	if out.Verdict != vPass {
		out.Detail = fmt.Sprintf("%s %s", res.message, truncBody(res.body))
	}
	lc.a.record(out)
	return res, nil
}

// state records a check on observed state rather than on a single response.
func (lc *lcRun) state(name string, ok bool, failVerdict, detail string) {
	out := checkResult{Layer: "lifecycle", Persona: "-", Target: name, Expected: "holds", Got: "holds", Verdict: vPass}
	if !ok {
		out.Got, out.Verdict, out.Detail = "does not hold", failVerdict, detail
	}
	lc.a.record(out)
}

// routed waits for tenant's alertmanager config to contain (or no longer
// contain) needle. A recipient that keeps receiving what it should not is a
// FAIL-OPEN; one that stops receiving what it should is a FAIL-CLOSED.
func (lc *lcRun) routed(name, tenant, needle string, present bool) {
	if id, ok := lc.a.orgLogtoID[tenant]; ok {
		tenant = id
	}
	out := checkResult{Layer: "lifecycle", Persona: "-", Target: name}
	if present {
		out.Expected = "routed in " + tenant
	} else {
		out.Expected = "not routed in " + tenant
	}
	if !lc.mimirOK {
		out.Verdict, out.Got, out.Detail = vSkip, "mimir unreachable", lc.mimirURL
		lc.a.record(out)
		return
	}
	deadline := time.Now().Add(lcMimirWait)
	for {
		cfg := lc.mimirConfig(tenant)
		if strings.Contains(cfg, needle) == present {
			out.Verdict, out.Got = vPass, out.Expected
			lc.a.record(out)
			return
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(time.Second)
	}
	out.Got = "the opposite, after " + lcMimirWait.String()
	if present {
		out.Verdict = vFailClosed
	} else {
		out.Verdict = vFailOpen
	}
	out.Detail = needle
	lc.a.record(out)
}

// ---------------------------------------------------------------------------
// Fixture helpers
// ---------------------------------------------------------------------------

func (lc *lcRun) org(key string) string {
	if id, ok := lc.a.orgLogtoID[key]; ok {
		return id
	}
	return key
}

func (lc *lcRun) system(key string) string {
	s, ok := lc.a.reg.Systems[fixtureRegKey(lc.a.spec.Fixture.Prefix, key)]
	if !ok {
		return key
	}
	return s.ID
}

func (lc *lcRun) systemOrg(id string) string {
	res, err := lc.call("owner", "GET", "/api/systems/"+id, nil)
	if err != nil || res.status != 200 {
		return ""
	}
	var parsed struct {
		Data struct {
			Organization struct {
				LogtoID string `json:"logto_id"`
			} `json:"organization"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(res.body), &parsed)
	return parsed.Data.Organization.LogtoID
}

func (lc *lcRun) createRegisteredSystem(as, orgID, name string) (string, error) {
	res, err := lc.call(as, "POST", "/api/systems", map[string]string{"name": name, "organization_id": orgID})
	if err != nil {
		return "", err
	}
	if res.status >= 300 {
		return "", fmt.Errorf("lifecycle setup: create system %s as %s: %d %s", name, as, res.status, res.message)
	}
	id, secret := dataField(res.body, "id"), dataField(res.body, "system_secret")
	lc.systems = append(lc.systems, id)
	reg, err := lc.a.request("", "POST", "/api/systems/register", fmt.Sprintf(`{"system_secret":%q}`, secret))
	if err != nil {
		return "", err
	}
	if reg.status >= 300 {
		return "", fmt.Errorf("lifecycle setup: register %s: %d %s", name, reg.status, reg.message)
	}
	return id, nil
}

func (lc *lcRun) createOrg(as, kind, name string) (string, error) {
	body := map[string]interface{}{
		"name":        name,
		"description": "authz lifecycle — safe to delete",
		"custom_data": map[string]string{"vat": syntheticVAT(int(time.Now().UnixNano() % 1e9))},
	}
	res, err := lc.call(as, "POST", "/api/"+kind, body)
	if err != nil {
		return "", err
	}
	if res.status >= 300 {
		return "", fmt.Errorf("lifecycle setup: create %s %s as %s: %d %s", kind, name, as, res.status, truncBody(res.body))
	}
	id := dataField(res.body, "logto_id")
	lc.orgs = append(lc.orgs, lcOrg{kind: kind, id: id})
	return id, nil
}

// createAdmin creates an Admin in orgID and makes it a persona for the rest of
// the run. Its credentials live in the in-memory registry only.
func (lc *lcRun) createAdmin(orgID, key string) (string, error) {
	client, err := loginAs(lc.a.reg, "owner")
	if err != nil {
		return "", err
	}
	roles, err := client.GetRoles()
	if err != nil {
		return "", err
	}
	email := renderEmail(lc.a.spec.Fixture.EmailTemplate, key)
	userID, err := client.CreateUser(email, "authz "+key, "", orgID, []string{roles["Admin"]})
	if err != nil {
		return "", fmt.Errorf("lifecycle setup: create admin: %w", err)
	}
	pw, err := generatePassword()
	if err != nil {
		return "", err
	}
	if err := client.ResetPassword(userID, pw); err != nil {
		return "", err
	}
	regKey := "authz-" + key
	lc.a.reg.Users[regKey] = User{Email: email, Password: pw, LogtoID: userID, OrgRole: "reseller",
		UserRoles: []string{"Admin"}, OrgID: orgID}
	lc.users = append(lc.users, regKey)
	p := &persona{regKey: regKey, id: key, orgRole: "reseller", orgID: orgID, userRoles: []string{"Admin"}}
	lc.a.byKey[key] = p
	return key, lc.relogin(key)
}

// relogin mints a fresh token: a promotion changes the org role it carries.
func (lc *lcRun) relogin(key string) error {
	p, err := lc.persona(key)
	if err != nil {
		return err
	}
	client, err := loginAs(lc.a.reg, p.regKey)
	if err != nil {
		return err
	}
	p.token = client.JWT()
	return nil
}

func (lc *lcRun) recipient(tag string, orgIDs, systemIDs []string) map[string]interface{} {
	scope := map[string][]string{}
	if len(orgIDs) > 0 {
		scope["organization_ids"] = orgIDs
	}
	if len(systemIDs) > 0 {
		scope["system_ids"] = systemIDs
	}
	return map[string]interface{}{
		"address": renderEmail(lc.a.spec.Fixture.EmailTemplate, tag),
		"scope":   scope,
	}
}

func layerBody(recipients []map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"enabled":             map[string]bool{"email": true},
		"email_recipients":    recipients,
		"webhook_recipients":  []interface{}{},
		"telegram_recipients": []interface{}{},
	}
}

func (lc *lcRun) saveLayerRaw(as string, recipients []map[string]interface{}) (*probeResult, error) {
	return lc.call(as, "POST", "/api/alerts/config", layerBody(recipients))
}

func (lc *lcRun) saveLayer(as string, recipients []map[string]interface{}) error {
	res, err := lc.saveLayerRaw(as, recipients)
	if err != nil {
		return err
	}
	if res.status != 200 {
		return fmt.Errorf("lifecycle setup: save layer as %s: %d %s", as, res.status, truncBody(res.body))
	}
	return nil
}

// cleanup empties the layers the run touched and destroys what it created.
func (lc *lcRun) cleanup() {
	empty := map[string]interface{}{"enabled": map[string]interface{}{}, "email_recipients": []interface{}{},
		"webhook_recipients": []interface{}{}, "telegram_recipients": []interface{}{}}
	if _, err := lc.call("d1-admin", "POST", "/api/alerts/config", empty); err != nil {
		fmt.Printf("  WARNING: could not reset d1's alert layer: %v\n", err)
	}
	for _, id := range lc.systems {
		_, _ = lc.call("owner", "DELETE", "/api/systems/"+id, nil)
		if res, err := lc.call("owner", "DELETE", "/api/systems/"+id+"/destroy", nil); err != nil || res.status >= 300 {
			fmt.Printf("  WARNING: could not destroy system %s\n", id)
		}
	}
	for i := len(lc.orgs) - 1; i >= 0; i-- {
		o := lc.orgs[i]
		_, _ = lc.call("owner", "DELETE", "/api/"+o.kind+"/"+o.id, nil)
		if res, err := lc.call("owner", "DELETE", "/api/"+o.kind+"/"+o.id+"/destroy", nil); err != nil || res.status >= 300 {
			fmt.Printf("  WARNING: could not destroy %s %s\n", o.kind, o.id)
		}
	}
	for _, key := range lc.users {
		delete(lc.a.reg.Users, key)
	}
}

// ---------------------------------------------------------------------------
// Mimir
// ---------------------------------------------------------------------------

func (lc *lcRun) probeMimir() bool {
	resp, err := http.Get(lc.mimirURL + "/ready")
	if err != nil {
		return false
	}
	_ = resp.Body.Close()
	return resp.StatusCode < 500
}

func (lc *lcRun) mimirConfig(tenant string) string {
	req, err := http.NewRequest("GET", lc.mimirURL+"/api/v1/alerts", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("X-Scope-OrgID", tenant)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	var parsed struct {
		AlertmanagerConfig string `yaml:"alertmanager_config"`
	}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		return ""
	}
	return parsed.AlertmanagerConfig
}

// ---------------------------------------------------------------------------
// Response helpers
// ---------------------------------------------------------------------------

func dataField(body, field string) string {
	var parsed struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ""
	}
	v, _ := parsed.Data[field].(string)
	return v
}

func scopeNames(body string) string {
	var parsed struct {
		Data struct {
			ScopeNames json.RawMessage `json:"scope_names"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return ""
	}
	return string(parsed.Data.ScopeNames)
}
