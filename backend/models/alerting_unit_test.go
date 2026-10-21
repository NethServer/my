/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipientScopeReadsLegacyOrganizationIDs(t *testing.T) {
	var s RecipientScope
	require.NoError(t, json.Unmarshal([]byte(`{"organization_ids":["a"],"managed_organization_ids":["b"],"assigned_organization_ids":["c"]}`), &s))
	assert.ElementsMatch(t, []string{"a", "b"}, s.ManagedOrganizationIDs)
	assert.Equal(t, []string{"c"}, s.AssignedOrganizationIDs)

	out, err := json.Marshal(s)
	require.NoError(t, err)
	assert.NotContains(t, string(out), `"organization_ids"`)
	assert.Contains(t, string(out), `"managed_organization_ids"`)
}

func TestValidateScopeAssignedOrganizations(t *testing.T) {
	_, ok := validateScope(&RecipientScope{AssignedOrganizationIDs: []string{"bad id"}})
	assert.False(t, ok)
	_, ok = validateScope(&RecipientScope{AssignedOrganizationIDs: []string{"dist_1"}})
	assert.True(t, ok)
}
