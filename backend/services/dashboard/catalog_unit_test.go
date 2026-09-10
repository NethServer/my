/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package dashboard

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

// permissionVocabulary is the permission set sync/configs/config.yml pushes to
// Logto. A catalog entry naming anything else would compile, pass every other
// test, and silently hide its widget from every persona forever — so the
// vocabulary is restated here rather than derived from the catalog.
var permissionVocabulary = []string{
	"config:alerts", "connect:systems",
	"destroy:customers", "destroy:distributors", "destroy:resellers",
	"destroy:systems", "destroy:users", "impersonate:users",
	"manage:alerts", "manage:applications", "manage:customers",
	"manage:distributors", "manage:entitlements", "manage:rebranding",
	"manage:resellers", "manage:systems", "manage:users",
	"read:alerts", "read:applications", "read:customers", "read:distributors",
	"read:entitlements", "read:rebranding", "read:resellers", "read:systems",
	"read:users",
}

var validTopics = []Topic{
	TopicOrganizations, TopicAlerts, TopicSystems, TopicUsers,
	TopicApplications, TopicEntitlements, TopicRebranding,
}

func userWith(permissions ...string) *models.User {
	return &models.User{
		ID:              "test-user",
		UserPermissions: permissions,
		OrgPermissions:  []string{},
	}
}

func TestCatalogEntriesAreWellFormed(t *testing.T) {
	seen := make(map[string]struct{})

	for _, widget := range Catalog() {
		t.Run(widget.ID, func(t *testing.T) {
			assert.NotEmpty(t, widget.ID)
			assert.NotEmpty(t, widget.Title)
			assert.NotEmpty(t, widget.Description)
			assert.Contains(t, validTopics, widget.Topic)
			assert.Contains(t, []Size{SizeSmall, SizeMedium, SizeLarge}, widget.DefaultSize)

			if widget.RequiredPermission != "" {
				assert.Contains(t, permissionVocabulary, widget.RequiredPermission,
					"permission is not in the vocabulary sync pushes to Logto")
			}

			_, duplicate := seen[widget.ID]
			assert.False(t, duplicate, "duplicate widget id")
			seen[widget.ID] = struct{}{}
		})
	}
}

func TestCatalogReturnsACopy(t *testing.T) {
	first := Catalog()
	original := first[0].ID
	first[0].ID = "mutated"

	assert.Equal(t, original, Catalog()[0].ID)
}

func TestCatalogForFiltersByEffectivePermissions(t *testing.T) {
	ids := func(widgets []Widget) []string {
		out := make([]string, 0, len(widgets))
		for _, widget := range widgets {
			out = append(out, widget.ID)
		}
		return out
	}

	tests := []struct {
		name        string
		user        *models.User
		wantPresent []string
		wantAbsent  []string
	}{
		{
			name: "an owner-level persona sees the organization widgets",
			user: userWith("read:systems", "read:users", "read:applications",
				"read:distributors", "read:resellers", "read:customers",
				"read:entitlements", "read:rebranding"),
			wantPresent: []string{"distributors_counter", "resellers_counter", "customers_counter", "third_party_apps"},
		},
		{
			name:        "a customer persona sees no organization widget",
			user:        userWith("read:systems", "read:applications", "read:entitlements"),
			wantPresent: []string{"alerts_counter", "systems_counter", "addons_expiring", "third_party_apps"},
			wantAbsent:  []string{"distributors_counter", "resellers_counter", "customers_counter", "users_counter"},
		},
		{
			name:        "a support persona without read:users sees no user widget",
			user:        userWith("read:systems", "read:applications"),
			wantPresent: []string{"alerts_firing_now", "systems_recent"},
			wantAbsent:  []string{"users_counter", "users_trend", "rebranding_summary"},
		},
		{
			name:        "organization permissions count as much as user ones",
			user:        &models.User{OrgPermissions: []string{"read:customers"}},
			wantPresent: []string{"customers_counter", "customers_trend"},
			wantAbsent:  []string{"systems_counter"},
		},
		{
			name:        "a persona with no permission still gets the unrestricted widget",
			user:        userWith(),
			wantPresent: []string{"third_party_apps"},
			wantAbsent:  []string{"alerts_counter", "systems_counter"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ids(CatalogFor(tt.user))
			for _, id := range tt.wantPresent {
				assert.Contains(t, got, id)
			}
			for _, id := range tt.wantAbsent {
				assert.NotContains(t, got, id)
			}
		})
	}
}

func TestCatalogForKeepsCatalogOrder(t *testing.T) {
	full := Catalog()
	filtered := CatalogFor(userWith("read:systems", "read:users", "read:applications",
		"read:distributors", "read:resellers", "read:customers",
		"read:entitlements", "read:rebranding"))

	assert.Len(t, filtered, len(full))
	for i := range full {
		assert.Equal(t, full[i].ID, filtered[i].ID)
	}
}

func TestTopicsAreDistinctAndInCatalogOrder(t *testing.T) {
	topics := Topics(CatalogFor(userWith("read:systems")))

	assert.Equal(t, []Topic{TopicAlerts, TopicSystems, TopicApplications}, topics)
	assert.Len(t, slices.Compact(slices.Clone(topics)), len(topics))
}

func TestByID(t *testing.T) {
	widget, ok := ByID("alerts_counter")
	assert.True(t, ok)
	assert.Equal(t, TopicAlerts, widget.Topic)

	_, ok = ByID("not_a_widget")
	assert.False(t, ok)
}
