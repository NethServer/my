/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"database/sql"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/models"
)

// parentOrganizationsQuery resolves the direct parent (custom_data.createdBy,
// the ownership key RBAC walks) of each requested organization, with the
// parent's current name and level. The child rows are read whatever their
// deleted_at, so the deleted systems of a deleted customer still show where
// they sat. A parent with no live partner row is the Owner organization: its
// name comes from the child's creator snapshot, the only place the database
// records it, and only when that snapshot names the parent itself (a promoted
// organization keeps the snapshot of its old parent).
const parentOrganizationsQuery = `
	WITH child AS (
		SELECT logto_id, custom_data->>'createdBy' AS parent_id, custom_data->'createdByUser'->>'organization_id' AS snapshot_org_id, custom_data->'createdByUser'->>'organization_name' AS snapshot_name FROM distributors WHERE logto_id = ANY($1)
		UNION ALL
		SELECT logto_id, custom_data->>'createdBy', custom_data->'createdByUser'->>'organization_id', custom_data->'createdByUser'->>'organization_name' FROM resellers WHERE logto_id = ANY($1)
		UNION ALL
		SELECT logto_id, custom_data->>'createdBy', custom_data->'createdByUser'->>'organization_id', custom_data->'createdByUser'->>'organization_name' FROM customers WHERE logto_id = ANY($1)
	), parent AS (
		SELECT logto_id, id::text AS db_id, name, 'distributor' AS org_type FROM distributors WHERE deleted_at IS NULL AND logto_id = ANY(ARRAY(SELECT parent_id FROM child))
		UNION ALL
		SELECT logto_id, id::text, name, 'reseller' FROM resellers WHERE deleted_at IS NULL AND logto_id = ANY(ARRAY(SELECT parent_id FROM child))
	)
	SELECT c.logto_id, c.parent_id, COALESCE(p.db_id, ''), COALESCE(p.name, CASE WHEN c.snapshot_org_id = c.parent_id THEN c.snapshot_name END, ''), COALESCE(p.org_type, 'owner')
	FROM child c
	LEFT JOIN parent p ON p.logto_id = c.parent_id
	WHERE COALESCE(c.parent_id, '') <> ''`

// lookupParentOrganizations returns the direct parent of each given
// organization logto ID, keyed by that ID, in one query for the whole page.
// An organization with no recorded parent (the Owner organization itself) or
// an unknown ID is absent. Like the creator
// organization types, the parent is a nice-to-have: on a query error the map
// is empty rather than failing the read.
func lookupParentOrganizations(db *sql.DB, orgIDs []string) map[string]*models.ParentOrganization {
	parents := make(map[string]*models.ParentOrganization)
	if db == nil {
		return parents
	}

	ids := make([]string, 0, len(orgIDs))
	seen := make(map[string]bool, len(orgIDs))
	for _, id := range orgIDs {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if len(ids) == 0 {
		return parents
	}

	rows, err := db.Query(parentOrganizationsQuery, pq.Array(ids))
	if err != nil {
		return parents
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var childID string
		parent := &models.ParentOrganization{}
		if err := rows.Scan(&childID, &parent.LogtoID, &parent.ID, &parent.Name, &parent.Type); err != nil {
			continue
		}
		parents[childID] = parent
	}
	if err := rows.Err(); err != nil {
		return make(map[string]*models.ParentOrganization)
	}
	return parents
}

// parentSlot pairs an organization logto ID with the field its parent goes in.
type parentSlot struct {
	orgID  string
	parent **models.ParentOrganization
}

// fillParentOrganizations resolves the parent of every slot's organization in
// one query and writes it into the slot.
func fillParentOrganizations(db *sql.DB, slots []parentSlot) {
	if len(slots) == 0 {
		return
	}
	orgIDs := make([]string, 0, len(slots))
	for _, slot := range slots {
		orgIDs = append(orgIDs, slot.orgID)
	}
	parents := lookupParentOrganizations(db, orgIDs)
	for _, slot := range slots {
		*slot.parent = parents[slot.orgID]
	}
}
