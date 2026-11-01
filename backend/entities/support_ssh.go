/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
	"time"
)

// SSHTargetSystem is the system an SSH access is for
type SSHTargetSystem struct {
	ID               string
	Name             string
	Type             string
	SystemKey        string
	OrganizationName string
}

// FindSSHSystem returns the live system with that ID, or with that key when
// systemID is empty, if the caller can see it; nil otherwise
func (r *SupportRepository) FindSSHSystem(systemID, systemKey, userOrgRole, userOrgID string) (*SSHTargetSystem, error) {
	var sys SSHTargetSystem
	var sysType, key, orgName sql.NullString
	err := r.db.QueryRow(
		`SELECT s.id, s.name, s.type, s.system_key, uo.name
		 FROM systems s
		 LEFT JOIN unified_organizations uo ON s.organization_id = uo.logto_id
		 WHERE s.deleted_at IS NULL
		   AND (($1 <> '' AND s.id = $1) OR ($1 = '' AND s.system_key = $2))`,
		systemID, systemKey,
	).Scan(&sys.ID, &sys.Name, &sysType, &key, &orgName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find system: %w", err)
	}
	sys.Type, sys.SystemKey, sys.OrganizationName = sysType.String, key.String, orgName.String
	if sys.SystemKey == "" {
		return nil, nil
	}

	inScope, err := r.SystemInScope(sys.ID, userOrgRole, userOrgID)
	if err != nil || !inScope {
		return nil, err
	}
	return &sys, nil
}

// LiveSessionForNode returns the active session of a system's node, or of its
// first node when nodeID is empty, with its expiry; "" when there is none
func (r *SupportRepository) LiveSessionForNode(systemID, nodeID string) (string, time.Time, error) {
	var sessionID string
	var expiresAt time.Time
	err := r.db.QueryRow(
		`SELECT id, expires_at FROM support_sessions
		 WHERE system_id = $1 AND status = 'active' AND expires_at > NOW()
		   AND ($2 = '' OR COALESCE(node_id, '') = $2)
		 ORDER BY length(COALESCE(node_id, '')), node_id, started_at DESC
		 LIMIT 1`,
		systemID, nodeID,
	).Scan(&sessionID, &expiresAt)
	if err == sql.ErrNoRows {
		return "", time.Time{}, nil
	}
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to find live session: %w", err)
	}
	return sessionID, expiresAt, nil
}
