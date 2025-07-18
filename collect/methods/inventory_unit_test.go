/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/models"
)

func TestCollectInventoryNoSystemID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/inventory", CollectInventory)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(`{"data": {"cpu": "Intel i7"}}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Authentication context error", response["message"])
}

func TestCollectInventoryInvalidSystemIDType(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", 123) // Set as int instead of string
		c.Next()
	})
	router.POST("/inventory", CollectInventory)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(`{"data": {"cpu": "Intel i7"}}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Authentication context error", response["message"])
}

func TestCollectInventoryRequestTooLarge(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set up configuration with small max request size
	configuration.Config.APIMaxRequestSize = 100

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", CollectInventory)

	// Create a large request that exceeds the limit
	largeData := make(map[string]string)
	for i := 0; i < 100; i++ {
		largeData[fmt.Sprintf("key%d", i)] = "very long value that makes the request exceed the size limit"
	}

	requestData := map[string]interface{}{
		"data": largeData,
	}

	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(jsonData))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Request too large", response["message"])
	assert.Contains(t, response["data"], "max_size_bytes")
	assert.Contains(t, response["data"], "received_bytes")
}

func TestCollectInventoryInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", CollectInventory)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(`invalid json`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid JSON payload", response["message"])
}

func TestCollectInventoryMissingDataField(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", CollectInventory)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Invalid inventory data", response["message"])
}

func TestCollectInventoryRequestValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		systemID       interface{}
		requestBody    string
		contentLength  int64
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "missing system_id in context",
			systemID:       nil,
			requestBody:    `{"data": {"cpu": "Intel i7"}}`,
			contentLength:  0,
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Authentication context error",
		},
		{
			name:           "invalid system_id type",
			systemID:       12345,
			requestBody:    `{"data": {"cpu": "Intel i7"}}`,
			contentLength:  0,
			expectedStatus: http.StatusInternalServerError,
			expectedMsg:    "Authentication context error",
		},
		{
			name:           "invalid json",
			systemID:       "test-system-001",
			requestBody:    `{"data": {"cpu": "Intel i7"`,
			contentLength:  0,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid JSON payload",
		},
		{
			name:           "missing data field",
			systemID:       "test-system-001",
			requestBody:    `{"other": "value"}`,
			contentLength:  0,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid inventory data",
		},
		{
			name:           "invalid data json",
			systemID:       "test-system-001",
			requestBody:    `{"data": {"cpu": "Intel i7", "memory":}}`,
			contentLength:  0,
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "Invalid JSON payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			if tt.systemID != nil {
				router.Use(func(c *gin.Context) {
					c.Set("system_id", tt.systemID)
					c.Next()
				})
			}
			router.POST("/inventory", CollectInventory)

			req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(tt.requestBody)))
			req.Header.Set("Content-Type", "application/json")
			if tt.contentLength > 0 {
				req.ContentLength = tt.contentLength
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedMsg != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, response["message"])
			}
		})
	}
}

func TestInventoryDataCreation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test that InventoryData is created correctly from request
	systemID := "test-system-001"
	requestData := models.InventorySubmissionRequest{
		Data: json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB"}`),
	}

	// Simulate the data creation logic from the handler
	now := time.Now()
	inventoryData := models.InventoryData{
		SystemID:  systemID,
		Timestamp: now,
		Data:      requestData.Data,
	}

	assert.Equal(t, systemID, inventoryData.SystemID)
	assert.Equal(t, requestData.Data, inventoryData.Data)
	assert.False(t, inventoryData.Timestamp.IsZero())

	// Test validation
	err := inventoryData.ValidateInventoryData()
	assert.NoError(t, err)
}

func TestInventoryDataValidationInHandler(t *testing.T) {
	// Test various validation scenarios that would be caught in the handler
	tests := []struct {
		name        string
		systemID    string
		data        json.RawMessage
		expectError bool
	}{
		{
			name:        "valid data",
			systemID:    "test-system-001",
			data:        json.RawMessage(`{"cpu": "Intel i7"}`),
			expectError: false,
		},
		{
			name:        "empty system id",
			systemID:    "",
			data:        json.RawMessage(`{"cpu": "Intel i7"}`),
			expectError: true,
		},
		{
			name:        "empty data",
			systemID:    "test-system-001",
			data:        json.RawMessage(``),
			expectError: true,
		},
		{
			name:        "invalid json data",
			systemID:    "test-system-001",
			data:        json.RawMessage(`{"cpu": "Intel i7", "memory":}`),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inventoryData := models.InventoryData{
				SystemID:  tt.systemID,
				Timestamp: time.Now(),
				Data:      tt.data,
			}

			err := inventoryData.ValidateInventoryData()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestContextTimeout(t *testing.T) {
	// Test that the context timeout is applied correctly
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify timeout is set correctly
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, time.Until(deadline) <= 5*time.Second)
	assert.True(t, time.Until(deadline) > 4*time.Second)
}
