/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package validators

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/stretchr/testify/assert"
)

func setupGinTest() (*gin.Engine, sqlmock.Sqlmock, *sql.DB, func()) {
	gin.SetMode(gin.TestMode)

	// Store original DB
	originalDB := database.DB

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}

	// Set mock DB
	database.DB = mockDB

	router := gin.New()

	// Cleanup function
	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}

	return router, mock, mockDB, cleanup
}

func TestValidateVAT_Success(t *testing.T) {
	_, mock, _, cleanup := setupGinTest()
	defer cleanup()

	tests := []struct {
		name           string
		entityType     string
		vatParam       string
		excludeIDParam string
		mockSetup      func()
		expectedExists bool
	}{
		{
			name:           "VAT exists in customers",
			entityType:     "customers",
			vatParam:       "12345678901",
			excludeIDParam: "",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
					WithArgs("12345678901", "").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedExists: true,
		},
		{
			name:           "VAT does not exist in distributors",
			entityType:     "distributors",
			vatParam:       "98765432109",
			excludeIDParam: "",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM distributors`).
					WithArgs("98765432109", "").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			expectedExists: false,
		},
		{
			name:           "VAT with exclude ID",
			entityType:     "resellers",
			vatParam:       "11111111111",
			excludeIDParam: "reseller-123",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM resellers`).
					WithArgs("11111111111", "reseller-123").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh router for each test
			testRouter := gin.New()

			// Setup mock expectations
			tt.mockSetup()

			// Setup route
			testRouter.GET("/validators/vat/:entity_type", ValidateVAT)

			// Create request
			url := "/validators/vat/" + tt.entityType + "?vat=" + tt.vatParam
			if tt.excludeIDParam != "" {
				url += "&exclude_id=" + tt.excludeIDParam
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()

			// Execute request
			testRouter.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "VAT validation completed", response.Message)

			// Verify data
			dataBytes, err := json.Marshal(response.Data)
			assert.NoError(t, err)

			var result models.VATValidationResponse
			err = json.Unmarshal(dataBytes, &result)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedExists, result.Exists)

			// Verify all mock expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestValidateVAT_ValidationErrors(t *testing.T) {
	_, _, _, cleanup := setupGinTest()
	defer cleanup()

	tests := []struct {
		name           string
		url            string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Missing entity_type parameter",
			url:            "/validators/vat/?vat=12345678901",
			expectedStatus: http.StatusNotFound, // Gin returns 404 for missing route params
			expectedError:  "page not found",    // Gin's default 404 message
		},
		{
			name:           "Invalid entity_type",
			url:            "/validators/vat/invalid?vat=12345678901",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "entity_type must be one of: distributors, resellers, customers",
		},
		{
			name:           "Missing vat parameter",
			url:            "/validators/vat/customers",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "vat query parameter is required",
		},
		{
			name:           "Empty vat parameter",
			url:            "/validators/vat/customers?vat=",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "vat query parameter is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh router for each test
			testRouter := gin.New()

			// Setup route
			testRouter.GET("/validators/vat/:entity_type", ValidateVAT)

			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			w := httptest.NewRecorder()

			// Execute request
			testRouter.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusNotFound {
				// For 404, Gin returns plain text, not JSON
				assert.Contains(t, w.Body.String(), tt.expectedError)
			} else {
				var response response.Response
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, response.Code)
				assert.Contains(t, response.Message, tt.expectedError)
			}
		})
	}
}

func TestValidateVAT_DatabaseError(t *testing.T) {
	_, mock, _, cleanup := setupGinTest()
	defer cleanup()

	// Create fresh router
	testRouter := gin.New()

	// Setup mock to return error
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
		WithArgs("12345678901", "").
		WillReturnError(sql.ErrConnDone)

	// Setup route
	testRouter.GET("/validators/vat/:entity_type", ValidateVAT)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/validators/vat/customers?vat=12345678901", nil)
	w := httptest.NewRecorder()

	// Execute request
	testRouter.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response response.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.Contains(t, response.Message, "failed to check VAT in customers")

	// Verify all mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestValidateVAT_AllEntityTypes(t *testing.T) {
	_, mock, _, cleanup := setupGinTest()
	defer cleanup()

	entityTypes := []string{"distributors", "resellers", "customers"}

	for _, entityType := range entityTypes {
		t.Run("Valid entity type: "+entityType, func(t *testing.T) {
			// Create fresh router
			testRouter := gin.New()

			// Setup mock
			mock.ExpectQuery(`SELECT COUNT\(\*\) FROM `+entityType).
				WithArgs("12345678901", "").
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

			// Setup route
			testRouter.GET("/validators/vat/:entity_type", ValidateVAT)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/validators/vat/"+entityType+"?vat=12345678901", nil)
			w := httptest.NewRecorder()

			// Execute request
			testRouter.ServeHTTP(w, req)

			// Verify response
			assert.Equal(t, http.StatusOK, w.Code)

			var response response.Response
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, response.Code)

			// Verify all mock expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestValidateVAT_ResponseFormat(t *testing.T) {
	_, mock, _, cleanup := setupGinTest()
	defer cleanup()

	// Create fresh router
	testRouter := gin.New()

	// Setup mock
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
		WithArgs("12345678901", "").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	// Setup route
	testRouter.GET("/validators/vat/:entity_type", ValidateVAT)

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/validators/vat/customers?vat=12345678901", nil)
	w := httptest.NewRecorder()

	// Execute request
	testRouter.ServeHTTP(w, req)

	// Verify response structure
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var responseMap map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &responseMap)
	assert.NoError(t, err)

	// Verify response has required fields
	assert.Contains(t, responseMap, "code")
	assert.Contains(t, responseMap, "message")
	assert.Contains(t, responseMap, "data")

	// Verify data structure
	data, ok := responseMap["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, data, "exists")

	exists, ok := data["exists"].(bool)
	assert.True(t, ok)
	assert.True(t, exists)

	// Verify all mock expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}
