package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupTestLogger() (*bytes.Buffer, func()) {
	var buf bytes.Buffer
	originalLogger := Logger
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	return &buf, func() {
		Logger = originalLogger
	}
}

func setupTestGinContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	return c, w
}

func TestHTTPErrorLogger(t *testing.T) {
	_, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()
	httpLogger := NewHTTPErrorLogger(c, "test-component")

	assert.NotNil(t, httpLogger)
	assert.Equal(t, "test-component", httpLogger.component)
	assert.NotNil(t, httpLogger.logger)
}

func TestHTTPErrorLoggerLogError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		operation  string
		err        error
		message    string
	}{
		{
			name:       "500_server_error",
			statusCode: 500,
			operation:  "database_query",
			err:        errors.New("connection failed"),
			message:    "Internal server error",
		},
		{
			name:       "400_client_error",
			statusCode: 400,
			operation:  "validate_input",
			err:        errors.New("invalid format"),
			message:    "Bad request",
		},
		{
			name:       "200_success_logged_as_info",
			statusCode: 200,
			operation:  "get_data",
			err:        nil,
			message:    "Success",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			c, _ := setupTestGinContext()
			httpLogger := NewHTTPErrorLogger(c, "test-component")

			httpLogger.LogError(tt.err, tt.operation, tt.statusCode, tt.message)

			logOutput := buf.String()
			assert.Contains(t, logOutput, tt.operation)
			assert.Contains(t, logOutput, tt.message)
			assert.Contains(t, logOutput, "test-component")

			// Parse JSON to verify structure
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, float64(tt.statusCode), logEntry["status_code"])
			assert.Equal(t, tt.operation, logEntry["operation"])
			assert.Equal(t, tt.message, logEntry["user_message"])
		})
	}
}

func TestHTTPErrorLoggerLogSuccess(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()
	httpLogger := NewHTTPErrorLogger(c, "test-component")

	testData := map[string]interface{}{"result": "success"}
	httpLogger.LogSuccess("user_creation", 201, testData)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "user_creation")
	assert.Contains(t, logOutput, "completed successfully")
	assert.Contains(t, logOutput, "test-component")

	// Verify it doesn't log the actual data content (security)
	assert.NotContains(t, logOutput, "result")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, float64(201), logEntry["status_code"])
	assert.Equal(t, "user_creation", logEntry["operation"])
	assert.Equal(t, true, logEntry["success"])
}

func TestLogAuthAttempt(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()

	LogAuthAttempt(c, "auth", "jwt", "user@example.com")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "auth_attempt")
	assert.Contains(t, logOutput, "jwt")
	assert.Contains(t, logOutput, "user@example.com")
	assert.Contains(t, logOutput, "Authentication attempt")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "auth_attempt", logEntry["operation"])
	assert.Equal(t, "jwt", logEntry["method"])
}

func TestLogAuthSuccess(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()

	LogAuthSuccess(c, "auth", "oauth", "user-123", "org-456")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "auth_success")
	assert.Contains(t, logOutput, "oauth")
	assert.Contains(t, logOutput, "user-123")
	assert.Contains(t, logOutput, "org-456")
	assert.Contains(t, logOutput, "Authentication successful")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "auth_success", logEntry["operation"])
	assert.Equal(t, "oauth", logEntry["method"])
	assert.Equal(t, "user-123", logEntry["user_id"])
	assert.Equal(t, "org-456", logEntry["organization_id"])
}

func TestLogAuthFailure(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()
	testErr := errors.New("invalid credentials")

	LogAuthFailure(c, "auth", "basic", "invalid_password", testErr)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "auth_failure")
	assert.Contains(t, logOutput, "basic")
	assert.Contains(t, logOutput, "invalid_password")
	assert.Contains(t, logOutput, "Authentication failed")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "auth_failure", logEntry["operation"])
	assert.Equal(t, "basic", logEntry["method"])
	assert.Equal(t, "invalid_password", logEntry["reason"])
}

