/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/collect/database"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupMockDB(t *testing.T) (sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	database.DB = mockDB
	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return mock, cleanup
}

func TestReceiveAlertHistory_ResolvedAlert(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO alert_history`).
		WithArgs(
			"SYS-KEY-001",
			"DiskFull",
			sqlmock.AnyArg(), // severity
			"resolved",
			"abc123",
			sqlmock.AnyArg(), // startsAt
			sqlmock.AnyArg(), // endsAt
			sqlmock.AnyArg(), // summary
			sqlmock.AnyArg(), // labels JSON
			sqlmock.AnyArg(), // annotations JSON
			sqlmock.AnyArg(), // receiver
		).WillReturnResult(sqlmock.NewResult(1, 1))

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "default",
		"alerts": []map[string]interface{}{
			{
				"status":      "resolved",
				"labels":      map[string]string{"alertname": "DiskFull", "severity": "critical", "system_key": "SYS-KEY-001"},
				"annotations": map[string]string{"summary": "Disk full"},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "abc123",
			},
		},
	}
	body, _ := json.Marshal(payload)

	router := gin.New()
	router.POST("/alert_history", ReceiveAlertHistory)

	req := httptest.NewRequest(http.MethodPost, "/alert_history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestReceiveAlertHistory_FiringAlertSkipped(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	// No DB expectations — firing alerts are skipped

	payload := map[string]interface{}{
		"status":   "firing",
		"receiver": "default",
		"alerts": []map[string]interface{}{
			{
				"status":      "firing",
				"labels":      map[string]string{"alertname": "HighCPU", "severity": "warning", "system_key": "SYS-KEY-001"},
				"annotations": map[string]string{"summary": "CPU high"},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "0001-01-01T00:00:00Z",
				"fingerprint": "def456",
			},
		},
	}
	body, _ := json.Marshal(payload)

	router := gin.New()
	router.POST("/alert_history", ReceiveAlertHistory)

	req := httptest.NewRequest(http.MethodPost, "/alert_history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestReceiveAlertHistory_MissingSystemKey(t *testing.T) {
	_, cleanup := setupMockDB(t)
	defer cleanup()

	// No DB expectations — alerts without system_key are skipped

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "default",
		"alerts": []map[string]interface{}{
			{
				"status":      "resolved",
				"labels":      map[string]string{"alertname": "TestAlert"},
				"annotations": map[string]string{},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "ghi789",
			},
		},
	}
	body, _ := json.Marshal(payload)

	router := gin.New()
	router.POST("/alert_history", ReceiveAlertHistory)

	req := httptest.NewRequest(http.MethodPost, "/alert_history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestReceiveAlertHistory_InvalidBody(t *testing.T) {
	router := gin.New()
	router.POST("/alert_history", ReceiveAlertHistory)

	req := httptest.NewRequest(http.MethodPost, "/alert_history", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceiveAlertHistory_DBError(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectExec(`INSERT INTO alert_history`).
		WillReturnError(sql.ErrConnDone)

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "default",
		"alerts": []map[string]interface{}{
			{
				"status":      "resolved",
				"labels":      map[string]string{"alertname": "DiskFull", "severity": "critical", "system_key": "SYS-KEY-001"},
				"annotations": map[string]string{"summary": "Disk full"},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "err123",
			},
		},
	}
	body, _ := json.Marshal(payload)

	router := gin.New()
	router.POST("/alert_history", ReceiveAlertHistory)

	req := httptest.NewRequest(http.MethodPost, "/alert_history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestReceiveAlertHistory_ZeroTimeEndsAt(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// The resolved alert with zero-time endsAt should store NULL
	mock.ExpectExec(`INSERT INTO alert_history`).
		WithArgs(
			"SYS-KEY-001",
			"TestAlert",
			sqlmock.AnyArg(),
			"resolved",
			"zero123",
			sqlmock.AnyArg(),
			nil, // endsAt should be nil for zero time
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
			sqlmock.AnyArg(),
		).WillReturnResult(sqlmock.NewResult(1, 1))

	zeroTime := time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "default",
		"alerts": []map[string]interface{}{
			{
				"status":      "resolved",
				"labels":      map[string]string{"alertname": "TestAlert", "severity": "info", "system_key": "SYS-KEY-001"},
				"annotations": map[string]string{},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      zeroTime.Format(time.RFC3339),
				"fingerprint": "zero123",
			},
		},
	}
	body, _ := json.Marshal(payload)

	router := gin.New()
	router.POST("/alert_history", ReceiveAlertHistory)

	req := httptest.NewRequest(http.MethodPost, "/alert_history", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNullableString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantVal string
	}{
		{name: "empty string returns nil", input: "", wantNil: true},
		{name: "non-empty string returns pointer", input: "hello", wantNil: false, wantVal: "hello"},
		{name: "whitespace is not nil", input: " ", wantNil: false, wantVal: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nullableString(tt.input)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantVal, *result)
			}
		})
	}
}
