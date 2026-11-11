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
	"time"

	"github.com/rs/zerolog"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/services/logto"
)

// RunContactsSync is the entry point of `backend freshdesk-contacts`: it
// renames the helpdesk contacts still named after the Logto username to the
// display name of the my user with the same email, and returns the process
// exit code. Freshworks copies the username claim into the contact name at
// the first login and never updates it, so the helpdesk is corrected from
// here.
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

	users, err := logto.NewManagementClient().GetAllUsers()
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

	log.Info().
		Bool("apply", *apply).
		Int("users", len(identities)).
		Int("contacts", len(contacts)).
		Int("to_rename", len(renames)).
		Int("renamed", renamed).
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
