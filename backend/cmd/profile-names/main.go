/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

// profile-names fills the Logto user profile (givenName / familyName) from
// the display name of every user of a tenant, so the ID token carries the
// given_name / family_name claims. It only prints the plan unless -apply is
// given. Users whose profile already matches are left untouched.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

func main() {
	envFile := flag.String("env", "", "env file with LOGTO_TENANT_ID, LOGTO_BACKEND_APP_ID and LOGTO_BACKEND_APP_SECRET (default: process environment)")
	apply := flag.Bool("apply", false, "write the profiles to Logto (default: print the plan only)")
	userID := flag.String("user", "", "limit the run to one Logto user id")
	delay := flag.Duration("delay", 100*time.Millisecond, "pause between writes")
	flag.Parse()

	if *envFile != "" {
		if err := godotenv.Load(*envFile); err != nil {
			fail("cannot load %s: %v", *envFile, err)
		}
	}
	tenant := os.Getenv("LOGTO_TENANT_ID")
	clientID := os.Getenv("LOGTO_BACKEND_APP_ID")
	clientSecret := os.Getenv("LOGTO_BACKEND_APP_SECRET")
	if tenant == "" || clientID == "" || clientSecret == "" {
		fail("LOGTO_TENANT_ID, LOGTO_BACKEND_APP_ID and LOGTO_BACKEND_APP_SECRET are required")
	}
	if err := logger.InitFromEnv("profile-names"); err != nil {
		fail("cannot initialize logger: %v", err)
	}
	configuration.Config.LogtoIssuer = fmt.Sprintf("https://%s.logto.app", tenant)
	configuration.Config.LogtoManagementBaseURL = configuration.Config.LogtoIssuer + "/api"
	configuration.Config.LogtoManagementClientID = clientID
	configuration.Config.LogtoManagementClientSecret = clientSecret

	client := logto.NewManagementClient()

	var users []models.LogtoUser
	if *userID != "" {
		user, err := client.GetUserByID(*userID)
		if err != nil {
			fail("cannot fetch user %s: %v", *userID, err)
		}
		users = []models.LogtoUser{*user}
	} else {
		all, err := client.GetAllUsers()
		if err != nil {
			fail("cannot list users: %v", err)
		}
		users = all
	}

	var unchanged, planned, written, failed, skipped int
	for _, user := range users {
		desired := models.UserProfileFromName(user.Name)
		if desired == nil {
			skipped++
			fmt.Printf("skip    %s %-30s no name\n", user.ID, user.Username)
			continue
		}
		if stringField(user.Profile, "givenName") == desired.GivenName &&
			stringField(user.Profile, "familyName") == desired.FamilyName {
			unchanged++
			continue
		}

		planned++
		fmt.Printf("update  %s %-30s %q -> given=%q family=%q%s\n",
			user.ID, user.Username, user.Name, desired.GivenName, desired.FamilyName, droppedKeys(user.Profile))
		if !*apply {
			continue
		}
		if _, err := client.UpdateUser(user.ID, models.UpdateUserRequest{Profile: desired}); err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "error   %s %-30s %v\n", user.ID, user.Username, err)
			continue
		}
		written++
		time.Sleep(*delay)
	}

	mode := "dry run"
	if *apply {
		mode = "applied"
	}
	fmt.Printf("\n%s: %d users, %d already set, %d to update, %d written, %d failed, %d without name\n",
		mode, len(users), unchanged, planned, written, failed, skipped)
	if failed > 0 {
		os.Exit(1)
	}
}

func stringField(profile map[string]interface{}, key string) string {
	value, _ := profile[key].(string)
	return value
}

// droppedKeys names the profile fields a write would wipe: Logto replaces
// the whole profile object on PATCH /users/{id}.
func droppedKeys(profile map[string]interface{}) string {
	var keys []string
	for key := range profile {
		if key != "givenName" && key != "familyName" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	return " (drops " + strings.Join(keys, ", ") + ")"
}

func fail(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
