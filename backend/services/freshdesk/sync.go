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

// Rename is a contact whose name differs from the my user with the same email
type Rename struct {
	Contact Contact
	Name    string
}

// NamesByEmail indexes the display names of the users that have both an
// email and a name; the first user wins on a duplicate email
func NamesByEmail(users []models.LogtoUser) map[string]string {
	names := make(map[string]string, len(users))
	for _, user := range users {
		email := normalizeEmail(user.PrimaryEmail)
		name := strings.TrimSpace(user.Name)
		if email == "" || name == "" {
			continue
		}
		if _, seen := names[email]; !seen {
			names[email] = name
		}
	}
	return names
}

// PlanRenames matches the contacts to the names by email and keeps the ones
// whose name differs, ordered by email
func PlanRenames(contacts []Contact, names map[string]string) []Rename {
	var renames []Rename
	for _, contact := range contacts {
		name, ok := names[normalizeEmail(contact.Email)]
		if !ok || strings.TrimSpace(contact.Name) == name {
			continue
		}
		renames = append(renames, Rename{Contact: contact, Name: name})
	}
	sort.Slice(renames, func(i, j int) bool {
		return normalizeEmail(renames[i].Contact.Email) < normalizeEmail(renames[j].Contact.Email)
	})
	return renames
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
