/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// ThirdPartyAppsService answers one question: which third-party portals may
// the users of a given organization see on their dashboard.
//
// The answer lives on the distributor at the top of the organization's branch
// (distributors.third_party_apps, set by the Owner organization) and binds the
// organizations BELOW it: a reseller reads the list of the distributor that
// created it, a customer the list of the distributor above its reseller (or
// above itself, when a distributor created it directly). No list means no
// portal for them. The distributor's own users, like the Owner organization's,
// are never restricted by it: they see every portal their roles admit.
type ThirdPartyAppsService struct{}

// NewThirdPartyAppsService creates a new ThirdPartyAppsService.
func NewThirdPartyAppsService() *ThirdPartyAppsService {
	return &ThirdPartyAppsService{}
}

// ResolveAllowedApps returns the portal names the organization may use.
// restricted is false for the Owner organization and for distributors, whose
// users see every portal their roles admit; for resellers and customers it is
// true and allowed is the list to intersect with (possibly empty).
func (s *ThirdPartyAppsService) ResolveAllowedApps(orgRole, orgID string) (allowed []string, restricted bool, err error) {
	if models.IsGlobalOrgRole(orgRole) {
		return nil, false, nil
	}

	var query string
	switch strings.ToLower(orgRole) {
	case "distributor":
		// The list a distributor carries is for its resellers and customers,
		// not for itself.
		return nil, false, nil
	case "reseller":
		query = `
			SELECT d.third_party_apps
			FROM resellers r
			JOIN distributors d ON d.logto_id = r.custom_data->>'createdBy' AND d.deleted_at IS NULL
			WHERE r.logto_id = $1 AND r.deleted_at IS NULL`
	case "customer":
		// A customer hangs either from a reseller (then the distributor is two
		// levels up) or directly from a distributor.
		query = `
			SELECT COALESCE(d_direct.third_party_apps, d_via_reseller.third_party_apps)
			FROM customers c
			LEFT JOIN distributors d_direct
			       ON d_direct.logto_id = c.custom_data->>'createdBy' AND d_direct.deleted_at IS NULL
			LEFT JOIN resellers r
			       ON r.logto_id = c.custom_data->>'createdBy' AND r.deleted_at IS NULL
			LEFT JOIN distributors d_via_reseller
			       ON d_via_reseller.logto_id = r.custom_data->>'createdBy' AND d_via_reseller.deleted_at IS NULL
			WHERE c.logto_id = $1 AND c.deleted_at IS NULL`
	default:
		// Unknown organization role: fail closed.
		return []string{}, true, nil
	}

	var apps pq.StringArray
	err = database.DB.QueryRow(query, orgID).Scan(&apps)
	if err != nil {
		if err == sql.ErrNoRows {
			// The organization has no distributor above it (or does not
			// exist): nothing is granted.
			return []string{}, true, nil
		}
		return nil, true, fmt.Errorf("failed to resolve third-party apps for organization %s: %w", orgID, err)
	}

	return models.NormalizeThirdPartyAppNames(apps), true, nil
}
