/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalSystemRepository implements SystemRepository for local database
type LocalSystemRepository struct {
	db *sql.DB
}

// NewLocalSystemRepository creates a new local system repository
func NewLocalSystemRepository() *LocalSystemRepository {
	return &LocalSystemRepository{
		db: database.DB,
	}
}

// GetTotals returns total counts and status for systems visible to the user
func (r *LocalSystemRepository) GetTotals(userOrgRole, userOrgID string, timeoutMinutes int) (*models.SystemTotals, error) {
	// Calculate cutoff time for alive/dead determination
	timeout := time.Duration(timeoutMinutes) * time.Minute
	cutoff := time.Now().Add(-timeout)

	var baseQuery string
	var args []interface{}

	// Base query with heartbeat status calculation
	baseQuery = `
		SELECT 
			COUNT(*) as total,
			SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat > $1 THEN 1 ELSE 0 END) as alive,
			SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat <= $1 THEN 1 ELSE 0 END) as dead,
			SUM(CASE WHEN h.last_heartbeat IS NULL THEN 1 ELSE 0 END) as zombie
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		JOIN customers c ON s.customer_id = c.id
		WHERE c.active = TRUE
	`
	args = append(args, cutoff)

	// Apply RBAC filtering
	switch userOrgRole {
	case "owner":
		// Owner sees all systems - no additional filtering needed

	case "distributor":
		// Distributor sees systems from customers created by them or their resellers
		baseQuery += ` AND (
			c.created_by_distributor_id = $2 OR 
			c.created_by_reseller_id IN (
				SELECT id FROM resellers WHERE distributor_id = $2 AND active = TRUE
			)
		)`
		args = append(args, userOrgID)

	case "reseller":
		// Reseller sees systems from customers they created
		baseQuery += ` AND c.created_by_reseller_id = $2`
		args = append(args, userOrgID)

	case "customer":
		// Customer sees only their own systems
		if userOrgID != "" {
			baseQuery += ` AND c.id = $2`
			args = append(args, userOrgID)
		} else {
			// No access if userOrgID is empty
			return &models.SystemTotals{
				Total:          0,
				Alive:          0,
				Dead:           0,
				Zombie:         0,
				TimeoutMinutes: timeoutMinutes,
			}, nil
		}

	default:
		// Unknown role - no access
		return &models.SystemTotals{
			Total:          0,
			Alive:          0,
			Dead:           0,
			Zombie:         0,
			TimeoutMinutes: timeoutMinutes,
		}, nil
	}

	var total, alive, dead, zombie int
	err := r.db.QueryRow(baseQuery, args...).Scan(&total, &alive, &dead, &zombie)
	if err != nil {
		return nil, fmt.Errorf("failed to get systems totals: %w", err)
	}

	return &models.SystemTotals{
		Total:          total,
		Alive:          alive,
		Dead:           dead,
		Zombie:         zombie,
		TimeoutMinutes: timeoutMinutes,
	}, nil
}
