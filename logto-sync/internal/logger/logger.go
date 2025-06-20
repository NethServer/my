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
	"log"
	"os"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	// DebugLevel for debug messages
	DebugLevel LogLevel = iota
	// InfoLevel for info messages
	InfoLevel
	// WarnLevel for warning messages
	WarnLevel
	// ErrorLevel for error messages
	ErrorLevel
)

var (
	currentLevel LogLevel = InfoLevel
	logger       *log.Logger
)

// Init initializes the logger with the specified level
func Init(level LogLevel) {
	currentLevel = level
	logger = log.New(os.Stderr, "", 0)
}

// Debug logs a debug message
func Debug(format string, args ...interface{}) {
	if currentLevel <= DebugLevel {
		logMessage("DEBUG", format, args...)
	}
}

// Info logs an info message
func Info(format string, args ...interface{}) {
	if currentLevel <= InfoLevel {
		logMessage("INFO", format, args...)
	}
}

// Warn logs a warning message
func Warn(format string, args ...interface{}) {
	if currentLevel <= WarnLevel {
		logMessage("WARN", format, args...)
	}
}

// Error logs an error message
func Error(format string, args ...interface{}) {
	if currentLevel <= ErrorLevel {
		logMessage("ERROR", format, args...)
	}
}

// Fatal logs a fatal message and exits
func Fatal(format string, args ...interface{}) {
	logMessage("FATAL", format, args...)
	os.Exit(1)
}

func logMessage(level, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logger.Printf("[%s] %s %s", level, timestamp, message)
}

// SetLevel sets the current logging level
func SetLevel(level LogLevel) {
	currentLevel = level
}

// GetLevel returns the current logging level
func GetLevel() LogLevel {
	return currentLevel
}
