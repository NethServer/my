/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"strings"
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

// The organization lists build their counter columns from a single helper so
// that the four reseller and eight customer query variants cannot drift apart.
// What each mode is allowed to cost is the whole point of the parameter, so it
// is asserted here rather than left to a reading of the SQL.

func TestResellerCountColumnsPerMode(t *testing.T) {
	assert.Empty(t, resellerCountColumns(models.CountsNone), "no counters means no extra columns at all")

	basic := resellerCountColumns(models.CountsBasic)
	assert.True(t, strings.HasPrefix(basic, ","), "columns must append to the preceding select list")
	assert.Contains(t, basic, "as systems_count")
	assert.Contains(t, basic, "as legacy_systems_count")
	assert.Contains(t, basic, "as customers_count")
	assert.NotContains(t, basic, "as applications_count", "the expensive counter must stay out of the basic mode")

	all := resellerCountColumns(models.CountsAll)
	assert.Contains(t, all, "as systems_count")
	assert.Contains(t, all, "as applications_count")
	assert.True(t, strings.HasPrefix(all, basic), "all must be basic plus applications, not a different set")
}

func TestCustomerCountColumnsPerMode(t *testing.T) {
	assert.Empty(t, customerCountColumns(models.CountsNone))

	basic := customerCountColumns(models.CountsBasic)
	assert.True(t, strings.HasPrefix(basic, ","))
	assert.Contains(t, basic, "as systems_count")
	assert.NotContains(t, basic, "as applications_count")

	all := customerCountColumns(models.CountsAll)
	assert.Contains(t, all, "as applications_count")
	assert.True(t, strings.HasPrefix(all, basic))
}

// The reseller applications count is the one that made the list unusable: its
// organization set is a correlated subquery, so it must never be built for a
// mode the portal actually uses.
func TestResellerApplicationsCountUsesTheSubtreeOrgSet(t *testing.T) {
	all := resellerCountColumns(models.CountsAll)
	assert.Contains(t, all, resellerOrgSet)
	assert.NotContains(t, all, "COALESCE(a.organization_id", "COALESCE on the two organization columns forces a sequential scan")
}
