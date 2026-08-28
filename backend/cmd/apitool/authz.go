/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// The authz suite is an authorization regression harness: it provisions a real
// hierarchy through the API, mints a real token per persona, and then asserts
// what each persona may and may not do.
//
// Expectations are hand-authored in backend/authz/*.yml from product intent and
// from sync/configs/config.yml (the role→permission source of truth that `sync`
// pushes to Logto). They are deliberately NOT derived from the middleware wiring
// in main.go: if the two disagree, that is the finding we are looking for.

const (
	authzDefaultSpecDir = "authz"
	authzRBACConfigRel  = "../sync/configs/config.yml"
	// bogusID is syntactically plausible for every id kind in the API (org
	// logto ids, user logto ids, system/application UUIDs) and matches nothing.
	// Every by-id route answers 404 for it, so the gate is observable without
	// mutating any real object — which is what makes the matrix idempotent.
	bogusID = "00000000-0000-4000-8000-000000000000"
)

// ---------------------------------------------------------------------------
// Spec types
// ---------------------------------------------------------------------------

// FixtureSpec describes the hierarchy the suite needs. Names are namespaced with
// Prefix so provisioning never collides with hand-made test data.
type FixtureSpec struct {
	Prefix string `yaml:"prefix"`
	// EmailTemplate builds every fixture user's address; "{key}" is replaced
	// with the fixture user key. Use plus sub-addressing on a real inbox so
	// password-reset mail actually arrives and stays sortable.
	EmailTemplate string          `yaml:"email_template"`
	Orgs          []FixtureOrg    `yaml:"orgs"`
	Users         []FixtureUser   `yaml:"users"`
	Systems       []FixtureSystem `yaml:"systems"`
}

type FixtureOrg struct {
	Key string `yaml:"key"`
	// Type is distributor|reseller|customer.
	Type string `yaml:"type"`
	// CreatedBy is the fixture user key that creates the org, which is what
	// puts it in that user's branch of the hierarchy. "owner" for the top.
	CreatedBy string `yaml:"created_by"`
	VAT       string `yaml:"vat"`
	Note      string `yaml:"note"`
}

type FixtureUser struct {
	Key string `yaml:"key"`
	// Org is a fixture org key, or "owner" for the owner organization.
	Org string `yaml:"org"`
	// Role is the technical role name as GET /api/roles exposes it:
	// Admin, Support, Backoffice, Reader, Super Admin.
	Role string `yaml:"role"`
	// CreatedBy is the fixture user key that creates this user (default owner).
	CreatedBy string `yaml:"created_by"`
	// Matrix marks the representative persona for the route×persona matrix.
	// Helper users used only by scope scenarios leave it false.
	Matrix bool   `yaml:"matrix"`
	Note   string `yaml:"note"`
}

type FixtureSystem struct {
	Key      string `yaml:"key"`
	Org      string `yaml:"org"`
	Register bool   `yaml:"register"`
	Note     string `yaml:"note"`
}

