/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"fmt"
	"strings"

	"github.com/nethesis/my/backend/database"
)

// CheckVATExists checks if a VAT exists in the specified entity table,
// restricted to the organizations visible to the caller (same RBAC scope as
// the List endpoints): owner sees everything, a distributor sees its own
// subtree, a reseller sees the customers it created. Callers with no
// visibility on the entity type get false, so the validator never leaks the
// existence of a VAT in an unrelated hierarchy.
func CheckVATExists(vat, entityType, excludeID, userOrgRole, userOrgID string) (bool, error) {
	trimmedVAT := strings.TrimSpace(vat)
	if trimmedVAT == "" {
		return false, nil
	}

	role := strings.ToLower(userOrgRole)

	scopeClause := ""
	switch entityType {
	case "distributors":
		if role != "owner" {
			return false, nil
		}
	case "resellers":
		switch role {
		case "owner":
		case "distributor":
			scopeClause = " AND custom_data->>'createdBy' = $3"
		default:
			return false, nil
		}
	case "customers":
		switch role {
		case "owner":
		case "distributor":
			scopeClause = ` AND (custom_data->>'createdBy' = $3 OR custom_data->>'createdBy' IN (SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $3 AND deleted_at IS NULL))`
		case "reseller":
			scopeClause = " AND custom_data->>'createdBy' = $3"
		default:
			return false, nil
		}
	default:
		return false, fmt.Errorf("invalid entity type: %s", entityType)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE TRIM(custom_data->>'vat') = $1
		  AND deleted_at IS NULL
		  AND ($2 = '' OR id != $2)%s
	`, entityType, scopeClause)

	args := []interface{}{trimmedVAT, excludeID}
	if scopeClause != "" {
		args = append(args, userOrgID)
	}

	var count int
	err := database.DB.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check VAT in %s: %w", entityType, err)
	}

	return count > 0, nil
}
