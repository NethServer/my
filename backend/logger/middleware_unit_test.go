package logger

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func setupTestGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router
}

func TestDefaultGinLoggerConfig(t *testing.T) {
	config := DefaultGinLoggerConfig()

	assert.Equal(t, []string{"/api/health"}, config.SkipPaths)
	assert.Equal(t, []string{"/static/", "/assets/", "/favicon."}, config.SkipPathPrefixes)
	assert.NotNil(t, config.Logger)
}

func TestGinLoggerConfigShouldSkipPath(t *testing.T) {
	config := DefaultGinLoggerConfig()

	tests := []struct {
		path     string
		expected bool
	}{
		{"/api/health", true},         // Exact match
		{"/api/users", false},         // No match
		{"/static/css/app.css", true}, // Prefix match
		{"/assets/js/main.js", true},  // Prefix match
		{"/favicon.ico", true},        // Prefix match
		{"/favicon.png", true},        // Prefix match
		{"/api/favicon", false},       // Not a prefix match
		{"/normal/path", false},       // No match
		{"", false},                   // Empty path
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.shouldSkipPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGinLoggerConfigCustomSkipPaths(t *testing.T) {
	config := GinLoggerConfig{
		SkipPaths:        []string{"/custom/health", "/admin/status"},
		SkipPathPrefixes: []string{"/internal/", "/debug/"},
		Logger:           Logger,
	}

	tests := []struct {
		path     string
		expected bool
	}{
		{"/custom/health", true},      // Custom exact match
		{"/admin/status", true},       // Custom exact match
		{"/internal/metrics", true},   // Custom prefix match
		{"/debug/profiler", true},     // Custom prefix match
		{"/api/users", false},         // No match
		{"/custom/health/sub", false}, // Not exact match
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := config.shouldSkipPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGinLogger(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(GinLogger())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "/test")
	assert.Contains(t, logOutput, "200")
	assert.Contains(t, logOutput, "test-agent")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "http", logEntry["component"])
	assert.Equal(t, "GET", logEntry["method"])
	assert.Equal(t, "/test", logEntry["path"])
	assert.Equal(t, float64(200), logEntry["status_code"])
	assert.Contains(t, logEntry, "latency_ms")
	assert.Contains(t, logEntry, "client_ip")
}

func TestGinLoggerWithQueryParameters(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(GinLogger())
	router.GET("/search", func(c *gin.Context) {
		c.JSON(200, gin.H{"results": []string{}})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/search?q=test&token=secret123", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "/search")
	assert.Contains(t, logOutput, "q=test")
	// Token should be sanitized
	assert.Contains(t, logOutput, "[******]")
	assert.NotContains(t, logOutput, "secret123")

	// Parse JSON to verify structure
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	pathWithQuery := logEntry["path"].(string)
	assert.Contains(t, pathWithQuery, "/search")
	assert.Contains(t, pathWithQuery, "q=test")
	assert.Contains(t, pathWithQuery, "[******]")
}

func TestGinLoggerWithUserContext(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(GinLogger())
	router.GET("/protected", func(c *gin.Context) {
		c.Set("user_id", "user-123")
		c.Set("organization_id", "org-456")
		c.JSON(200, gin.H{"data": "protected"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "/protected")

	// Parse JSON to verify user context is included
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "user-123", logEntry["user_id"])
	assert.Equal(t, "org-456", logEntry["organization_id"])
}

func TestGinLoggerWithErrors(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(GinLogger())
	router.GET("/error", func(c *gin.Context) {
		_ = c.Error(gin.Error{
			Err:  assert.AnError,
			Type: gin.ErrorTypePublic,
		})
		c.JSON(500, gin.H{"error": "internal error"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 500, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "/error")

	// Parse JSON to verify error is included
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, float64(500), logEntry["status_code"])
	assert.Equal(t, "error", logEntry["level"])
	assert.Contains(t, logEntry, "error")
}

func TestGinLoggerStatusCodeLevels(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		expectedLevel string
	}{
		{"success_200", 200, "info"},
		{"created_201", 201, "info"},
		{"bad_request_400", 400, "warn"},
		{"unauthorized_401", 401, "warn"},
		{"not_found_404", 404, "warn"},
		{"internal_error_500", 500, "error"},
		{"bad_gateway_502", 502, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			router := setupTestGin()
			router.Use(GinLogger())
			router.GET("/status", func(c *gin.Context) {
				c.Status(tt.statusCode)
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/status", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)

			logOutput := buf.String()
			assert.Contains(t, logOutput, "/status")

			// Parse JSON to verify log level
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedLevel, logEntry["level"])
			assert.Equal(t, float64(tt.statusCode), logEntry["status_code"])
		})
	}
}

func TestGinLoggerSkipPaths(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(GinLogger())
	router.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.GET("/api/users", func(c *gin.Context) {
		c.JSON(200, gin.H{"users": []string{}})
	})

	// Request to skipped path
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/health", nil)
	router.ServeHTTP(w1, req1)

	// Request to non-skipped path
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/users", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)

	logOutput := buf.String()
	// Health endpoint should not be logged
	assert.NotContains(t, logOutput, "/api/health")
	// Users endpoint should be logged
	assert.Contains(t, logOutput, "/api/users")
}

func TestGinLoggerCustomConfig(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	config := GinLoggerConfig{
		SkipPaths:        []string{"/custom/skip"},
		SkipPathPrefixes: []string{},
		Logger:           Logger,
	}

	router := setupTestGin()
	router.Use(GinLoggerWithConfig(config))
	router.GET("/custom/skip", func(c *gin.Context) {
		c.JSON(200, gin.H{"skipped": true})
	})
	router.GET("/custom/log", func(c *gin.Context) {
		c.JSON(200, gin.H{"logged": true})
	})

	// Request to custom skipped path
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/custom/skip", nil)
	router.ServeHTTP(w1, req1)

	// Request to logged path
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/custom/log", nil)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, 200, w1.Code)
	assert.Equal(t, 200, w2.Code)

	logOutput := buf.String()
	assert.NotContains(t, logOutput, "/custom/skip")
	assert.Contains(t, logOutput, "/custom/log")
}

func TestSecurityMiddleware(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	// Set debug level to capture debug logs
	originalLogger := Logger
	originalLevel := zerolog.GlobalLevel()
	defer func() {
		Logger = originalLogger
		zerolog.SetGlobalLevel(originalLevel)
	}()

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	Logger = zerolog.New(buf).Level(zerolog.DebugLevel).With().Timestamp().Logger()

	router := setupTestGin()
	router.Use(SecurityMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer token123")
	req.Header.Set("User-Agent", "test-agent") // Add user agent to prevent empty user agent warning
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "auth_header_present")
	assert.Contains(t, logOutput, "/test")

	// Should not log the actual token
	assert.NotContains(t, logOutput, "token123")
	assert.NotContains(t, logOutput, "Bearer token123")
}

func TestSecurityMiddlewareEmptyUserAgent(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(SecurityMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	// No User-Agent header set (empty)
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "empty_user_agent")
	assert.Contains(t, logOutput, "/test")
}

func TestSecurityMiddlewareSuspiciousRequests(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		shouldDetect bool
	}{
		{"path_traversal", "/api/../etc/passwd", true},
		{"windows_path_traversal", "/api/..\\windows\\system32", true},
		{"xss_attempt", "/search?q=%3Cscript%3Ealert%281%29%3C%2Fscript%3E", true},
		{"javascript_injection", "/page?redirect=javascript%3Aalert%281%29", true},
		{"sql_injection", "/users?id=1%20union%20select%20*%20from%20users", true},
		{"wordpress_scan", "/wp-admin/login.php", true},
		{"phpmyadmin_scan", "/phpmyadmin/index.php", true},
		{"env_file_access", "/.env", true},
		{"git_access", "/.git/config", true},
		{"normal_request", "/api/users", false},
		{"safe_search", "/search?q=hello%20world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			router := setupTestGin()
			router.Use(SecurityMiddleware())
			router.Any("/*path", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "ok"})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", tt.path, nil)
			req.Header.Set("User-Agent", "test-agent")
			router.ServeHTTP(w, req)

			logOutput := buf.String()
			if tt.shouldDetect {
				assert.Contains(t, logOutput, "suspicious_request")
				assert.Contains(t, logOutput, "Potentially malicious request detected")
			} else {
				assert.NotContains(t, logOutput, "suspicious_request")
				assert.NotContains(t, logOutput, "Potentially malicious request detected")
			}
		})
	}
}

func TestIsLikelyAttack(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"../etc/passwd", true},
		{"..\\windows\\system32", true},
		{"/normal/path", false},
		{"/search?q=<script>alert(1)</script>", true},
		{"/redirect?url=javascript:void(0)", true},
		{"/page.html", false},
		{"/api/users?id=1 union select password from users", true},
		{"/admin?cmd=drop table users", true},
		{"/api/users", false},
		{"/wp-admin/admin.php", true},
		{"/phpmyadmin/", true},
		{"/.env", true},
		{"/.git/config", true},
		{"/api/config.json", false},
		{"/assets/app.js", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isLikelyAttack(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGinLoggerSanitization(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(GinLogger())
	router.GET("/login", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/login?username=john&password=secret123", nil)
	req.Header.Set("User-Agent", "MyApp/1.0 (secret_key=abc123)")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "/login")
	assert.Contains(t, logOutput, "username=john")
	// Sensitive data should be redacted
	assert.Contains(t, logOutput, "[******]")
	assert.NotContains(t, logOutput, "secret123")
	assert.NotContains(t, logOutput, "abc123")
}

func TestMiddlewareEdgeCases(t *testing.T) {
	t.Run("empty_path", func(t *testing.T) {
		config := DefaultGinLoggerConfig()
		result := config.shouldSkipPath("")
		assert.False(t, result)
	})

	t.Run("nil_skip_paths", func(t *testing.T) {
		config := GinLoggerConfig{
			SkipPaths:        nil,
			SkipPathPrefixes: nil,
			Logger:           Logger,
		}
		result := config.shouldSkipPath("/any/path")
		assert.False(t, result)
	})

	t.Run("empty_skip_paths", func(t *testing.T) {
		config := GinLoggerConfig{
			SkipPaths:        []string{},
			SkipPathPrefixes: []string{},
			Logger:           Logger,
		}
		result := config.shouldSkipPath("/any/path")
		assert.False(t, result)
	})
}

func TestSecurityMiddlewareWithValidUserAgent(t *testing.T) {
	buf, cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestGin()
	router.Use(SecurityMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "success"})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyBot/1.0)")
	router.ServeHTTP(w, req)

	assert.Equal(t, 200, w.Code)

	logOutput := buf.String()
	// Should not log anything for normal user agent
	assert.Equal(t, "", strings.TrimSpace(logOutput))
}

func TestGinLoggerBehaviorWithDifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			buf, cleanup := setupTestLogger()
			defer cleanup()

			router := setupTestGin()
			router.Use(GinLogger())
			router.Any("/test", func(c *gin.Context) {
				c.JSON(200, gin.H{"method": method})
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, 200, w.Code)

			logOutput := buf.String()
			assert.Contains(t, logOutput, method)
			assert.Contains(t, logOutput, "/test")

			// Parse JSON to verify method is logged correctly
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, method, logEntry["method"])
		})
	}
}
