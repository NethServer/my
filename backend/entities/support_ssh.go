/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
)

// SSHTargetSystem is the system an SSH access request points to
type SSHTargetSystem struct {
	ID        string
	Name      string
	Type      string
	SystemKey string
}

// FindSystemByKey returns the live system with the given key when the caller
// can see it, nil otherwise
func (r *SupportRepository) FindSystemByKey(systemKey, userOrgRole, userOrgID string) (*SSHTargetSystem, error) {
	var sys SSHTargetSystem
	var sysType sql.NullString
	err := r.db.QueryRow(
		`SELECT id, name, type, system_key FROM systems
		 WHERE system_key = $1 AND deleted_at IS NULL`,
		systemKey,
	).Scan(&sys.ID, &sys.Name, &sysType, &sys.SystemKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find system: %w", err)
	}
	sys.Type = sysType.String

	inScope, err := r.SystemInScope(sys.ID, userOrgRole, userOrgID)
	if err != nil || !inScope {
		return nil, err
	}
	return &sys, nil
}

// LiveSessionForNode returns the active session of a system's node, or of its
// first node when nodeID is empty; "" when there is none
func (r *SupportRepository) LiveSessionForNode(systemID, nodeID string) (string, error) {
	var sessionID string
	err := r.db.QueryRow(
		`SELECT id FROM support_sessions
		 WHERE system_id = $1 AND status = 'active'
		   AND ($2 = '' OR COALESCE(node_id, '') = $2)
		 ORDER BY length(COALESCE(node_id, '')), node_id, started_at DESC
		 LIMIT 1`,
		systemID, nodeID,
	).Scan(&sessionID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to find live session: %w", err)
	}
	return sessionID, nil
}
