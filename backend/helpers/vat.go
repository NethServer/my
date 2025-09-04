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

// CheckVATExists checks if a VAT exists in the specified entity table
func CheckVATExists(vat, entityType, excludeID string) (bool, error) {
	trimmedVAT := strings.TrimSpace(vat)
	if trimmedVAT == "" {
		return false, nil
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s 
		WHERE TRIM(custom_data->>'vat') = $1 
		  AND deleted_at IS NULL 
		  AND ($2 = '' OR id != $2)
	`, entityType)

	var count int
	err := database.DB.QueryRow(query, trimmedVAT, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check VAT in %s: %w", entityType, err)
	}

	return count > 0, nil
}
