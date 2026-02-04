/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package helpers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHandleAccessError_NilError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handled := HandleAccessError(c, nil, "system", "sys-1")

	assert.False(t, handled, "nil error should not be handled")
	assert.Equal(t, http.StatusOK, w.Code, "Response code should remain default")
}

func TestHandleAccessError_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handled := HandleAccessError(c, errors.New("system not found"), "system", "sys-1")

	assert.True(t, handled)
	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "system not found", resp["message"])
}

func TestHandleAccessError_AccessDenied(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handled := HandleAccessError(c, errors.New("access denied for this resource"), "system", "sys-123")

	assert.True(t, handled)
	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "access denied to system", resp["message"])

	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "sys-123", data["system_id"])
}

func TestHandleAccessError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	handled := HandleAccessError(c, errors.New("database connection failed"), "application", "app-1")

	assert.True(t, handled)
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "failed to process application request", resp["message"])

	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "database connection failed", data["error"])
}

func TestHandleAccessError_DifferentEntityTypes(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		entityID   string
		err        error
		statusCode int
	}{
		{
			name:       "application not found",
			entityType: "application",
			entityID:   "app-1",
			err:        errors.New("application not found"),
			statusCode: http.StatusNotFound,
		},
		{
			name:       "system access denied",
			entityType: "system",
			entityID:   "sys-1",
			err:        errors.New("access denied"),
			statusCode: http.StatusForbidden,
		},
		{
			name:       "user not found",
			entityType: "user",
			entityID:   "usr-1",
			err:        errors.New("user not found"),
			statusCode: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			handled := HandleAccessError(c, tt.err, tt.entityType, tt.entityID)

			assert.True(t, handled)
			assert.Equal(t, tt.statusCode, w.Code)
		})
	}
}
