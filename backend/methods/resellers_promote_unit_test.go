/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

// Promotion detaches an organization from the distributor that manages it, so
// only owner-level authority may trigger it.
func TestIsOwnerOrSuperAdmin(t *testing.T) {
	tests := []struct {
		name     string
		user     *models.User
		expected bool
	}{
		{
			name:     "owner organization",
			user:     &models.User{OrgRole: "Owner", UserRoles: []string{"Admin"}},
			expected: true,
		},
		{
			name:     "owner organization lowercase claim",
			user:     &models.User{OrgRole: "owner"},
			expected: true,
		},
		{
			name:     "super admin outside the owner organization",
			user:     &models.User{OrgRole: "Distributor", UserRoles: []string{"Super Admin"}},
			expected: true,
		},
		{
			name:     "distributor admin",
			user:     &models.User{OrgRole: "Distributor", UserRoles: []string{"Admin"}},
			expected: false,
		},
		{
			name:     "reseller admin",
			user:     &models.User{OrgRole: "Reseller", UserRoles: []string{"Admin", "Support"}},
			expected: false,
		},
		{
			name:     "customer with no user role",
			user:     &models.User{OrgRole: "Customer"},
			expected: false,
		},
		{
			name:     "no claims at all",
			user:     &models.User{},
			expected: false,
		},
		{
			name:     "nil user",
			user:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsOwnerOrSuperAdmin(tt.user))
		})
	}
}
