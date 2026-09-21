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

// ownerOrgType is the organization_type of the Owner organization: the one
// organization outside the three levels, so the one the organization tables
// cannot answer for. It is the value the organization_type of users and systems
// carries for it, so a client draws it with the same icon everywhere.
const ownerOrgType = "owner"

// creatorOrgTypesQuery looks the creator organizations up in the three
// organization tables rather than in unified_organizations: the materialized
// view refreshes asynchronously and leaves soft-deleted rows out, so it cannot
// tell a fresh insert, a deleted organization and the Owner organization apart.
// Live and deleted rows are probed separately so every branch runs on an index:
// the partial unique index on logto_id for the live rows, the deleted_at index
// for the few deleted ones.
const creatorOrgTypesQuery = `
	SELECT logto_id, 'distributor' AS org_type, TRUE AS live FROM distributors WHERE logto_id = ANY($1) AND deleted_at IS NULL
	UNION ALL
	SELECT logto_id, 'reseller', TRUE FROM resellers WHERE logto_id = ANY($1) AND deleted_at IS NULL
	UNION ALL
	SELECT logto_id, 'customer', TRUE FROM customers WHERE logto_id = ANY($1) AND deleted_at IS NULL
	UNION ALL
	SELECT logto_id, 'distributor', FALSE FROM distributors WHERE logto_id = ANY($1) AND deleted_at IS NOT NULL
	UNION ALL
	SELECT logto_id, 'reseller', FALSE FROM resellers WHERE logto_id = ANY($1) AND deleted_at IS NOT NULL
	UNION ALL
	SELECT logto_id, 'customer', FALSE FROM customers WHERE logto_id = ANY($1) AND deleted_at IS NOT NULL`

// fillCreatorOrgTypes resolves the current level of every organization
// referenced by the given creator snapshots and fills it into
// organization_type, so a client can pick the organization's icon and build
// the link to its detail page without a second lookup.
//
// The level is resolved on every read instead of being stored with the
// snapshot: an organization promoted after the fact (reseller -> distributor)
// is then labelled with its current level, with no retroactive backfill of the
// stored snapshots. The whole page costs one query, whatever the number of
// rows.
//
// Each organization gets one of three answers:
//   - a live row in one of the organization tables: its level;
//   - a soft-deleted row only: no type at all. A deleted organization has no
//     detail page to link to and no current level to assert, so the field is
//     omitted rather than guessed;
//   - no row anywhere: "owner". The Owner organization is the only one outside
//     the three levels, and this is the value the organization_type of users
//     and systems answers for it. An organization destroyed outright lands
//     here too, as it does in those joins.
//
// The enrichment is a nice-to-have: on a query error the field is left empty
// rather than failing the read.
func fillCreatorOrgTypes(db *sql.DB, creators ...models.CreatorOrgRef) {
	if db == nil {
		return
	}

	ids := make([]string, 0, len(creators))
	seen := make(map[string]bool, len(creators))
	for _, creator := range creators {
		if creator == nil {
			continue
		}
		id := creator.CreatorOrgID()
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return
	}

	rows, err := db.Query(creatorOrgTypesQuery, pq.Array(ids))
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	live := make(map[string]string, len(ids))
	deleted := make(map[string]bool)
	for rows.Next() {
		var logtoID, orgType string
		var isLive bool
		if err := rows.Scan(&logtoID, &orgType, &isLive); err != nil {
			continue
		}
		if isLive {
			live[logtoID] = orgType
		} else {
			deleted[logtoID] = true
		}
	}
	if err := rows.Err(); err != nil {
		return
	}

	for _, creator := range creators {
		if creator == nil {
			continue
		}
		id := creator.CreatorOrgID()
		if id == "" {
			continue
		}
		if orgType, found := live[id]; found {
			creator.SetCreatorOrgType(orgType)
			continue
		}
		if deleted[id] {
			continue
		}
		creator.SetCreatorOrgType(ownerOrgType)
	}
}

// creatorRefsOf collects the creator snapshots of a page of entities so
// fillCreatorOrgTypes can resolve them all in one query. Entities without a
// snapshot are harmless: a nil snapshot resolves to an empty organization ID.
func creatorRefsOf[T any](items []T, creatorOf func(T) models.CreatorOrgRef) []models.CreatorOrgRef {
	refs := make([]models.CreatorOrgRef, 0, len(items))
	for _, item := range items {
		refs = append(refs, creatorOf(item))
	}
	return refs
}
