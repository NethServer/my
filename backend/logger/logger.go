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
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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

	// Sensitive field names that should be redacted
	sensitiveFields = map[string]bool{
		"password":      true,
		"secret":        true,
		"token":         true,
		"access_token":  true,
		"refresh_token": true,
		"id_token":      true,
		"client_secret": true,
		"api_key":       true,
		"apikey":        true,
		"authorization": true,
		"auth":          true,
		"bearer":        true,
		"jwt":           true,
		"private_key":   true,
		"cert":          true,
		"certificate":   true,
	}
)

// LogLevel represents the logging level
type LogLevel string

const (
	TraceLevel LogLevel = "trace"
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

// LogFormat represents the logging format
type LogFormat string

const (
	JSONFormat    LogFormat = "json"
	ConsoleFormat LogFormat = "console"
)

// LogOutput represents where logs are written
type LogOutput string

const (
	StdoutOutput LogOutput = "stdout"
	StderrOutput LogOutput = "stderr"
	FileOutput   LogOutput = "file"
)

// Config holds the logger configuration
type Config struct {
	Level      LogLevel  `json:"level"`
	Format     LogFormat `json:"format"`
	Output     LogOutput `json:"output"`
	FilePath   string    `json:"file_path"`
	AppName    string    `json:"app_name"`
	TimeFormat string    `json:"time_format"`
}

// DefaultConfig returns a default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      InfoLevel,
		Format:     JSONFormat,
		Output:     StdoutOutput,
		FilePath:   "",
		AppName:    "nethesis-backend",
		TimeFormat: time.RFC3339,
	}
}

// Init initializes the global logger with the given configuration
func Init(config *Config) error {
	// Set global log level
	switch config.Level {
	case TraceLevel:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case DebugLevel:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case InfoLevel:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case WarnLevel:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case ErrorLevel:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case FatalLevel:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case PanicLevel:
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Configure time format
	zerolog.TimeFieldFormat = config.TimeFormat

	// Choose output destination
	var output io.Writer
	switch config.Output {
	case StdoutOutput:
		output = os.Stdout
	case StderrOutput:
		output = os.Stderr
	case FileOutput:
		if config.FilePath == "" {
			output = os.Stderr
		} else {
			file, err := os.OpenFile(config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err != nil {
				return err
			}
			output = file
		}
	default:
		output = os.Stdout
	}

	// Configure format
	if config.Format == ConsoleFormat && config.Output != FileOutput {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339,
			NoColor:    false,
		}
	}

	// Create logger with context
	Logger = zerolog.New(output).
		Level(zerolog.GlobalLevel()).
		With().
		Timestamp().
		Str("service", config.AppName).
		Logger()

	// Replace global logger
	log.Logger = Logger

	return nil
}

// InitFromEnv initializes the logger from environment variables
func InitFromEnv(appName string) error {
	config := DefaultConfig()
	config.AppName = appName

	// Read configuration from environment
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		config.Level = LogLevel(strings.ToLower(level))
	}

	if format := os.Getenv("LOG_FORMAT"); format != "" {
		config.Format = LogFormat(strings.ToLower(format))
	}

	if output := os.Getenv("LOG_OUTPUT"); output != "" {
		config.Output = LogOutput(strings.ToLower(output))
	}

	if filePath := os.Getenv("LOG_FILE_PATH"); filePath != "" {
		config.FilePath = filePath
	}

	return Init(config)
}

// SanitizeString removes sensitive data from log messages
func SanitizeString(input string) string {
	result := input
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			// Handle JSON format: "field": "value"
			if strings.Contains(match, `":`) {
				parts := strings.SplitN(match, `":`, 2)
				if len(parts) == 2 {
					return parts[0] + `": "[******]"`
				}
			}
			// Handle key=value format
			if strings.Contains(match, "=") {
				parts := strings.SplitN(match, "=", 2)
				if len(parts) == 2 {
					return parts[0] + "=[******]"
				}
			}
			// Handle key: value format
			if strings.Contains(match, ":") {
				parts := strings.SplitN(match, ":", 2)
				if len(parts) == 2 {
					return parts[0] + ": [******]"
				}
			}
			// Fallback
			return "[******]"
		})
	}
	return result
}

// SanitizeField checks if a field name is sensitive and should be redacted
func SanitizeField(fieldName string) bool {
	return sensitiveFields[strings.ToLower(fieldName)]
}

// SafeStr safely logs a string field, redacting it if the field name is sensitive
func SafeStr(fieldName, value string) (string, interface{}) {
	if SanitizeField(fieldName) {
		return fieldName, "[******]"
	}
	return fieldName, SanitizeString(value)
}

// SafeInterface safely logs an interface field, redacting it if the field name is sensitive
func SafeInterface(fieldName string, value interface{}) (string, interface{}) {
	if SanitizeField(fieldName) {
		return fieldName, "[******]"
	}

	// If it's a string, sanitize it
	if str, ok := value.(string); ok {
		return fieldName, SanitizeString(str)
	}

	return fieldName, value
}

// ComponentLogger creates a logger with a component field
func ComponentLogger(component string) *zerolog.Logger {
	logger := Logger.With().Str("component", component).Logger()
	return &logger
}

// RequestLogger creates a logger with request context from Gin
func RequestLogger(c *gin.Context, component string) *zerolog.Logger {
	logger := *ComponentLogger(component)

	// Add request context
	if c != nil {
		logger = logger.With().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("client_ip", c.ClientIP()).
			Logger()

		// Add user context if available
		if userID, exists := c.Get("user_id"); exists {
			if uid, ok := userID.(string); ok {
				logger = logger.With().Str("user_id", uid).Logger()
			}
		}

		if orgID, exists := c.Get("organization_id"); exists {
			if oid, ok := orgID.(string); ok {
				logger = logger.With().Str("organization_id", oid).Logger()
			}
		}
	}

	return &logger
}

// Helper functions for common logging patterns

// Trace logs a trace level message
func Trace() *zerolog.Event {
	return Logger.Trace()
}

// Debug logs a debug level message
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info logs an info level message
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn logs a warning level message
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error logs an error level message
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal logs a fatal level message and calls os.Exit(1)
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}

// Panic logs a panic level message and panics
func Panic() *zerolog.Event {
	return Logger.Panic()
}
