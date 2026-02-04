/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package helpers

import (
	"strings"
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
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "distributor", "org-dist-1", "", args, 1)

	assert.Contains(t, resultQuery, "organization_id = $1")
	assert.Contains(t, resultQuery, "resellers")
	assert.Contains(t, resultQuery, "customers")
	assert.Equal(t, []interface{}{"org-dist-1"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_ResellerRole(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "reseller", "org-res-1", "", args, 1)

	assert.Contains(t, resultQuery, "organization_id = $1")
	assert.Contains(t, resultQuery, "customers")
	assert.NotContains(t, resultQuery, "resellers")
	assert.Equal(t, []interface{}{"org-res-1"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_CustomerRole(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "customer", "org-cust-1", "", args, 1)

	assert.Contains(t, resultQuery, "AND organization_id = $1")
	assert.NotContains(t, resultQuery, "IN (")
	assert.Equal(t, []interface{}{"org-cust-1"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_UnknownRole(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "unknown", "org-x", "", args, 1)

	// Unknown roles should be treated like customer (most restrictive)
	assert.Contains(t, resultQuery, "AND organization_id = $1")
	assert.NotContains(t, resultQuery, "IN (")
	assert.Equal(t, []interface{}{"org-x"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_WithTableAlias(t *testing.T) {
	query := "SELECT * FROM systems s WHERE 1=1"
	args := []interface{}{}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "customer", "org-1", "s.", args, 1)

	assert.Contains(t, resultQuery, "s.organization_id = $1")
	assert.Equal(t, []interface{}{"org-1"}, resultArgs)
	assert.Equal(t, 2, nextIdx)
}

func TestAppendOrgFilter_WithTableAlias_Distributor(t *testing.T) {
	query := "SELECT * FROM systems s WHERE 1=1"
	args := []interface{}{}

	resultQuery, _, _ := AppendOrgFilter(query, "distributor", "org-1", "s.", args, 1)

	assert.Contains(t, resultQuery, "s.organization_id = $1")
	assert.Contains(t, resultQuery, "s.organization_id IN (")
}

func TestAppendOrgFilter_WithExistingArgs(t *testing.T) {
	query := "SELECT * FROM systems WHERE name = $1"
	args := []interface{}{"test-name"}

	resultQuery, resultArgs, nextIdx := AppendOrgFilter(query, "customer", "org-1", "", args, 2)

	assert.Contains(t, resultQuery, "organization_id = $2")
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

func TestAppendOrgFilter_DistributorSubqueryStructure(t *testing.T) {
	query := "SELECT * FROM systems WHERE 1=1"
	args := []interface{}{}

	resultQuery, _, _ := AppendOrgFilter(query, "distributor", "org-dist", "", args, 1)

	// Verify the subquery has UNION of resellers and customers
	assert.True(t, strings.Contains(resultQuery, "UNION"), "Distributor filter should use UNION")
	assert.True(t, strings.Contains(resultQuery, "deleted_at IS NULL"), "Subquery should filter deleted records")
}
