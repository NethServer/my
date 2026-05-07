/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

// apitool is a dev CLI that produces real, hierarchy-aware backend JWTs by
// running the full OIDC login + /auth/exchange flow (the same a browser does).
// It also creates test orgs/users autonomously and persists their credentials
// in backend/.api-registry.json so subsequent token requests are one command.
//
// Replaces cmd/gentoken (which only signed locally and produced tokens whose
// embedded org_id had no counterpart in the database).
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	defaultLogtoEndpoint = "https://o3izgd.logto.app"
	defaultLogtoAppID    = "dkmw3j3ansfj0wybhhgjr"
	defaultAuthBaseURL   = "https://my.localtest.me"
	defaultBackendURL    = "https://my.localtest.me/api"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "init":
		err = cmdInit(args)
	case "token":
		err = cmdToken(args)
	case "create-org":
		err = cmdCreateOrg(args)
	case "create-user":
		err = cmdCreateUser(args)
	case "list":
		err = cmdList(args)
	case "delete-user":
		err = cmdDeleteUser(args)
	case "delete-org":
		err = cmdDeleteOrg(args)
	case "create-system":
		err = cmdCreateSystem(args)
	case "cleanup-orphans":
		err = cmdCleanupOrphans(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		usage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `apitool — manage real OIDC test tokens via login + exchange

Usage:
  apitool init
      Set/refresh owner credentials and OIDC config.

  apitool token <name>
      Print a fresh JWT for a registered user. Use "owner" for the owner.

  apitool create-org <type> <name> --vat=<12 digits> [--description=...]
                                    [--data-<key>=<value>] [--as=<user-key>]
      Create distributor|reseller|customer.
      Repeat --data-<key>=<value> for any extra custom_data field
      (e.g. --data-address='Via Roma 1' --data-language=it --data-email=...).
      --as  acts as a non-owner user (e.g., the parent's admin), so the new
            org becomes a child in the caller's hierarchy. Default: owner.

  apitool create-user --org=<name> --email=<email> --name=<name>
                       [--role=Admin] [--username=...] [--key=<reg-name>]
                       [--as=<user-key>]
      Create a user under an org. Generates a strong password, fixes it via
      PATCH /users/:id/password and saves credentials to the registry.
      --key  alias under which the user is saved (default: email).
      --as   acts as a non-owner user. Default: owner.

  apitool list
      Show registered config, owner, orgs and users.

  apitool delete-user <key>
      Soft-delete a registered user and remove from registry.

  apitool delete-org <name>
      Soft-delete a registered org and remove from registry. Will fail if it
      still has child orgs/users; clean those out first.

  apitool create-system --org=<name> <system-name>
      Create a system under a customer org. Prints the system_key.

  apitool cleanup-orphans --org=<name>
      Soft-delete every user listed in <org> whose email is NOT in registry.
      Useful to clean up users left over from earlier failed runs.

Registry: backend/.api-registry.json (gitignored, file mode 0600)`)
}

func loadOrInit() (*Registry, error) {
	r, err := LoadRegistry()
	if err != nil {
		return nil, err
	}
	if r.Config.LogtoEndpoint == "" {
		r.Config = Config{
			LogtoEndpoint: defaultLogtoEndpoint,
			LogtoAppID:    defaultLogtoAppID,
			AuthBaseURL:   defaultAuthBaseURL,
			BackendURL:    defaultBackendURL,
		}
	}
	return r, nil
}

// loginAs returns an authenticated client logged in as the given registry key.
// "" or "owner" means use the saved owner credentials.
func loginAs(r *Registry, key string) (*Client, error) {
	var email, password string
	if key == "" || key == "owner" {
		if r.Owner.Email == "" {
			return nil, fmt.Errorf("not initialized; run: apitool init")
		}
		email = r.Owner.Email
		password = r.Owner.Password
	} else {
		u, ok := r.Users[key]
		if !ok {
			return nil, fmt.Errorf("user %q not found in registry", key)
		}
		email = u.Email
		password = u.Password
	}
	client, err := NewClient(r.Config)
	if err != nil {
		return nil, err
	}
	if err := client.Login(email, password); err != nil {
		return nil, fmt.Errorf("login as %q failed: %w", key, err)
	}
	return client, nil
}

