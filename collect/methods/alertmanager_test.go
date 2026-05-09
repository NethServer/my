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

// resolveOrgsRegex matches the bulk SELECT used by resolveOrganizationIDs.
const resolveOrgsRegex = `SELECT system_key, organization_id\s+FROM systems\s+WHERE system_key = ANY\(\$1\) AND deleted_at IS NULL`

// linkFailedUpdateRegex matches the bulk LinkFailed UPDATE-with-RETURNING CTE.
const linkFailedUpdateRegex = `WITH inputs AS.+UPDATE alert_history.+RETURNING m\.idx`

// bulkInsertRegex matches the bulk INSERT INTO alert_history … FROM unnest(…).
const bulkInsertRegex = `INSERT INTO alert_history.+FROM unnest`

func TestReceiveAlertHistory_ResolvedAlert(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}).
			AddRow("SYS-KEY-001", "org-1"))

	// Non-LinkFailed alerts skip the LinkFailed UPDATE entirely.
	mock.ExpectExec(bulkInsertRegex).
		WithArgs(
			sqlmock.AnyArg(), // system_keys
			sqlmock.AnyArg(), // org_ids
			sqlmock.AnyArg(), // alertnames
			sqlmock.AnyArg(), // severity
			sqlmock.AnyArg(), // fingerprint
			sqlmock.AnyArg(), // starts_at
			sqlmock.AnyArg(), // ends_at_text
			sqlmock.AnyArg(), // summary
			sqlmock.AnyArg(), // labels
			sqlmock.AnyArg(), // annotations
			"default",        // receiver (scalar)
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

	// No DB expectations — firing alerts are filtered out before any query.

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

	// No DB expectations — alerts without system_key are dropped pre-DB.

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

func TestReceiveAlertHistory_UnknownSystemKey(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Bulk lookup returns no rows; the alert is dropped silently and no
	// further query runs.
	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}))

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "default",
		"alerts": []map[string]interface{}{
			{
				"status":      "resolved",
				"labels":      map[string]string{"alertname": "DiskFull", "system_key": "SYS-UNKNOWN"},
				"annotations": map[string]string{},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "unknown1",
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

	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}).
			AddRow("SYS-KEY-001", "org-1"))

	mock.ExpectExec(bulkInsertRegex).
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

	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}).
			AddRow("SYS-KEY-001", "org-1"))

	// We rely on the regex match for the INSERT; the ends_at_text array slot
	// is empty for this alert, which the SQL turns into NULL via NULLIF.
	mock.ExpectExec(bulkInsertRegex).
		WillReturnResult(sqlmock.NewResult(1, 1))

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

func TestReceiveAlertHistory_LinkFailedUpdatesExistingStart(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}).
			AddRow("SYS-KEY-001", "org-1"))

	// The bulk LinkFailed UPDATE returns the matched input idx, so the
	// caller knows there's nothing left to insert.
	mock.ExpectQuery(linkFailedUpdateRegex).
		WillReturnRows(sqlmock.NewRows([]string{"idx"}).AddRow(1))

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "builtin-history",
		"alerts": []map[string]interface{}{
			{
				"status": "resolved",
				"labels": map[string]string{
					"alertname":  "LinkFailed",
					"severity":   "critical",
					"system_key": "SYS-KEY-001",
				},
				"annotations": map[string]string{"summary": "No heartbeat received from system"},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "link123",
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

func TestReceiveAlertHistory_LinkFailedInsertsWhenStartNotSeen(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}).
			AddRow("SYS-KEY-001", "org-1"))

	// No idx returned → the LinkFailed alert flows into the bulk insert.
	mock.ExpectQuery(linkFailedUpdateRegex).
		WillReturnRows(sqlmock.NewRows([]string{"idx"}))

	mock.ExpectExec(bulkInsertRegex).
		WillReturnResult(sqlmock.NewResult(1, 1))

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "builtin-history",
		"alerts": []map[string]interface{}{
			{
				"status": "resolved",
				"labels": map[string]string{
					"alertname":  "LinkFailed",
					"severity":   "critical",
					"system_key": "SYS-KEY-001",
				},
				"annotations": map[string]string{"summary": "No heartbeat received from system"},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "link123",
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

func TestReceiveAlertHistory_BulkSplitsLinkFailedAndOthers(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	// Two distinct system_keys, two distinct alertnames in the same payload.
	mock.ExpectQuery(resolveOrgsRegex).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"system_key", "organization_id"}).
			AddRow("SYS-KEY-001", "org-1").
			AddRow("SYS-KEY-002", "org-2"))

	// LinkFailed UPDATE absorbs idx=1 (first LinkFailed alert), no fallback.
	mock.ExpectQuery(linkFailedUpdateRegex).
		WillReturnRows(sqlmock.NewRows([]string{"idx"}).AddRow(1))

	// One INSERT for the non-LinkFailed alert.
	mock.ExpectExec(bulkInsertRegex).
		WillReturnResult(sqlmock.NewResult(1, 1))

	payload := map[string]interface{}{
		"status":   "resolved",
		"receiver": "builtin-history",
		"alerts": []map[string]interface{}{
			{
				"status": "resolved",
				"labels": map[string]string{
					"alertname":  "LinkFailed",
					"severity":   "critical",
					"system_key": "SYS-KEY-001",
				},
				"annotations": map[string]string{"summary": "Link down"},
				"startsAt":    "2026-04-09T10:00:00Z",
				"endsAt":      "2026-04-09T10:30:00Z",
				"fingerprint": "link001",
			},
			{
				"status": "resolved",
				"labels": map[string]string{
					"alertname":  "DiskFull",
					"severity":   "warning",
					"system_key": "SYS-KEY-002",
				},
				"annotations": map[string]string{"summary": "Disk almost full"},
				"startsAt":    "2026-04-09T11:00:00Z",
				"endsAt":      "2026-04-09T11:30:00Z",
				"fingerprint": "disk001",
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
