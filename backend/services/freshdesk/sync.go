/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package freshdesk

import (
	"sort"
	"strings"

	"github.com/nethesis/my/backend/models"
)

// Identity is the username and display name of a my user
type Identity struct {
	Username string
	Name     string
}

// Rename is a contact still named after the username of the my user with
// the same email
type Rename struct {
	Contact Contact
	Name    string
}

// IdentitiesByEmail indexes the users that have an email, a username and a
// name; the first user wins on a duplicate email
func IdentitiesByEmail(users []models.LogtoUser) map[string]Identity {
	identities := make(map[string]Identity, len(users))
	for _, user := range users {
		email := normalizeEmail(user.PrimaryEmail)
		identity := Identity{
			Username: strings.TrimSpace(user.Username),
			Name:     strings.Join(strings.Fields(user.Name), " "),
		}
		if email == "" || identity.Username == "" || identity.Name == "" {
			continue
		}
		if _, seen := identities[email]; !seen {
			identities[email] = identity
		}
	}
	return identities
}

// PlanRenames keeps the contacts whose name is still the username of the my
// user with the same email, ordered by email; a name edited by hand in the
// helpdesk is left alone
func PlanRenames(contacts []Contact, identities map[string]Identity) []Rename {
	var renames []Rename
	for _, contact := range contacts {
		identity, ok := identities[normalizeEmail(contact.Email)]
		if !ok {
			continue
		}
		name := strings.TrimSpace(contact.Name)
		if !strings.EqualFold(name, identity.Username) || name == identity.Name {
			continue
		}
		renames = append(renames, Rename{Contact: contact, Name: identity.Name})
	}
	sort.Slice(renames, func(i, j int) bool {
		return normalizeEmail(renames[i].Contact.Email) < normalizeEmail(renames[j].Contact.Email)
	})
	return renames
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
