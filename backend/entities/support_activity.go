/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/models"
)

// Bounds of a system's support history: enough for the workspace panel
const (
	activitySessionsLimit   = 50
	activityAccessLogsLimit = 500
	notesLimit              = 200
)

// SystemInScope reports whether a system with support history is visible to
// the caller
func (r *SupportRepository) SystemInScope(systemID, userOrgRole, userOrgID string) (bool, error) {
	conditions := []string{"s.id = $1"}
	args := []interface{}{systemID}
	rbacCondition, rbacArgs, _ := buildRBACFilter(userOrgRole, userOrgID, 2)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
	}
	var found bool
	err := r.db.QueryRow(
		fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM systems s WHERE %s)`, strings.Join(conditions, " AND ")),
		args...,
	).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("failed to check system scope: %w", err)
	}
	return found, nil
}

// GetSystemActivity returns the sessions of a system and the access log of
// all of them, newest first. The caller checks the system scope.
func (r *SupportRepository) GetSystemActivity(systemID string) (*models.SupportActivity, error) {
	activity := &models.SupportActivity{
		SystemID:   systemID,
		Sessions:   []models.SupportActivitySession{},
		AccessLogs: []models.SupportAccessLog{},
	}

	rows, err := r.db.Query(
		`SELECT id, node_id, status, started_at, expires_at, closed_at, closed_by
		 FROM support_sessions
		 WHERE system_id = $1
		 ORDER BY started_at DESC
		 LIMIT $2`,
		systemID, activitySessionsLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query system sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	sessionIDs := []string{}
	for rows.Next() {
		var s models.SupportActivitySession
		var nodeID, closedBy sql.NullString
		var closedAt sql.NullTime
		if err := rows.Scan(&s.ID, &nodeID, &s.Status, &s.StartedAt, &s.ExpiresAt, &closedAt, &closedBy); err != nil {
			return nil, fmt.Errorf("failed to scan system session: %w", err)
		}
		if nodeID.Valid {
			s.NodeID = &nodeID.String
		}
		if closedAt.Valid {
			s.ClosedAt = &closedAt.Time
		}
		if closedBy.Valid {
			s.ClosedBy = &closedBy.String
		}
		activity.Sessions = append(activity.Sessions, s)
		sessionIDs = append(sessionIDs, s.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate system sessions: %w", err)
	}
	if len(sessionIDs) == 0 {
		return activity, nil
	}

	logRows, err := r.db.Query(
		`SELECT id, session_id, operator_id, operator_name, access_type,
		        connected_at, disconnected_at, metadata
		 FROM support_access_logs
		 WHERE session_id = ANY($1::uuid[])
		 ORDER BY connected_at DESC
		 LIMIT $2`,
		pq.Array(sessionIDs), activityAccessLogsLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query system access logs: %w", err)
	}
	defer func() { _ = logRows.Close() }()

	for logRows.Next() {
		var log models.SupportAccessLog
		var operatorName, metadata sql.NullString
		var disconnectedAt sql.NullTime
		if err := logRows.Scan(
			&log.ID, &log.SessionID, &log.OperatorID, &operatorName,
			&log.AccessType, &log.ConnectedAt, &disconnectedAt, &metadata,
		); err != nil {
			return nil, fmt.Errorf("failed to scan access log: %w", err)
		}
		if operatorName.Valid {
			log.OperatorName = &operatorName.String
		}
		if disconnectedAt.Valid {
			log.DisconnectedAt = &disconnectedAt.Time
		}
		if metadata.Valid {
			log.Metadata = &metadata.String
		}
		activity.AccessLogs = append(activity.AccessLogs, log)
	}
	return activity, logRows.Err()
}

const noteColumns = `id, session_id, system_id, author_id, author_name, body, created_at, updated_at`

func scanNote(row scannable) (models.SupportSessionNote, error) {
	var n models.SupportSessionNote
	var authorName sql.NullString
	err := row.Scan(&n.ID, &n.SessionID, &n.SystemID, &n.AuthorID, &authorName, &n.Body, &n.CreatedAt, &n.UpdatedAt)
	if authorName.Valid {
		n.AuthorName = &authorName.String
	}
	return n, err
}

// GetSystemNotes returns the notes of a system, newest first. The caller
// checks the system scope.
func (r *SupportRepository) GetSystemNotes(systemID string) ([]models.SupportSessionNote, error) {
	rows, err := r.db.Query(
		`SELECT `+noteColumns+`
		 FROM support_session_notes
		 WHERE system_id = $1 AND deleted_at IS NULL
		 ORDER BY created_at DESC
		 LIMIT $2`,
		systemID, notesLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query notes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	notes := []models.SupportSessionNote{}
	for rows.Next() {
		n, err := scanNote(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan note: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// CreateNote adds a note to the system's current session: a live one if any,
// otherwise the latest. Returns nil when the system never had a session.
func (r *SupportRepository) CreateNote(systemID, authorID, authorName, body string) (*models.SupportSessionNote, error) {
	n, err := scanNote(r.db.QueryRow(
		`INSERT INTO support_session_notes (session_id, system_id, author_id, author_name, body)
		 SELECT ss.id, ss.system_id, $2, NULLIF($3, ''), $4
		 FROM support_sessions ss
		 WHERE ss.system_id = $1
		 ORDER BY ss.status IN ('active', 'pending') DESC, ss.started_at DESC
		 LIMIT 1
		 RETURNING `+noteColumns,
		systemID, authorID, authorName, body,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}
	return &n, nil
}

// GetNote returns a note that is not deleted, or nil
func (r *SupportRepository) GetNote(noteID string) (*models.SupportSessionNote, error) {
	n, err := scanNote(r.db.QueryRow(
		`SELECT `+noteColumns+` FROM support_session_notes WHERE id = $1 AND deleted_at IS NULL`,
		noteID,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}
	return &n, nil
}

// UpdateNote changes the body of a note written by the author
func (r *SupportRepository) UpdateNote(noteID, authorID, body string) (*models.SupportSessionNote, error) {
	n, err := scanNote(r.db.QueryRow(
		`UPDATE support_session_notes SET body = $3, updated_at = NOW()
		 WHERE id = $1 AND author_id = $2 AND deleted_at IS NULL
		 RETURNING `+noteColumns,
		noteID, authorID, body,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}
	return &n, nil
}

// DeleteNote soft-deletes a note written by the author
func (r *SupportRepository) DeleteNote(noteID, authorID string) (bool, error) {
	result, err := r.db.Exec(
		`UPDATE support_session_notes SET deleted_at = NOW()
		 WHERE id = $1 AND author_id = $2 AND deleted_at IS NULL`,
		noteID, authorID,
	)
	if err != nil {
		return false, fmt.Errorf("failed to delete note: %w", err)
	}
	rows, _ := result.RowsAffected()
	return rows > 0, nil
}

// SupportSystemService is a service an operator added to a system node
type SupportSystemService struct {
	Name   string
	Target string
	Label  string
	TLS    bool
}

// SaveSystemServices remembers services added to a system node, so later
// sessions get them again; a service with the same name is replaced
func (r *SupportRepository) SaveSystemServices(systemID, nodeID, userID, userName string, services []SupportSystemService) error {
	for _, svc := range services {
		_, err := r.db.Exec(
			`INSERT INTO support_system_services (system_id, node_id, name, target, label, tls, created_by_id, created_by_name)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 ON CONFLICT (system_id, node_id, name) DO UPDATE
			 SET target = EXCLUDED.target, label = EXCLUDED.label, tls = EXCLUDED.tls,
			     created_by_id = EXCLUDED.created_by_id, created_by_name = EXCLUDED.created_by_name,
			     updated_at = NOW()`,
			systemID, nodeID, svc.Name, svc.Target, svc.Label, svc.TLS, userID, userName,
		)
		if err != nil {
			return fmt.Errorf("failed to save system service: %w", err)
		}
	}
	return nil
}

// DeleteSystemService forgets a service added to a system node
func (r *SupportRepository) DeleteSystemService(systemID, nodeID, name string) error {
	if _, err := r.db.Exec(
		`DELETE FROM support_system_services WHERE system_id = $1 AND node_id = $2 AND name = $3`,
		systemID, nodeID, name,
	); err != nil {
		return fmt.Errorf("failed to delete system service: %w", err)
	}
	return nil
}