func cmdInit(_ []string) error {
	r, err := loadOrInit()
	if err != nil {
		return err
	}

	fmt.Println("Configuring apitool registry.")
	fmt.Println("Owner credentials are required to create orgs and users.")
	fmt.Println("They will be saved (in cleartext) to", registryPath())
	fmt.Println()

	email := prompt("Owner email", r.Owner.Email)
	pass := prompt("Owner password", "")
	if email == "" || pass == "" {
		return fmt.Errorf("email and password are required")
	}
	r.Owner.Email = email
	r.Owner.Password = pass

	fmt.Print("Verifying credentials... ")
	client, err := NewClient(r.Config)
	if err != nil {
		fmt.Println("FAILED")
		return err
	}
	if err := client.Login(email, pass); err != nil {
		fmt.Println("FAILED")
		return err
	}
	fmt.Println("OK")

	if err := r.Save(); err != nil {
		return err
	}
	fmt.Println("Registry saved to", registryPath())
	return nil
}

func cmdToken(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: apitool token <name>")
	}
	name := args[0]
	r, err := loadOrInit()
	if err != nil {
		return err
	}

	var email, pass string
	if name == "owner" {
		if r.Owner.Email == "" {
			return fmt.Errorf("owner not initialized; run: apitool init")
		}
		email = r.Owner.Email
		pass = r.Owner.Password
	} else {
		u, ok := r.Users[name]
		if !ok {
			return fmt.Errorf("user %q not found in registry (run: apitool list)", name)
		}
		email = u.Email
		pass = u.Password
	}

	client, err := NewClient(r.Config)
	if err != nil {
		return err
	}
	if err := client.Login(email, pass); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	fmt.Println(client.JWT())
	return nil
}

