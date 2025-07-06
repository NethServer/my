/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logger

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitFromEnv(t *testing.T) {
	// Capture the original logger
	originalLogger := Logger

	err := InitFromEnv("test-service")
	require.NoError(t, err)

	// Check that logger is initialized
	assert.NotEqual(t, originalLogger, Logger)
	assert.Equal(t, zerolog.InfoLevel, Logger.GetLevel())

	// Reset to original logger
	Logger = originalLogger
}

func TestSetLevel(t *testing.T) {
	// Setup
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	tests := []struct {
		input    string
		expected zerolog.Level
	}{
		{"debug", zerolog.DebugLevel},
		{"DEBUG", zerolog.DebugLevel},
		{"info", zerolog.InfoLevel},
		{"INFO", zerolog.InfoLevel},
		{"warn", zerolog.WarnLevel},
		{"warning", zerolog.WarnLevel},
		{"WARNING", zerolog.WarnLevel},
		{"error", zerolog.ErrorLevel},
		{"ERROR", zerolog.ErrorLevel},
		{"fatal", zerolog.FatalLevel},
		{"FATAL", zerolog.FatalLevel},
		{"invalid", zerolog.InfoLevel}, // Default fallback
		{"", zerolog.InfoLevel},        // Default fallback
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			SetLevel(tt.input)
			assert.Equal(t, tt.expected, Logger.GetLevel())
		})
	}
}

func TestComponentLogger(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	componentLogger := ComponentLogger("test-component")
	componentLogger.Info().Msg("test message")

	output := buf.String()
	assert.Contains(t, output, "test-component")
	assert.Contains(t, output, "test message")
}

func TestSanitizeMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON password field",
			input:    `{"password": "secret123", "username": "user"}`,
			expected: `{"password": "[******]", "username": "user"}`,
		},
		{
			name:     "JSON access_token field",
			input:    `{"access_token": "abc123xyz", "user": "test"}`,
			expected: `{"access_token": "[******]", "user": "test"}`,
		},
		{
			name:     "JSON client_secret field",
			input:    `{"client_secret": "very-secret-key", "name": "app"}`,
			expected: `{"client_secret": "[******]", "name": "app"}`,
		},
		{
			name:     "Key-value password",
			input:    "password=secret123 username=user",
			expected: "[******] username=user",
		},
		{
			name:     "Key-value token",
			input:    "token=abc123xyz data=value",
			expected: "[******] data=value",
		},
		{
			name:     "Bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: \"[******]\" eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9",
		},
		{
			name:     "Base64 token",
			input:    "Token: dGVzdC10b2tlbi12YWx1ZS1mb3ItdGVzdGluZy1wdXJwb3Nlcw==",
			expected: "Token: \"[******]\"",
		},
		{
			name:     "Multiple secrets",
			input:    `{"password": "secret", "token": "abc123"} key=value`,
			expected: `{"password": "[******]", "token": "[******]"} [******]`,
		},
		{
			name:     "JSON with escaped quotes",
			input:    `{"password": "secret\"with\"quotes", "data": "normal"}`,
			expected: `{"password": "[******]", "data": "normal"}`,
		},
		{
			name:     "No sensitive data",
			input:    `{"username": "user", "email": "test@example.com"}`,
			expected: `{"username": "user", "email": "test@example.com"}`,
		},
		{
			name:     "Case insensitive matching",
			input:    `{"PASSWORD": "secret", "Token": "abc123"}`,
			expected: `{"PASSWORD": "[******]", "Token": "[******]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeMessage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLegacyAPIFunctions(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Test that legacy functions work and sanitize messages
	Debug("Debug message with password=secret123")
	Info("Info message with token=abc123")
	Warn("Warning message with key=value")
	Error("Error message with client_secret=secret")

	output := buf.String()

	// Check that messages were logged
	assert.Contains(t, output, "Debug message")
	assert.Contains(t, output, "Info message")
	assert.Contains(t, output, "Warning message")
	assert.Contains(t, output, "Error message")

	// Check that sensitive data was sanitized
	assert.NotContains(t, output, "secret123")
	assert.NotContains(t, output, "abc123")
	assert.NotContains(t, output, "secret")
	assert.Contains(t, output, "[******]")
}

func TestLegacyLevelFunctions(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Test Init with different levels
	tests := []struct {
		level    LogLevel
		expected zerolog.Level
	}{
		{DebugLevel, zerolog.DebugLevel},
		{InfoLevel, zerolog.InfoLevel},
		{WarnLevel, zerolog.WarnLevel},
		{ErrorLevel, zerolog.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			// Save original stderr
			originalStderr := os.Stderr

			// Create a pipe to capture stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Use SetLevelLegacy instead of Init to avoid logger reinitialization
			SetLevelLegacy(tt.level)
			assert.Equal(t, tt.expected, Logger.GetLevel())

			// Test GetLevel
			assert.Equal(t, tt.level, GetLevel())

			// Restore stderr
			_ = w.Close()
			os.Stderr = originalStderr
			_ = r.Close()
		})
	}
}

func TestStructuredLoggingFunctions(t *testing.T) {
	var buf bytes.Buffer
	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	t.Run("LogSyncOperation success", func(t *testing.T) {
		buf.Reset()
		LogSyncOperation("sync", "user", "create", true, nil)

		output := buf.String()
		assert.Contains(t, output, "sync")
		assert.Contains(t, output, "user")
		assert.Contains(t, output, "create")
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "Sync operation completed")
	})

	t.Run("LogSyncOperation failure", func(t *testing.T) {
		buf.Reset()
		LogSyncOperation("sync", "role", "update", false, assert.AnError)

		output := buf.String()
		assert.Contains(t, output, "sync")
		assert.Contains(t, output, "role")
		assert.Contains(t, output, "update")
		assert.Contains(t, output, "false")
		assert.Contains(t, output, "Sync operation failed")
		assert.Contains(t, output, assert.AnError.Error())
	})

	t.Run("LogAPICall success", func(t *testing.T) {
		buf.Reset()
		LogAPICall("GET", "/api/test", 200, 100*time.Millisecond)

		output := buf.String()
		assert.Contains(t, output, "GET")
		assert.Contains(t, output, "/api/test")
		assert.Contains(t, output, "200")
		assert.Contains(t, output, "100")
		assert.Contains(t, output, "API call completed")
	})

	t.Run("LogAPICall client error", func(t *testing.T) {
		buf.Reset()
		LogAPICall("POST", "/api/test", 400, 50*time.Millisecond)

		output := buf.String()
		assert.Contains(t, output, "POST")
		assert.Contains(t, output, "400")
		assert.Contains(t, output, "50")
	})

	t.Run("LogAPICall server error", func(t *testing.T) {
		buf.Reset()
		LogAPICall("PUT", "/api/test", 500, 200*time.Millisecond)

		output := buf.String()
		assert.Contains(t, output, "PUT")
		assert.Contains(t, output, "500")
		assert.Contains(t, output, "200")
	})

	t.Run("LogAPICall with sensitive endpoint", func(t *testing.T) {
		buf.Reset()
		LogAPICall("POST", "/api/auth?token=secret123", 200, 100*time.Millisecond)

		output := buf.String()
		assert.Contains(t, output, "/api/auth")
		assert.NotContains(t, output, "secret123")
		assert.Contains(t, output, "[******]")
	})

	t.Run("LogConfigLoad valid", func(t *testing.T) {
		buf.Reset()
		LogConfigLoad("/path/to/config.yml", 5, 3, true)

		output := buf.String()
		assert.Contains(t, output, "/path/to/config.yml")
		assert.Contains(t, output, "5")
		assert.Contains(t, output, "3")
		assert.Contains(t, output, "true")
		assert.Contains(t, output, "Configuration loaded")
	})

	t.Run("LogConfigLoad invalid", func(t *testing.T) {
		buf.Reset()
		LogConfigLoad("/path/to/bad-config.yml", 0, 0, false)

		output := buf.String()
		assert.Contains(t, output, "/path/to/bad-config.yml")
		assert.Contains(t, output, "0")
		assert.Contains(t, output, "false")
		assert.Contains(t, output, "Configuration loaded")
	})
}

func TestSensitivePatterns(t *testing.T) {
	// Test that our regex patterns are working correctly
	testCases := []struct {
		name        string
		input       string
		shouldMatch bool
	}{
		{"JSON password", `"password": "test123"`, true},
		{"JSON token", `"access_token": "abc123"`, true},
		{"Key-value password", "password=test123", true},
		{"Key-value token", "token=abc123", true},
		{"Bearer token", "Bearer eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9", true},
		{"Base64 token", "dGVzdC10b2tlbi12YWx1ZS1mb3ItdGVzdGluZy1wdXJwb3Nlcw==", true},
		{"Normal text", "This is normal text", false},
		{"Username field", `"username": "user123"`, false},
		{"Short string", "abc123", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matched := false
			for _, pattern := range sensitivePatterns {
				if pattern.MatchString(tc.input) {
					matched = true
					break
				}
			}
			assert.Equal(t, tc.shouldMatch, matched, "Pattern matching failed for: %s", tc.input)
		})
	}
}
