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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// NethSecurityProvisioner creates ephemeral users on NethSecurity (OpenWrt):
//   - A local user via the nethsec Python module
//   - Promoted to admin via ubus call ns.users set-admin
type NethSecurityProvisioner struct{}

func (p *NethSecurityProvisioner) Create(sessionID string, record RecordFunc) (*SessionUsers, error) {
	username := generateUsername(sessionID)
	password := generatePassword()
	result := &SessionUsers{
		SessionID: sessionID,
		Platform:  "nethsecurity",
		CreatedAt: time.Now(),
	}

	// Remove existing user first to ensure password matches on restart
	_ = p.removeAdmin(username)
	_ = p.deleteLocalUser(username)

	// Create local user via Python (ubus has timeout issues)
	if err := p.addLocalUser(username, password); err != nil {
		return result, fmt.Errorf("failed to create local user: %w", err)
	}

	result.LocalUsers = append(result.LocalUsers, UserCredential{
		Username: username,
		Password: password,
	})
	if err := record(result); err != nil {
		result.LocalUsers = nil
		if delErr := p.deleteLocalUser(username); delErr != nil {
			log.Printf("NethSecurity user provisioning: rollback of unrecorded user %q failed: %v", username, delErr)
		}
		return result, fmt.Errorf("cannot record local user: %w", err)
	}

	// Promote to admin (web UI access)
	if err := p.setAdmin(username); err != nil {
		if delErr := p.Delete(result); delErr != nil {
			log.Printf("NethSecurity user provisioning: rollback of %q failed: %v", username, delErr)
		}
		if recErr := record(result); recErr != nil {
			log.Printf("NethSecurity user provisioning: cannot update users state: %v", recErr)
		}
		return result, fmt.Errorf("failed to set admin role: %w", err)
	}

	log.Printf("NethSecurity user provisioning: admin user %q created", username)
	return result, nil
}

func (p *NethSecurityProvisioner) Delete(users *SessionUsers) error {
	if users == nil {
		return nil
	}

	var errs []error
	var remaining []UserCredential
	for _, u := range users.LocalUsers {
		if err := p.removeAdmin(u.Username); err != nil {
			log.Printf("NethSecurity user cleanup: failed to remove admin role for %q: %v", u.Username, err)
		}
		if err := p.deleteLocalUser(u.Username); err != nil {
			log.Printf("NethSecurity user cleanup: failed to delete user %q: %v", u.Username, err)
			errs = append(errs, err)
			remaining = append(remaining, u)
		} else {
			log.Printf("NethSecurity user cleanup: user %q removed", u.Username)
		}
	}
	users.LocalUsers = remaining

	return errors.Join(errs...)
}

// nethsecScript runs one nethsec.users call. Arguments are read as JSON from
// stdin so the password never appears in the process argv.
const nethsecScript = `
import json, sys
from nethsec import users
from euci import EUci
req = json.load(sys.stdin)
u = EUci()
op = req["op"]
name = req["username"]
if op == "add":
    users.add_local_user(u, name, req["password"], "Support Session", "main")
elif op == "set_admin":
    users.set_admin(u, name, "main")
elif op == "remove_admin":
    users.remove_admin(u, name)
elif op == "delete":
    users.delete_local_user(u, name, "main")
else:
    sys.exit("unknown op")
`

func (p *NethSecurityProvisioner) addLocalUser(username, password string) error {
	return runNethsec(map[string]string{"op": "add", "username": username, "password": password})
}

func (p *NethSecurityProvisioner) setAdmin(username string) error {
	return runNethsec(map[string]string{"op": "set_admin", "username": username})
}

func (p *NethSecurityProvisioner) removeAdmin(username string) error {
	return runNethsec(map[string]string{"op": "remove_admin", "username": username})
}

func (p *NethSecurityProvisioner) deleteLocalUser(username string) error {
	return runNethsec(map[string]string{"op": "delete", "username": username})
}

// runNethsec executes nethsecScript with req on stdin.
func runNethsec(req map[string]string) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	cmd := exec.Command("python3", "-c", nethsecScript) //nolint:gosec // python3 is a trusted system binary
	cmd.Stdin = bytes.NewReader(data)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python3 failed: %w: %s", err, output)
	}
	return nil
}