// RouteSpec is the hand-authored authorization intent of one endpoint.
type RouteSpec struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
	// Intent is the permission the endpoint is meant to require
	// ("read:systems"), or one of the pseudo-values:
	//   public        — reachable with no token at all
	//   authenticated — any valid token, no permission needed
	//   owner_only    — only the owner organization, whatever the user role
	Intent string `yaml:"intent"`
	// IntentAny: any one of these permissions is enough.
	IntentAny []string `yaml:"intent_any"`
	// SelfOrgGet records the intent "a user may always read its own
	// organization", so an org-scoped GET on the caller's own org is expected
	// to pass even when the caller lacks the read permission for that level.
	SelfOrgGet bool `yaml:"self_org_get"`
	// ObjectScoped declares that this endpoint intentionally has no route-level
	// permission gate: it is reachable by any authenticated user and the
	// decision belongs to the handler, per object. A handler refusal is then a
	// correct answer rather than something to review.
	ObjectScoped bool `yaml:"object_scoped"`
	// OrgRole narrows the intent to one organization role, ANDed with the
	// permission: "manage:rebranding, and only from the owner organization".
	// Needed where a route-level permission is held across the hierarchy but the
	// action belongs to one level only — owner_only alone would wrongly expect a
	// Reader of that organization to pass the permission gate.
	OrgRole string `yaml:"org_role"`
	// OrUserRole widens OrgRole with a technical role that satisfies it from any
	// organization: "the owner organization, or a Super Admin wherever they sit".
	// That is how the entitlement administrative surface is defined — Nethesis
	// staff hold Super Admin inside the Nethesis Italia distributor.
	OrUserRole string `yaml:"or_user_role"`
	// EnforcedBy names where the authorization decision is expected to live.
	// "handler" means the check is deliberately inside the handler rather than in
	// route middleware, so a handler refusal is the expected denial. The
	// assertion stays sharp: if that check disappears the probe starts returning
	// 404/2xx and the suite reports a fail-open.
	EnforcedBy string `yaml:"enforced_by"`
	// ExpectAnonymous overrides the unauthenticated expectation. Needed for the
	// endpoints that ARE the authenticator: a missing credential legitimately
	// yields 400/401 there, which says nothing about a route gate.
	ExpectAnonymous []int `yaml:"expect_anonymous"`
	// Body overrides the request body. Default for write verbs is "{}", which
	// a correctly-gated endpoint rejects with 400 *after* the gate passed.
	Body string `yaml:"body"`
	// Query is appended verbatim (without "?").
	Query string `yaml:"query"`
	// IDs overrides path params by name, e.g. {entity_type: distributor}.
	IDs map[string]string `yaml:"ids"`
	// ExpectAuthorized, when set, tightens the accepted status codes for
	// personas that are supposed to get through the gate.
	ExpectAuthorized []int `yaml:"expect_authorized"`
	// Skip excludes the route from the gate layer, with a stated reason.
	Skip string `yaml:"skip"`
	Note string `yaml:"note"`
}

func (r RouteSpec) id() string { return r.Method + " " + r.Path }

// AuthzSpec is everything loaded from the spec directory.
type AuthzSpec struct {
	Fixture   FixtureSpec
	Routes    []RouteSpec
	Scenarios []ScenarioSpec
	Apps      AppsSpec
	Special   []SpecialSpec
	Model     ModelSpec
	rbac      *rbacConfig
}

// ModelSpec records reviewed deviations between config.yml's declared
// role→permission model and the permissions the backend really embeds in the
// JWT. See authz/model.yml for why each one exists.
type ModelSpec struct {
	PermissionExceptions []PermissionException `yaml:"permission_exceptions"`
}

type PermissionException struct {
	// Roles are the technical role names the exception applies to.
	Roles  []string `yaml:"roles"`
	Remove []string `yaml:"remove"`
	Code   string   `yaml:"code"`
	Reason string   `yaml:"reason"`
	// ActionRequired states what should change so the exception can go away.
	ActionRequired string `yaml:"action_required"`
}

