/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

var (
	ErrSeatNotFound     = errors.New("seat not found")
	ErrSeatAlreadyHeld  = errors.New("instance already holds a seat")
	ErrNoFreeSeat       = errors.New("no free seat")
	ErrInstanceNotFound = errors.New("application instance not found")
)

// Seat release reasons.
const (
	SeatReleasedManual          = "manual"
	SeatReleasedInstanceRemoved = "instance_removed"
)

// LocalEntitlementSeatRepository reads / writes entitlement_seats.
type LocalEntitlementSeatRepository struct {
	db *sql.DB
}

func NewLocalEntitlementSeatRepository() *LocalEntitlementSeatRepository {
	return &LocalEntitlementSeatRepository{db: database.DB}
}

// ListByGrants maps each grant to its held seats, oldest first. A seat is in
// quota while its rank is within the seats bought.
func (r *LocalEntitlementSeatRepository) ListByGrants(grantIDs []string) (map[string][]models.EntitlementSeat, error) {
	out := make(map[string][]models.EntitlementSeat, len(grantIDs))
	if len(grantIDs) == 0 {
		return out, nil
	}

	rows, err := r.db.Query(`
		SELECT s.grant_id, s.id, s.module_id, s.assigned_at, s.assigned_by,
		       ROW_NUMBER() OVER (PARTITION BY s.grant_id ORDER BY s.assigned_at, s.id) <= COALESCE(e.seats, 0) AS in_quota
		FROM entitlement_seats s
		JOIN system_entitlements e ON e.id = s.grant_id
		WHERE s.grant_id = ANY($1::uuid[]) AND s.released_at IS NULL
		ORDER BY s.grant_id, s.assigned_at, s.id`, pq.Array(grantIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to list seats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var grantID string
		var seat models.EntitlementSeat
		var assignedBy []byte
		if err := rows.Scan(&grantID, &seat.ID, &seat.ModuleID, &seat.AssignedAt, &assignedBy, &seat.InQuota); err != nil {
			return nil, fmt.Errorf("failed to scan seat: %w", err)
		}
		if len(assignedBy) > 0 {
			_ = json.Unmarshal(assignedBy, &seat.AssignedBy)
		}
		out[grantID] = append(out[grantID], seat)
	}
	return out, rows.Err()
}

// Assign gives a seat of the grant to a live instance of appliesTo on the
// system. The grant row is locked so that concurrent assignments cannot go
// beyond the seats bought.
func (r *LocalEntitlementSeatRepository) Assign(grantID, systemID, appliesTo, moduleID string, assignedBy map[string]interface{}) (*models.EntitlementSeat, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin seat assignment: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var seats sql.NullInt64
	if err := tx.QueryRow(`SELECT seats FROM system_entitlements WHERE id = $1 FOR UPDATE`, grantID).Scan(&seats); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEntitlementNotFound
		}
		return nil, fmt.Errorf("failed to lock grant: %w", err)
	}

	var live bool
	if err := tx.QueryRow(`
		SELECT EXISTS (SELECT 1 FROM applications
		                WHERE system_id = $1 AND module_id = $2 AND instance_of = $3 AND deleted_at IS NULL)`,
		systemID, moduleID, appliesTo).Scan(&live); err != nil {
		return nil, fmt.Errorf("failed to look up instance: %w", err)
	}
	if !live {
		return nil, ErrInstanceNotFound
	}

	var held int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM entitlement_seats WHERE grant_id = $1 AND released_at IS NULL`, grantID).Scan(&held); err != nil {
		return nil, fmt.Errorf("failed to count seats: %w", err)
	}
	if int64(held) >= seats.Int64 {
		var already bool
		if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM entitlement_seats WHERE grant_id = $1 AND module_id = $2 AND released_at IS NULL)`,
			grantID, moduleID).Scan(&already); err == nil && already {
			return nil, ErrSeatAlreadyHeld
		}
		return nil, ErrNoFreeSeat
	}

	var assignedByJSON interface{}
	if assignedBy != nil {
		assignedByJSON, _ = json.Marshal(assignedBy)
	}

	seat := models.EntitlementSeat{ModuleID: moduleID, AssignedBy: assignedBy, InQuota: true}
	err = tx.QueryRow(`
		INSERT INTO entitlement_seats (grant_id, module_id, assigned_by)
		VALUES ($1, $2, $3)
		RETURNING id, assigned_at`, grantID, moduleID, assignedByJSON).Scan(&seat.ID, &seat.AssignedAt)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, ErrSeatAlreadyHeld
		}
		return nil, fmt.Errorf("failed to assign seat: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit seat assignment: %w", err)
	}
	return &seat, nil
}

// Release frees the seat an instance holds on the grant; the row is kept for
// audit.
func (r *LocalEntitlementSeatRepository) Release(grantID, moduleID string, releasedBy map[string]interface{}) error {
	var releasedByJSON interface{}
	if releasedBy != nil {
		releasedByJSON, _ = json.Marshal(releasedBy)
	}
	res, err := r.db.Exec(`
		UPDATE entitlement_seats
		SET released_at = NOW(), released_reason = $3, released_by = $4
		WHERE grant_id = $1 AND module_id = $2 AND released_at IS NULL`,
		grantID, moduleID, SeatReleasedManual, releasedByJSON)
	if err != nil {
		return fmt.Errorf("failed to release seat: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrSeatNotFound
	}
	return nil
}
