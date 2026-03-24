/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package users

import (
	"fmt"
	"log"
	"os/exec"
	"time"
)

// NethSecurityProvisioner creates ephemeral users on NethSecurity (OpenWrt):
//   - A local user via the nethsec Python module
//   - Promoted to admin via ubus call ns.users set-admin
type NethSecurityProvisioner struct{}

func (p *NethSecurityProvisioner) Create(sessionID string) (*SessionUsers, error) {
	username := generateUsername(sessionID)
	password := generatePassword()
	result := &SessionUsers{
		SessionID: sessionID,
		Platform:  "nethsecurity",
		CreatedAt: time.Now(),
	}

	// Create local user via Python (ubus has timeout issues)
	if err := p.addLocalUser(username, password); err != nil {
		return result, fmt.Errorf("failed to create local user: %w", err)
	}

	// Promote to admin (web UI access)
	if err := p.setAdmin(username); err != nil {
		// Rollback: remove the user we just created
		_ = p.deleteLocalUser(username)
		return result, fmt.Errorf("failed to set admin role: %w", err)
	}

	result.LocalUsers = append(result.LocalUsers, UserCredential{
		Username: username,
		Password: password,
	})
	log.Printf("NethSecurity user provisioning: admin user %q created", username)

	return result, nil
}

func (p *NethSecurityProvisioner) Delete(users *SessionUsers) error {
	if users == nil {
		return nil
	}

	for _, u := range users.LocalUsers {
		if err := p.removeAdmin(u.Username); err != nil {
			log.Printf("NethSecurity user cleanup: failed to remove admin role for %q: %v", u.Username, err)
		}
		if err := p.deleteLocalUser(u.Username); err != nil {
			log.Printf("NethSecurity user cleanup: failed to delete user %q: %v", u.Username, err)
		} else {
			log.Printf("NethSecurity user cleanup: user %q removed", u.Username)
		}
	}

	return nil
}

func (p *NethSecurityProvisioner) addLocalUser(username, password string) error {
	script := fmt.Sprintf(`
from nethsec import users
from euci import EUci
u = EUci()
users.add_local_user(u, %q, %q, "Support Session", "main")
`, username, password)
	return runPython(script)
}

func (p *NethSecurityProvisioner) setAdmin(username string) error {
	script := fmt.Sprintf(`
from nethsec import users
from euci import EUci
u = EUci()
users.set_admin(u, %q, "main")
`, username)
	return runPython(script)
}

func (p *NethSecurityProvisioner) removeAdmin(username string) error {
	script := fmt.Sprintf(`
from nethsec import users
from euci import EUci
u = EUci()
users.remove_admin(u, %q)
`, username)
	return runPython(script)
}

func (p *NethSecurityProvisioner) deleteLocalUser(username string) error {
	script := fmt.Sprintf(`
from nethsec import users
from euci import EUci
u = EUci()
users.delete_local_user(u, %q, "main")
`, username)
	return runPython(script)
}

// runPython executes a Python3 script on NethSecurity.
func runPython(script string) error {
	cmd := exec.Command("python3", "-c", script) //nolint:gosec // python3 is a trusted system binary
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python3 failed: %w: %s", err, output)
	}
	return nil
}