func TestLogTokenExchange(t *testing.T) {
	tests := []struct {
		name      string
		success   bool
		tokenType string
		err       error
	}{
		{
			name:      "successful_exchange",
			success:   true,
			tokenType: "access_token",
			err:       nil,
		},
		{
			name:      "failed_exchange",
			success:   false,
			tokenType: "refresh_token",
			err:       errors.New("token expired"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			c, _ := setupTestGinContext()

			LogTokenExchange(c, "auth", tt.tokenType, tt.success, tt.err)

			logOutput := buf.String()
			assert.Contains(t, logOutput, "token_exchange")
			assert.Contains(t, logOutput, tt.tokenType)

			if tt.success {
				assert.Contains(t, logOutput, "Token exchange successful")
			} else {
				assert.Contains(t, logOutput, "Token exchange failed")
			}

			// Parse JSON to verify structure
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "token_exchange", logEntry["operation"])
			assert.Equal(t, tt.tokenType, logEntry["token_type"])
			assert.Equal(t, tt.success, logEntry["success"])
		})
	}
}

func TestLogBusinessOperation(t *testing.T) {
	tests := []struct {
		name       string
		operation  string
		entityType string
		entityID   string
		success    bool
		err        error
	}{
		{
			name:       "successful_create",
			operation:  "create",
			entityType: "user",
			entityID:   "user-123",
			success:    true,
			err:        nil,
		},
		{
			name:       "failed_update",
			operation:  "update",
			entityType: "customer",
			entityID:   "customer-456",
			success:    false,
			err:        errors.New("validation failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			c, _ := setupTestGinContext()

			LogBusinessOperation(c, "business", tt.operation, tt.entityType, tt.entityID, tt.success, tt.err)

			logOutput := buf.String()
			assert.Contains(t, logOutput, tt.operation)
			assert.Contains(t, logOutput, tt.entityType)
			assert.Contains(t, logOutput, tt.entityID)

			// Parse JSON to verify structure
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, tt.operation, logEntry["operation"])
			assert.Equal(t, tt.entityType, logEntry["entity_type"])
			assert.Equal(t, tt.entityID, logEntry["entity_id"])
			assert.Equal(t, tt.success, logEntry["success"])
		})
	}
}

func TestLogAccountOperation(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()

	LogAccountOperation(c, "create", "target-user-123", "target-org-456",
		"actor-user-789", "actor-org-101", true, nil)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "create")
	assert.Contains(t, logOutput, "target-user-123")
	assert.Contains(t, logOutput, "actor-user-789")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "create", logEntry["operation"])
	assert.Equal(t, "target-user-123", logEntry["target_user_id"])
	assert.Equal(t, "target-org-456", logEntry["target_organization_id"])
	assert.Equal(t, "actor-user-789", logEntry["actor_user_id"])
	assert.Equal(t, "actor-org-101", logEntry["actor_organization_id"])
	assert.Equal(t, true, logEntry["success"])
}

func TestLogExternalAPICall(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		duration    int64
		err         error
		expectLevel string
	}{
		{
			name:        "successful_call",
			statusCode:  200,
			duration:    150,
			err:         nil,
			expectLevel: "info",
		},
		{
			name:        "redirect_call",
			statusCode:  302,
			duration:    50,
			err:         nil,
			expectLevel: "warn",
		},
		{
			name:        "client_error",
			statusCode:  404,
			duration:    75,
			err:         errors.New("not found"),
			expectLevel: "error",
		},
		{
			name:        "server_error",
			statusCode:  500,
			duration:    1000,
			err:         errors.New("internal error"),
			expectLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			LogExternalAPICall("api-client", "Logto", "GET", "/api/users",
				tt.statusCode, tt.duration, tt.err)

			logOutput := buf.String()
			assert.Contains(t, logOutput, "external_api_call")
			assert.Contains(t, logOutput, "Logto")
			assert.Contains(t, logOutput, "GET")

			// Parse JSON to verify structure
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, "external_api_call", logEntry["operation"])
			assert.Equal(t, "Logto", logEntry["service"])
			assert.Equal(t, "GET", logEntry["method"])
			assert.Equal(t, float64(tt.statusCode), logEntry["status_code"])
			assert.Equal(t, float64(tt.duration), logEntry["duration_ms"])
			assert.Equal(t, tt.expectLevel, logEntry["level"])
		})
	}
}

func TestLogExternalAPIResponse(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	LogExternalAPIResponse("api-client", "Logto", 200, true)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "external_api_response")
	assert.Contains(t, logOutput, "Logto")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "external_api_response", logEntry["operation"])
	assert.Equal(t, "Logto", logEntry["service"])
	assert.Equal(t, float64(200), logEntry["status_code"])
	assert.Equal(t, true, logEntry["has_data"])
	assert.Equal(t, "debug", logEntry["level"])
}

