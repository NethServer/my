/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendOrgFilter_OwnerRole(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "owner", "org-owner", "", args, 1)

	assert.Equal(t, query, resultQuery, "Owner role should not modify query")
	assert.Empty(t, resultArgs, "Owner role should not add args")
	assert.Equal(t, 1, nextIdx, "Owner role should not increment arg index")
}

func TestAppendOrgFilter_OwnerCaseInsensitive(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, _ := AppendOrgFilter(query, "Owner", "org-owner", "", args, 1)

	assert.Equal(t, query, resultQuery, "Owner role (capitalized) should not modify query")
	assert.Empty(t, resultArgs)
}

func TestAppendOrgFilter_DistributorRole(t *testing.T) {
	// Override the callback to simulate a distributor with multiple allowed orgs
	origCallback := GetAllowedOrgIDsForFilter
	defer func() { GetAllowedOrgIDsForFilter = origCallback }()
	GetAllowedOrgIDsForFilter = func(role, orgID string) []string {
		return []string{"org-dist-1", "org-res-1", "org-cust-1"}
	}

	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "distributor", "org-dist-1", "", args, 1)

	assert.Contains(t, resultQuery, "organization_id IN ($1,$2,$3)")
	assert.Equal(t, []interface{}{"org-dist-1", "org-res-1", "org-cust-1"}, resultArgs)
	assert.Equal(t, 4, nextIdx)
}

func TestAppendOrgFilter_ResellerRole(t *testing.T) {
	// Override to simulate a reseller with its own org + customer orgs
	origCallback := GetAllowedOrgIDsForFilter
	defer func() { GetAllowedOrgIDsForFilter = origCallback }()
	GetAllowedOrgIDsForFilter = func(role, orgID string) []string {
		return []string{"org-res-1", "org-cust-1"}
	}

	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "reseller", "org-res-1", "", args, 1)

	assert.Contains(t, resultQuery, "organization_id IN ($1,$2)")
	assert.Equal(t, []interface{}{"org-res-1", "org-cust-1"}, resultArgs)
	assert.Equal(t, 3, nextIdx)
}

func TestAppendOrgFilter_CustomerRole(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	// Default callback returns just the user's own org ID
	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "customer", "org-cust-1", "", args, 1)

	assert.Contains(t, resultQuery, "AND organization_id IN ($1)")
	assert.Equal(t, []interface{}{"org-cust-1"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_UnknownRole(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	// Default callback returns just the user's own org ID for unknown roles too
	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "unknown", "org-x", "", args, 1)

	assert.Contains(t, resultQuery, "AND organization_id IN ($1)")
	assert.Equal(t, []interface{}{"org-x"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_WithTableAlias(t *testing.T) {
	query := "SELECT * FROM systems s WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "customer", "org-1", "s.", args, 1)

	assert.Contains(t, resultQuery, "s.organization_id IN ($1)")
	assert.Equal(t, []interface{}{"org-1"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_WithTableAlias_Distributor(t *testing.T) {
	origCallback := GetAllowedOrgIDsForFilter
	defer func() { GetAllowedOrgIDsForFilter = origCallback }()
	GetAllowedOrgIDsForFilter = func(role, orgID string) []string {
		return []string{"org-1", "org-child-1"}
	}

	query := "SELECT * FROM systems s WHERE 1=1"
	args := []interface{}{}

	resultQuery, _, _ := AppendOrgFilter(query, "distributor", "org-1", "s.", args, 1)

	assert.Contains(t, resultQuery, "s.organization_id IN ($1,$2)")
}

func TestAppendOrgFilter_WithExistingArgs(t *testing.T) {
	query := "SELECT * FROM systems WHERE name = $1"
	args := []interface{}{"test-name"}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "customer", "org-1", "", args, 2)

	assert.Contains(t, resultQuery, "organization_id IN ($2)")
	assert.Equal(t, []interface{}{"test-name", "org-1"}, resultArgs)
	assert.Equal(t, 3, nextIdx)
}

func TestAppendOrgFilter_OwnerWithExistingArgs(t *testing.T) {
	query := "SELECT * FROM systems WHERE name = $1"
	args := []interface{}{"test-name"}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "owner", "org-owner", "", args, 2)

	// Owner should not add anything
	assert.Equal(t, "SELECT * FROM systems WHERE name = $1", resultQuery)
	assert.Equal(t, []interface{}{"test-name"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_EmptyAllowedOrgs(t *testing.T) {
	origCallback := GetAllowedOrgIDsForFilter
	defer func() { GetAllowedOrgIDsForFilter = origCallback }()
	GetAllowedOrgIDsForFilter = func(role, orgID string) []string {
		return []string{}
	}

	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "distributor", "org-dist", "", args, 1)

	// Empty allowed orgs should add impossible condition
	assert.Contains(t, resultQuery, "organization_id IS NULL AND organization_id IS NOT NULL")
	assert.Empty(t, resultArgs)
	assert.Equal(t, 1, nextIdx)
}
