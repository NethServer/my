/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/nethesis/my/backend/models"
)

// createdByIDPattern matches the Logto ID charset used for user and organization
// IDs, so matching values can be inlined into SQL without risk of injection.
var createdByIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// createdByFilterClause builds a SQL fragment restricting organizations to those
// whose creator snapshot (custom_data.createdByUser) matches any of the given
// user or organization logto IDs, mirroring the systems created_by filter which
// matches either the creating user or their organization. Values are strictly
// validated against the Logto ID charset so they can be inlined safely; invalid
// or duplicate values are ignored. Returns "" when no usable value is provided.
//
// The fragment uses a bare custom_data reference so it works in both the
// single-table COUNT query and the aliased list query.
func createdByFilterClause(createdBy []string) string {
	seen := make(map[string]bool)
	quoted := make([]string, 0, len(createdBy))
	for _, v := range createdBy {
		if v == "" || seen[v] || !createdByIDPattern.MatchString(v) {
			continue
		}
		seen[v] = true
		quoted = append(quoted, "'"+v+"'")
	}
	if len(quoted) == 0 {
		return ""
	}
	list := strings.Join(quoted, ", ")
	return fmt.Sprintf(" AND (custom_data->'createdByUser'->>'user_id' IN (%s) OR custom_data->'createdByUser'->>'organization_id' IN (%s))", list, list)
}

// ownedByFilterClause builds a SQL fragment restricting resellers/customers to
// those owned by any of the given organization logto IDs (custom_data.createdBy,
// the ownership key RBAC visibility walks — not the creator snapshot). Backs
// the organization_id list filter; combined with ExpandOrganizationIDs it
// covers a whole hierarchy. Same validation and inlining rules as
// createdByFilterClause. Returns "" when no usable value is provided.
func ownedByFilterClause(ownedBy []string) string {
	seen := make(map[string]bool)
	quoted := make([]string, 0, len(ownedBy))
	for _, v := range ownedBy {
		if v == "" || seen[v] || !createdByIDPattern.MatchString(v) {
			continue
		}
		seen[v] = true
		quoted = append(quoted, "'"+v+"'")
	}
	if len(quoted) == 0 {
		return ""
	}
	return fmt.Sprintf(" AND custom_data->>'createdBy' IN (%s)", strings.Join(quoted, ", "))
}

// queryOrgCreators returns the distinct creator snapshots (custom_data.createdByUser)
// of the organizations in the given table matching scopeClause. It powers the
// created_by filter dropdown on the distributor/reseller/customer lists without
// paying for the per-row hierarchy counts the full List query computes: it reads
// only the snapshot from one pass over the (small) org table.
//
// table is a trusted constant supplied by the caller (never user input) and
// scopeClause carries only positional placeholders, so both interpolate safely.
// Results are deduplicated by user_id (a creator may own many orgs) and sorted
// by name, matching LocalUserRepository.ListCreators.
func queryOrgCreators(db *sql.DB, table, scopeClause string, args []interface{}) ([]models.OrgCreator, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT
			custom_data->'createdByUser'->>'user_id'                        AS user_id,
			custom_data->'createdByUser'->>'name'                           AS name,
			COALESCE(custom_data->'createdByUser'->>'email', '')            AS email,
			COALESCE(custom_data->'createdByUser'->>'organization_id', '')  AS organization_id,
			COALESCE(custom_data->'createdByUser'->>'organization_name', '') AS organization_name
		FROM %s
		WHERE deleted_at IS NULL
			AND custom_data->'createdByUser'->>'user_id' IS NOT NULL
			AND custom_data->'createdByUser'->>'user_id' != ''
			AND custom_data->'createdByUser'->>'name' IS NOT NULL
			AND custom_data->'createdByUser'->>'name' != ''%s
		ORDER BY name ASC`, table, scopeClause)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s creators: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	creators := make([]models.OrgCreator, 0)
	seen := make(map[string]bool)
	for rows.Next() {
		var c models.OrgCreator
		if err := rows.Scan(&c.UserID, &c.Name, &c.Email, &c.OrganizationID, &c.OrganizationName); err != nil {
			continue
		}
		if seen[c.UserID] {
			continue
		}
		seen[c.UserID] = true
		creators = append(creators, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating %s creators: %w", table, err)
	}

	return creators, nil
}

// ListCreators returns the distinct creators of the distributors visible to the
// caller (owner sees all; no other role can see distributors), for the
// created_by filter on GET /api/distributors.
func (r *LocalDistributorRepository) ListCreators(userOrgRole, userOrgID string) ([]models.OrgCreator, error) {
	if strings.ToLower(userOrgRole) != "owner" {
		return []models.OrgCreator{}, nil
	}
	return queryOrgCreators(r.db, "distributors", "", nil)
}

// ListCreators returns the distinct creators of the resellers visible to the
// caller, mirroring the RBAC scope of LocalResellerRepository.List.
func (r *LocalResellerRepository) ListCreators(userOrgRole, userOrgID string) ([]models.OrgCreator, error) {
	switch strings.ToLower(userOrgRole) {
	case "owner":
		return queryOrgCreators(r.db, "resellers", "", nil)
	case "distributor":
		return queryOrgCreators(r.db, "resellers", " AND custom_data->>'createdBy' = $1", []interface{}{userOrgID})
	default:
		return []models.OrgCreator{}, nil
	}
}

// ListCreators returns the distinct creators of the customers visible to the
// caller, mirroring the RBAC scope of LocalCustomerRepository.List.
func (r *LocalCustomerRepository) ListCreators(userOrgRole, userOrgID string) ([]models.OrgCreator, error) {
	switch strings.ToLower(userOrgRole) {
	case "owner":
		return queryOrgCreators(r.db, "customers", "", nil)
	case "distributor":
		return queryOrgCreators(r.db, "customers", " AND (custom_data->>'createdBy' = $1 OR custom_data->>'createdBy' IN (SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL))", []interface{}{userOrgID})
	case "reseller":
		return queryOrgCreators(r.db, "customers", " AND custom_data->>'createdBy' = $1", []interface{}{userOrgID})
	case "customer":
		return queryOrgCreators(r.db, "customers", " AND id = $1", []interface{}{userOrgID})
	default:
		return []models.OrgCreator{}, nil
	}
}
