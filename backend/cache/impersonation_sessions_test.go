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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/testutils"
)

func TestNewImpersonationSessionManager(t *testing.T) {
	manager := NewImpersonationSessionManager()
	assert.NotNil(t, manager)
	// Redis client can be nil if Redis is not available - this is expected behavior
}

func TestImpersonationSessionManager_getSessionKey(t *testing.T) {
	manager := &ImpersonationSessionManager{}

	tests := []struct {
		name               string
		impersonatorUserID string
		expected           string
	}{
		{
			name:               "standard user ID",
			impersonatorUserID: "user123",
			expected:           "impersonation_session:user123",
		},
		{
			name:               "empty user ID",
			impersonatorUserID: "",
			expected:           "impersonation_session:",
		},
		{
			name:               "special characters in user ID",
			impersonatorUserID: "user@123.com",
			expected:           "impersonation_session:user@123.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.getSessionKey(tt.impersonatorUserID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImpersonationSessionManager_Methods_Structure(t *testing.T) {
	testutils.SetupLogger()

	manager := NewImpersonationSessionManager()

	// Test that methods exist and have correct signatures (compilation test)
	assert.NotNil(t, manager.CreateSession)
	assert.NotNil(t, manager.GetActiveSession)
	assert.NotNil(t, manager.HasActiveSession)
	assert.NotNil(t, manager.ClearSession)
	assert.NotNil(t, manager.ClearSessionByID)
	assert.NotNil(t, manager.GetStats)

	// Test basic method calls (will fail without Redis but tests structure)
	err := manager.CreateSession("admin123", "session456", "user789", time.Hour)
	// Should succeed with graceful fallback when Redis is unavailable
	assert.NoError(t, err) // This should fail-open

	session, err := manager.GetActiveSession("admin123")
	// Should fail-open and return nil session
	assert.NoError(t, err)
	assert.Nil(t, session)

	hasSession, err := manager.HasActiveSession("admin123")
	assert.NoError(t, err)
	assert.False(t, hasSession)

	err = manager.ClearSession("admin123")
	assert.NoError(t, err) // Should fail-open

	err = manager.ClearSessionByID("admin123", "session456")
	assert.NoError(t, err) // Should fail-open through GetActiveSession

	stats, err := manager.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, false, stats["redis_available"])
}

func TestImpersonationSessionManager_EdgeCases(t *testing.T) {
	testutils.SetupLogger()

	// Test with nil Redis client
	manager := &ImpersonationSessionManager{redis: nil}

	// All operations should fail gracefully
	err := manager.CreateSession("user1", "session1", "user2", time.Hour)
	assert.NoError(t, err) // Should fail-open

	session, err := manager.GetActiveSession("user1")
	assert.NoError(t, err) // Should fail-open
	assert.Nil(t, session)

	hasSession, err := manager.HasActiveSession("user1")
	assert.NoError(t, err) // Should fail-open
	assert.False(t, hasSession)

	err = manager.ClearSession("user1")
	assert.NoError(t, err) // Should fail-open

	err = manager.ClearSessionByID("user1", "session1")
	assert.NoError(t, err) // Should fail-open

	stats, err := manager.GetStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, false, stats["redis_available"])
}

func TestImpersonationSession_Struct(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)

	session := ImpersonationSession{
		SessionID:          "test_session",
		ImpersonatedUserID: "user123",
		CreatedAt:          now,
		ExpiresAt:          expiresAt,
	}

	assert.Equal(t, "test_session", session.SessionID)
	assert.Equal(t, "user123", session.ImpersonatedUserID)
	assert.Equal(t, now, session.CreatedAt)
	assert.Equal(t, expiresAt, session.ExpiresAt)

	// Test JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(session)
	assert.NoError(t, err)

	var unmarshaled ImpersonationSession
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, session.SessionID, unmarshaled.SessionID)
	assert.Equal(t, session.ImpersonatedUserID, unmarshaled.ImpersonatedUserID)
}