func cmdCreateOrg(args []string) error {
	flags, pos := parseFlags(args)
	if len(pos) < 2 {
		return fmt.Errorf("usage: apitool create-org <type> <name> [--vat=...] [--description=...]")
	}
	orgType := pos[0]
	name := pos[1]
	if !validOrgType(orgType) {
		return fmt.Errorf("invalid org type %q (use distributor|reseller|customer)", orgType)
	}

	r, err := loadOrInit()
	if err != nil {
		return err
	}

	customData := map[string]interface{}{}
	if vat := flags["vat"]; vat != "" {
		customData["vat"] = vat
	}
	if customData["vat"] == nil {
		return fmt.Errorf("--vat=<12 digits> is required for all org types")
	}
	// Free-form custom_data fields via --data-<key>=<value>. Anything beyond
	// the dedicated flags (vat, description, as) goes through this prefix so
	// new fields don't require parser changes.
	for k, v := range flags {
		if key, ok := strings.CutPrefix(k, "data-"); ok && key != "" {
			customData[key] = v
		}
	}

	client, err := loginAs(r, flags["as"])
	if err != nil {
		return err
	}

	logtoID, err := client.CreateOrg(orgType, name, flags["description"], customData)
	if err != nil {
		return err
	}

	r.Orgs[name] = Org{
		Type:      orgType,
		LogtoID:   logtoID,
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	if err := r.Save(); err != nil {
		return err
	}
	fmt.Printf("Created %s %q (logto_id=%s) as %q\n", orgType, name, logtoID, defaultAs(flags["as"]))
	return nil
}

func cmdCreateUser(args []string) error {
	flags, _ := parseFlags(args)

	orgKey := flags["org"]
	email := flags["email"]
	name := flags["name"]
	roleName := flags["role"]
	username := flags["username"]
	regKey := flags["key"]

	if orgKey == "" || email == "" || name == "" {
		return fmt.Errorf("usage: apitool create-user --org=<name> --email=<email> --name=<name> [--role=Admin] [--username=...] [--key=<registry-name>]")
	}
	if roleName == "" {
		roleName = "Admin"
	}

	r, err := loadOrInit()
	if err != nil {
		return err
	}
	org, ok := r.Orgs[orgKey]
	if !ok {
		return fmt.Errorf("org %q not in registry (run: apitool list)", orgKey)
	}

	client, err := loginAs(r, flags["as"])
	if err != nil {
		return err
	}

	roles, err := client.GetRoles()
	if err != nil {
		return err
	}
	roleID, ok := roles[roleName]
	if !ok {
		var names []string
		for k := range roles {
			names = append(names, k)
		}
		sort.Strings(names)
		return fmt.Errorf("role %q not found (available: %s)", roleName, strings.Join(names, ", "))
	}

	pw, err := generatePassword()
	if err != nil {
		return err
	}

	userID, err := client.CreateUser(email, name, username, org.LogtoID, []string{roleID})
	if err != nil {
		return err
	}
	if err := client.ResetPassword(userID, pw); err != nil {
		return fmt.Errorf("user created (id=%s) but password reset failed: %w", userID, err)
	}

	if regKey == "" {
		regKey = email
	}
	r.Users[regKey] = User{
		Email:     email,
		Username:  username,
		Password:  pw,
		LogtoID:   userID,
		OrgRole:   org.Type,
		OrgID:     org.LogtoID,
		OrgName:   org.Name,
		CreatedAt: time.Now().UTC(),
	}
	if err := r.Save(); err != nil {
		return err
	}
	fmt.Printf("Created user %q (logto_id=%s)\n", email, userID)
	fmt.Printf("Registry key: %s\n", regKey)
	return nil
}

func cmdList(_ []string) error {
	r, err := loadOrInit()
	if err != nil {
		return err
	}
	fmt.Println("Registry:", registryPath())
	fmt.Println()
	fmt.Println("=== Config ===")
	fmt.Printf("  logto_endpoint: %s\n", r.Config.LogtoEndpoint)
	fmt.Printf("  logto_app_id:   %s\n", r.Config.LogtoAppID)
	fmt.Printf("  backend_url:    %s\n", r.Config.BackendURL)
	fmt.Println()
	fmt.Println("=== Owner ===")
	if r.Owner.Email != "" {
		fmt.Printf("  email: %s\n", r.Owner.Email)
	} else {
		fmt.Println("  (not initialized — run: apitool init)")
	}
	fmt.Println()
	fmt.Println("=== Orgs ===")
	if len(r.Orgs) == 0 {
		fmt.Println("  (none)")
	} else {
		var keys []string
		for k := range r.Orgs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			o := r.Orgs[k]
			fmt.Printf("  %-30s %-12s %s\n", o.Name, o.Type, o.LogtoID)
		}
	}
	fmt.Println()
	fmt.Println("=== Users ===")
	if len(r.Users) == 0 {
		fmt.Println("  (none)")
	} else {
		var keys []string
		for k := range r.Users {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			u := r.Users[k]
			fmt.Printf("  %-30s %-30s %-12s in %s\n", k, u.Email, u.OrgRole, u.OrgName)
		}
	}
	return nil
}

func cmdDeleteUser(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: apitool delete-user <key>")
	}
	key := args[0]
	r, err := loadOrInit()
	if err != nil {
		return err
	}
	u, ok := r.Users[key]
	if !ok {
		return fmt.Errorf("user %q not in registry", key)
	}
	client, err := loginAs(r, "")
	if err != nil {
		return err
	}
	if err := client.DeleteUser(u.LogtoID); err != nil {
		return err
	}
	delete(r.Users, key)
	if err := r.Save(); err != nil {
		return err
	}
	fmt.Printf("Deleted user %q (logto_id=%s)\n", key, u.LogtoID)
	return nil
}

func cmdDeleteOrg(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: apitool delete-org <name>")
	}
	name := args[0]
	r, err := loadOrInit()
	if err != nil {
		return err
	}
	org, ok := r.Orgs[name]
	if !ok {
		return fmt.Errorf("org %q not in registry", name)
	}
	client, err := loginAs(r, "")
	if err != nil {
		return err
	}
	if err := client.DeleteOrg(org.Type, org.LogtoID); err != nil {
		return err
	}
	delete(r.Orgs, name)
	if err := r.Save(); err != nil {
		return err
	}
	fmt.Printf("Deleted %s %q (logto_id=%s)\n", org.Type, name, org.LogtoID)
	return nil
}

