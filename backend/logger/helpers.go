/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logger

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// HTTPErrorLogger standardizes HTTP error logging with response
type HTTPErrorLogger struct {
	logger    *zerolog.Logger
	component string
}

// NewHTTPErrorLogger creates a new HTTP error logger
func NewHTTPErrorLogger(c *gin.Context, component string) *HTTPErrorLogger {
	return &HTTPErrorLogger{
		logger:    RequestLogger(c, component),
		component: component,
	}
}

// LogError logs an error with HTTP context and returns the appropriate HTTP status
func (h *HTTPErrorLogger) LogError(err error, operation string, statusCode int, userMessage string) {
	var logEvent *zerolog.Event

	// Choose log level based on HTTP status code
	switch {
	case statusCode >= 500:
		logEvent = h.logger.Error()
	case statusCode >= 400:
		logEvent = h.logger.Warn()
	default:
		logEvent = h.logger.Info()
	}

	logEvent.
		Err(err).
		Str("operation", operation).
		Int("status_code", statusCode).
		Str("user_message", userMessage).
		Msg(fmt.Sprintf("HTTP %d: %s", statusCode, operation))
}

// LogSuccess logs a successful HTTP operation
func (h *HTTPErrorLogger) LogSuccess(operation string, statusCode int, data interface{}) {
	// Don't log the actual data to avoid sensitive information
	h.logger.Info().
		Str("operation", operation).
		Int("status_code", statusCode).
		Bool("success", true).
		Msg(fmt.Sprintf("HTTP %d: %s completed successfully", statusCode, operation))
}

// Authentication logging helpers

// LogAuthAttempt logs an authentication attempt
func LogAuthAttempt(c *gin.Context, component, method, userIdentifier string) {
	RequestLogger(c, component).Info().
		Str("operation", "auth_attempt").
		Str("method", method).
		Str("user_identifier", SanitizeString(userIdentifier)).
		Msg("Authentication attempt")
}

// LogAuthSuccess logs successful authentication
func LogAuthSuccess(c *gin.Context, component, method, userID, orgID string) {
	RequestLogger(c, component).Info().
		Str("operation", "auth_success").
		Str("method", method).
		Str("user_id", userID).
		Str("organization_id", orgID).
		Msg("Authentication successful")
}

// LogAuthFailure logs failed authentication
func LogAuthFailure(c *gin.Context, component, method, reason string, err error) {
	RequestLogger(c, component).Warn().
		Str("operation", "auth_failure").
		Str("method", method).
		Str("reason", reason).
		Err(err).
		Msg("Authentication failed")
}

// LogTokenExchange logs token exchange operations
func LogTokenExchange(c *gin.Context, component, tokenType string, success bool, err error) {
	logger := RequestLogger(c, component)

	if success {
		logger.Info().
			Str("operation", "token_exchange").
			Str("token_type", tokenType).
			Bool("success", true).
			Msg("Token exchange successful")
	} else {
		logger.Error().
			Str("operation", "token_exchange").
			Str("token_type", tokenType).
			Bool("success", false).
			Err(err).
			Msg("Token exchange failed")
	}
}

// Business operation logging helpers

// LogBusinessOperation logs business operations (CRUD operations)
func LogBusinessOperation(c *gin.Context, component, operation, entityType, entityID string, success bool, err error) {
	logger := RequestLogger(c, component)

	event := logger.Info()
	if !success {
		event = logger.Error()
	}

	event.
		Str("operation", operation).
		Str("entity_type", entityType).
		Str("entity_id", entityID).
		Bool("success", success)

	if err != nil {
		event.Err(err)
	}

	event.Msg(fmt.Sprintf("%s %s: %s", operation, entityType, entityID))
}

// LogAccountOperation logs account management operations with additional security context
func LogAccountOperation(c *gin.Context, operation, targetUserID, targetOrgID, actorUserID, actorOrgID string, success bool, err error) {
	logger := RequestLogger(c, "accounts")

	event := logger.Info()
	if !success {
		event = logger.Error()
	}

	event.
		Str("operation", operation).
		Str("target_user_id", targetUserID).
		Str("target_organization_id", targetOrgID).
		Str("actor_user_id", actorUserID).
		Str("actor_organization_id", actorOrgID).
		Bool("success", success)

	if err != nil {
		event.Err(err)
	}

	event.Msg(fmt.Sprintf("Account %s by %s", operation, actorUserID))
}

// External API logging helpers

// LogExternalAPICall logs calls to external APIs (like Logto Management API)
func LogExternalAPICall(component, service, method, endpoint string, statusCode int, duration int64, err error) {
	logger := ComponentLogger(component)

	event := logger.Info()
	if statusCode >= 400 || err != nil {
		event = logger.Error()
	} else if statusCode >= 300 {
		event = logger.Warn()
	}

	event.
		Str("operation", "external_api_call").
		Str("service", service).
		Str("method", method).
		Str("endpoint", SanitizeString(endpoint)). // Remove potential tokens from URL
		Int("status_code", statusCode).
		Int64("duration_ms", duration)

	if err != nil {
		event.Err(err)
	}

	event.Msg(fmt.Sprintf("%s API call: %s %s", service, method, endpoint))
}

// LogExternalAPIResponse logs external API responses (without sensitive data)
func LogExternalAPIResponse(component, service string, statusCode int, hasData bool) {
	ComponentLogger(component).Debug().
		Str("operation", "external_api_response").
		Str("service", service).
		Int("status_code", statusCode).
		Bool("has_data", hasData).
		Msg(fmt.Sprintf("%s API response: %d", service, statusCode))
}

// System operation logging helpers

// LogSystemOperation logs system operations (restart, backup, etc.)
func LogSystemOperation(c *gin.Context, operation, systemID string, success bool, err error) {
	logger := RequestLogger(c, "systems")

	event := logger.Info()
	if !success {
		event = logger.Error()
	}

	// Critical operations should be logged at higher level
	if operation == "factory_reset" || operation == "destroy" || operation == "restore" {
		if success {
			event = logger.Warn() // Successful destructive operations as warnings
		} else {
			event = logger.Error()
		}
	}

	event.
		Str("operation", operation).
		Str("system_id", systemID).
		Bool("success", success)

	if err != nil {
		event.Err(err)
	}

	event.Msg(fmt.Sprintf("System %s: %s", operation, systemID))
}

// Configuration and startup logging helpers

// LogConfigLoad logs configuration loading
func LogConfigLoad(component, configType string, success bool, err error) {
	logger := ComponentLogger(component)

	if success {
		logger.Info().
			Str("operation", "config_load").
			Str("config_type", configType).
			Bool("success", true).
			Msg(fmt.Sprintf("%s configuration loaded", configType))
	} else {
		logger.Fatal(). // Configuration failures are fatal
			Str("operation", "config_load").
			Str("config_type", configType).
			Bool("success", false).
			Err(err).
			Msg(fmt.Sprintf("Failed to load %s configuration", configType))
	}
}

// LogServiceStart logs service startup
func LogServiceStart(serviceName, version, listenAddress string) {
	Logger.Info().
		Str("operation", "service_start").
		Str("service", serviceName).
		Str("version", version).
		Str("listen_address", listenAddress).
		Msg(fmt.Sprintf("%s starting on %s", serviceName, listenAddress))
}

// LogServiceStop logs service shutdown
func LogServiceStop(serviceName string, reason string) {
	Logger.Info().
		Str("operation", "service_stop").
		Str("service", serviceName).
		Str("reason", reason).
		Msg(fmt.Sprintf("%s shutting down: %s", serviceName, reason))
}