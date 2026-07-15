/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
)

// AlertAssignment is the current assignee of one alert. One row per
// (organization_id, fingerprint); rows are deleted by collect when the alert
// resolves, so presence means "someone is working on this right now".
// AssignedUserOrg* describe the assignee's own organization (from the JWT),
// which with cross-hierarchy takeover may differ from the alert's org.
type AlertAssignment struct {
	OrganizationID      string    `json:"organization_id"`
	Fingerprint         string    `json:"fingerprint"`
	AssignedUserID      string    `json:"assigned_user_id"`
	AssignedUserName    string    `json:"assigned_user_name"`
	AssignedUserOrgID   string    `json:"assigned_user_org_id"`
	AssignedUserOrgName string    `json:"assigned_user_org_name"`
	AssignedAt          time.Time `json:"assigned_at"`
}

// LocalAlertAssignmentRepository reads / writes alert_assignments.
type LocalAlertAssignmentRepository struct {
	db *sql.DB
}

func NewLocalAlertAssignmentRepository() *LocalAlertAssignmentRepository {
	return &LocalAlertAssignmentRepository{db: database.DB}
}

// Upsert assigns the alert to the given user, replacing any existing assignee
// (takeover). Returns the previous assignee's name (empty when the alert was
// unassigned or already assigned to the same user) so the caller can record
// the handover in the activity timeline.
func (r *LocalAlertAssignmentRepository) Upsert(orgID, fingerprint, userID, userName, userOrgID, userOrgName string) (previousName string, assignment *AlertAssignment, err error) {
	var prevName sql.NullString
	var out AlertAssignment
	err = r.db.QueryRow(
		`WITH previous AS (
		     SELECT assigned_user_id, assigned_user_name
		     FROM alert_assignments
		     WHERE organization_id = $1 AND fingerprint = $2
		 ), upserted AS (
		     INSERT INTO alert_assignments (organization_id, fingerprint, assigned_user_id, assigned_user_name, assigned_user_org_id, assigned_user_org_name, assigned_at)
		     VALUES ($1, $2, $3, NULLIF($4,''), NULLIF($5,''), NULLIF($6,''), NOW())
		     ON CONFLICT (organization_id, fingerprint) DO UPDATE
		     SET assigned_user_id       = EXCLUDED.assigned_user_id,
		         assigned_user_name     = EXCLUDED.assigned_user_name,
		         assigned_user_org_id   = EXCLUDED.assigned_user_org_id,
		         assigned_user_org_name = EXCLUDED.assigned_user_org_name,
		         assigned_at            = EXCLUDED.assigned_at
		     RETURNING organization_id, fingerprint, assigned_user_id,
		               COALESCE(assigned_user_name,'')     AS assigned_user_name,
		               COALESCE(assigned_user_org_id,'')   AS assigned_user_org_id,
		               COALESCE(assigned_user_org_name,'') AS assigned_user_org_name,
		               assigned_at
		 )
		 SELECT u.organization_id, u.fingerprint, u.assigned_user_id, u.assigned_user_name, u.assigned_user_org_id, u.assigned_user_org_name, u.assigned_at,
		        COALESCE((SELECT CASE WHEN p.assigned_user_id <> $3 THEN COALESCE(p.assigned_user_name, p.assigned_user_id) END FROM previous p), '')
		 FROM upserted u`,
		orgID, fingerprint, userID, userName, userOrgID, userOrgName,
	).Scan(&out.OrganizationID, &out.Fingerprint, &out.AssignedUserID, &out.AssignedUserName, &out.AssignedUserOrgID, &out.AssignedUserOrgName, &out.AssignedAt, &prevName)
	if err != nil {
		return "", nil, fmt.Errorf("upsert alert_assignment: %w", err)
	}
	return prevName.String, &out, nil
}

// GetByFingerprints returns the assignments for the given (orgID, fingerprint)
// pairs in one round-trip, keyed by "orgID\x00fingerprint" via AssignmentKey.
// Fingerprints missing from the map are unassigned. Both slices must be the
// same length (pairwise).
func (r *LocalAlertAssignmentRepository) GetByFingerprints(orgIDs, fingerprints []string) (map[string]AlertAssignment, error) {
	out := make(map[string]AlertAssignment, len(fingerprints))
	if len(fingerprints) == 0 {
		return out, nil
	}
	rows, err := r.db.Query(
		`SELECT a.organization_id, a.fingerprint, a.assigned_user_id,
		        COALESCE(a.assigned_user_name,''), COALESCE(a.assigned_user_org_id,''), COALESCE(a.assigned_user_org_name,''),
		        a.assigned_at
		 FROM alert_assignments a
		 JOIN unnest($1::text[], $2::text[]) AS t(organization_id, fingerprint)
		   ON a.organization_id = t.organization_id AND a.fingerprint = t.fingerprint`,
		pq.Array(orgIDs), pq.Array(fingerprints),
	)
	if err != nil {
		return nil, fmt.Errorf("query alert_assignments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var a AlertAssignment
		if err := rows.Scan(&a.OrganizationID, &a.Fingerprint, &a.AssignedUserID, &a.AssignedUserName, &a.AssignedUserOrgID, &a.AssignedUserOrgName, &a.AssignedAt); err != nil {
			return nil, fmt.Errorf("scan alert_assignment: %w", err)
		}
		out[AssignmentKey(a.OrganizationID, a.Fingerprint)] = a
	}
	return out, rows.Err()
}

// AssignmentKey builds the map key used by GetByFingerprints. NUL is not a
// valid character in either component so the concatenation is unambiguous.
func AssignmentKey(orgID, fingerprint string) string {
	return orgID + "\x00" + fingerprint
}
