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

// CustomerOrgTransfer mirrors a row in customer_org_transfers. The counters
// record what the transfer touched besides the customer itself.
type CustomerOrgTransfer struct {
	CustomerID          string
	FromOrgID           string
	ToOrgID             string
	ActorUserID         sql.NullString
	ActorOrganizationID sql.NullString
	ActorIP             sql.NullString
	UserAgent           sql.NullString
	SystemsCount        int
	RenewalsStopped     int
	SilencesCleared     int
	AssignmentsReleased int
}

// LocalCustomerOrgTransfersRepository writes the audit table of customer
// "managed by" transfers.
type LocalCustomerOrgTransfersRepository struct {
	db *sql.DB
}

func NewLocalCustomerOrgTransfersRepository() *LocalCustomerOrgTransfersRepository {
	return &LocalCustomerOrgTransfersRepository{db: database.DB}
}

// Insert appends a single transfer row. Returns the generated id.
func (r *LocalCustomerOrgTransfersRepository) Insert(t *CustomerOrgTransfer) (string, error) {
	if t == nil {
		return "", fmt.Errorf("transfer is required")
	}
	if t.CustomerID == "" || t.FromOrgID == "" || t.ToOrgID == "" {
		return "", fmt.Errorf("customer_id, from_org_id and to_org_id are required")
	}
	var id string
	err := r.db.QueryRow(
		`INSERT INTO customer_org_transfers
		    (customer_id, from_org_id, to_org_id,
		     actor_user_id, actor_organization_id, actor_ip, user_agent,
		     systems_count, renewals_stopped, silences_cleared, assignments_released)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id`,
		t.CustomerID, t.FromOrgID, t.ToOrgID,
		t.ActorUserID, t.ActorOrganizationID, t.ActorIP, t.UserAgent,
		t.SystemsCount, t.RenewalsStopped, t.SilencesCleared, t.AssignmentsReleased,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert customer_org_transfers: %w", err)
	}
	return id, nil
}
