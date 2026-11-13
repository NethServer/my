/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"

	"github.com/nethesis/my/backend/database"
)

// ResellerOrgTransfer mirrors a row in reseller_org_transfers. The counters
// record what followed the reseller and what the transfer dropped.
type ResellerOrgTransfer struct {
	ResellerID          string
	FromOrgID           string
	ToOrgID             string
	ActorUserID         sql.NullString
	ActorOrganizationID sql.NullString
	ActorIP             sql.NullString
	UserAgent           sql.NullString
	CustomersCount      int
	SystemsCount        int
	RenewalsStopped     int
	AssignmentsReleased int
}

// LocalResellerOrgTransfersRepository writes the audit table of reseller
// "managed by" transfers.
type LocalResellerOrgTransfersRepository struct {
	db *sql.DB
}

func NewLocalResellerOrgTransfersRepository() *LocalResellerOrgTransfersRepository {
	return &LocalResellerOrgTransfersRepository{db: database.DB}
}

// Insert appends a single transfer row. Returns the generated id.
func (r *LocalResellerOrgTransfersRepository) Insert(t *ResellerOrgTransfer) (string, error) {
	if t == nil {
		return "", fmt.Errorf("transfer is required")
	}
	if t.ResellerID == "" || t.FromOrgID == "" || t.ToOrgID == "" {
		return "", fmt.Errorf("reseller_id, from_org_id and to_org_id are required")
	}
	var id string
	err := r.db.QueryRow(
		`INSERT INTO reseller_org_transfers
		    (reseller_id, from_org_id, to_org_id,
		     actor_user_id, actor_organization_id, actor_ip, user_agent,
		     customers_count, systems_count, renewals_stopped, assignments_released)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		t.ResellerID, t.FromOrgID, t.ToOrgID,
		t.ActorUserID, t.ActorOrganizationID, t.ActorIP, t.UserAgent,
		t.CustomersCount, t.SystemsCount, t.RenewalsStopped, t.AssignmentsReleased,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert reseller_org_transfers: %w", err)
	}
	return id, nil
}
