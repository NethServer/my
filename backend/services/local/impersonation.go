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
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// ImpersonationService provides database operations for impersonation consent and audit
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
// CONSENT MANAGEMENT
// =============================================================================

// EnableConsent creates or updates impersonation consent for a user
func (s *ImpersonationService) EnableConsent(userID string, durationHours int) (*models.ImpersonationConsent, error) {
	// First, disable any existing consent
	if err := s.DisableConsent(userID); err != nil {
		logger.ComponentLogger("impersonation").Warn().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to disable existing consent before enabling new one")
	}

	consent := &models.ImpersonationConsent{
		ID:                 uuid.New().String(),
		UserID:             userID,
		ExpiresAt:          time.Now().Add(time.Duration(durationHours) * time.Hour),
		MaxDurationMinutes: durationHours * 60,
		CreatedAt:          time.Now(),
		Active:             true,
	}

	query := `
		INSERT INTO impersonation_consents (id, user_id, expires_at, max_duration_minutes, created_at, active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.Exec(query, consent.ID, consent.UserID, consent.ExpiresAt, consent.MaxDurationMinutes, consent.CreatedAt, consent.Active)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Int("duration_hours", durationHours).
			Msg("Failed to enable impersonation consent")
		return nil, err
	}

	logger.ComponentLogger("impersonation").Info().
		Str("consent_id", consent.ID).
		Str("user_id", userID).
		Int("duration_hours", durationHours).
		Time("expires_at", consent.ExpiresAt).
		Msg("Impersonation consent enabled successfully")

	return consent, nil
}

// DisableConsent disables all active consent for a user
func (s *ImpersonationService) DisableConsent(userID string) error {
	query := `UPDATE impersonation_consents SET active = FALSE WHERE user_id = $1 AND active = TRUE`

	result, err := s.db.Exec(query, userID)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to disable impersonation consent")
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	logger.ComponentLogger("impersonation").Info().
		Str("user_id", userID).
		Int64("rows_affected", rowsAffected).
		Msg("Impersonation consent disabled")

	return nil
}

// GetConsentStatus retrieves the current consent status for a user
func (s *ImpersonationService) GetConsentStatus(userID string) (*models.ImpersonationConsent, error) {
	query := `
		SELECT id, user_id, expires_at, max_duration_minutes, created_at, active
		FROM impersonation_consents
		WHERE user_id = $1 AND active = TRUE AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var consent models.ImpersonationConsent
	err := s.db.QueryRow(query, userID).Scan(
		&consent.ID,
		&consent.UserID,
		&consent.ExpiresAt,
		&consent.MaxDurationMinutes,
		&consent.CreatedAt,
		&consent.Active,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No active consent
	}
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get consent status")
		return nil, err
	}

	return &consent, nil
}

// CanBeImpersonated checks if a user can currently be impersonated
func (s *ImpersonationService) CanBeImpersonated(userID string) (bool, error) {
	consent, err := s.GetConsentStatus(userID)
	if err != nil {
		return false, err
	}
	return consent != nil, nil
}

// CanBeImpersonatedBatch checks multiple users at once for impersonation consent
// Returns a map[userID]bool for efficient lookup. Much faster than N individual queries.
func (s *ImpersonationService) CanBeImpersonatedBatch(userIDs []string) (map[string]bool, error) {
	if len(userIDs) == 0 {
		return make(map[string]bool), nil
	}

	result := make(map[string]bool)

	// Initialize all users to false
	for _, userID := range userIDs {
		result[userID] = false
	}

	// Build placeholders for IN query ($1, $2, $3, ...)
	placeholders := make([]string, len(userIDs))
	params := make([]interface{}, len(userIDs))
	for i, userID := range userIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		params[i] = userID
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT user_id
		FROM impersonation_consents
		WHERE user_id IN (%s) AND active = TRUE AND expires_at > NOW()
	`, strings.Join(placeholders, ","))

	rows, err := s.db.Query(query, params...)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Int("user_count", len(userIDs)).
			Msg("Failed to batch check impersonation consent")
		return result, err
	}
	defer func() { _ = rows.Close() }()

	// Mark users with active consent as true
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			logger.ComponentLogger("impersonation").Error().
				Err(err).
				Msg("Failed to scan user ID from batch consent check")
			continue
		}
		result[userID] = true
	}

	logger.ComponentLogger("impersonation").Debug().
		Int("user_count", len(userIDs)).
		Int("consents_found", len(result)).
		Msg("Batch impersonation consent check completed")

	return result, nil
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
			api_endpoint, http_method, request_data, response_status, timestamp,
			impersonator_username, impersonated_username
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
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
		entry.Timestamp,
		entry.ImpersonatorUsername,
		entry.ImpersonatedUsername,
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

// GetUserAuditHistory retrieves audit history for a specific user (being impersonated)
func (s *ImpersonationService) GetUserAuditHistory(userID string, limit, offset int) ([]models.ImpersonationAuditEntry, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM impersonation_audit WHERE impersonated_user_id = $1`
	var total int
	err := s.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get audit history count")
		return nil, 0, err
	}

	// Get entries
	query := `
		SELECT id, session_id, impersonator_user_id, impersonated_user_id, action_type,
			   api_endpoint, http_method, request_data, response_status, timestamp,
			   impersonator_username, impersonated_username
		FROM impersonation_audit
		WHERE impersonated_user_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.Query(query, userID, limit, offset)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("user_id", userID).
			Msg("Failed to get audit history")
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
			&entry.Timestamp,
			&entry.ImpersonatorUsername,
			&entry.ImpersonatedUsername,
		)
		if err != nil {
			logger.ComponentLogger("impersonation").Error().
				Err(err).
				Str("user_id", userID).
				Msg("Failed to scan audit entry")
			continue
		}
		entries = append(entries, entry)
	}

	return entries, total, nil
}

// GetSessionAuditHistory retrieves audit history for a specific impersonation session
func (s *ImpersonationService) GetSessionAuditHistory(sessionID string) ([]models.ImpersonationAuditEntry, error) {
	query := `
		SELECT id, session_id, impersonator_user_id, impersonated_user_id, action_type,
			   api_endpoint, http_method, request_data, response_status, timestamp,
			   impersonator_username, impersonated_username
		FROM impersonation_audit
		WHERE session_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, sessionID)
	if err != nil {
		logger.ComponentLogger("impersonation").Error().
			Err(err).
			Str("session_id", sessionID).
			Msg("Failed to get session audit history")
		return nil, err
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
			&entry.Timestamp,
			&entry.ImpersonatorUsername,
			&entry.ImpersonatedUsername,
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

	return entries, nil
}
