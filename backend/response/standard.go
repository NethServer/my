/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package response

import (
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Response represents the standard API response format
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// ValidationError represents a structured validation error
type ValidationError struct {
	Key     string `json:"key"`
	Message string `json:"message"`
	Input   string `json:"input,omitempty"`
}

// Success creates a success response
func Success(code int, message string, data interface{}) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// Error creates an error response
func Error(code int, message string, data interface{}) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// OK creates a 200 OK response
func OK(message string, data interface{}) Response {
	return Success(200, message, data)
}

// Created creates a 201 Created response
func Created(message string, data interface{}) Response {
	return Success(201, message, data)
}

// BadRequest creates a 400 Bad Request response
func BadRequest(message string, data interface{}) Response {
	return Error(400, message, data)
}

// Unauthorized creates a 401 Unauthorized response
func Unauthorized(message string, data interface{}) Response {
	return Error(401, message, data)
}

// Forbidden creates a 403 Forbidden response
func Forbidden(message string, data interface{}) Response {
	return Error(403, message, data)
}

// NotFound creates a 404 Not Found response
func NotFound(message string, data interface{}) Response {
	return Error(404, message, data)
}

// Conflict creates a 409 Conflict response
func Conflict(message string, data interface{}) Response {
	return Error(409, message, data)
}

// UnprocessableEntity creates a 422 Unprocessable Entity response
func UnprocessableEntity(message string, data interface{}) Response {
	return Error(422, message, data)
}

// InternalServerError creates a 500 Internal Server Error response
func InternalServerError(message string, data interface{}) Response {
	return Error(500, message, data)
}

// ParseValidationError parses validation errors from Gin binding and returns a structured ValidationError
func ParseValidationError(err error) ValidationError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		if len(validationErrors) > 0 {
			ve := validationErrors[0]

			// Extract field name from struct tag
			fieldName := ve.Field()

			// Get the validation tag that failed
			tag := ve.Tag()

			// Get the actual value that failed validation
			value := ""
			if ve.Value() != nil {
				value = strings.TrimSpace(reflect.ValueOf(ve.Value()).String())
			}

			return ValidationError{
				Key:     fieldName,
				Message: tag,
				Input:   value,
			}
		}
	}

	// Fallback for other error types
	return ValidationError{
		Key:     "unknown",
		Message: err.Error(),
		Input:   "",
	}
}

// ValidationBadRequest creates a 400 Bad Request response with structured validation error
func ValidationBadRequest(err error) Response {
	validationError := ParseValidationError(err)
	return BadRequest("validation failed", validationError)
}