func cmdCreateSystem(args []string) error {
	flags, pos := parseFlags(args)
	orgKey := flags["org"]
	if orgKey == "" || len(pos) < 1 {
		return fmt.Errorf("usage: apitool create-system --org=<name> <system-name>")
	}
	systemName := pos[0]
	r, err := loadOrInit()
	if err != nil {
		return err
	}
	org, ok := r.Orgs[orgKey]
	if !ok {
		return fmt.Errorf("org %q not in registry", orgKey)
	}
	client, err := loginAs(r, flags["as"])
	if err != nil {
		return err
	}
	systemKey, err := client.CreateSystem(systemName, org.LogtoID)
	if err != nil {
		return err
	}
	fmt.Printf("Created system %q in org %q (system_key=%s)\n", systemName, orgKey, systemKey)
	return nil
}

func cmdCleanupOrphans(args []string) error {
	flags, _ := parseFlags(args)
	orgKey := flags["org"]
	if orgKey == "" {
		return fmt.Errorf("usage: apitool cleanup-orphans --org=<name>")
	}
	r, err := loadOrInit()
	if err != nil {
		return err
	}
	org, ok := r.Orgs[orgKey]
	if !ok {
		return fmt.Errorf("org %q not in registry", orgKey)
	}
	client, err := loginAs(r, "")
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, u := range r.Users {
		if u.OrgID == org.LogtoID {
			known[strings.ToLower(u.Email)] = true
		}
	}
	users, err := client.ListUsersInOrg(org.LogtoID)
	if err != nil {
		return err
	}
	for _, u := range users {
		if known[strings.ToLower(u.Email)] {
			continue
		}
		if err := client.DeleteUser(u.LogtoID); err != nil {
			fmt.Printf("  FAILED to delete %s: %v\n", u.Email, err)
			continue
		}
		fmt.Printf("  deleted %s (logto_id=%s)\n", u.Email, u.LogtoID)
	}
	return nil
}

func defaultAs(s string) string {
	if s == "" {
		return "owner"
	}
	return s
}

// parseFlags splits "--key=value" args from positional ones. A bare "--flag"
// becomes flags["flag"]="true".
func parseFlags(args []string) (map[string]string, []string) {
	flags := map[string]string{}
	var pos []string
	for _, a := range args {
		if strings.HasPrefix(a, "--") {
			kv := strings.SplitN(a[2:], "=", 2)
			if len(kv) == 2 {
				flags[kv[0]] = kv[1]
			} else {
				flags[kv[0]] = "true"
			}
		} else {
			pos = append(pos, a)
		}
	}
	return flags, pos
}

// generatePassword produces a strong password matching the backend policy
// (12+ chars, upper/lower/digit/special, no 4+ repeats, no weak ascending
// triplets like "abc" or "123" — case-insensitive). Fixed prefix supplies
// the 4 character classes; entropy comes from random bytes encoded as
// URL-safe base64. Retries until the candidate has no forbidden triplet.
func generatePassword() (string, error) {
	for attempt := 0; attempt < 64; attempt++ {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		candidate := "Aa9!" + base64.RawURLEncoding.EncodeToString(b)
		if !hasAscendingTriplet(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not generate clean password after 64 attempts")
}

// hasAscendingTriplet reports whether the password (case-insensitive) contains
// any 3-character ascending alphabetical or numeric sequence the backend's
// password validator rejects (012-890, abc-xyz).
func hasAscendingTriplet(s string) bool {
	low := strings.ToLower(s)
	for i := 0; i+2 < len(low); i++ {
		a, b, c := low[i], low[i+1], low[i+2]
		isAlpha := a >= 'a' && a <= 'z' && b >= 'a' && b <= 'z' && c >= 'a' && c <= 'z'
		isDigit := a >= '0' && a <= '9' && b >= '0' && b <= '9' && c >= '0' && c <= '9'
		if (isAlpha || isDigit) && b == a+1 && c == b+1 {
			return true
		}
	}
	return false
}

// stdinReader is shared across prompt() calls so a buffered Reader doesn't
// swallow input that belongs to subsequent prompts when stdin is a pipe.
var stdinReader = bufio.NewReader(os.Stdin)

func prompt(label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, err := stdinReader.ReadString('\n')
	if err != nil && line == "" {
		return def
	}
	s := strings.TrimSpace(line)
	if s == "" {
		return def
	}
	return s
}
