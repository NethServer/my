/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/models"
)

// tokenDef defines a test token with its role and output filename
type tokenDef struct {
	Filename string
	User     models.User
}

func main() {
	// Initialize configuration (reads JWT_SECRET, TENANT_ID, TENANT_DOMAIN from env)
	configuration.Init()

	// Determine output directory (same directory as the backend project root)
	outputDir := "."

	// Define test users for each RBAC role
	tokens := []tokenDef{
		{
			Filename: "token-owner",
			User: models.User{
				ID:          "owner-test-id",
				Username:    "owner",
				Email:       "owner@nethesis.it",
				Name:        "Owner User",
				UserRoles:   []string{"Super Admin"},
				UserRoleIDs: []string{"super-admin-role-id"},
				UserPermissions: []string{
					"destroy:systems", "read:systems", "manage:systems",
					"impersonate:users", "read:users", "manage:users",
					"read:applications", "manage:applications",
				},
				OrgRole:   "owner",
				OrgRoleID: "owner-role-id",
				OrgPermissions: []string{
					"read:distributors", "manage:distributors",
					"read:resellers", "manage:resellers",
					"read:customers", "manage:customers",
				},
				OrganizationID:   "nethesis-org-id",
				OrganizationName: "Nethesis",
			},
		},
		{
			Filename: "token-distributor",
			User: models.User{
				ID:          "distributor-test-id",
				Username:    "distributor",
				Email:       "distributor@example.com",
				Name:        "Distributor User",
				UserRoles:   []string{"Admin"},
				UserRoleIDs: []string{"admin-role-id"},
				UserPermissions: []string{
					"read:systems", "manage:systems",
					"read:users", "manage:users",
					"read:applications", "manage:applications",
				},
				OrgRole:   "distributor",
				OrgRoleID: "distributor-role-id",
				OrgPermissions: []string{
					"read:resellers", "manage:resellers",
					"read:customers", "manage:customers",
				},
				OrganizationID:   "distributor-org-id",
				OrganizationName: "Distributor Org",
			},
		},
		{
			Filename: "token-reseller",
			User: models.User{
				ID:          "reseller-test-id",
				Username:    "reseller",
				Email:       "reseller@example.com",
				Name:        "Reseller User",
				UserRoles:   []string{"Admin"},
				UserRoleIDs: []string{"admin-role-id"},
				UserPermissions: []string{
					"read:systems", "manage:systems",
					"read:users", "manage:users",
					"read:applications", "manage:applications",
				},
				OrgRole:   "reseller",
				OrgRoleID: "reseller-role-id",
				OrgPermissions: []string{
					"read:customers", "manage:customers",
				},
				OrganizationID:   "reseller-org-id",
				OrganizationName: "Reseller Org",
			},
		},
		{
			Filename: "token-customer",
			User: models.User{
				ID:          "customer-test-id",
				Username:    "customer",
				Email:       "customer@example.com",
				Name:        "Customer User",
				UserRoles:   []string{"Admin"},
				UserRoleIDs: []string{"admin-role-id"},
				UserPermissions: []string{
					"read:systems", "manage:systems",
					"read:users", "manage:users",
					"read:applications", "manage:applications",
				},
				OrgRole:          "customer",
				OrgRoleID:        "customer-role-id",
				OrgPermissions:   []string{},
				OrganizationID:   "customer-org-id",
				OrganizationName: "Customer Org",
			},
		},
	}

	for _, td := range tokens {
		tokenString, err := jwt.GenerateCustomToken(td.User)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error generating token for %s: %v\n", td.Filename, err)
			os.Exit(1)
		}

		outPath := filepath.Join(outputDir, td.Filename)
		if err := os.WriteFile(outPath, []byte(tokenString+"\n"), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "error writing %s: %v\n", outPath, err)
			os.Exit(1)
		}

		fmt.Printf("  %s written\n", td.Filename)
	}
}
