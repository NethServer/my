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
	"unicode"

	"github.com/nethesis/my/backend/models"
)

// Identity is the username and display name of a my user
type Identity struct {
	UserID   string
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
			UserID:   user.ID,
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

// Orphan is a contact with no company whose email belongs to a my user
type Orphan struct {
	Contact Contact
	UserID  string
}

// PlanOrphans keeps the contacts with no company that are my users, ordered
// by email; a company set in the helpdesk is left alone
func PlanOrphans(contacts []Contact, identities map[string]Identity) []Orphan {
	var orphans []Orphan
	seen := make(map[int64]bool)
	for _, contact := range contacts {
		identity, ok := identities[normalizeEmail(contact.Email)]
		if !ok || contact.CompanyID != nil || seen[contact.ID] {
			continue
		}
		seen[contact.ID] = true
		orphans = append(orphans, Orphan{Contact: contact, UserID: identity.UserID})
	}
	sort.Slice(orphans, func(i, j int) bool {
		return normalizeEmail(orphans[i].Contact.Email) < normalizeEmail(orphans[j].Contact.Email)
	})
	return orphans
}

// CompanyIndex finds a Freshdesk company by VAT number or, failing that, by
// name
type CompanyIndex struct {
	byVAT  map[string]Company
	byName map[string]Company
}

// NewCompanyIndex indexes the companies; the first one wins on a duplicate
func NewCompanyIndex(companies []Company) *CompanyIndex {
	index := &CompanyIndex{byVAT: make(map[string]Company), byName: make(map[string]Company)}
	for _, company := range companies {
		index.Add(company)
	}
	return index
}

// Add indexes one more company
func (i *CompanyIndex) Add(company Company) {
	if vat := normalizeVAT(company.CustomFields.VAT); vat != "" {
		if _, seen := i.byVAT[vat]; !seen {
			i.byVAT[vat] = company
		}
	}
	if name := normalizeName(company.Name); name != "" {
		if _, seen := i.byName[name]; !seen {
			i.byName[name] = company
		}
	}
}

// Find returns the company of an organization: the VAT number decides, the
// name only when no company has that VAT number
func (i *CompanyIndex) Find(vat, name string) (Company, bool) {
	if vat = normalizeVAT(vat); vat != "" {
		if company, ok := i.byVAT[vat]; ok {
			return company, true
		}
	}
	company, ok := i.byName[normalizeName(name)]
	return company, ok
}

func normalizeVAT(vat string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, vat)
}

func normalizeName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}
