/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package entities

import (
	"fmt"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// orgTypesByLogtoID maps organization logto ids to their current level
// (distributor, reseller, customer) in one pass over unified_organizations,
// which carries a unique index on logto_id. Ids absent from the view — the
// Owner organization, deleted orgs, orgs not yet synced with Logto — are simply
// missing from the result.
func orgTypesByLogtoID(ids []string) (map[string]string, error) {
	out := make(map[string]string, len(ids))
	if len(ids) == 0 {
		return out, nil
	}

	rows, err := database.DB.Query(`
		SELECT logto_id, org_type
		FROM unified_organizations
		WHERE logto_id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve creator organization types: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var logtoID, orgType string
		if err := rows.Scan(&logtoID, &orgType); err != nil {
			continue
		}
		out[logtoID] = orgType
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating creator organization types: %w", err)
	}
	return out, nil
}

// ResolveCreatorOrgTypes fills OrganizationType on every creator snapshot given,
// with a single query for the whole batch — call it once per page of results,
// not once per row.
//
// The level is resolved here rather than stored alongside the rest of the
// snapshot because an organization can change it after the snapshot was taken:
// PromoteResellerToDistributor moves a reseller up in place, keeping its
// logto_id, so a stored copy would keep saying "reseller" and send clients to
// the wrong detail page. This mirrors OrganizationName, which the rename
// propagation keeps live for the same reason.
//
// Creators with no resolvable organization keep an empty type, which is what the
// Owner organization and deleted orgs should get: they have no detail page.
func ResolveCreatorOrgTypes(creators ...*models.OrgCreator) error {
	ids := make([]string, 0, len(creators))
	for _, creator := range creators {
		if creator != nil && creator.OrganizationID != "" {
			ids = append(ids, creator.OrganizationID)
		}
	}

	types, err := orgTypesByLogtoID(ids)
	if err != nil {
		return err
	}

	for _, creator := range creators {
		if creator == nil {
			continue
		}
		creator.OrganizationType = types[creator.OrganizationID]
	}
	return nil
}

// ResolveSystemCreatorOrgTypes is ResolveCreatorOrgTypes for the system creator
// snapshot, which carries the same organization fields under its own type.
func ResolveSystemCreatorOrgTypes(creators ...*models.SystemCreator) error {
	ids := make([]string, 0, len(creators))
	for _, creator := range creators {
		if creator != nil && creator.OrganizationID != "" {
			ids = append(ids, creator.OrganizationID)
		}
	}

	types, err := orgTypesByLogtoID(ids)
	if err != nil {
		return err
	}

	for _, creator := range creators {
		if creator == nil {
			continue
		}
		creator.OrganizationType = types[creator.OrganizationID]
	}
	return nil
}
