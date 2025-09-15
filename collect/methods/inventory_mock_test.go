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

// MockQueueManager implements queue operations for testing
type MockQueueManager struct {
	ShouldFail  bool
	EnqueueFunc func(ctx context.Context, data *models.InventoryData) error
}

func (m *MockQueueManager) EnqueueInventory(ctx context.Context, data *models.InventoryData) error {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(ctx, data)
	}
	if m.ShouldFail {
		return fmt.Errorf("queue unavailable")
	}
	return nil
}

func CollectInventoryWithMockQueue(c *gin.Context, queueManager interface{}) {
	systemID, exists := c.Get("system_id")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "authentication context error"})
		return
	}

	systemIDStr, ok := systemID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "authentication context error"})
		return
	}

	// Check request size limit
	if c.Request.ContentLength > configuration.Config.APIMaxRequestSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"message": "request too large",
			"data": map[string]interface{}{
				"max_size_bytes": configuration.Config.APIMaxRequestSize,
				"received_bytes": c.Request.ContentLength,
			},
		})
		return
	}

	// Parse request body
	var inventoryRequest models.InventorySubmissionRequest
	if err := c.ShouldBindJSON(&inventoryRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid JSON payload",
			"data": map[string]interface{}{
				"error": err.Error(),
			},
		})
		return
	}

	// Create inventory data
	inventoryData := models.InventoryData{
		SystemID:  systemIDStr,
		Timestamp: time.Now(),
		Data:      inventoryRequest.Data,
	}

	// Validate inventory data
	if err := inventoryData.ValidateInventoryData(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid inventory data",
			"data": map[string]interface{}{
				"validation_error": err.Error(),
			},
		})
		return
	}

	// Quick validation of JSON structure
	var testData interface{}
	if err := json.Unmarshal(inventoryData.Data, &testData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "invalid data structure",
			"data": map[string]interface{}{
				"error": "Data field must contain valid JSON",
			},
		})
		return
	}

	// Use mock queue manager if provided
	if mockQueue, ok := queueManager.(*MockQueueManager); ok {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		if err := mockQueue.EnqueueInventory(ctx, &inventoryData); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": "failed to process inventory",
				"data": map[string]interface{}{
					"error": "Processing queue unavailable",
				},
			})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
			"message": "Inventory received and queued for processing",
			"data": map[string]interface{}{
				"system_id":    systemIDStr,
				"timestamp":    inventoryData.Timestamp,
				"data_size":    len(inventoryData.Data),
				"queue_status": "queued",
			},
		})
		return
	}

	// Default case - would normally use real queue
	c.JSON(http.StatusInternalServerError, gin.H{
		"message": "Queue manager not available",
	})
}

func TestCollectInventoryInvalidDataJSONWithMock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set a reasonable max request size for testing
	configuration.Config.APIMaxRequestSize = 10 * 1024 * 1024

	// Create mock queue manager
	mockQueue := &MockQueueManager{ShouldFail: false}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", func(c *gin.Context) {
		CollectInventoryWithMockQueue(c, mockQueue)
	})

	// Send data field with invalid JSON - this should fail JSON parsing
	invalidJSON := `{"data": invalid json}`

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(invalidJSON)))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(invalidJSON))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	assert.Equal(t, "invalid JSON payload", response["message"])
}

func TestCollectInventoryValidRequestWithMock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set reasonable configuration
	configuration.Config.APIMaxRequestSize = 10 * 1024 * 1024 // 10MB

	// Create mock queue manager that succeeds
	mockQueue := &MockQueueManager{ShouldFail: false}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", func(c *gin.Context) {
		CollectInventoryWithMockQueue(c, mockQueue)
	})

	// Send valid data
	requestData := models.InventorySubmissionRequest{
		Data: json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB", "disk": "1TB SSD"}`),
	}

	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(jsonData))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "Inventory received and queued for processing", response["message"])

	// Check response data
	data := response["data"].(map[string]interface{})
	assert.Equal(t, "test-system-001", data["system_id"])
	assert.Equal(t, "queued", data["queue_status"])
	assert.NotNil(t, data["timestamp"])
	assert.Greater(t, data["data_size"], float64(40)) // Should be around 51-56 chars
}

func TestCollectInventoryQueueFailureWithMock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set reasonable configuration
	configuration.Config.APIMaxRequestSize = 10 * 1024 * 1024 // 10MB

	// Create mock queue manager that fails
	mockQueue := &MockQueueManager{ShouldFail: true}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", func(c *gin.Context) {
		CollectInventoryWithMockQueue(c, mockQueue)
	})

	// Send valid data
	requestData := models.InventorySubmissionRequest{
		Data: json.RawMessage(`{"cpu": "Intel i7", "memory": "16GB"}`),
	}

	jsonData, err := json.Marshal(requestData)
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(jsonData))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "failed to process inventory", response["message"])
}

func TestRequestSizeValidationWithMock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test request size validation logic
	tests := []struct {
		name           string
		maxSize        int64
		contentLength  int64
		expectRejected bool
	}{
		{
			name:           "request within limit",
			maxSize:        1024,
			contentLength:  500,
			expectRejected: false,
		},
		{
			name:           "request at limit",
			maxSize:        1024,
			contentLength:  1024,
			expectRejected: false,
		},
		{
			name:           "request exceeds limit",
			maxSize:        1024,
			contentLength:  1025,
			expectRejected: true,
		},
		{
			name:           "zero content length",
			maxSize:        1024,
			contentLength:  0,
			expectRejected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configuration.Config.APIMaxRequestSize = tt.maxSize

			// Create mock queue manager
			mockQueue := &MockQueueManager{ShouldFail: false}

			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("system_id", "test-system-001")
				c.Next()
			})
			router.POST("/inventory", func(c *gin.Context) {
				CollectInventoryWithMockQueue(c, mockQueue)
			})

			req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer([]byte(`{"data": {"cpu": "Intel i7"}}`)))
			req.Header.Set("Content-Type", "application/json")
			req.ContentLength = tt.contentLength

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectRejected {
				assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "request too large", response["message"])
			} else {
				assert.NotEqual(t, http.StatusRequestEntityTooLarge, w.Code)
			}
		})
	}
}

func TestCollectInventoryDataValidationWithMock(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Set reasonable configuration
	configuration.Config.APIMaxRequestSize = 10 * 1024 * 1024

	// Create mock queue manager
	mockQueue := &MockQueueManager{ShouldFail: false}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("system_id", "test-system-001")
		c.Next()
	})
	router.POST("/inventory", func(c *gin.Context) {
		CollectInventoryWithMockQueue(c, mockQueue)
	})

	tests := []struct {
		name           string
		requestData    interface{}
		expectedStatus int
		expectedMsg    string
	}{
		{
			name:           "valid data",
			requestData:    map[string]interface{}{"data": json.RawMessage(`{"cpu": "Intel i7"}`)},
			expectedStatus: http.StatusAccepted,
			expectedMsg:    "Inventory received and queued for processing",
		},
		{
			name:           "missing data field",
			requestData:    map[string]interface{}{"other": "value"},
			expectedStatus: http.StatusBadRequest,
			expectedMsg:    "invalid inventory data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData, err := json.Marshal(tt.requestData)
			require.NoError(t, err)

			req := httptest.NewRequest("POST", "/inventory", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			req.ContentLength = int64(len(jsonData))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			var response map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedMsg, response["message"])
		})
	}
}
