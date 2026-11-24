/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/local"
)

// The extend and status handlers keep the session in Redis and mint JWTs: an
// in-memory Redis and a test signing key stand in for both. Logto, the user
// resolver and the audit table are replaced through the package seams.
var (
	impersonationRedisOnce sync.Once
	impersonationRedis     *miniredis.Miniredis
)

const (
	testImpersonatorID = "staff-1"
	testTargetLogtoID  = "target-logto"
	testSessionID      = "session-1"
)

type impersonationStubs struct {
	permitted    bool
	permittedErr error
	target       *models.User
	audited      []string
}

func setupImpersonationHandlers(t *testing.T) *impersonationStubs {
	t.Helper()
	gin.SetMode(gin.TestMode)
	impersonationRedisOnce.Do(func() {
		srv, err := miniredis.Run()
		require.NoError(t, err)
		impersonationRedis = srv
		configuration.Config.RedisURL = "redis://" + srv.Addr()
		configuration.Config.RedisDialTimeout = 2 * time.Second
		configuration.Config.RedisReadTimeout = 2 * time.Second
		configuration.Config.RedisWriteTimeout = 2 * time.Second
		configuration.Config.RedisOperationTimeout = 2 * time.Second
		configuration.Config.RedisPoolSize = 4
		require.NoError(t, cache.InitRedis())
	})
	impersonationRedis.FlushAll()
	impersonationRedis.SetError("")
	configuration.Config.JWTSecret = "test-secret-key-for-testing-only"
	configuration.Config.JWTIssuer = "test-issuer"
	configuration.Config.ImpersonationSessionDuration = time.Hour

	targetLogtoID := testTargetLogtoID
	stubs := &impersonationStubs{
		permitted: true,
		target:    &models.User{ID: "target-1", LogtoID: &targetLogtoID, Username: "target", UserRoles: []string{"Admin"}},
	}

	prevPermitted, prevTarget, prevAudit := impersonatorStillPermitted, getUserForImpersonation, logImpersonationAction
	impersonatorStillPermitted = func(string) (bool, error) { return stubs.permitted, stubs.permittedErr }
	getUserForImpersonation = func(string) (*models.User, error) { return stubs.target, nil }
	logImpersonationAction = func(entry *models.ImpersonationAuditEntry) error {
		stubs.audited = append(stubs.audited, entry.ActionType)
		return nil
	}
	t.Cleanup(func() {
		impersonatorStillPermitted, getUserForImpersonation, logImpersonationAction = prevPermitted, prevTarget, prevAudit
	})
	return stubs
}

func impersonator() *models.User {
	logtoID := "staff-logto"
	return &models.User{ID: testImpersonatorID, LogtoID: &logtoID, Username: "staff", UserRoles: []string{"Staff"}}
}

func createTestSession(t *testing.T, duration time.Duration) *cache.ImpersonationSession {
	t.Helper()
	manager := cache.NewImpersonationSessionManager()
	require.NoError(t, manager.CreateSession(testImpersonatorID, testSessionID, testTargetLogtoID, duration))
	session, err := manager.GetActiveSession(testImpersonatorID)
	require.NoError(t, err)
	require.NotNil(t, session)
	return session
}

// runExtend calls the handler as the JWT middleware would for an
// impersonation token of the given session.
func runExtend(sessionID string, presented string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/impersonate/extend", nil)
	c.Request.Header.Set("Authorization", "Bearer "+presented)
	targetLogtoID := testTargetLogtoID
	c.Set("is_impersonated", true)
	c.Set("session_id", sessionID)
	c.Set("impersonated_by", impersonator())
	c.Set("user", &models.User{ID: "target-1", LogtoID: &targetLogtoID})
	ExtendImpersonation(c)
	return w
}

func runStatus() *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/impersonate/status", nil)
	c.Set("user", impersonator())
	GetImpersonationStatus(c)
	return w
}

