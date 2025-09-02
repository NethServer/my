/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// ImpersonationSession represents an active impersonation session
type ImpersonationSession struct {
	SessionID          string    `json:"session_id"`
	ImpersonatedUserID string    `json:"impersonated_user_id"`
	CreatedAt          time.Time `json:"created_at"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// ImpersonationSessionManager manages active impersonation sessions in Redis
type ImpersonationSessionManager struct {
	redis *RedisClient
}

// NewImpersonationSessionManager creates a new session manager
func NewImpersonationSessionManager() *ImpersonationSessionManager {
	return &ImpersonationSessionManager{
		redis: GetRedisClient(),
	}
}

// getSessionKey generates Redis key for impersonator's active session
func (m *ImpersonationSessionManager) getSessionKey(impersonatorUserID string) string {
	return fmt.Sprintf("impersonation_session:%s", impersonatorUserID)
}

// CreateSession creates a new active impersonation session
func (m *ImpersonationSessionManager) CreateSession(impersonatorUserID, sessionID, impersonatedUserID string, duration time.Duration) error {
	if m.redis == nil {
		logger.ComponentLogger("impersonation_sessions").Warn().
			Msg("Redis client not available - skipping session creation")
		return nil // Fail-open
	}

	session := ImpersonationSession{
		SessionID:          sessionID,
		ImpersonatedUserID: impersonatedUserID,
		CreatedAt:          time.Now(),
		ExpiresAt:          time.Now().Add(duration),
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		logger.ComponentLogger("impersonation_sessions").Error().
			Err(err).
			Str("impersonator_user_id", impersonatorUserID).
			Str("session_id", sessionID).
			Msg("Failed to marshal session data")
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	key := m.getSessionKey(impersonatorUserID)
	err = m.redis.Set(key, string(sessionData), duration)
	if err != nil {
		logger.ComponentLogger("impersonation_sessions").Error().
			Err(err).
			Str("impersonator_user_id", impersonatorUserID).
			Str("session_id", sessionID).
			Str("key", key).
			Msg("Failed to store active impersonation session")
		return fmt.Errorf("failed to store session: %w", err)
	}

	logger.ComponentLogger("impersonation_sessions").Info().
		Str("impersonator_user_id", impersonatorUserID).
		Str("session_id", sessionID).
		Str("impersonated_user_id", impersonatedUserID).
		Time("expires_at", session.ExpiresAt).
		Msg("Active impersonation session created")

	return nil
}

// GetActiveSession retrieves the active impersonation session for an impersonator
func (m *ImpersonationSessionManager) GetActiveSession(impersonatorUserID string) (*ImpersonationSession, error) {
	if m.redis == nil {
		logger.ComponentLogger("impersonation_sessions").Warn().
			Msg("Redis client not available - assuming no active session")
		return nil, nil // Fail-open
	}

	key := m.getSessionKey(impersonatorUserID)
	var sessionData string
	err := m.redis.Get(key, &sessionData)
	if err != nil {
		if err == ErrCacheMiss {
			// No active session - this is normal
			return nil, nil
		}
		logger.ComponentLogger("impersonation_sessions").Error().
			Err(err).
			Str("impersonator_user_id", impersonatorUserID).
			Str("key", key).
			Msg("Failed to retrieve active impersonation session")
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session ImpersonationSession
	err = json.Unmarshal([]byte(sessionData), &session)
	if err != nil {
		logger.ComponentLogger("impersonation_sessions").Error().
			Err(err).
			Str("impersonator_user_id", impersonatorUserID).
			Str("session_data", sessionData).
			Msg("Failed to unmarshal session data")
		return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		logger.ComponentLogger("impersonation_sessions").Info().
			Str("impersonator_user_id", impersonatorUserID).
			Str("session_id", session.SessionID).
			Time("expired_at", session.ExpiresAt).
			Msg("Active impersonation session has expired - removing")

		// Remove expired session
		_ = m.ClearSession(impersonatorUserID)
		return nil, nil
	}

	return &session, nil
}

// HasActiveSession checks if an impersonator has an active session
func (m *ImpersonationSessionManager) HasActiveSession(impersonatorUserID string) (bool, error) {
	session, err := m.GetActiveSession(impersonatorUserID)
	if err != nil {
		return false, err
	}
	return session != nil, nil
}

// ClearSession removes the active impersonation session for an impersonator
func (m *ImpersonationSessionManager) ClearSession(impersonatorUserID string) error {
	if m.redis == nil {
		logger.ComponentLogger("impersonation_sessions").Warn().
			Msg("Redis client not available - skipping session removal")
		return nil // Fail-open
	}

	key := m.getSessionKey(impersonatorUserID)
	err := m.redis.Delete(key)
	if err != nil {
		logger.ComponentLogger("impersonation_sessions").Error().
			Err(err).
			Str("impersonator_user_id", impersonatorUserID).
			Str("key", key).
			Msg("Failed to clear active impersonation session")
		return fmt.Errorf("failed to clear session: %w", err)
	}

	logger.ComponentLogger("impersonation_sessions").Info().
		Str("impersonator_user_id", impersonatorUserID).
		Msg("Active impersonation session cleared")

	return nil
}

// ClearSessionByID removes an active impersonation session by session ID
// This is used when we need to clean up by session ID instead of impersonator ID
func (m *ImpersonationSessionManager) ClearSessionByID(impersonatorUserID, sessionID string) error {
	// First check if the session matches
	activeSession, err := m.GetActiveSession(impersonatorUserID)
	if err != nil {
		return err
	}

	if activeSession != nil && activeSession.SessionID == sessionID {
		return m.ClearSession(impersonatorUserID)
	}

	// Session doesn't match or doesn't exist - this is fine
	return nil
}

// GetStats returns statistics about active impersonation sessions
func (m *ImpersonationSessionManager) GetStats() (map[string]interface{}, error) {
	if m.redis == nil {
		return map[string]interface{}{
			"redis_available": false,
		}, nil
	}

	// Get basic Redis stats
	redisStats, err := m.redis.GetStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis stats: %w", err)
	}

	stats := map[string]interface{}{
		"redis_available": true,
		"redis_stats":     redisStats,
	}

	return stats, nil
}
