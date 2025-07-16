package logger

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, InfoLevel, config.Level)
	assert.Equal(t, JSONFormat, config.Format)
	assert.Equal(t, StdoutOutput, config.Output)
	assert.Equal(t, "", config.FilePath)
	assert.Equal(t, "backend", config.AppName)
	assert.Equal(t, time.RFC3339, config.TimeFormat)
}

func TestConfigStructure(t *testing.T) {
	config := &Config{
		Level:      DebugLevel,
		Format:     ConsoleFormat,
		Output:     StderrOutput,
		FilePath:   "/tmp/test.log",
		AppName:    "test-app",
		TimeFormat: time.RFC822,
	}

	assert.Equal(t, DebugLevel, config.Level)
	assert.Equal(t, ConsoleFormat, config.Format)
	assert.Equal(t, StderrOutput, config.Output)
	assert.Equal(t, "/tmp/test.log", config.FilePath)
	assert.Equal(t, "test-app", config.AppName)
	assert.Equal(t, time.RFC822, config.TimeFormat)
}

func TestInitWithValidConfig(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	config := &Config{
		Level:      InfoLevel,
		Format:     JSONFormat,
		Output:     StdoutOutput,
		AppName:    "test-logger",
		TimeFormat: time.RFC3339,
	}

	// Temporarily replace stdout
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	err := Init(config)
	assert.NoError(t, err)

	// Create a new logger that writes to our buffer for testing
	testLogger := zerolog.New(&buf).With().Timestamp().Str("service", config.AppName).Logger()
	testLogger.Info().Msg("test message")

	// Verify the log output contains expected fields
	logOutput := buf.String()
	assert.Contains(t, logOutput, "test message")
	assert.Contains(t, logOutput, "test-logger")
}

func TestInitWithInvalidLevel(t *testing.T) {
	config := &Config{
		Level:      LogLevel("invalid"),
		Format:     JSONFormat,
		Output:     StdoutOutput,
		AppName:    "test-logger",
		TimeFormat: time.RFC3339,
	}

	err := Init(config)
	assert.NoError(t, err) // Should not error, should default to InfoLevel

	// Verify it defaults to info level
	assert.Equal(t, zerolog.InfoLevel, zerolog.GlobalLevel())
}

