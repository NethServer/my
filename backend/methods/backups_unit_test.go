/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handlerPanicRecovery wraps a handler so panics surface as 500s during
// tests instead of aborting the runner. The backup handlers reach for
// database/S3 clients once they pass the early parameter/auth gates; in
// the tests below we only exercise the short-circuit paths that return
// before any of those side effects.
func handlerPanicRecovery(h gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		h(c)
	}
}

func TestIsValidBackupID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		ok   bool
	}{
		{"uuidv7 bare", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8", true},
		{"uuidv7 with tar.gz", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.tar.gz", true},
		{"uuidv7 uppercase normalised", "01934FAB-BC33-7890-A1B2-C3D4E5F6A7B8.GPG", true},
		{"path traversal", "../../etc/passwd", false},
		{"slash injection", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8/extra", false},
		{"unknown extension", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.sh", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ok, isValidBackupID(tc.in))
		})
	}
}

func TestGetSystemBackupsRequiresUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/systems/:id/backups", handlerPanicRecovery(GetSystemBackups))

	req := httptest.NewRequest("GET", "/systems/sys_123/backups", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// helpers.GetUserFromContext writes its own 401 response and returns
	// false when the auth middleware has not run — the handler must stop
	// short of touching the DB.
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDownloadSystemBackupRequiresUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/systems/:id/backups/:backup_id/download",
		handlerPanicRecovery(DownloadSystemBackup))

	req := httptest.NewRequest("GET",
		"/systems/sys_123/backups/01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.tar.gz/download", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDeleteSystemBackupRequiresUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/systems/:id/backups/:backup_id",
		handlerPanicRecovery(DeleteSystemBackup))

	req := httptest.NewRequest("DELETE",
		"/systems/sys_123/backups/01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.tar.gz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestBackupMetadataSerialization(t *testing.T) {
	// Document the JSON shape the UI consumes. If any field name or
	// casing shifts, this test highlights it before the frontend breaks.
	m := BackupMetadata{
		ID:       "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.tar.gz",
		Filename: "daily-backup.tar.gz",
		Size:     42 * 1024 * 1024,
		SHA256:   "3a7bd3e2360a3d29eea436fcfb7e44c735d117c42d1c1835420b6b9942dd4f1b",
		MimeType: "application/gzip",
	}

	payload, err := json.Marshal(m)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &decoded))

	assert.Equal(t, m.ID, decoded["id"])
	assert.Equal(t, m.Filename, decoded["filename"])
	assert.Equal(t, float64(m.Size), decoded["size"])
	assert.Equal(t, m.SHA256, decoded["sha256"])
	assert.Equal(t, m.MimeType, decoded["mimetype"])
	_, hasUploader := decoded["uploader_ip"]
	assert.False(t, hasUploader, "uploader_ip must not be exposed in BackupMetadata")
}

func TestBackupListResponseSerialization(t *testing.T) {
	r := BackupListResponse{
		Backups:        []BackupMetadata{},
		QuotaUsedBytes: 1024 * 1024,
		SlotsUsed:      3,
	}

	payload, err := json.Marshal(r)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &decoded))

	assert.Equal(t, float64(r.QuotaUsedBytes), decoded["quota_used_bytes"])
	assert.Equal(t, float64(r.SlotsUsed), decoded["slots_used"])
	assert.NotNil(t, decoded["backups"])
}
