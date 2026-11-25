/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// ImpersonationService provides database operations for the impersonation audit
type ImpersonationService struct {
	db *sql.DB
}

// NewImpersonationService creates a new impersonation service
func NewImpersonationService() *ImpersonationService {
	return &ImpersonationService{
		db: database.DB,
	}
}

// =============================================================================
// AUDIT MANAGEMENT
// =============================================================================

// LogImpersonationAction logs an action performed during impersonation
func (s *ImpersonationService) LogImpersonationAction(entry *models.ImpersonationAuditEntry) error {
	entry.ID = uuid.New().String()
	entry.Timestamp = time.Now()

	query := `
		INSERT INTO impersonation_audit (
			id, session_id, impersonator_user_id, impersonated_user_id, action_type,
			api_endpoint, http_method, request_data, response_status, response_status_text, timestamp,
			impersonator_username, impersonated_username, impersonator_name, impersonated_name
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := s.db.Exec(query,
		entry.ID,
		entry.SessionID,
		entry.ImpersonatorUserID,
		entry.ImpersonatedUserID,
		entry.ActionType,
		entry.APIEndpoint,
		entry.HTTPMethod,
		entry.RequestData,
		entry.ResponseStatus,
		entry.ResponseStatusText,
		entry.Timestamp,
		entry.ImpersonatorUsername,
		entry.ImpersonatedUsername,
		entry.ImpersonatorName,
		entry.ImpersonatedName,
	)

	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("session_id", entry.SessionID).
			Str("action_type", entry.ActionType).
			Msg("Failed to log impersonation action")
		return err
	}

	return nil
}

// GetSessionAuditHistory retrieves audit history for a specific impersonation session with pagination
func (s *ImpersonationService) GetSessionAuditHistory(sessionID string, page, pageSize int) ([]models.ImpersonationAuditEntry, int, error) {
	// Get total count of audit entries
	countQuery := `
		SELECT COUNT(*)
		FROM impersonation_audit
		WHERE session_id = $1
	`

	var totalCount int
	err := s.db.QueryRow(countQuery, sessionID).Scan(&totalCount)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("session_id", sessionID).
			Msg("Failed to get audit entry count")
		return nil, 0, err
	}

	// Calculate offset
	offset := (page - 1) * pageSize

	// Get paginated entries
	query := `
		SELECT id, session_id, impersonator_user_id, impersonated_user_id, action_type,
			   api_endpoint, http_method, request_data, response_status, response_status_text, timestamp,
			   impersonator_username, impersonated_username, impersonator_name, impersonated_name
		FROM impersonation_audit
		WHERE session_id = $1
		ORDER BY timestamp ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, sessionID, pageSize, offset)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("session_id", sessionID).
			Msg("Failed to get session audit history")
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var entries []models.ImpersonationAuditEntry
	for rows.Next() {
		var entry models.ImpersonationAuditEntry
		err := rows.Scan(
			&entry.ID,
			&entry.SessionID,
			&entry.ImpersonatorUserID,
			&entry.ImpersonatedUserID,
			&entry.ActionType,
			&entry.APIEndpoint,
			&entry.HTTPMethod,
			&entry.RequestData,
			&entry.ResponseStatus,
			&entry.ResponseStatusText,
			&entry.Timestamp,
			&entry.ImpersonatorUsername,
			&entry.ImpersonatedUsername,
			&entry.ImpersonatorName,
			&entry.ImpersonatedName,
		)
		if err != nil {
			logger.ComponentLogger("impersonation").Error().
				Err(err).
				Str("session_id", sessionID).
				Msg("Failed to scan session audit entry")
			continue
		}
		entries = append(entries, entry)
	}

	return entries, totalCount, nil
}

// SessionExists checks if a session exists in the database (regardless of ownership)
func (s *ImpersonationService) SessionExists(sessionID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM impersonation_audit WHERE session_id = $1)`

	var exists bool
	err := s.db.QueryRow(query, sessionID).Scan(&exists)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("session_id", sessionID).
			Msg("Failed to check if session exists")
		return false, err
	}

	return exists, nil
}

// GetUserSessions retrieves all impersonation sessions for a specific user (being impersonated)
func (s *ImpersonationService) GetUserSessions(userID string, page, pageSize int) ([]models.ImpersonationSession, int, error) {
	// Get total count of sessions
	countQuery := `
		SELECT COUNT(DISTINCT session_id)
		FROM impersonation_audit
		WHERE impersonated_user_id = $1
	`
	var total int
	err := s.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get sessions count")
		return nil, 0, err
	}

	// Get sessions with aggregated data
	query := `
		WITH session_data AS (
			SELECT
				session_id,
				impersonator_user_id,
				impersonated_user_id,
				impersonator_username,
				impersonated_username,
				MAX(impersonator_name) as impersonator_name,
				MAX(impersonated_name) as impersonated_name,
				MIN(timestamp) as start_time,
				MAX(CASE WHEN action_type = 'session_end' THEN timestamp END) as end_time,
				COUNT(*) as action_count,
				CASE
					WHEN MAX(CASE WHEN action_type = 'session_end' THEN timestamp END) IS NULL THEN 'active'
					ELSE 'completed'
				END as status
			FROM impersonation_audit
			WHERE impersonated_user_id = $1
			GROUP BY session_id, impersonator_user_id, impersonated_user_id,
					 impersonator_username, impersonated_username
		)
		SELECT
			session_id,
			impersonator_user_id,
			impersonated_user_id,
			impersonator_username,
			impersonated_username,
			impersonator_name,
			impersonated_name,
			start_time,
			end_time,
			CASE
				WHEN end_time IS NOT NULL THEN EXTRACT(EPOCH FROM (end_time - start_time))/60
				ELSE NULL
			END as duration_minutes,
			action_count,
			status
		FROM session_data
		ORDER BY start_time DESC
		LIMIT $2 OFFSET $3
	`

	// Convert page/pageSize to limit/offset for SQL query
	limit := pageSize
	offset := (page - 1) * pageSize

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get user sessions")
		return nil, 0, err
	}
	defer func() { _ = rows.Close() }()

	var sessions []models.ImpersonationSession
	for rows.Next() {
		var session models.ImpersonationSession
		var durationMinutes *float64 // Use float64 to handle EXTRACT result

		err := rows.Scan(
			&session.SessionID,
			&session.ImpersonatorUserID,
			&session.ImpersonatedUserID,
			&session.ImpersonatorUsername,
			&session.ImpersonatedUsername,
			&session.ImpersonatorName,
			&session.ImpersonatedName,
			&session.StartTime,
			&session.EndTime,
			&durationMinutes,
			&session.ActionCount,
			&session.Status,
		)
		if err != nil {
			logger.ComponentLogger("impersonation").Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to scan session entry")
			continue
		}

		// Convert duration from float64 to *int (rounded)
		if durationMinutes != nil {
			duration := int(*durationMinutes + 0.5) // Round to nearest minute
			session.Duration = &duration
		}

		sessions = append(sessions, session)
	}

	return sessions, total, nil
}