func responseData(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body struct {
		Data map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Data
}

func TestExtendImpersonation_RestartsTheSession(t *testing.T) {
	stubs := setupImpersonationHandlers(t)
	before := createTestSession(t, 5*time.Minute)

	w := runExtend(testSessionID, "old-token")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := responseData(t, w)
	assert.NotEmpty(t, data["token"])
	assert.Equal(t, float64(3600), data["expires_in"])

	after, err := cache.NewImpersonationSessionManager().GetActiveSession(testImpersonatorID)
	require.NoError(t, err)
	require.NotNil(t, after)
	assert.True(t, after.ExpiresAt.After(before.ExpiresAt.Add(50*time.Minute)), "the session gets a full new duration")
	assert.Equal(t, before.CreatedAt.Unix(), after.CreatedAt.Unix(), "the start of the session is kept")

	revoked, _, err := cache.GetTokenBlacklist().IsTokenBlacklisted("old-token")
	require.NoError(t, err)
	assert.True(t, revoked, "the presented token is revoked")
	assert.Equal(t, []string{"session_extended"}, stubs.audited)
}

func TestExtendImpersonation_Refusals(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		setup     func(s *impersonationStubs)
		want      int
	}{
		{"the impersonator lost impersonate:users", testSessionID, func(s *impersonationStubs) { s.permitted = false }, http.StatusForbidden},
		{"the impersonator was suspended", testSessionID, func(s *impersonationStubs) { s.permittedErr = local.ErrUserInactive }, http.StatusUnauthorized},
		{"the impersonator cannot be resolved", testSessionID, func(s *impersonationStubs) { s.permittedErr = errors.New("db down") }, http.StatusInternalServerError},
		{"the target got the Owner role", testSessionID, func(s *impersonationStubs) { s.target.UserRoles = []string{"Owner"} }, http.StatusForbidden},
		{"another session", "session-2", func(*impersonationStubs) {}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := setupImpersonationHandlers(t)
			before := createTestSession(t, 5*time.Minute)
			tt.setup(stubs)

			w := runExtend(tt.sessionID, "old-token")

			assert.Equal(t, tt.want, w.Code, w.Body.String())
			assert.NotContains(t, w.Body.String(), `"token"`)
			after, err := cache.NewImpersonationSessionManager().GetActiveSession(testImpersonatorID)
			require.NoError(t, err)
			require.NotNil(t, after)
			assert.Equal(t, before.ExpiresAt.Unix(), after.ExpiresAt.Unix(), "the session is not extended")
			revoked, _, _ := cache.GetTokenBlacklist().IsTokenBlacklisted("old-token")
			assert.False(t, revoked, "a refused extension leaves the token alone")
			assert.Empty(t, stubs.audited)
		})
	}
}

func TestExtendImpersonation_FailsClosedWhenRedisFails(t *testing.T) {
	stubs := setupImpersonationHandlers(t)
	createTestSession(t, 5*time.Minute)
	impersonationRedis.SetError("redis unavailable")
	t.Cleanup(func() { impersonationRedis.SetError("") })

	w := runExtend(testSessionID, "old-token")

	assert.Equal(t, http.StatusInternalServerError, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), `"token"`)
	assert.Empty(t, stubs.audited)
}

func TestGetImpersonationStatus_ResumesAPermittedSession(t *testing.T) {
	setupImpersonationHandlers(t)
	createTestSession(t, 30*time.Minute)

	w := runStatus()

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	data := responseData(t, w)
	assert.Equal(t, true, data["is_impersonating"])
	assert.NotEmpty(t, data["token"])
}

func TestGetImpersonationStatus_EndsASessionNoLongerAllowed(t *testing.T) {
	tests := []struct {
		name  string
		setup func(s *impersonationStubs)
	}{
		{"the impersonator lost impersonate:users", func(s *impersonationStubs) { s.permitted = false }},
		{"the target got the Owner role", func(s *impersonationStubs) { s.target.UserRoles = []string{"Owner"} }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stubs := setupImpersonationHandlers(t)
			createTestSession(t, 30*time.Minute)
			tt.setup(stubs)

			w := runStatus()

			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			data := responseData(t, w)
			assert.Equal(t, false, data["is_impersonating"])
			assert.Nil(t, data["token"])
			session, err := cache.NewImpersonationSessionManager().GetActiveSession(testImpersonatorID)
			require.NoError(t, err)
			assert.Nil(t, session, "the session is cleared")
		})
	}
}

func TestGetImpersonationStatus_RefusesASuspendedImpersonator(t *testing.T) {
	stubs := setupImpersonationHandlers(t)
	createTestSession(t, 30*time.Minute)
	stubs.permittedErr = local.ErrUserInactive

	w := runStatus()

	assert.Equal(t, http.StatusUnauthorized, w.Code, w.Body.String())
	assert.NotContains(t, w.Body.String(), `"token"`)
}
