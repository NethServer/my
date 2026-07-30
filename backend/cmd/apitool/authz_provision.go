/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// Provisioning is idempotent: anything already in the registry is left alone, so
// the command can be re-run after a partial failure without duplicating orgs or
// burning Logto writes. Orgs and users are created *by* the fixture user named in
// created_by, because that is what places them in the right branch of the
// hierarchy — creating everything as the owner would produce a flat tree and the
// scope layer would prove nothing.

type provisioner struct {
	spec       *AuthzSpec
	reg        *Registry
	clients    map[string]*Client
	ownerOrgID string
}

func newProvisioner(spec *AuthzSpec, reg *Registry) *provisioner {
	return &provisioner{spec: spec, reg: reg, clients: map[string]*Client{}}
}

// clientFor logs in once per actor and caches the client.
func (p *provisioner) clientFor(fixtureKey string) (*Client, error) {
	regKey := "owner"
	if fixtureKey != "" && fixtureKey != "owner" {
		regKey = fixtureRegKey(p.spec.Fixture.Prefix, fixtureKey)
	}
	if c, ok := p.clients[regKey]; ok {
		return c, nil
	}
	c, err := loginAs(p.reg, regKey)
	if err != nil {
		return nil, fmt.Errorf("login as %q: %w", regKey, err)
	}
	p.clients[regKey] = c
	return c, nil
}

// resolveOrgID maps a fixture org key to a Logto organization id. "owner"
// resolves to the owner's own organization, read from the owner token.
func (p *provisioner) resolveOrgID(orgKey string) (string, error) {
	if orgKey == "owner" {
		if p.ownerOrgID != "" {
			return p.ownerOrgID, nil
		}
		c, err := p.clientFor("owner")
		if err != nil {
			return "", err
		}
		id := jwtClaimString(c.JWT(), "organization_id")
		if id == "" {
			return "", fmt.Errorf("cannot determine owner organization id from token")
		}
		p.ownerOrgID = id
		return id, nil
	}
	regKey := fixtureRegKey(p.spec.Fixture.Prefix, orgKey)
	org, ok := p.reg.Orgs[regKey]
	if !ok {
		return "", fmt.Errorf("org %q (%s) not provisioned yet", orgKey, regKey)
	}
	return org.LogtoID, nil
}

