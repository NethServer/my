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

// fillCreatorOrgTypes resolves the current level (distributor, reseller,
// customer) of every organization referenced by the given creator snapshots and
// fills it into organization_type, so a client can build the link to the
// creator's organization without a second lookup.
//
// The level is resolved on every read instead of being stored with the
// snapshot: an organization promoted after the fact (reseller -> distributor)
// is then labelled with its current level, with no retroactive backfill of the
// stored snapshots. The whole page costs one indexed query on
// unified_organizations, whatever the number of rows.
//
// An organization the view does not carry gets no type at all: the field is
// omitted rather than guessed. Three different cases land there, and none of
// them is linkable: the owner organization, which is not one of the three
// levels; a soft-deleted organization, which the view filters out; and the
// window in which the asynchronous refresh has not caught up with a fresh
// insert. Answering "owner" would be a link the client cannot follow in all
// three, and a false attribution in the last two. The enrichment is a
// nice-to-have: on a query error the field is left empty rather than failing
// the read.
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

	rows, err := db.Query(`SELECT logto_id, org_type FROM unified_organizations WHERE logto_id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return
	}
	defer func() { _ = rows.Close() }()

	types := make(map[string]string, len(ids))
	for rows.Next() {
		var logtoID, orgType string
		if err := rows.Scan(&logtoID, &orgType); err != nil {
			continue
		}
		types[logtoID] = orgType
	}
	if err := rows.Err(); err != nil {
		return
	}

	for _, creator := range creators {
		if creator == nil || creator.CreatorOrgID() == "" {
			continue
		}
		orgType, found := types[creator.CreatorOrgID()]
		if !found {
			continue
		}
		creator.SetCreatorOrgType(orgType)
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
