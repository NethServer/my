/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/nethesis/my/backend/database"
)

// SystemOrgTransfer mirrors a row in system_org_transfers. Counter columns are
// captured at the end of the migration so a single row tells the whole story
// of what landed where.
type SystemOrgTransfer struct {
	ID                    string
	SystemID              string
	SystemKey             string
	FromOrgID             string
	ToOrgID               string
	ActorUserID           sql.NullString
	ActorUserEmail        sql.NullString
	ActorOrganizationID   sql.NullString
	ActorIP               sql.NullString
	UserAgent             sql.NullString
	BackupsCopied         int
	BackupsDeleted        int
	SilencesCleared       int
	HistoryRowsReassigned int
	AppsUnassigned        int
	OccurredAt            time.Time
}

// LocalSystemOrgTransfersRepository writes and reads the audit table for
// cross-organization system reassignments. Reads exist for forensics and for
// the GDPR purge path that iterates every prior org of a system on destroy.
type LocalSystemOrgTransfersRepository struct {
	db *sql.DB
}

func NewLocalSystemOrgTransfersRepository() *LocalSystemOrgTransfersRepository {
	return &LocalSystemOrgTransfersRepository{db: database.DB}
}

// Insert appends a single transfer row. Returns the generated id.
func (r *LocalSystemOrgTransfersRepository) Insert(t *SystemOrgTransfer) (string, error) {
	if t == nil {
		return "", fmt.Errorf("transfer is required")
	}
	if t.SystemID == "" || t.SystemKey == "" || t.FromOrgID == "" || t.ToOrgID == "" {
		return "", fmt.Errorf("system_id, system_key, from_org_id and to_org_id are required")
	}
	var id string
	err := r.db.QueryRow(
		`INSERT INTO system_org_transfers
		    (system_id, system_key, from_org_id, to_org_id,
		     actor_user_id, actor_user_email, actor_organization_id,
		     actor_ip, user_agent,
		     backups_copied, backups_deleted, silences_cleared,
		     history_rows_reassigned, apps_unassigned)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		 RETURNING id`,
		t.SystemID, t.SystemKey, t.FromOrgID, t.ToOrgID,
		t.ActorUserID, t.ActorUserEmail, t.ActorOrganizationID,
		t.ActorIP, t.UserAgent,
		t.BackupsCopied, t.BackupsDeleted, t.SilencesCleared,
		t.HistoryRowsReassigned, t.AppsUnassigned,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("insert system_org_transfers: %w", err)
	}
	return id, nil
}

// PriorOrgIDsForSystem returns the distinct from_org_id values where the
// system used to live, ordered most-recent first. DestroySystem uses this to
// run DeleteBackupPrefix on every historical prefix in addition to the
// current one — a partial cleanup failure during reassignment cannot leak
// ciphertext past the destroy.
func (r *LocalSystemOrgTransfersRepository) PriorOrgIDsForSystem(systemID string) ([]string, error) {
	if systemID == "" {
		return nil, fmt.Errorf("systemID is required")
	}
	rows, err := r.db.Query(
		`SELECT DISTINCT from_org_id
		 FROM system_org_transfers
		 WHERE system_id = $1
		 ORDER BY from_org_id`,
		systemID,
	)
	if err != nil {
		return nil, fmt.Errorf("query system_org_transfers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []string{}
	for rows.Next() {
		var orgID string
		if err := rows.Scan(&orgID); err != nil {
			return nil, fmt.Errorf("scan from_org_id: %w", err)
		}
		out = append(out, orgID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
