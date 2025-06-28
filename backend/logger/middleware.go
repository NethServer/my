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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// GinLoggerConfig defines the configuration for Gin logging middleware
type GinLoggerConfig struct {
	// SkipPaths defines the paths to skip logging
	SkipPaths []string
	// SkipPathPrefixes defines path prefixes to skip logging
	SkipPathPrefixes []string
	// Logger is the zerolog logger to use
	Logger zerolog.Logger
}

// DefaultGinLoggerConfig returns a default configuration for Gin logging
func DefaultGinLoggerConfig() GinLoggerConfig {
	return GinLoggerConfig{
		SkipPaths: []string{
			"/api/health", // Skip health checks
		},
		SkipPathPrefixes: []string{
			"/static/",  // Skip static files
			"/assets/",  // Skip asset files
			"/favicon.", // Skip favicon requests
		},
		Logger: Logger,
	}
}

// shouldSkipPath checks if a path should be skipped from logging
func (config GinLoggerConfig) shouldSkipPath(path string) bool {
	// Check exact paths
	for _, skipPath := range config.SkipPaths {
		if path == skipPath {
			return true
		}
	}

	// Check path prefixes
	for _, prefix := range config.SkipPathPrefixes {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// GinLogger returns a Gin middleware that logs HTTP requests using zerolog
func GinLogger() gin.HandlerFunc {
	return GinLoggerWithConfig(DefaultGinLoggerConfig())
}

// GinLoggerWithConfig returns a Gin middleware that logs HTTP requests with custom configuration
func GinLoggerWithConfig(config GinLoggerConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for specified paths
		if config.shouldSkipPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate request duration
		latency := time.Since(start)

		// Get client IP
		clientIP := c.ClientIP()

		// Get method
		method := c.Request.Method

		// Get status code
		statusCode := c.Writer.Status()

		// Get user agent (sanitized)
		userAgent := SanitizeString(c.Request.UserAgent())

		// Build the path with query parameters
		if raw != "" {
			path = path + "?" + SanitizeString(raw) // Sanitize query parameters
		}

		// Determine log level based on status code
		var logEvent *zerolog.Event
		switch {
		case statusCode >= 500:
			logEvent = config.Logger.Error()
		case statusCode >= 400:
			logEvent = config.Logger.Warn()
		default:
			logEvent = config.Logger.Info()
		}

		// Add common fields
		logEvent = logEvent.
			Str("component", "http").
			Str("method", method).
			Str("path", path).
			Int("status_code", statusCode).
			Int64("latency_ms", latency.Milliseconds()).
			Str("client_ip", clientIP).
			Str("user_agent", userAgent)

		// Add user context if available
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				logEvent = logEvent.Str("user_id", uid)
			}
		}

		if orgID, exists := c.Get("organization_id"); exists {
			if oid, ok := orgID.(string); ok {
				logEvent = logEvent.Str("organization_id", oid)
			}
		}

		// Add error information for error responses
		if len(c.Errors) > 0 {
			// Log the first error (sanitized)
			errorMsg := SanitizeString(c.Errors[0].Error())
			logEvent = logEvent.Str("error", errorMsg)
		}

		// Log the request
		logEvent.Msg(fmt.Sprintf("HTTP %d %s %s", statusCode, method, c.Request.URL.Path))
	}
}

// SecurityMiddleware logs security-related events
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Log potential security concerns

		// Check for suspicious headers
		if authHeader := c.GetHeader("Authorization"); authHeader != "" {
			// Log authorization header presence (without the actual token)
			ComponentLogger("security").Debug().
				Str("operation", "auth_header_present").
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("client_ip", c.ClientIP()).
				Bool("has_auth_header", true).
				Msg("Request with authorization header")
		}

		// Log potentially suspicious user agents
		userAgent := c.Request.UserAgent()
		if userAgent == "" {
			ComponentLogger("security").Warn().
				Str("operation", "empty_user_agent").
				Str("method", c.Request.Method).
				Str("path", c.Request.URL.Path).
				Str("client_ip", c.ClientIP()).
				Msg("Request with empty user agent")
		}

		// Check for common attack patterns in URL
		if isLikelyAttack(c.Request.URL.Path) {
			ComponentLogger("security").Warn().
				Str("operation", "suspicious_request").
				Str("method", c.Request.Method).
				Str("path", SanitizeString(c.Request.URL.Path)).
				Str("client_ip", c.ClientIP()).
				Str("user_agent", SanitizeString(userAgent)).
				Msg("Potentially malicious request detected")
		}

		c.Next()
	}
}

// isLikelyAttack checks if a URL path contains common attack patterns
func isLikelyAttack(path string) bool {
	suspiciousPatterns := []string{
		"../",          // Path traversal
		"..\\",         // Windows path traversal
		"<script",      // XSS attempt
		"javascript:",  // JavaScript injection
		"<iframe",      // Iframe injection
		"union select", // SQL injection
		"drop table",   // SQL injection
		"exec(",        // Command injection
		"system(",      // Command injection
		"passwd",       // File access attempt
		"/etc/",        // Linux system files
		"wp-admin",     // WordPress scanning
		"phpmyadmin",   // Database admin scanning
		".env",         // Environment file access
		".git",         // Git repository access
	}

	pathLower := strings.ToLower(path)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}

	return false
}