func TestLogSystemOperation(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		systemID    string
		success     bool
		err         error
		expectLevel string
	}{
		{
			name:        "successful_restart",
			operation:   "restart",
			systemID:    "system-123",
			success:     true,
			err:         nil,
			expectLevel: "info",
		},
		{
			name:        "successful_destroy",
			operation:   "destroy",
			systemID:    "system-456",
			success:     true,
			err:         nil,
			expectLevel: "warn", // Destructive operations logged as warnings
		},
		{
			name:        "failed_factory_reset",
			operation:   "factory_reset",
			systemID:    "system-789",
			success:     false,
			err:         errors.New("operation failed"),
			expectLevel: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			c, _ := setupTestGinContext()

			LogSystemOperation(c, tt.operation, tt.systemID, tt.success, tt.err)

			logOutput := buf.String()
			assert.Contains(t, logOutput, tt.operation)
			assert.Contains(t, logOutput, tt.systemID)

			// Parse JSON to verify structure
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, tt.operation, logEntry["operation"])
			assert.Equal(t, tt.systemID, logEntry["system_id"])
			assert.Equal(t, tt.success, logEntry["success"])
			assert.Equal(t, tt.expectLevel, logEntry["level"])
		})
	}
}

func TestLogConfigLoad(t *testing.T) {
	t.Run("successful_load", func(t *testing.T) {
		buf, cleanup := setupTestLogger()
		defer cleanup()

		LogConfigLoad("config", "database", true, nil)

		logOutput := buf.String()
		assert.Contains(t, logOutput, "config_load")
		assert.Contains(t, logOutput, "database")

		// Parse JSON to verify structure
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
		assert.NoError(t, err)
		assert.Equal(t, "config_load", logEntry["operation"])
		assert.Equal(t, "database", logEntry["config_type"])
		assert.Equal(t, true, logEntry["success"])
		assert.Equal(t, "info", logEntry["level"])
	})

	// Note: We skip testing the failed case because LogConfigLoad calls logger.Fatal()
	// which would terminate the test process. In a real application, this is the desired
	// behavior for critical configuration failures.
}

func TestLogServiceStart(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	LogServiceStart("nethesis-backend", "1.0.0", ":8080")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service_start")
	assert.Contains(t, logOutput, "nethesis-backend")
	assert.Contains(t, logOutput, "1.0.0")
	assert.Contains(t, logOutput, ":8080")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "service_start", logEntry["operation"])
	assert.Equal(t, "nethesis-backend", logEntry["service"])
	assert.Equal(t, "1.0.0", logEntry["version"])
	assert.Equal(t, ":8080", logEntry["listen_address"])
}

func TestLogServiceStop(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	LogServiceStop("nethesis-backend", "graceful shutdown")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "service_stop")
	assert.Contains(t, logOutput, "nethesis-backend")
	assert.Contains(t, logOutput, "graceful shutdown")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "service_stop", logEntry["operation"])
	assert.Equal(t, "nethesis-backend", logEntry["service"])
	assert.Equal(t, "graceful shutdown", logEntry["reason"])
}

func TestRequestLoggerWithContext(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()
	c.Set("user_id", "user-123")
	c.Set("organization_id", "org-456")

	logger := RequestLogger(c, "test-component")
	assert.NotNil(t, logger)

	logger.Info().Msg("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test-component")
	assert.Contains(t, logOutput, "test message")

	// Parse JSON to verify context is included
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "test-component", logEntry["component"])
	assert.Equal(t, "GET", logEntry["method"])
	assert.Equal(t, "/test", logEntry["path"])
	assert.Equal(t, "user-123", logEntry["user_id"])
	assert.Equal(t, "org-456", logEntry["organization_id"])
}

func TestRequestLoggerWithNilContext(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	logger := RequestLogger(nil, "test-component")
	assert.NotNil(t, logger)

	logger.Info().Msg("test message")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "test-component")
	assert.Contains(t, logOutput, "test message")

	// Should not contain request-specific fields
	assert.NotContains(t, logOutput, "method")
	assert.NotContains(t, logOutput, "path")
}

func TestHelperFunctionsWithSensitiveData(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	c, _ := setupTestGinContext()

	// Test that sensitive data in user identifier is sanitized
	LogAuthAttempt(c, "auth", "jwt", "user@example.com password=secret123")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "user@example.com")
	assert.Contains(t, logOutput, "[******]") // Password should be redacted
	assert.NotContains(t, logOutput, "secret123")
}
