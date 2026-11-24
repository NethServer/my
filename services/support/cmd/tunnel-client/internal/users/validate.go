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
	"log"
	"regexp"
)

var (
	supportUsernameRegex = regexp.MustCompile(`^support-[a-z0-9-]+$`)
	passwordRegex        = regexp.MustCompile(`^[A-Za-z0-9!@#%^&*]{12,128}$`)
	moduleIDRegex        = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}[0-9]$`)
	domainRegex          = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.-]{0,251}[A-Za-z0-9])?$`)
)

// IsSupportUsername reports whether name is a valid ephemeral support username.
func IsSupportUsername(name string) bool {
	return len(name) <= 64 && supportUsernameRegex.MatchString(name)
}

func validCredential(username, password string) bool {
	return IsSupportUsername(username) && passwordRegex.MatchString(password)
}

// SanitizeFetched returns the subset of server-supplied NS8 credentials that
// is safe to hand to users.d plugins. Anything that does not look like an
// identity this software would generate is dropped. The result is never
// recorded in the state file nor passed to Delete.
func SanitizeFetched(fetched *SessionUsers, sessionID string) *SessionUsers {
	if fetched == nil {
		return nil
	}
	out := &SessionUsers{
		SessionID: sessionID,
		Platform:  "ns8",
		CreatedAt: fetched.CreatedAt,
	}
	if ca := fetched.ClusterAdmin; ca != nil {
		if validCredential(ca.Username, ca.Password) {
			out.ClusterAdmin = &UserCredential{Username: ca.Username, Password: ca.Password}
		} else {
			log.Println("Discarding fetched cluster-admin credential: invalid username or password")
		}
	}
	for _, du := range fetched.DomainUsers {
		if !validCredential(du.Username, du.Password) || !moduleIDRegex.MatchString(du.Module) || !domainRegex.MatchString(du.Domain) {
			log.Println("Discarding fetched domain user: invalid username, password, module or domain")
			continue
		}
		out.DomainUsers = append(out.DomainUsers, du)
	}
	if len(fetched.LocalUsers) > 0 {
		log.Println("Discarding fetched local users: not supported on NS8")
	}
	if !HasIdentities(out) {
		return nil
	}
	return out
}
