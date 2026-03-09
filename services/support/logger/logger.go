/*
 * Copyright (C) 2026 Nethesis S.r.l.
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
)

// Logger is the global logger instance
var Logger zerolog.Logger

// InitFromEnv initializes the logger from environment variables
func InitFromEnv(appName string) error {
	level := os.Getenv("LOG_LEVEL")
	format := os.Getenv("LOG_FORMAT")

	logLevel, err := zerolog.ParseLevel(strings.ToLower(level))
	if err != nil || level == "" {
		logLevel = zerolog.InfoLevel
	}

	var output io.Writer
	if strings.ToLower(format) == "json" {
		output = os.Stderr
	} else {
		output = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
	}

	Logger = zerolog.New(output).With().
		Timestamp().
		Str("app", appName).
		Logger().
		Level(logLevel)

	zerolog.DefaultContextLogger = &Logger

	return nil
}

// ComponentLogger returns a logger scoped to a component
func ComponentLogger(component string) *zerolog.Logger {
	l := Logger.With().Str("component", component).Logger()
	return &l
}

// RequestLogger returns a logger enriched with request context
func RequestLogger(c *gin.Context, component string) *zerolog.Logger {
	l := Logger.With().
		Str("component", component).
		Str("method", c.Request.Method).
		Str("path", c.Request.URL.Path).
		Str("client_ip", c.ClientIP()).
		Logger()
	return &l
}

// Package-level convenience functions
func Trace() *zerolog.Event { return Logger.Trace() }
func Debug() *zerolog.Event { return Logger.Debug() }
func Info() *zerolog.Event  { return Logger.Info() }
func Warn() *zerolog.Event  { return Logger.Warn() }
func Error() *zerolog.Event { return Logger.Error() }
func Fatal() *zerolog.Event { return Logger.Fatal() }

// SanitizeConnectionURL redacts credentials from connection URLs
func SanitizeConnectionURL(url string) string {
	if url == "" {
		return ""
	}
	re := regexp.MustCompile(`://([^:]+):([^@]+)@`)
	return re.ReplaceAllString(url, "://$1:***@")
}

// LogConfigLoad logs a configuration loading event
func LogConfigLoad(component, configType string, success bool, err error) {
	logger := ComponentLogger(component)

	if success {
		logger.Info().
			Str("operation", "config_load").
			Str("config_type", configType).
			Msg("configuration loaded")
	} else {
		logger.Warn().
			Str("operation", "config_load").
			Str("config_type", configType).
			Err(err).
			Msg("configuration load issue")
	}
}

// LogServiceStart logs service startup information
func LogServiceStart(serviceName, version, listenAddress string) {
	Logger.Info().
		Str("operation", "service_start").
		Str("service", serviceName).
		Str("version", version).
		Str("listen_address", listenAddress).
		Msg("service starting")
}

// GinLogger returns a gin middleware that logs requests using zerolog
func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		event := Logger.Info()
		if c.Writer.Status() >= 500 {
			event = Logger.Error()
		} else if c.Writer.Status() >= 400 {
			event = Logger.Warn()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("client_ip", c.ClientIP()).
			Msg("request")
	}
}

// SecurityMiddleware returns a gin middleware that sets security headers
func SecurityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Next()
	}
}
