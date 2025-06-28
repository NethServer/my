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
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// Global logger instance
	Logger zerolog.Logger

	// Patterns for sensitive data detection
	sensitivePatterns = []*regexp.Regexp{
		// JSON format with quotes: "password": "any_value_including_special_chars_and_escaped_quotes"
		regexp.MustCompile(`(?i)"(password|pwd|secret|token|key|auth|bearer|authorization)":\s*"(\\.|[^"\\])*"`),
		regexp.MustCompile(`(?i)"(access_token|refresh_token|id_token|client_secret|client_id|api_key|apikey)":\s*"(\\.|[^"\\])*"`),
		// Key-value without quotes - stop at whitespace, comma, brace, or newline
		regexp.MustCompile(`(?i)(password|pwd|secret|token|key|auth|bearer|authorization)[:=]\s*\S+`),
		regexp.MustCompile(`(?i)(access_token|refresh_token|id_token|client_secret|client_id|api_key|apikey)[:=]\s*\S+`),
		// Bearer tokens
		regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9+/=_-]{20,}`),
		// Base64 tokens (standalone)
		regexp.MustCompile(`\b[a-zA-Z0-9+/]{40,}={0,2}\b`),
	}
)

// InitFromEnv initializes the logger from environment variables
func InitFromEnv(serviceName string) error {
	// Set up console writer for development
	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
		NoColor:    false,
	}

	// Create logger with service name
	Logger = zerolog.New(output).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger()

	// Set global logger
	log.Logger = Logger

	return nil
}

// SetLevel sets the logging level
func SetLevel(level string) {
	var zLevel zerolog.Level
	switch strings.ToLower(level) {
	case "debug":
		zLevel = zerolog.DebugLevel
	case "info":
		zLevel = zerolog.InfoLevel
	case "warn", "warning":
		zLevel = zerolog.WarnLevel
	case "error":
		zLevel = zerolog.ErrorLevel
	case "fatal":
		zLevel = zerolog.FatalLevel
	default:
		zLevel = zerolog.InfoLevel
	}

	Logger = Logger.Level(zLevel)
	log.Logger = Logger
}

// ComponentLogger returns a logger with component context
func ComponentLogger(component string) zerolog.Logger {
	return Logger.With().Str("component", component).Logger()
}

// SanitizeMessage removes sensitive information from log messages
func SanitizeMessage(message string) string {
	for _, pattern := range sensitivePatterns {
		message = pattern.ReplaceAllStringFunc(message, func(match string) string {
			// For JSON-style matches, preserve the structure but redact the value
			if strings.Contains(match, ":") {
				parts := strings.SplitN(match, ":", 2)
				if len(parts) == 2 {
					return parts[0] + ": \"[******]\""
				}
			}
			// For other patterns, replace with placeholder
			return "[******]"
		})
	}
	return message
}

// Legacy API for backward compatibility

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	Logger.Debug().Msgf(SanitizeMessage(format), args...)
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	Logger.Info().Msgf(SanitizeMessage(format), args...)
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	Logger.Warn().Msgf(SanitizeMessage(format), args...)
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	Logger.Error().Msgf(SanitizeMessage(format), args...)
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	Logger.Fatal().Msgf(SanitizeMessage(format), args...)
}

// Legacy LogLevel for backward compatibility
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Init initializes the logger with the specified level (legacy compatibility)
func Init(level LogLevel) {
	err := InitFromEnv("sync-tool")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	var levelStr string
	switch level {
	case DebugLevel:
		levelStr = "debug"
	case InfoLevel:
		levelStr = "info"
	case WarnLevel:
		levelStr = "warn"
	case ErrorLevel:
		levelStr = "error"
	default:
		levelStr = "info"
	}

	SetLevel(levelStr)
}

// SetLevel sets the current logging level (legacy compatibility)
func SetLevelLegacy(level LogLevel) {
	var levelStr string
	switch level {
	case DebugLevel:
		levelStr = "debug"
	case InfoLevel:
		levelStr = "info"
	case WarnLevel:
		levelStr = "warn"
	case ErrorLevel:
		levelStr = "error"
	default:
		levelStr = "info"
	}

	SetLevel(levelStr)
}

// GetLevel returns the current logging level (legacy compatibility)
func GetLevel() LogLevel {
	switch Logger.GetLevel() {
	case zerolog.DebugLevel:
		return DebugLevel
	case zerolog.InfoLevel:
		return InfoLevel
	case zerolog.WarnLevel:
		return WarnLevel
	case zerolog.ErrorLevel:
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// LogSyncOperation logs a sync operation with structured data
func LogSyncOperation(operation, entity, action string, success bool, err error) {
	event := Logger.Info()
	if !success {
		event = Logger.Error()
	}

	event.
		Str("operation", operation).
		Str("entity", entity).
		Str("action", action).
		Bool("success", success)

	if err != nil {
		event.Err(err)
	}

	if success {
		event.Msg("Sync operation completed")
	} else {
		event.Msg("Sync operation failed")
	}
}

// LogAPICall logs an API call with structured data
func LogAPICall(method, endpoint string, statusCode int, duration time.Duration) {
	event := Logger.Debug()
	if statusCode >= 400 {
		event = Logger.Warn()
	}
	if statusCode >= 500 {
		event = Logger.Error()
	}

	event.
		Str("component", "api-client").
		Str("method", method).
		Str("endpoint", SanitizeMessage(endpoint)).
		Int("status_code", statusCode).
		Dur("duration", duration).
		Msg("API call completed")
}

// LogConfigLoad logs configuration loading with validation results
func LogConfigLoad(configPath string, resourceCount, roleCount int, isValid bool) {
	event := Logger.Info()
	if !isValid {
		event = Logger.Error()
	}

	event.
		Str("component", "config").
		Str("path", configPath).
		Int("resources", resourceCount).
		Int("roles", roleCount).
		Bool("valid", isValid).
		Msg("Configuration loaded")
}
