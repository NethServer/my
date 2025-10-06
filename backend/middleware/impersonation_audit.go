/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/local"
)

// responseBodyWriter is a wrapper to capture response data
type responseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r responseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

// ImpersonationAuditMiddleware logs all API calls made during impersonation
func ImpersonationAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if this is an impersonation session
		isImpersonated, exists := c.Get("is_impersonated")
		if !exists || !isImpersonated.(bool) {
			// Not an impersonation session, continue normally
			c.Next()
			return
		}

		// Get impersonation context
		sessionID, sessionExists := c.Get("session_id")
		impersonator, impersonatorExists := c.Get("impersonated_by")
		impersonatedUser, userExists := helpers.GetUserFromContext(c)

		if !sessionExists || !impersonatorExists || !userExists {
			// Missing context data, log warning but continue
			logger.RequestLogger(c, "impersonate").Warn().
				Bool("session_exists", sessionExists).
				Bool("impersonator_exists", impersonatorExists).
				Bool("user_exists", userExists).
				Str("operation", "audit_missing_context").
				Msg("Missing impersonation context for audit logging")
			c.Next()
			return
		}

		sessionIDStr, ok := sessionID.(string)
		if !ok || sessionIDStr == "" {
			// Invalid session ID, continue but log warning
			logger.RequestLogger(c, "impersonate").Warn().
				Interface("session_id", sessionID).
				Str("operation", "audit_invalid_session_id").
				Msg("Invalid session ID for audit logging")
			c.Next()
			return
		}

		impersonatorUser, ok := impersonator.(*models.User)
		if !ok || impersonatorUser == nil {
			// Invalid impersonator, continue but log warning
			logger.RequestLogger(c, "impersonate").Warn().
				Interface("impersonator", impersonator).
				Str("operation", "audit_invalid_impersonator").
				Msg("Invalid impersonator data for audit logging")
			c.Next()
			return
		}

		// Skip audit logging for the impersonation endpoints themselves to avoid recursion
		if strings.HasPrefix(c.Request.URL.Path, "/api/impersonate/audit") {
			c.Next()
			return
		}

		// Capture request data
		var requestData string
		if c.Request.Method != "GET" && c.Request.Method != "HEAD" {
			// Only capture request body for methods that typically have one
			if c.Request.Body != nil {
				bodyBytes, err := io.ReadAll(c.Request.Body)
				if err != nil {
					logger.RequestLogger(c, "impersonate").Warn().
						Err(err).
						Str("operation", "audit_read_request_body").
						Msg("Failed to read request body for audit")
					requestData = "Failed to read request body"
				} else {
					// Restore the request body for the actual handler
					c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

					// Sanitize request data - remove any potential sensitive information
					requestData = sanitizeRequestData(string(bodyBytes))
				}
			}
		}

		// Capture response
		responseWriter := &responseBodyWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = responseWriter

		// Process the request
		c.Next()

		// Create audit entry
		responseStatus := c.Writer.Status()
		responseStatusText := http.StatusText(responseStatus)
		auditEntry := &models.ImpersonationAuditEntry{
			SessionID:            sessionIDStr,
			ImpersonatorUserID:   helpers.GetEffectiveUserID(impersonatorUser),
			ImpersonatedUserID:   impersonatedUser.ID,
			ActionType:           "api_call",
			APIEndpoint:          &c.Request.URL.Path,
			HTTPMethod:           &c.Request.Method,
			ResponseStatus:       &responseStatus,
			ResponseStatusText:   &responseStatusText,
			ImpersonatorUsername: impersonatorUser.Username,
			ImpersonatedUsername: impersonatedUser.Username,
			ImpersonatorName:     impersonatorUser.Name,
			ImpersonatedName:     impersonatedUser.Name,
		}

		// Add request data if not empty
		if requestData != "" {
			auditEntry.RequestData = &requestData
		}

		// Log the audit entry
		impersonationService := local.NewImpersonationService()
		err := impersonationService.LogImpersonationAction(auditEntry)
		if err != nil {
			logger.RequestLogger(c, "impersonate").Error().
				Err(err).
				Str("operation", "audit_log_failed").
				Str("session_id", sessionIDStr).
				Str("api_endpoint", c.Request.URL.Path).
				Str("http_method", c.Request.Method).
				Int("response_status", c.Writer.Status()).
				Msg("Failed to log impersonation audit entry")
		} else {
			logger.RequestLogger(c, "impersonate").Debug().
				Str("operation", "audit_logged").
				Str("session_id", sessionIDStr).
				Str("api_endpoint", c.Request.URL.Path).
				Str("http_method", c.Request.Method).
				Int("response_status", c.Writer.Status()).
				Str("impersonator_id", impersonatorUser.ID).
				Str("impersonated_id", impersonatedUser.ID).
				Msg("Impersonation action logged successfully")
		}
	}
}

// sanitizeRequestData removes or masks sensitive information from request data
func sanitizeRequestData(data string) string {
	// Limit size to prevent huge audit logs
	maxSize := 10000 // 10KB limit
	if len(data) > maxSize {
		data = data[:maxSize] + "... (truncated)"
	}

	// Try to parse as JSON to sanitize sensitive fields
	var jsonData interface{}
	if err := json.Unmarshal([]byte(data), &jsonData); err == nil {
		// Successfully parsed as JSON, sanitize it
		sanitized := sanitizeJSONData(jsonData)
		if sanitizedBytes, err := json.Marshal(sanitized); err == nil {
			return string(sanitizedBytes)
		}
	}

	// If not JSON or failed to sanitize, return as is (already size-limited)
	return data
}

// sanitizeJSONData recursively sanitizes JSON data to remove sensitive information
func sanitizeJSONData(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		sanitized := make(map[string]interface{})
		for key, value := range v {
			// List of sensitive field names to mask
			sensitiveFields := []string{
				"password", "secret", "token", "api_key", "private_key",
				"current_password", "new_password", "refresh_token",
				"access_token", "authorization", "auth", "key",
			}

			isSensitive := false
			keyLower := strings.ToLower(key)
			for _, sensitiveField := range sensitiveFields {
				if strings.Contains(keyLower, sensitiveField) {
					isSensitive = true
					break
				}
			}

			if isSensitive {
				sanitized[key] = "[REDACTED]"
			} else {
				sanitized[key] = sanitizeJSONData(value)
			}
		}
		return sanitized
	case []interface{}:
		sanitized := make([]interface{}, len(v))
		for i, item := range v {
			sanitized[i] = sanitizeJSONData(item)
		}
		return sanitized
	default:
		return v
	}
}
