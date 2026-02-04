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
	case "distributor":
		query += fmt.Sprintf(`
			AND (
				%s = $%d
				OR %s IN (
					SELECT logto_id FROM resellers WHERE deleted_at IS NULL
					UNION
					SELECT logto_id FROM customers WHERE deleted_at IS NULL
				)
			)
		`, colRef, nextArgIdx, colRef)
	case "reseller":
		query += fmt.Sprintf(`
			AND (
				%s = $%d
				OR %s IN (
					SELECT logto_id FROM customers WHERE deleted_at IS NULL
				)
			)
		`, colRef, nextArgIdx, colRef)
	default:
		// Customer or unknown role — only their organization
		query += fmt.Sprintf(` AND %s = $%d`, colRef, nextArgIdx)
	}

	args = append(args, orgID)
	nextArgIdx++
	return query, args, nextArgIdx
}