// authzProvision walks the fixture until nothing is left to create. A reseller
// must be created by a user of its distributor, and that user must exist first,
// so orgs and users cannot be provisioned in two separate passes. Rather than
// forcing the spec into a dependency-sorted order, this repeatedly creates
// whatever is currently creatable until it makes no more progress.
func authzProvision(dir string, reg *Registry, flags map[string]string) error {
	spec, err := loadAuthzSpec(dir)
	if err != nil {
		return err
	}
	p := newProvisioner(spec, reg)
	prefix := spec.Fixture.Prefix

	orgReady := func(orgKey string) bool {
		if orgKey == "owner" {
			return true
		}
		_, ok := reg.Orgs[fixtureRegKey(prefix, orgKey)]
		return ok
	}
	actorReady := func(userKey string) bool {
		if userKey == "" || userKey == "owner" {
			return true
		}
		_, ok := reg.Users[fixtureRegKey(prefix, userKey)]
		return ok
	}

	todoOrgs := map[string]bool{}
	for _, fo := range spec.Fixture.Orgs {
		if _, ok := reg.Orgs[fixtureRegKey(prefix, fo.Key)]; !ok {
			todoOrgs[fo.Key] = true
		}
	}
	todoUsers := map[string]bool{}
	for _, fu := range spec.Fixture.Users {
		if existing, ok := reg.Users[fixtureRegKey(prefix, fu.Key)]; ok {
			if len(existing.UserRoles) == 0 {
				existing.UserRoles = []string{fu.Role}
				reg.Users[fixtureRegKey(prefix, fu.Key)] = existing
			}
			continue
		}
		todoUsers[fu.Key] = true
	}
	todoSystems := map[string]bool{}
	for _, fs := range spec.Fixture.Systems {
		if _, ok := reg.Systems[fixtureRegKey(prefix, fs.Key)]; !ok {
			todoSystems[fs.Key] = true
		}
	}
	if err := reg.Save(); err != nil {
		return err
	}
	fmt.Printf("To create: %d org(s), %d user(s), %d system(s)\n\n",
		len(todoOrgs), len(todoUsers), len(todoSystems))

	for progress := true; progress; {
		progress = false

		for i, fo := range spec.Fixture.Orgs {
			if !todoOrgs[fo.Key] || !actorReady(fo.CreatedBy) {
				continue
			}
			regKey := fixtureRegKey(prefix, fo.Key)
			client, err := p.clientFor(fo.CreatedBy)
			if err != nil {
				return err
			}
			vat := fo.VAT
			if vat == "" {
				vat = syntheticVAT(i)
			}
			customData := map[string]interface{}{
				"vat":   vat,
				"notes": "authz suite fixture — safe to delete",
			}
			logtoID, err := client.CreateOrg(fo.Type, regKey, "authz suite fixture", customData)
			if err != nil {
				return fmt.Errorf("create org %s: %w", regKey, err)
			}
			reg.Orgs[regKey] = Org{Type: fo.Type, LogtoID: logtoID, Name: regKey, CreatedAt: time.Now().UTC()}
			if err := reg.Save(); err != nil {
				return err
			}
			delete(todoOrgs, fo.Key)
			progress = true
			fmt.Printf("  org    %-26s %-12s by %-14s %s\n", regKey, fo.Type, defaultAs(fo.CreatedBy), logtoID)
		}

		for _, fu := range spec.Fixture.Users {
			if !todoUsers[fu.Key] || !orgReady(fu.Org) || !actorReady(fu.CreatedBy) {
				continue
			}
			regKey := fixtureRegKey(prefix, fu.Key)
			orgID, err := p.resolveOrgID(fu.Org)
			if err != nil {
				return err
			}
			client, err := p.clientFor(fu.CreatedBy)
			if err != nil {
				return err
			}
			roles, err := client.GetRoles()
			if err != nil {
				return err
			}
			roleID, ok := roles[fu.Role]
			if !ok {
				return fmt.Errorf("user %s: role %q not exposed by GET /roles", regKey, fu.Role)
			}
			pw, err := generatePassword()
			if err != nil {
				return err
			}
			email := renderEmail(spec.Fixture.EmailTemplate, fu.Key)
			userID, err := client.CreateUser(email, "authz "+fu.Key, "", orgID, []string{roleID})
			if err != nil {
				return fmt.Errorf("create user %s: %w", regKey, err)
			}
			if err := client.ResetPassword(userID, pw); err != nil {
				return fmt.Errorf("user %s created (%s) but password reset failed: %w", regKey, userID, err)
			}
			orgRole, orgName := "owner", "owner"
			if fu.Org != "owner" {
				o := reg.Orgs[fixtureRegKey(prefix, fu.Org)]
				orgRole, orgName = o.Type, o.Name
			}
			reg.Users[regKey] = User{
				Email: email, Password: pw, LogtoID: userID,
				OrgRole: orgRole, UserRoles: []string{fu.Role},
				OrgID: orgID, OrgName: orgName, CreatedAt: time.Now().UTC(),
			}
			if err := reg.Save(); err != nil {
				return err
			}
			delete(todoUsers, fu.Key)
			progress = true
			fmt.Printf("  user   %-26s %-12s in %s\n", regKey, fu.Role, orgName)
		}

		for _, fs := range spec.Fixture.Systems {
			if !todoSystems[fs.Key] || !orgReady(fs.Org) {
				continue
			}
			regKey := fixtureRegKey(prefix, fs.Key)
			orgID, err := p.resolveOrgID(fs.Org)
			if err != nil {
				return err
			}
			client, err := p.clientFor("owner")
			if err != nil {
				return err
			}
			id, key, secret, err := client.CreateSystem(regKey, orgID)
			if err != nil {
				return fmt.Errorf("create system %s: %w", regKey, err)
			}
			if fs.Register {
				if _, err := client.RegisterSystem(secret); err != nil {
					return fmt.Errorf("system %s created but registration failed: %w", regKey, err)
				}
			}
			orgName := fs.Org
			if fs.Org != "owner" {
				orgName = reg.Orgs[fixtureRegKey(prefix, fs.Org)].Name
			}
			reg.Systems[regKey] = System{
				Name: regKey, ID: id, SystemKey: key, Secret: secret,
				OrgID: orgID, OrgName: orgName, CreatedAt: time.Now().UTC(),
			}
			if err := reg.Save(); err != nil {
				return err
			}
			delete(todoSystems, fs.Key)
			progress = true
			fmt.Printf("  system %-26s in %-16s registered=%v\n", regKey, orgName, fs.Register)
		}
	}

	if len(todoOrgs)+len(todoUsers)+len(todoSystems) > 0 {
		return fmt.Errorf("could not resolve dependencies for orgs=%v users=%v systems=%v "+
			"(a created_by refers to a user that is never created?)",
			keysOf(todoOrgs), keysOf(todoUsers), keysOf(todoSystems))
	}

	fmt.Println("\nFixture ready. Next: apitool authz personas   (verifies token permissions match config.yml)")
	return nil
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// authzTeardown removes the fixture in reverse dependency order. Users and
// systems are destroyed rather than soft-deleted so a later provision can reuse
// the same email addresses; pass --soft to keep them recoverable.
func authzTeardown(dir string, reg *Registry, flags map[string]string) error {
	spec, err := loadAuthzSpec(dir)
	if err != nil {
		return err
	}
	if flags["yes"] != "true" {
		return fmt.Errorf("this deletes every %q fixture org, user and system — re-run with --yes", spec.Fixture.Prefix)
	}
	hard := flags["soft"] != "true"
	prefix := spec.Fixture.Prefix
	client, err := loginAs(reg, "owner")
	if err != nil {
		return err
	}

	fmt.Println("=== Systems ===")
	for _, fs := range spec.Fixture.Systems {
		regKey := fixtureRegKey(prefix, fs.Key)
		s, ok := reg.Systems[regKey]
		if !ok {
			continue
		}
		if err := client.DeleteSystem(s.ID); err != nil {
			fmt.Printf("  %-28s FAILED: %v\n", regKey, err)
			continue
		}
		delete(reg.Systems, regKey)
		fmt.Printf("  %-28s deleted\n", regKey)
	}
	if err := reg.Save(); err != nil {
		return err
	}

	fmt.Println("\n=== Users ===")
	for i := len(spec.Fixture.Users) - 1; i >= 0; i-- {
		regKey := fixtureRegKey(prefix, spec.Fixture.Users[i].Key)
		u, ok := reg.Users[regKey]
		if !ok {
			continue
		}
		var err error
		if hard {
			err = client.DestroyUser(u.LogtoID)
		} else {
			err = client.DeleteUser(u.LogtoID)
		}
		if err != nil {
			fmt.Printf("  %-28s FAILED: %v\n", regKey, err)
			continue
		}
		delete(reg.Users, regKey)
		fmt.Printf("  %-28s removed\n", regKey)
	}
	if err := reg.Save(); err != nil {
		return err
	}

	fmt.Println("\n=== Organizations ===")
	for i := len(spec.Fixture.Orgs) - 1; i >= 0; i-- {
		fo := spec.Fixture.Orgs[i]
		regKey := fixtureRegKey(prefix, fo.Key)
		o, ok := reg.Orgs[regKey]
		if !ok {
			continue
		}
		if err := client.DeleteOrg(fo.Type, o.LogtoID); err != nil {
			fmt.Printf("  %-28s FAILED: %v\n", regKey, err)
			continue
		}
		delete(reg.Orgs, regKey)
		fmt.Printf("  %-28s removed\n", regKey)
	}
	return reg.Save()
}

// syntheticVAT produces a distinct, well-formed 12-digit VAT per fixture org.
// Real validation only checks shape for these test entities.
func syntheticVAT(i int) string {
	return fmt.Sprintf("9%011d", 90000000000+int64(i))[:12]
}

func renderEmail(template, key string) string {
	if template == "" {
		template = "authz+{key}@example.com"
	}
	return strings.ReplaceAll(template, "{key}", key)
}
