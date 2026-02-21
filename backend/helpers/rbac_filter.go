/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package helpers

import (
	"fmt"
	"strings"
)

// AppendOrgFilter appends RBAC hierarchical organization filtering to a SQL query.
// It handles all four organization roles: owner, distributor, reseller, and customer.
//
// Parameters:
//   - query: the base SQL query to append to
//   - orgRole: the user's organization role (case-insensitive)
//   - orgID: the user's organization ID
//   - tableAlias: optional table alias prefix for organization_id (e.g. "s." for "s.organization_id")
//   - args: existing query arguments
//   - nextArgIdx: the next positional argument index ($1, $2, etc.)
//
// Returns the modified query, updated args, and the next argument index.
func AppendOrgFilter(query string, orgRole, orgID, tableAlias string, args []interface{}, nextArgIdx int) (string, []interface{}, int) {
	orgRoleLower := strings.ToLower(orgRole)
	colRef := tableAlias + "organization_id"

	switch orgRoleLower {
	case "owner":
		// Owner sees everything — no filter added
		return query, args, nextArgIdx
	default:
		// For all non-owner roles, use pre-computed allowed org IDs from the RBAC cache.
		// This avoids the security bug where distributor/reseller subqueries selected
		// ALL resellers/customers instead of only those in the user's hierarchy.
		allowedOrgIDs := GetAllowedOrgIDsForFilter(orgRoleLower, orgID)
		if len(allowedOrgIDs) == 0 {
			// No allowed orgs - add impossible condition
			query += fmt.Sprintf(` AND %s IS NULL AND %s IS NOT NULL`, colRef, colRef)
			return query, args, nextArgIdx
		}

		placeholders := make([]string, len(allowedOrgIDs))
		for i, id := range allowedOrgIDs {
			placeholders[i] = fmt.Sprintf("$%d", nextArgIdx)
			args = append(args, id)
			nextArgIdx++
			_ = id
		}
		query += fmt.Sprintf(` AND %s IN (%s)`, colRef, strings.Join(placeholders, ","))
	}

	return query, args, nextArgIdx
}

// GetAllowedOrgIDsForFilter computes allowed org IDs for RBAC filtering.
// It uses the applications service cached lookup to get the correct hierarchy.
// This is a lightweight wrapper that avoids circular imports by using a callback.
var GetAllowedOrgIDsForFilter func(role, orgID string) []string

func init() {
	// Default implementation returns just the user's own org ID.
	// This is overridden at startup from the applications service.
	GetAllowedOrgIDsForFilter = func(role, orgID string) []string {
		return []string{orgID}
	}
}