// appliesTo reports whether a persona holding these roles is subject to the
// exception.
func (e PermissionException) appliesTo(userRoles []string) bool {
	for _, want := range e.Roles {
		for _, have := range userRoles {
			if strings.EqualFold(want, have) {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// RBAC config (sync/configs/config.yml) — the role→permission source of truth
// ---------------------------------------------------------------------------

type rbacPermission struct {
	ID string `yaml:"id"`
}

type rbacRole struct {
	ID          string           `yaml:"id"`
	Name        string           `yaml:"name"`
	Permissions []rbacPermission `yaml:"permissions"`
}

type rbacAccessControl struct {
	OrganizationIDs   []string `yaml:"organization_ids"`
	OrganizationRoles []string `yaml:"organization_roles"`
	UserRoles         []string `yaml:"user_roles"`
}

type rbacThirdPartyApp struct {
	Name          string            `yaml:"name"`
	DisplayName   string            `yaml:"display_name"`
	LoginURL      string            `yaml:"login_url"`
	RedirectURIs  []string          `yaml:"redirect_uris"`
	AccessControl rbacAccessControl `yaml:"access_control"`
}

type rbacConfig struct {
	OrganizationRoles []rbacRole          `yaml:"organization_roles"`
	UserRoles         []rbacRole          `yaml:"user_roles"`
	ThirdPartyApps    []rbacThirdPartyApp `yaml:"third_party_apps"`
}

// orgPerms returns the permissions an organization role grants.
func (c *rbacConfig) orgPerms(orgRole string) []string {
	for _, r := range c.OrganizationRoles {
		if strings.EqualFold(r.ID, orgRole) {
			return permIDs(r.Permissions)
		}
	}
	return nil
}

// userPermsByName resolves a technical role by the display name the API exposes
// ("Super Admin") rather than by its config id ("super").
func (c *rbacConfig) userPermsByName(roleName string) ([]string, bool) {
	for _, r := range c.UserRoles {
		if strings.EqualFold(r.Name, roleName) || strings.EqualFold(r.ID, roleName) {
			return permIDs(r.Permissions), true
		}
	}
	return nil, false
}

func (c *rbacConfig) userRoleID(roleName string) string {
	for _, r := range c.UserRoles {
		if strings.EqualFold(r.Name, roleName) || strings.EqualFold(r.ID, roleName) {
			return r.ID
		}
	}
	return strings.ToLower(roleName)
}

func permIDs(ps []rbacPermission) []string {
	out := make([]string, 0, len(ps))
	for _, p := range ps {
		if p.ID != "" {
			out = append(out, p.ID)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// Personas
// ---------------------------------------------------------------------------

// persona is a (organization role, technical role) pair backed by a real user
// with a real token. perms is what config.yml says the pair should be able to
// do; jwtPerms is what the token actually carries, so drift between Logto and
// the config is reported instead of silently changing every expectation.
type persona struct {
	regKey    string
	id        string
	orgRole   string
	orgKey    string
	orgID     string
	orgName   string
	userRoles []string
	perms     map[string]bool
	jwtPerms  map[string]bool
	token     string
	matrix    bool
}

func (p *persona) has(perm string) bool { return p.perms[perm] }

func (p *persona) hasUserRole(role string) bool {
	if role == "" {
		return false
	}
	for _, r := range p.userRoles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

func (p *persona) hasAny(perms []string) bool {
	for _, perm := range perms {
		if p.perms[perm] {
			return true
		}
	}
	return false
}

func (p *persona) sortedPerms() []string {
	out := make([]string, 0, len(p.perms))
	for k := range p.perms {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ---------------------------------------------------------------------------
// Runner
// ---------------------------------------------------------------------------

type authzRunner struct {
	spec     *AuthzSpec
	reg      *Registry
	baseURL  string
	http     *http.Client
	personas []*persona
	byKey    map[string]*persona
	results  []checkResult
	verbose  bool
	// idpProbe forces the (slow) real OIDC probe in the apps layer.
	idpProbe bool
	// apiKeys caches the throwaway keys minted by the special layer.
	apiKeys map[string]*apiKeyRef
	// roleIDs caches the tenant's technical role name → id catalogue.
	roleIDs map[string]string
	// fixture resolution: fixture key -> real registry key / logto id
	orgLogtoID map[string]string
	orgName    map[string]string
}

type checkResult struct {
	Layer    string `json:"layer"`
	Persona  string `json:"persona"`
	Target   string `json:"target"`
	Expected string `json:"expected"`
	Got      string `json:"got"`
	Status   int    `json:"status"`
	Verdict  string `json:"verdict"`
	Detail   string `json:"detail,omitempty"`
}

// Verdicts.
const (
	vPass = "PASS"
	// vFailOpen: the persona was supposed to be blocked and was not. This is
	// the class of finding that leaks data across the hierarchy.
	vFailOpen = "FAIL-OPEN"
	// vFailClosed: the persona was supposed to be allowed and was blocked.
	// Not a data leak, but a broken product promise.
	vFailClosed = "FAIL-CLOSED"
	// vReview: denied/allowed in a way that needs a human look (denied by the
	// handler rather than the gate, or a 5xx that hides the real answer).
	vReview = "REVIEW"
	vSkip   = "SKIP"
)

// Response classification.
const (
	outAllowed       = "ALLOWED"        // 2xx
	outGateDenied    = "GATE-DENIED"    // 403 from the RBAC middleware
	outHandlerDenied = "HANDLER-DENIED" // 403 from handler-level checks
	outNotFound      = "NOT-FOUND"      // 404: gate passed, object absent
	// outNoRoute is the router's own 404 ("api not found"). It means the request
	// never reached any endpoint, so it proves nothing about authorization. It is
	// never a PASS: a mistyped path in routes.yml would otherwise look green.
	outNoRoute     = "NO-ROUTE"
	outBadRequest  = "BAD-REQUEST"  // 400/422: gate passed, body rejected
	outUnauth      = "UNAUTH"       // 401
	outRateLimited = "RATE-LIMITED" // 429: authenticated, then throttled
	outServerError = "SERVER-ERROR" // 5xx
	outOther       = "OTHER"
)

// gateDeniedMessage is the exact message the RBAC middleware returns. It is what
// separates a route-level denial from a handler-level ("access denied…") one.
const gateDeniedMessage = "insufficient permissions"

// noRouteMessage is what the router answers when no endpoint matches at all.
const noRouteMessage = "api not found"

type probeResult struct {
	status  int
	message string
	body    string
	class   string
}

func classify(status int, message string) string {
	switch {
	case status >= 200 && status < 300:
		return outAllowed
	case status == 401:
		return outUnauth
	case status == 403:
		if strings.EqualFold(strings.TrimSpace(message), gateDeniedMessage) {
			return outGateDenied
		}
		return outHandlerDenied
	case status == 404:
		if strings.EqualFold(strings.TrimSpace(message), noRouteMessage) {
			return outNoRoute
		}
		return outNotFound
	case status == 400 || status == 422:
		return outBadRequest
	case status == 429:
		return outRateLimited
	case status >= 500:
		return outServerError
	}
	return outOther
}

// request fires one API call as a persona. An empty token means anonymous.
func (a *authzRunner) request(token, method, path, body string) (*probeResult, error) {
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, a.baseURL+path, rdr)
	if err != nil {
		return nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := a.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)
	pr := &probeResult{status: resp.StatusCode, body: string(raw)}
	var envelope struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &envelope); err == nil {
		pr.message = envelope.Message
	}
	pr.class = classify(pr.status, pr.message)
	return pr, nil
}

func (a *authzRunner) record(r checkResult) {
	a.results = append(a.results, r)
	if a.verbose && r.Verdict != vPass {
		fmt.Printf("  %-11s %-22s %-52s expected %s, got %s (%d) %s\n",
			r.Verdict, r.Persona, r.Target, r.Expected, r.Got, r.Status, r.Detail)
	}
}

// ---------------------------------------------------------------------------
// Spec loading
// ---------------------------------------------------------------------------

func authzSpecDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	for _, cand := range []string{authzDefaultSpecDir, filepath.Join("backend", authzDefaultSpecDir)} {
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("spec dir %q not found (run from backend/ or repo root)", authzDefaultSpecDir)
}

func loadYAMLFile(path string, into interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(data, into); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	return nil
}

func loadAuthzSpec(dir string) (*AuthzSpec, error) {
	spec := &AuthzSpec{}

	if err := loadYAMLFile(filepath.Join(dir, "fixture.yml"), &spec.Fixture); err != nil {
		return nil, err
	}
	var routesDoc struct {
		Routes []RouteSpec `yaml:"routes"`
	}
	if err := loadYAMLFile(filepath.Join(dir, "routes.yml"), &routesDoc); err != nil {
		return nil, err
	}
	spec.Routes = routesDoc.Routes

	// Optional files: layers can be developed independently.
	var scenariosDoc struct {
		Scenarios []ScenarioSpec `yaml:"scenarios"`
	}
	if err := loadYAMLFile(filepath.Join(dir, "scenarios.yml"), &scenariosDoc); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	spec.Scenarios = scenariosDoc.Scenarios

	if err := loadYAMLFile(filepath.Join(dir, "apps.yml"), &spec.Apps); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	var specialDoc struct {
		Cases []SpecialSpec `yaml:"cases"`
	}
	if err := loadYAMLFile(filepath.Join(dir, "special.yml"), &specialDoc); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	spec.Special = specialDoc.Cases

	if err := loadYAMLFile(filepath.Join(dir, "model.yml"), &spec.Model); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	rbacPath := authzRBACPath(dir)
	var rbac rbacConfig
	if err := loadYAMLFile(rbacPath, &rbac); err != nil {
		return nil, fmt.Errorf("loading RBAC config: %w", err)
	}
	spec.rbac = &rbac

	if err := spec.validate(); err != nil {
		return nil, err
	}
	return spec, nil
}

// authzRBACPath locates sync/configs/config.yml relative to the spec dir, so the
// suite works both from backend/ and from the repo root.
func authzRBACPath(specDir string) string {
	cands := []string{
		filepath.Join(specDir, "..", authzRBACConfigRel),
		filepath.Join("sync", "configs", "config.yml"),
		filepath.Join("..", "sync", "configs", "config.yml"),
	}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return cands[0]
}

func (s *AuthzSpec) validate() error {
	var problems []string

	orgKeys := map[string]bool{"owner": true}
	for _, o := range s.Fixture.Orgs {
		if orgKeys[o.Key] {
			problems = append(problems, fmt.Sprintf("duplicate fixture org key %q", o.Key))
		}
		orgKeys[o.Key] = true
		if o.Type != "distributor" && o.Type != "reseller" && o.Type != "customer" {
			problems = append(problems, fmt.Sprintf("org %q: invalid type %q", o.Key, o.Type))
		}
	}
	userKeys := map[string]bool{"owner": true}
	for _, u := range s.Fixture.Users {
		if userKeys[u.Key] {
			problems = append(problems, fmt.Sprintf("duplicate fixture user key %q", u.Key))
		}
		userKeys[u.Key] = true
		if !orgKeys[u.Org] {
			problems = append(problems, fmt.Sprintf("user %q: unknown org %q", u.Key, u.Org))
		}
		if _, ok := s.rbac.userPermsByName(u.Role); !ok {
			problems = append(problems, fmt.Sprintf("user %q: role %q not in config.yml", u.Key, u.Role))
		}
	}
	for _, o := range s.Fixture.Orgs {
		if o.CreatedBy != "" && !userKeys[o.CreatedBy] {
			problems = append(problems, fmt.Sprintf("org %q: created_by %q is not a fixture user", o.Key, o.CreatedBy))
		}
	}
	for _, sy := range s.Fixture.Systems {
		if !orgKeys[sy.Org] {
			problems = append(problems, fmt.Sprintf("system %q: unknown org %q", sy.Key, sy.Org))
		}
	}

	// Route intents must reference permissions that exist in config.yml,
	// otherwise a typo would silently expect "denied for everybody".
	known := map[string]bool{}
	for _, r := range s.rbac.OrganizationRoles {
		for _, p := range r.Permissions {
			known[p.ID] = true
		}
	}
	for _, r := range s.rbac.UserRoles {
		for _, p := range r.Permissions {
			known[p.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, rt := range s.Routes {
		if seen[rt.id()] {
			problems = append(problems, fmt.Sprintf("duplicate route spec %s", rt.id()))
		}
		seen[rt.id()] = true
		if rt.Intent == "" && len(rt.IntentAny) == 0 {
			problems = append(problems, fmt.Sprintf("%s: no intent declared", rt.id()))
		}
		for _, p := range append([]string{}, rt.IntentAny...) {
			if !known[p] {
				problems = append(problems, fmt.Sprintf("%s: unknown permission %q", rt.id(), p))
			}
		}
		switch rt.Intent {
		case "", "public", "authenticated", "owner_only":
		default:
			if !known[rt.Intent] {
				problems = append(problems, fmt.Sprintf("%s: unknown permission %q", rt.id(), rt.Intent))
			}
		}
		switch rt.OrgRole {
		case "", "owner", "distributor", "reseller", "customer":
		default:
			problems = append(problems, fmt.Sprintf("%s: unknown organization role %q", rt.id(), rt.OrgRole))
		}
		if rt.OrUserRole != "" && rt.OrgRole == "" {
			problems = append(problems, fmt.Sprintf("%s: or_user_role without org_role", rt.id()))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("spec problems:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// ---------------------------------------------------------------------------
// Guard
// ---------------------------------------------------------------------------

// assertLocal refuses to touch a shared environment. The suite creates orgs and
// users and fires every endpoint of the API, including destructive verbs in the
// scope layer; pointing it at QA or production would pollute real data.
func assertLocal(reg *Registry, flags map[string]string) error {
	if flags["i-know-this-is-not-local"] == "true" {
		fmt.Fprintln(os.Stderr, "WARNING: authz suite running against a non-local backend by explicit override")
		return nil
	}
	u := reg.Config.BackendURL
	if strings.Contains(u, "localhost") || strings.Contains(u, "127.0.0.1") || strings.Contains(u, "localtest.me") {
		return nil
	}
	return fmt.Errorf("backend_url %q is not local: the authz suite creates orgs/users and fires every endpoint.\n"+
		"Point apitool at a local stack (apitool init) or pass --i-know-this-is-not-local", u)
}

// ---------------------------------------------------------------------------
// Persona construction
// ---------------------------------------------------------------------------

// buildPersonas resolves every fixture user to a registry entry, computes the
// permissions config.yml says it should have, and mints a token.
func (a *authzRunner) buildPersonas(onlyMatrix bool) error {
	prefix := a.spec.Fixture.Prefix
	a.byKey = map[string]*persona{}

	// The owner persona is the registry owner itself: it has no fixture entry.
	ownerP, err := a.ownerPersona()
	if err != nil {
		return err
	}
	a.personas = append(a.personas, ownerP)
	a.byKey["owner"] = ownerP

	for _, fu := range a.spec.Fixture.Users {
		if onlyMatrix && !fu.Matrix {
			continue
		}
		regKey := fixtureRegKey(prefix, fu.Key)
		ru, ok := a.reg.Users[regKey]
		if !ok {
			return fmt.Errorf("fixture user %q (registry key %q) not provisioned — run: apitool authz provision", fu.Key, regKey)
		}
		p := &persona{
			regKey:    regKey,
			id:        fu.Key,
			orgRole:   ru.OrgRole,
			orgKey:    fu.Org,
			orgID:     ru.OrgID,
			orgName:   ru.OrgName,
			userRoles: []string{fu.Role},
			matrix:    fu.Matrix,
			perms:     map[string]bool{},
		}
		for _, perm := range a.spec.rbac.orgPerms(ru.OrgRole) {
			p.perms[perm] = true
		}
		up, ok := a.spec.rbac.userPermsByName(fu.Role)
		if !ok {
			return fmt.Errorf("fixture user %q: role %q not in config.yml", fu.Key, fu.Role)
		}
		for _, perm := range up {
			p.perms[perm] = true
		}
		a.applyModelExceptions(p)
		a.personas = append(a.personas, p)
		a.byKey[fu.Key] = p
	}
	return nil
}

// applyModelExceptions subtracts the reviewed deviations in model.yml from the
// permissions config.yml grants, so expectations reflect the model the product
// actually intends — with every departure from config.yml written down.
func (a *authzRunner) applyModelExceptions(p *persona) {
	for _, ex := range a.spec.Model.PermissionExceptions {
		if !ex.appliesTo(p.userRoles) {
			continue
		}
		for _, perm := range ex.Remove {
			delete(p.perms, perm)
		}
	}
}

func (a *authzRunner) ownerPersona() (*persona, error) {
	p := &persona{
		regKey:    "owner",
		id:        "owner-super",
		orgRole:   "owner",
		orgKey:    "owner",
		userRoles: []string{"Super Admin"},
		matrix:    true,
		perms:     map[string]bool{},
	}
	for _, perm := range a.spec.rbac.orgPerms("owner") {
		p.perms[perm] = true
	}
	up, _ := a.spec.rbac.userPermsByName("Super Admin")
	for _, perm := range up {
		p.perms[perm] = true
	}
	a.applyModelExceptions(p)
	return p, nil
}

// mintTokens logs every persona in once. Tokens are reused for the whole run;
// the backend JWT lives 30 minutes, comfortably longer than a full matrix.
func (a *authzRunner) mintTokens() error {
	for _, p := range a.personas {
		client, err := loginAs(a.reg, p.regKey)
		if err != nil {
			return fmt.Errorf("login as %s (%s): %w", p.id, p.regKey, err)
		}
		p.token = client.JWT()
		p.jwtPerms = jwtPermissions(p.token)
		if p.orgID == "" {
			p.orgID = jwtClaimString(p.token, "organization_id")
		}
		if p.orgRole == "" {
			p.orgRole = jwtClaimString(p.token, "org_role")
		}
	}
	return nil
}

// jwtPermissions decodes the permission claims the backend embedded in the
// token. Comparing them against config.yml catches a Logto that is out of sync
// with the config, which would otherwise look like a middleware bug.
func jwtPermissions(token string) map[string]bool {
	out := map[string]bool{}
	claims := jwtUserClaims(token)
	if claims == nil {
		return out
	}
	for _, key := range []string{"user_permissions", "org_permissions", "permissions"} {
		raw, ok := claims[key]
		if !ok {
			continue
		}
		list, ok := raw.([]interface{})
		if !ok {
			continue
		}
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				continue
			}
			// role-access-control scopes ("owner:role-access-control") are
			// derived by sync from the organization role, not listed on any role
			// in config.yml. Comparing them would report permanent false drift.
			if strings.HasSuffix(s, ":role-access-control") {
				continue
			}
			out[s] = true
		}
	}
	return out
}

func jwtClaimString(token, claim string) string {
	claims := jwtUserClaims(token)
	if claims == nil {
		return ""
	}
	if s, ok := claims[claim].(string); ok {
		return s
	}
	return ""
}

// jwtUserClaims returns the object the backend nests its identity claims in. The
// custom JWT carries everything under "user"; falling back to the top level keeps
// this working if that ever flattens.
func jwtUserClaims(token string) map[string]interface{} {
	claims := jwtClaims(token)
	if claims == nil {
		return nil
	}
	if nested, ok := claims["user"].(map[string]interface{}); ok {
		return nested
	}
	return claims
}

func jwtClaims(token string) map[string]interface{} {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil
	}
	return claims
}

func fixtureRegKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "-" + key
}

// ---------------------------------------------------------------------------
// Command dispatch
// ---------------------------------------------------------------------------

func cmdAuthz(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: apitool authz <provision|run|coverage|personas|teardown> [flags]")
	}
	sub := args[0]
	flags, _ := parseFlags(args[1:])

	reg, err := loadOrInit()
	if err != nil {
		return err
	}
	if err := assertLocal(reg, flags); err != nil {
		return err
	}
	dir, err := authzSpecDir(flags["spec-dir"])
	if err != nil {
		return err
	}

	switch sub {
	case "coverage":
		return authzCoverage(dir, flags)
	case "provision":
		return authzProvision(dir, reg, flags)
	case "teardown":
		return authzTeardown(dir, reg, flags)
	case "personas":
		return authzPersonas(dir, reg, flags)
	case "run":
		return authzRun(dir, reg, flags)
	}
	return fmt.Errorf("unknown authz subcommand %q", sub)
}

// newRunner loads the spec, resolves personas and mints their tokens.
func newRunner(dir string, reg *Registry, flags map[string]string, onlyMatrix bool) (*authzRunner, error) {
	spec, err := loadAuthzSpec(dir)
	if err != nil {
		return nil, err
	}
	// Spec paths are absolute and include the /api prefix (they come from the
	// route inventory), while backend_url already ends in /api. Strip it so the
	// two do not stack into /api/api/… — which the router answers with a 404
	// that would read as "the gate let me through".
	baseURL := strings.TrimSuffix(strings.TrimSuffix(reg.Config.BackendURL, "/"), "/api")

	a := &authzRunner{
		spec:    spec,
		reg:     reg,
		baseURL: baseURL,
		http: &http.Client{
			Timeout:   60 * time.Second,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		},
		verbose:    flags["verbose"] == "true" || flags["v"] == "true",
		idpProbe:   flags["idp-probe"] == "true",
		orgLogtoID: map[string]string{},
		orgName:    map[string]string{},
	}
	for _, fo := range spec.Fixture.Orgs {
		regKey := fixtureRegKey(spec.Fixture.Prefix, fo.Key)
		if o, ok := reg.Orgs[regKey]; ok {
			a.orgLogtoID[fo.Key] = o.LogtoID
			a.orgName[fo.Key] = o.Name
		}
	}
	if err := a.buildPersonas(onlyMatrix); err != nil {
		return nil, err
	}
	return a, nil
}

// authzPersonas prints each persona with the permissions config.yml grants and
// flags any difference against the permissions its real token carries.
func authzPersonas(dir string, reg *Registry, flags map[string]string) error {
	a, err := newRunner(dir, reg, flags, false)
	if err != nil {
		return err
	}
	if err := a.mintTokens(); err != nil {
		return err
	}
	drift := 0
	for _, p := range a.personas {
		fmt.Printf("\n%-22s org=%-12s roles=%-14s org_id=%s\n", p.id, p.orgRole, strings.Join(p.userRoles, ","), p.orgID)
		fmt.Printf("  config.yml perms (%d): %s\n", len(p.perms), strings.Join(p.sortedPerms(), " "))
		var missing, extra []string
		for perm := range p.perms {
			if !p.jwtPerms[perm] {
				missing = append(missing, perm)
			}
		}
		for perm := range p.jwtPerms {
			if !p.perms[perm] {
				extra = append(extra, perm)
			}
		}
		sort.Strings(missing)
		sort.Strings(extra)
		if len(missing) > 0 || len(extra) > 0 {
			drift++
			fmt.Printf("  DRIFT vs token: missing=%v extra=%v\n", missing, extra)
		}
	}
	fmt.Printf("\n%d persona(s), %d with config/token drift\n", len(a.personas), drift)
	if drift > 0 {
		return fmt.Errorf("token permissions disagree with config.yml — run `sync push` before trusting the matrix")
	}
	return nil
}

// authzCoverage reports which live routes have no hand-authored intent. It reads
// the route inventory from main.go: only method+path, never the middleware.
func authzCoverage(dir string, flags map[string]string) error {
	spec, err := loadAuthzSpec(dir)
	if err != nil {
		return err
	}
	inv, err := routeInventory(flags["main-go"])
	if err != nil {
		return err
	}
	specified := map[string]bool{}
	for _, r := range spec.Routes {
		specified[r.Method+" "+r.Path] = true
	}
	live := map[string]bool{}
	var missing []string
	for _, r := range inv {
		live[r.Method+" "+r.Path] = true
		if !specified[r.Method+" "+r.Path] {
			missing = append(missing, fmt.Sprintf("%-6s %s", r.Method, r.Path))
		}
	}
	var stale []string
	for _, r := range spec.Routes {
		if !live[r.Method+" "+r.Path] {
			stale = append(stale, fmt.Sprintf("%-6s %s", r.Method, r.Path))
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)

	fmt.Printf("Routes in main.go: %d\nRoutes in routes.yml: %d\n", len(inv), len(spec.Routes))
	if len(missing) > 0 {
		fmt.Printf("\nMISSING intent (%d) — every new endpoint must declare one:\n", len(missing))
		for _, m := range missing {
			fmt.Println("  " + m)
		}
	}
	if len(stale) > 0 {
		fmt.Printf("\nSTALE spec entries (%d) — route no longer exists:\n", len(stale))
		for _, m := range stale {
			fmt.Println("  " + m)
		}
	}
	if len(missing) == 0 && len(stale) == 0 {
		fmt.Println("\nCoverage complete: every route has a hand-authored intent.")
		return nil
	}
	return fmt.Errorf("route coverage incomplete: %d missing, %d stale", len(missing), len(stale))
}