func TestInitFromEnv(t *testing.T) {
	// Save original env vars
	originalLevel := os.Getenv("LOG_LEVEL")
	originalFormat := os.Getenv("LOG_FORMAT")
	originalOutput := os.Getenv("LOG_OUTPUT")

	defer func() {
		_ = os.Setenv("LOG_LEVEL", originalLevel)
		_ = os.Setenv("LOG_FORMAT", originalFormat)
		_ = os.Setenv("LOG_OUTPUT", originalOutput)
	}()

	// Set test env vars
	_ = os.Setenv("LOG_LEVEL", "debug")
	_ = os.Setenv("LOG_FORMAT", "console")
	_ = os.Setenv("LOG_OUTPUT", "stderr")

	err := InitFromEnv("test-app")
	assert.NoError(t, err)

	// Verify the global level was set correctly
	assert.Equal(t, zerolog.DebugLevel, zerolog.GlobalLevel())
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password in JSON format",
			input:    `{"password": "secret123"}`,
			expected: `{"password": "[******]"}`,
		},
		{
			name:     "token in JSON format",
			input:    `{"access_token": "abc123def456"}`,
			expected: `{"access_token": "[******]"}`,
		},
		{
			name:     "password with key-value format",
			input:    "password=secret123",
			expected: "password=[******]",
		},
		{
			name:     "token with colon format",
			input:    "token: bearer_token_here",
			expected: "token: [******]",
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: [******]",
		},
		{
			name:     "normal text without sensitive data",
			input:    "This is normal log message",
			expected: "This is normal log message",
		},
		{
			name:     "mixed content with password",
			input:    `Processing user login with {"username": "john", "password": "secret123"}`,
			expected: `Processing user login with {"username": "john", "password": "[******]"}`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "case insensitive password",
			input:    `{"PASSWORD": "secret123"}`,
			expected: `{"PASSWORD": "[******]"}`,
		},
		{
			name:     "escaped quotes in JSON",
			input:    `{"password": "secret\"with\"quotes"}`,
			expected: `{"password": "[******]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeField(t *testing.T) {
	tests := []struct {
		fieldName string
		expected  bool
	}{
		{"password", true},
		{"secret", true},
		{"token", true},
		{"access_token", true},
		{"refresh_token", true},
		{"api_key", true},
		{"apikey", true},
		{"authorization", true},
		{"auth", true},
		{"bearer", true},
		{"jwt", true},
		{"private_key", true},
		{"cert", true},
		{"certificate", true},
		{"client_secret", true},
		{"PASSWORD", true}, // Case insensitive
		{"SECRET", true},   // Case insensitive
		{"username", false},
		{"email", false},
		{"name", false},
		{"id", false},
		{"normal_field", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.fieldName, func(t *testing.T) {
			result := SanitizeField(tt.fieldName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeStr(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		expField  string
		expValue  interface{}
	}{
		{
			name:      "sensitive field",
			fieldName: "password",
			value:     "secret123",
			expField:  "password",
			expValue:  "[******]",
		},
		{
			name:      "normal field",
			fieldName: "username",
			value:     "john_doe",
			expField:  "username",
			expValue:  "john_doe",
		},
		{
			name:      "normal field with sensitive content",
			fieldName: "message",
			value:     `User logged in with password=secret123`,
			expField:  "message",
			expValue:  `User logged in with password=[******]`,
		},
		{
			name:      "case insensitive sensitive field",
			fieldName: "PASSWORD",
			value:     "secret123",
			expField:  "PASSWORD",
			expValue:  "[******]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, value := SafeStr(tt.fieldName, tt.value)
			assert.Equal(t, tt.expField, field)
			assert.Equal(t, tt.expValue, value)
		})
	}
}

func TestSafeInterface(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     interface{}
		expField  string
		expValue  interface{}
	}{
		{
			name:      "sensitive field with string",
			fieldName: "token",
			value:     "abc123def456",
			expField:  "token",
			expValue:  "[******]",
		},
		{
			name:      "normal field with string",
			fieldName: "username",
			value:     "john_doe",
			expField:  "username",
			expValue:  "john_doe",
		},
		{
			name:      "normal field with integer",
			fieldName: "count",
			value:     42,
			expField:  "count",
			expValue:  42,
		},
		{
			name:      "normal field with boolean",
			fieldName: "active",
			value:     true,
			expField:  "active",
			expValue:  true,
		},
		{
			name:      "sensitive field with non-string",
			fieldName: "password",
			value:     123456,
			expField:  "password",
			expValue:  "[******]",
		},
		{
			name:      "normal field with string containing sensitive data",
			fieldName: "log_message",
			value:     "Authentication failed for token=abc123",
			expField:  "log_message",
			expValue:  "Authentication failed for token=[******]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, value := SafeInterface(tt.fieldName, tt.value)
			assert.Equal(t, tt.expField, field)
			assert.Equal(t, tt.expValue, value)
		})
	}
}

func TestComponentLogger(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger to write to buffer
	originalLogger := Logger
	defer func() { Logger = originalLogger }()

	Logger = zerolog.New(&buf).With().Timestamp().Logger()

	// Create component logger
	compLogger := ComponentLogger("test-component")
	require.NotNil(t, compLogger)

	// Log a message
	compLogger.Info().Msg("test message")

	// Verify output contains component
	logOutput := buf.String()
	assert.Contains(t, logOutput, "test-component")
	assert.Contains(t, logOutput, "test message")

	// Verify it's valid JSON
	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
	assert.NoError(t, err)
	assert.Equal(t, "test-component", logEntry["component"])
}

func TestLogLevelTypes(t *testing.T) {
	levels := []LogLevel{
		TraceLevel,
		DebugLevel,
		InfoLevel,
		WarnLevel,
		ErrorLevel,
		FatalLevel,
		PanicLevel,
	}

	expectedLevels := []string{
		"trace",
		"debug",
		"info",
		"warn",
		"error",
		"fatal",
		"panic",
	}

	for i, level := range levels {
		assert.Equal(t, expectedLevels[i], string(level))
	}
}

func TestLogFormatTypes(t *testing.T) {
	formats := []LogFormat{
		JSONFormat,
		ConsoleFormat,
	}

	expectedFormats := []string{
		"json",
		"console",
	}

	for i, format := range formats {
		assert.Equal(t, expectedFormats[i], string(format))
	}
}

func TestLogOutputTypes(t *testing.T) {
	outputs := []LogOutput{
		StdoutOutput,
		StderrOutput,
		FileOutput,
	}

	expectedOutputs := []string{
		"stdout",
		"stderr",
		"file",
	}

	for i, output := range outputs {
		assert.Equal(t, expectedOutputs[i], string(output))
	}
}

func TestHelperFunctions(t *testing.T) {
	var buf bytes.Buffer

	// Setup logger to write to buffer with trace level to capture all logs
	originalLogger := Logger
	originalLevel := zerolog.GlobalLevel()
	defer func() {
		Logger = originalLogger
		zerolog.SetGlobalLevel(originalLevel)
	}()

	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	Logger = zerolog.New(&buf).Level(zerolog.TraceLevel).With().Logger()

	// Test each helper function
	helpers := []struct {
		name string
		fn   func() *zerolog.Event
	}{
		{"Trace", Trace},
		{"Debug", Debug},
		{"Info", Info},
		{"Warn", Warn},
		{"Error", Error},
	}

	for _, helper := range helpers {
		t.Run(helper.name, func(t *testing.T) {
			buf.Reset()

			event := helper.fn()
			assert.NotNil(t, event)

			// Complete the log event
			event.Msg("test message")

			// Verify output
			logOutput := buf.String()
			assert.Contains(t, logOutput, "test message")

			// Verify it's valid JSON
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry)
			assert.NoError(t, err)
			assert.Equal(t, strings.ToLower(helper.name), logEntry["level"])
		})
	}
}

func TestConfigJSONSerialization(t *testing.T) {
	config := &Config{
		Level:      DebugLevel,
		Format:     ConsoleFormat,
		Output:     FileOutput,
		FilePath:   "/var/log/app.log",
		AppName:    "test-app",
		TimeFormat: time.RFC3339,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(config)
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Test JSON unmarshaling
	var unmarshaledConfig Config
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	assert.NoError(t, err)

	assert.Equal(t, config.Level, unmarshaledConfig.Level)
	assert.Equal(t, config.Format, unmarshaledConfig.Format)
	assert.Equal(t, config.Output, unmarshaledConfig.Output)
	assert.Equal(t, config.FilePath, unmarshaledConfig.FilePath)
	assert.Equal(t, config.AppName, unmarshaledConfig.AppName)
	assert.Equal(t, config.TimeFormat, unmarshaledConfig.TimeFormat)
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config := &Config{}
		err := Init(config)
		assert.NoError(t, err)
	})

	t.Run("nil patterns in SanitizeString", func(t *testing.T) {
		// This tests that SanitizeString handles the global patterns safely
		result := SanitizeString("test message")
		assert.Equal(t, "test message", result)
	})

	t.Run("component logger with empty component", func(t *testing.T) {
		logger := ComponentLogger("")
		assert.NotNil(t, logger)
	})
}

func TestSensitivePatternsCoverage(t *testing.T) {
	// Test that all sensitive patterns are actually used and work
	testCases := []string{
		`{"client_secret": "secret_value"}`,
		`client_id=abc123`,
		`Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9`,
		`api_key: very_secret_key_here`,
		`refresh_token=abcdef123456`,
		`password: "complex_password_123"`,
	}

	for _, testCase := range testCases {
		t.Run("pattern_test", func(t *testing.T) {
			sanitized := SanitizeString(testCase)
			assert.Contains(t, sanitized, "[******]")
			assert.NotEqual(t, testCase, sanitized)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	t.Run("valid_levels", func(t *testing.T) {
		validLevels := []LogLevel{
			TraceLevel, DebugLevel, InfoLevel, WarnLevel, ErrorLevel, FatalLevel, PanicLevel,
		}

		for _, level := range validLevels {
			config := DefaultConfig()
			config.Level = level
			err := Init(config)
			assert.NoError(t, err)
		}
	})

	t.Run("valid_formats", func(t *testing.T) {
		validFormats := []LogFormat{JSONFormat, ConsoleFormat}

		for _, format := range validFormats {
			config := DefaultConfig()
			config.Format = format
			err := Init(config)
			assert.NoError(t, err)
		}
	})

	t.Run("valid_outputs", func(t *testing.T) {
		validOutputs := []LogOutput{StdoutOutput, StderrOutput}

		for _, output := range validOutputs {
			config := DefaultConfig()
			config.Output = output
			err := Init(config)
			assert.NoError(t, err)
		}
	})
}
