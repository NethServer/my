package testutils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/logger"
)

// SetupTestGin configures Gin for testing
func SetupTestGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

// MakeRequest performs an HTTP request for testing
func MakeRequest(t *testing.T, router *gin.Engine, method, url string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	var reqBody io.Reader

	if body != nil {
		jsonBody, err := json.Marshal(body)
		assert.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	assert.NoError(t, err)

	// Set default content type
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set custom headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// AssertJSONResponse checks if response is valid JSON and matches expected status
func AssertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	assert.Equal(t, expectedStatus, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	return response
}

// MockHTTPServer creates a test HTTP server for external API mocking
func MockHTTPServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// SetupLogger initializes logger for testing
func SetupLogger() {
	_ = logger.Init(&logger.Config{
		Level:   logger.InfoLevel,
		Format:  logger.JSONFormat,
		Output:  logger.StdoutOutput,
		AppName: "[TEST]",
	})
}
