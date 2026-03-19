/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package session

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/database"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/models"
)

// GenerateToken creates a cryptographically secure session token.
// Tokens are stored in plaintext in the database because they serve as a shared
// secret for inter-service communication (backend reads them to authenticate
// requests to the support service). The 256-bit entropy makes brute-forcing
// infeasible; the primary risk is a full database compromise, which would
// expose all session data regardless of token hashing.
func GenerateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// CreateSession creates a new support session for a system.
// nodeID identifies the cluster node (empty for single-node systems).
// Enforces a maximum number of active sessions per system atomically within a transaction.
// Closes any existing active/pending sessions for the same system+node to prevent orphans.
func CreateSession(systemID, nodeID string) (*models.SupportSession, error) {
	log := logger.ComponentLogger("session")

	token, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	reconnectToken, err := GenerateToken()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	expiresAt := now.Add(configuration.Config.SessionDefaultDuration)
	maxSessions := configuration.Config.MaxSessionsPerSystem

	// Use NULL for empty node_id
	var nodeIDParam interface{}
	if nodeID != "" {
		nodeIDParam = nodeID
	}

	// Use a transaction to atomically close orphans, check limits, and insert
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Close any existing active/pending sessions for this system+node combination
	var closeResult sql.Result
	if nodeID == "" {
		closeResult, err = tx.Exec(
			`UPDATE support_sessions
			 SET status = 'closed', closed_at = NOW(), closed_by = 'replaced', updated_at = NOW()
			 WHERE system_id = $1 AND node_id IS NULL AND status IN ('pending', 'active')`,
			systemID,
		)
	} else {
		closeResult, err = tx.Exec(
			`UPDATE support_sessions
			 SET status = 'closed', closed_at = NOW(), closed_by = 'replaced', updated_at = NOW()
			 WHERE system_id = $1 AND node_id = $2 AND status IN ('pending', 'active')`,
			systemID, nodeID,
		)
	}
	if err != nil {
		log.Warn().Err(err).
			Str("system_id", systemID).Str("node_id", nodeID).
			Msg("failed to close existing sessions before creating new one")
	} else if rows, _ := closeResult.RowsAffected(); rows > 0 {
		log.Info().
			Str("system_id", systemID).Str("node_id", nodeID).
			Int64("closed_count", rows).
			Msg("closed orphaned sessions before creating new one")
	}

	// Check session limit within the transaction (atomic with the close + insert)
	if maxSessions > 0 {
		var activeCount int
		err = tx.QueryRow(
			`SELECT COUNT(*) FROM support_sessions WHERE system_id = $1 AND status IN ('pending', 'active')`,
			systemID,
		).Scan(&activeCount)
		if err != nil {
			return nil, fmt.Errorf("failed to check session count: %w", err)
		}
		if activeCount >= maxSessions {
			return nil, fmt.Errorf("maximum active sessions per system reached (%d)", maxSessions)
		}
	}

	var session models.SupportSession
	var scannedNodeID sql.NullString
	err = tx.QueryRow(
		`INSERT INTO support_sessions (system_id, node_id, session_token, reconnect_token, started_at, expires_at, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		 RETURNING id, system_id, node_id, session_token, reconnect_token, started_at, expires_at, status, created_at, updated_at`,
		systemID, nodeIDParam, token, reconnectToken, now, expiresAt,
	).Scan(
		&session.ID, &session.SystemID, &scannedNodeID, &session.SessionToken, &session.ReconnectToken,
		&session.StartedAt, &session.ExpiresAt, &session.Status,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit session creation: %w", err)
	}

	if scannedNodeID.Valid {
		session.NodeID = scannedNodeID.String
	}

	log.Info().
		Str("session_id", session.ID).
		Str("system_id", systemID).
		Str("node_id", nodeID).
		Msg("session created")

	return &session, nil
}

// ActivateSession marks a session as active (tunnel connected)
func ActivateSession(sessionID string) error {
	_, err := database.DB.Exec(
		`UPDATE support_sessions SET status = 'active', updated_at = NOW()
		 WHERE id = $1 AND status = 'pending'`,
		sessionID,
	)
	return err
}

