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
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

// RunContactsSync is the entry point of `backend freshdesk-contacts`: it
// renames the helpdesk contacts still named after the Logto username to the
// display name of the my user with the same email, gives the contacts with no
// company the one of their my organization, and returns the process exit
// code. Freshworks copies the username claim into the contact name at every
// login and carries no organization, so the helpdesk is corrected from here.
func RunContactsSync(args []string) int {
	flags := flag.NewFlagSet("freshdesk-contacts", flag.ContinueOnError)
	apply := flags.Bool("apply", false, "write the names to Freshdesk (default: log the plan only)")
	email := flags.String("email", "", "limit the run to the contact with this email")
	delay := flags.Duration("delay", 200*time.Millisecond, "pause between writes")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	log := logger.ComponentLogger("freshdesk-contacts")

	domain := os.Getenv("FRESHDESK_DOMAIN")
	apiKey := os.Getenv("FRESHDESK_API_KEY")
	tenant := os.Getenv("LOGTO_TENANT_ID")
	clientID := os.Getenv("LOGTO_BACKEND_APP_ID")
	clientSecret := os.Getenv("LOGTO_BACKEND_APP_SECRET")
	if domain == "" || apiKey == "" || tenant == "" || clientID == "" || clientSecret == "" {
		log.Error().Msg("FRESHDESK_DOMAIN, FRESHDESK_API_KEY, LOGTO_TENANT_ID, LOGTO_BACKEND_APP_ID and LOGTO_BACKEND_APP_SECRET are required")
		return 2
	}
	configuration.Config.LogtoIssuer = fmt.Sprintf("https://%s.logto.app", tenant)
	configuration.Config.LogtoManagementBaseURL = configuration.Config.LogtoIssuer + "/api"
	configuration.Config.LogtoManagementClientID = clientID
	configuration.Config.LogtoManagementClientSecret = clientSecret

	management := logto.NewManagementClient()
	users, err := management.GetAllUsers()
	if err != nil {
		log.Error().Err(err).Msg("cannot list Logto users")
		return 2
	}
	identities := IdentitiesByEmail(users)

	client := NewClient(domain, apiKey)
	var contacts []Contact
	if *email != "" {
		contacts, err = client.FindContactsByEmail(*email)
	} else {
		contacts, err = client.ListContacts()
	}
	if err != nil {
		log.Error().Err(err).Msg("cannot list Freshdesk contacts")
		return 2
	}

	renames := PlanRenames(contacts, identities)
	var renamed, failed int
	for _, rename := range renames {
		if !*apply {
			withRename(log.Info(), rename).Msg("contact to rename")
			continue
		}
		if err := client.UpdateContactName(rename.Contact.ID, rename.Name); err != nil {
			failed++
			withRename(log.Error().Err(err), rename).Msg("contact rename failed")
			continue
		}
		renamed++
		withRename(log.Info(), rename).Msg("contact renamed")
		sleep(*delay)
	}

	companies := syncCompanies(log, client, management, PlanOrphans(contacts, identities), *apply, *delay)
	failed += companies.failed

	log.Info().
		Bool("apply", *apply).
		Int("users", len(identities)).
		Int("contacts", len(contacts)).
		Int("to_rename", len(renames)).
		Int("renamed", renamed).
		Int("without_company", companies.orphans).
		Int("companies_created", companies.created).
		Int("companies_assigned", companies.assigned).
		Int("failed", failed).
		Msg("freshdesk contacts sync done")
	if failed > 0 {
		return 1
	}
	return 0
}

func withRename(event *zerolog.Event, rename Rename) *zerolog.Event {
	return event.
		Int64("contact_id", rename.Contact.ID).
		Str("email", rename.Contact.Email).
		Str("from", rename.Contact.Name).
		Str("to", rename.Name)
}

type companiesResult struct {
	orphans, created, assigned, failed int
}

// syncCompanies gives each orphan the Freshdesk company of its my
// organization, creating the company when the helpdesk has none
func syncCompanies(log *zerolog.Logger, client *Client, management *logto.LogtoManagementClient, orphans []Orphan, apply bool, delay time.Duration) companiesResult {
	result := companiesResult{orphans: len(orphans)}
	if len(orphans) == 0 {
		return result
	}

	listed, err := client.ListCompanies()
	if err != nil {
		log.Error().Err(err).Msg("cannot list Freshdesk companies")
		result.failed++
		return result
	}
	index := NewCompanyIndex(listed)

	for _, orphan := range orphans {
		event := func(e *zerolog.Event) *zerolog.Event {
			return e.Int64("contact_id", orphan.Contact.ID).Str("email", orphan.Contact.Email)
		}

		orgs, err := management.GetUserOrganizations(orphan.UserID)
		if err != nil {
			result.failed++
			event(log.Error().Err(err)).Msg("cannot read the user organization")
			continue
		}
		name, vat, ok := helpdeskOrganization(orgs)
		if !ok {
			continue
		}

		company, found := index.Find(vat, name)
		if !apply {
			event(log.Info()).Str("organization", name).Str("vat", vat).Bool("create", !found).Str("company", company.Name).Msg("contact to give a company")
			if !found {
				index.Add(Company{Name: name, CustomFields: CompanyCustomFields{VAT: vat}})
			}
			continue
		}

		if !found {
			created, err := client.CreateCompany(name, vat)
			if err != nil {
				result.failed++
				event(log.Error().Err(err)).Str("organization", name).Msg("company creation failed")
				continue
			}
			company = *created
			index.Add(company)
			result.created++
			log.Info().Int64("company_id", company.ID).Str("company", company.Name).Str("vat", vat).Msg("company created")
			sleep(delay)
		}

		if err := client.UpdateContactCompany(orphan.Contact.ID, company.ID); err != nil {
			result.failed++
			event(log.Error().Err(err)).Str("company", company.Name).Msg("contact company update failed")
			continue
		}
		result.assigned++
		event(log.Info()).Int64("company_id", company.ID).Str("company", company.Name).Msg("contact given a company")
		sleep(delay)
	}
	return result
}

// helpdeskOrganization picks the distributor, reseller or customer the user
// belongs to; the owner organization has no company in the helpdesk
func helpdeskOrganization(orgs []models.LogtoOrganization) (name, vat string, ok bool) {
	for _, org := range orgs {
		switch org.CustomData["type"] {
		case "distributor", "reseller", "customer":
		default:
			continue
		}
		name = strings.Join(strings.Fields(org.Name), " ")
		if name == "" {
			continue
		}
		vat, _ = org.CustomData["vat"].(string)
		return name, strings.TrimSpace(vat), true
	}
	return "", "", false
}
