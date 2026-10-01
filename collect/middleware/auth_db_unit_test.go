/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package middleware

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/helpers"
)

// swapMockDB replaces database.DB with a sqlmock and returns the mock plus a
// restore function.
func swapMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	t.Helper()
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	database.DB = mockDB
	return mock, func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
}

// basicAuthHeader builds the Authorization header for a system_key/secret pair.
func basicAuthHeader(systemKey, systemSecret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(systemKey+":"+systemSecret))
}

// TestBasicAuthMiddleware_DBOutcomes pins the status-code contract of the
// credentials lookup: a missing row is 401, while a database that cannot
// answer (down, or timing out) must be 503 — never 401. During the 2026-09-08
// incident a saturated Postgres made lookups exceed their deadline; mapping
// that to 401 made every appliance silently drop its heartbeat and fired
// ~370 false LinkFailed alerts.
func TestBasicAuthMiddleware_DBOutcomes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	_ = os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	defer func() { _ = os.Unsetenv("DATABASE_URL") }()
	configuration.Init()

	// Well-formed secret: my_<public>.<secret with min length>
	secret := "my_pub." + strings.Repeat("s", configuration.Config.SystemSecretMinLength)

	credsQueryRegex := `SELECT s.id, s.system_secret_public, s.system_secret_sha256, s.registered_at, s.system_key_hash\s+FROM systems s`

	tests := []struct {
		name           string
		systemKey      string // unique per case so the in-process cache never interferes
		dbError        error
		expectedStatus int
	}{
		{
			name:           "unknown system key returns 401",
			systemKey:      "NETH-0000-0000-0000-0000-0000-0000-0000-0001",
			dbError:        sql.ErrNoRows,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "database timeout returns 503, not 401",
			systemKey:      "NETH-0000-0000-0000-0000-0000-0000-0000-0002",
			dbError:        context.DeadlineExceeded,
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "database connection failure returns 503, not 401",
			systemKey:      "NETH-0000-0000-0000-0000-0000-0000-0000-0003",
			dbError:        sql.ErrConnDone,
			expectedStatus: http.StatusServiceUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, restore := swapMockDB(t)
			defer restore()

			mock.ExpectQuery(credsQueryRegex).
				WithArgs(tt.systemKey).
				WillReturnError(tt.dbError)

			router := gin.New()
			router.Use(BasicAuthMiddleware())
			router.POST("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodPost, "/probe", nil)
			req.Header.Set("Authorization", basicAuthHeader(tt.systemKey, secret))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}

	// A DB failure must not poison the negative cache: once the database is
	// healthy again the same system authenticates without waiting out a
	// cached-invalid TTL. The second request here would be answered by the
	// in-process cache (bypassing sqlmock) if the failure had been cached.
	t.Run("database failure is not cached as invalid", func(t *testing.T) {
		mock, restore := swapMockDB(t)
		defer restore()

		const systemKey = "NETH-0000-0000-0000-0000-0000-0000-0000-0004"

		mock.ExpectQuery(credsQueryRegex).
			WithArgs(systemKey).
			WillReturnError(context.DeadlineExceeded)
		// The recovered DB still refuses the pair (no row): the point is that
		// the second request reaches the database at all.
		mock.ExpectQuery(credsQueryRegex).
			WithArgs(systemKey).
			WillReturnError(sql.ErrNoRows)

		router := gin.New()
		router.Use(BasicAuthMiddleware())
		router.POST("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })

		for _, expected := range []int{http.StatusServiceUnavailable, http.StatusUnauthorized} {
			req := httptest.NewRequest(http.MethodPost, "/probe", nil)
			req.Header.Set("Authorization", basicAuthHeader(systemKey, secret))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			assert.Equal(t, expected, w.Code)
		}
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

var credsColumns = []string{"id", "system_secret_public", "system_secret_sha256", "registered_at", "system_key_hash"}

// Bearer lane: known hash 200, unknown 401 (nginx falls back to legacy), DB failure 503.
func TestSystemKeyHashAuth_DBOutcomes(t *testing.T) {
	hash := strings.Repeat("ab", 32)
	for expected, result := range map[int]func(*sqlmock.ExpectedQuery){
		http.StatusOK: func(q *sqlmock.ExpectedQuery) {
			q.WillReturnRows(sqlmock.NewRows(credsColumns).AddRow("sys-1", nil, nil, time.Now(), hash))
		},
		http.StatusUnauthorized:       func(q *sqlmock.ExpectedQuery) { q.WillReturnError(sql.ErrNoRows) },
		http.StatusServiceUnavailable: func(q *sqlmock.ExpectedQuery) { q.WillReturnError(context.DeadlineExceeded) },
	} {
		mock, restore := swapMockDB(t)
		result(mock.ExpectQuery(`s.system_key_hash = \$1`).WithArgs(storedKeyHash(hash)))

		router := gin.New()
		router.Use(SystemKeyHashAuthMiddleware())
		router.GET("/auth-hash", func(c *gin.Context) { c.Status(http.StatusOK) })
		req := httptest.NewRequest(http.MethodGet, "/auth-hash", nil)
		req.Header.Set("Authorization", "Bearer "+hash)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, expected, w.Code)
		assert.NoError(t, mock.ExpectationsWereMet())
		restore()
	}
}

// The first uncached Basic request stores sha256 of the hex sha256("<key>:<token>").
func TestBasicAuth_LearnsSystemKeyHash(t *testing.T) {
	_ = os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	defer func() { _ = os.Unsetenv("DATABASE_URL") }()
	configuration.Init()

	const key = "NETH-0000-0000-0000-0000-0000-0000-0000-00B1"
	secret := strings.Repeat("s", configuration.Config.SystemSecretMinLength)
	token := "my_pub." + secret
	digest, _ := helpers.HashSystemSecretSHA256(secret)
	sum := sha256.Sum256([]byte(key + ":" + token))
	stored := sha256.Sum256([]byte(hex.EncodeToString(sum[:])))

	mock, restore := swapMockDB(t)
	defer restore()
	mock.ExpectQuery(`s.system_key = \$1`).WithArgs(key).
		WillReturnRows(sqlmock.NewRows(credsColumns).AddRow("sys-1", "pub", digest, time.Now(), nil))
	mock.ExpectExec(`UPDATE systems SET system_key_hash`).WithArgs("sys-1", hex.EncodeToString(stored[:])).
		WillReturnResult(sqlmock.NewResult(0, 1))

	router := gin.New()
	router.Use(BasicAuthMiddleware())
	router.POST("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/probe", nil)
	req.Header.Set("Authorization", basicAuthHeader(key, token))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
