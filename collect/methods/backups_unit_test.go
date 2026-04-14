/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/collect/configuration"
)

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		fallback string
		want     string
	}{
		{"empty falls back to backup-<id>", "", "abc-123", "backup-abc-123"},
		{"whitespace falls back", "   ", "abc-123", "backup-abc-123"},
		{"keeps plain filename", "daily.tar.gz", "x", "daily.tar.gz"},
		{"strips directory prefix", "../../etc/passwd", "x", "passwd"},
		{"strips windows-style prefix", `C:\evil\dump.gpg`, "x", "dump.gpg"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, extractFilename(tc.header, tc.fallback))
		})
	}
}

func TestIsValidBackupID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		ok   bool
	}{
		// UUIDv7 canonical form — version nibble 7, variant nibble 8/9/a/b.
		{"uuidv7 bare", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8", true},
		{"uuidv7 with tar.gz", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.tar.gz", true},
		{"uuidv7 with gpg", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.gpg", true},
		{"uuidv7 with bin", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.bin", true},
		{"uuidv7 uppercase normalised", "01934FAB-BC33-7890-A1B2-C3D4E5F6A7B8.TAR.GZ", true},
		// Rejected shapes.
		{"uuidv4 (wrong version nibble)", "01934fab-bc33-4890-a1b2-c3d4e5f6a7b8", false},
		{"path traversal", "../../etc/passwd", false},
		{"encoded slash traversal", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8..%2Fetc%2Fpasswd", false},
		{"slash injection", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8/extra", false},
		{"unknown extension", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8.sh", false},
		{"trailing whitespace", "01934fab-bc33-7890-a1b2-c3d4e5f6a7b8 ", false},
		{"empty", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.ok, isValidBackupID(tc.in))
		})
	}
}

func TestExtractExtension(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"tar.gz compound preserved", "dump.tar.gz", ".tar.gz"},
		{"tar.xz compound preserved", "dump.tar.xz", ".tar.xz"},
		{"tar.bz2 compound preserved", "dump.tar.bz2", ".tar.bz2"},
		{"tar.zst compound preserved", "dump.tar.zst", ".tar.zst"},
		{"simple gpg extension", "backup.gpg", ".gpg"},
		{"uppercase normalized", "backup.GPG", ".gpg"},
		{"no extension defaults to .bin", "backup", ".bin"},
		{"trailing dot defaults to .bin", "backup.", ".bin"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, extractExtension(tc.in))
		})
	}
}

func TestUploadBackupNoSystemID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/backups", UploadBackup)

	req := httptest.NewRequest("POST", "/backups", bytes.NewBufferString("payload"))
	req.Header.Set("Content-Type", "application/octet-stream")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "authentication required", body["message"])
}

func TestUploadBackupExceedsMaxSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	configuration.Config.BackupMaxUploadSize = 16

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "sys-123")
		c.Set("system_key", "my_sys_abc")
		c.Next()
	})
	router.POST("/backups", UploadBackup)

	payload := strings.Repeat("A", 128)
	req := httptest.NewRequest("POST", "/backups", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(payload))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "backup exceeds max upload size", body["message"])
}

func TestListBackupsNoSystemID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/backups", ListBackups)

	req := httptest.NewRequest("GET", "/backups", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "authentication required", body["message"])
}

func TestDownloadBackupNoSystemID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/backups/:id", DownloadBackup)

	req := httptest.NewRequest("GET", "/backups/abc.tar.gz", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