// GetActiveSession returns the active or pending session for a system+node combination.
// nodeID can be empty for single-node systems.
func GetActiveSession(systemID, nodeID string) (*models.SupportSession, error) {
	var session models.SupportSession
	var closedAt sql.NullTime
	var closedBy sql.NullString
	var reconnectToken sql.NullString
	var scannedNodeID sql.NullString

	var query string
	var args []interface{}

	if nodeID == "" {
		query = `SELECT id, system_id, node_id, session_token, reconnect_token, started_at, expires_at, status,
		                closed_at, closed_by, created_at, updated_at
		         FROM support_sessions
		         WHERE system_id = $1 AND node_id IS NULL AND status IN ('pending', 'active')
		         ORDER BY created_at DESC LIMIT 1`
		args = []interface{}{systemID}
	} else {
		query = `SELECT id, system_id, node_id, session_token, reconnect_token, started_at, expires_at, status,
		                closed_at, closed_by, created_at, updated_at
		         FROM support_sessions
		         WHERE system_id = $1 AND node_id = $2 AND status IN ('pending', 'active')
		         ORDER BY created_at DESC LIMIT 1`
		args = []interface{}{systemID, nodeID}
	}

	err := database.DB.QueryRow(query, args...).Scan(
		&session.ID, &session.SystemID, &scannedNodeID, &session.SessionToken, &reconnectToken,
		&session.StartedAt, &session.ExpiresAt, &session.Status,
		&closedAt, &closedBy, &session.CreatedAt, &session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if scannedNodeID.Valid {
		session.NodeID = scannedNodeID.String
	}
	if closedAt.Valid {
		session.ClosedAt = &closedAt.Time
	}
	if closedBy.Valid {
		session.ClosedBy = &closedBy.String
	}
	if reconnectToken.Valid {
		session.ReconnectToken = reconnectToken.String
	}

	return &session, nil
}

// ValidateReconnectToken checks if a reconnect token matches the session
func ValidateReconnectToken(sessionID, token string) bool {
	if token == "" {
		return false
	}
	var storedToken sql.NullString
	err := database.DB.QueryRow(
		`SELECT reconnect_token FROM support_sessions WHERE id = $1 AND status IN ('pending', 'active')`,
		sessionID,
	).Scan(&storedToken)
	if err != nil || !storedToken.Valid {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(storedToken.String), []byte(token)) == 1
}

// GetSessionTokenByID returns the session_token for a session (for internal auth)
func GetSessionTokenByID(sessionID string) (string, error) {
	var token string
	err := database.DB.QueryRow(
		`SELECT session_token FROM support_sessions WHERE id = $1 AND status IN ('pending', 'active')`,
		sessionID,
	).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("session not found or not active: %w", err)
	}
	return token, nil
}

// GetSessionByID returns a session by its ID
func GetSessionByID(sessionID string) (*models.SupportSession, error) {
	var session models.SupportSession
	var closedAt sql.NullTime
	var closedBy sql.NullString
	var scannedNodeID sql.NullString

	err := database.DB.QueryRow(
		`SELECT id, system_id, node_id, session_token, started_at, expires_at, status,
		        closed_at, closed_by, created_at, updated_at
		 FROM support_sessions
		 WHERE id = $1`,
		sessionID,
	).Scan(
		&session.ID, &session.SystemID, &scannedNodeID, &session.SessionToken,
		&session.StartedAt, &session.ExpiresAt, &session.Status,
		&closedAt, &closedBy, &session.CreatedAt, &session.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if scannedNodeID.Valid {
		session.NodeID = scannedNodeID.String
	}
	if closedAt.Valid {
		session.ClosedAt = &closedAt.Time
	}
	if closedBy.Valid {
		session.ClosedBy = &closedBy.String
	}

	return &session, nil
}

// CloseSession closes a support session
func CloseSession(sessionID, closedBy string) error {
	result, err := database.DB.Exec(
		`UPDATE support_sessions
		 SET status = 'closed', closed_at = NOW(), closed_by = $2, updated_at = NOW()
		 WHERE id = $1 AND status IN ('pending', 'active')`,
		sessionID, closedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found or already closed")
	}

	logger.ComponentLogger("session").Info().
		Str("session_id", sessionID).
		Str("closed_by", closedBy).
		Msg("session closed")

	return nil
}

// ExpireSessions marks expired sessions
func ExpireSessions() (int64, error) {
	result, err := database.DB.Exec(
		`UPDATE support_sessions
		 SET status = 'expired', closed_at = NOW(), closed_by = 'timeout', updated_at = NOW()
		 WHERE status IN ('pending', 'active') AND expires_at < NOW()`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to expire sessions: %w", err)
	}

	rows, _ := result.RowsAffected()
	return rows, nil
}

// GetActiveSessions returns the count of active sessions
func GetActiveSessions() (int, error) {
	var count int
	err := database.DB.QueryRow(
		`SELECT COUNT(*) FROM support_sessions WHERE status = 'active'`,
	).Scan(&count)
	return count, err
}

// SaveDiagnostics stores diagnostic report data on a session.
// This is best-effort: if the session no longer exists the error is ignored by the caller.
func SaveDiagnostics(sessionID string, data json.RawMessage) error {
	_, err := database.DB.Exec(
		`UPDATE support_sessions
		 SET diagnostics = $1, diagnostics_at = NOW(), updated_at = NOW()
		 WHERE id = $2`,
		string(data), sessionID,
	)
	return err
}
