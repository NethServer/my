/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"testing"

	"github.com/nethesis/my/backend/services/logto"
	"github.com/stretchr/testify/assert"
)

func TestSameOrganizationRules(t *testing.T) {
	desired := logto.ApplicationAccessControl{OrganizationIDs: []string{"a", "b", "c"}}

	assert.True(t, sameOrganizationRules(logto.ApplicationAccessControl{OrganizationIDs: []string{"c", "a", "b"}}, desired),
		"order does not matter")
	assert.False(t, sameOrganizationRules(logto.ApplicationAccessControl{OrganizationIDs: []string{"a", "b"}}, desired),
		"a missing organization is a difference")
	assert.False(t, sameOrganizationRules(logto.ApplicationAccessControl{OrganizationIDs: []string{"a", "b", "c", "d"}}, desired),
		"an extra organization is a difference")
	assert.False(t, sameOrganizationRules(logto.ApplicationAccessControl{
		OrganizationIDs: []string{"a", "b", "c"},
		UserRoleIDs:     []string{"admin-role"},
	}, desired), "a user-role rule next to the organizations widens the gate and must be rewritten")
	assert.False(t, sameOrganizationRules(logto.ApplicationAccessControl{
		OrganizationIDs: []string{"a", "b", "c"},
		UserIDs:         []string{"someone"},
	}, desired), "a user rule must be rewritten as well")
	assert.False(t, sameOrganizationRules(logto.ApplicationAccessControl{
		OrganizationIDs:       []string{"a", "b", "c"},
		OrganizationRoleRules: []logto.ApplicationOrganizationRoleRule{{OrganizationID: "a", OrganizationRoleIDs: []string{"r"}}},
	}, desired), "an organization-role rule must be rewritten as well")
}
