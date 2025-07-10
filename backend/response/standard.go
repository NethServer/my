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
	Value   string `json:"value,omitempty"`
}

// ErrorData represents standardized error data
type ErrorData struct {
	Type    string            `json:"type"`
	Errors  []ValidationError `json:"errors,omitempty"`
	Details interface{}       `json:"details,omitempty"`
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

// getJSONFieldName extracts the JSON field name from validator.FieldError
func getJSONFieldName(ve validator.FieldError) string {
	// For now, use a simple approach: convert field name to lowercase
	// This covers the most common case where Go field names are capitalized
	// but JSON tags are lowercase
	fieldName := ve.Field()

	// Simple conversion to lowercase for now
	// This works for most standard cases like "Username" -> "username"
	return strings.ToLower(fieldName)
}

// ParseValidationError parses validation errors from Gin binding and returns a structured ValidationError
func ParseValidationError(err error) ValidationError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		if len(validationErrors) > 0 {
			ve := validationErrors[0]

			// Extract field name from JSON tag, fallback to lowercase field name
			fieldName := getJSONFieldName(ve)

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
				Value:   value,
			}
		}
	}

	// Fallback for other error types
	return ValidationError{
		Key:     "unknown",
		Message: err.Error(),
		Value:   "",
	}
}

// ParseValidationErrors parses multiple validation errors from Gin binding and returns a slice of ValidationError
func ParseValidationErrors(err error) []ValidationError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		result := make([]ValidationError, 0, len(validationErrors))
		for _, ve := range validationErrors {
			// Extract field name from JSON tag, fallback to lowercase field name
			fieldName := getJSONFieldName(ve)

			// Get the validation tag that failed
			tag := ve.Tag()

			// Get the actual value that failed validation
			value := ""
			if ve.Value() != nil {
				value = strings.TrimSpace(reflect.ValueOf(ve.Value()).String())
			}

			result = append(result, ValidationError{
				Key:     fieldName,
				Message: tag,
				Value:   value,
			})
		}
		return result
	}

	// Fallback for other error types
	return []ValidationError{{
		Key:     "unknown",
		Message: err.Error(),
		Value:   "",
	}}
}

// ValidationBadRequest creates a 400 Bad Request response with structured validation error
func ValidationBadRequest(err error) Response {
	validationError := ParseValidationError(err)
	return BadRequest("validation failed", validationError)
}

// ValidationBadRequestMultiple creates a 400 Bad Request response with multiple validation errors
func ValidationBadRequestMultiple(err error) Response {
	validationErrors := ParseValidationErrors(err)
	return BadRequest("validation failed", ErrorData{
		Type:   "validation_error",
		Errors: validationErrors,
	})
}

// NormalizeLogtoError converts various Logto error formats to our standard format
func NormalizeLogtoError(logtoError interface{}) ErrorData {
	errorData := ErrorData{
		Type: "external_api_error",
	}

	switch err := logtoError.(type) {
	case map[string]interface{}:
		// Handle Logto simple errors (e.g., user.username_already_in_use)
		if code, exists := err["code"].(string); exists {
			errorData.Errors = []ValidationError{{
				Key:     MapLogtoCodeToField(code),
				Message: MapLogtoCodeToMessage(code),
				Value:   "",
			}}
			return errorData
		}

		// Handle Logto Zod validation errors (guard.invalid_input)
		if data, exists := err["data"].(map[string]interface{}); exists {
			if issues, exists := data["issues"].([]interface{}); exists {
				for _, issue := range issues {
					if issueMap, ok := issue.(map[string]interface{}); ok {
						field := "unknown"
						if path, exists := issueMap["path"].([]interface{}); exists && len(path) > 0 {
							if fieldName, ok := path[0].(string); ok {
								field = MapLogtoFieldToOurs(fieldName)
							}
						}

						code := "invalid"
						if codeVal, exists := issueMap["code"].(string); exists {
							code = codeVal
						}

						errorData.Errors = append(errorData.Errors, ValidationError{
							Key:     field,
							Message: code,
							Value:   "",
						})
					}
				}
				return errorData
			}
		}

		// Fallback for unknown Logto error structure
		errorData.Details = err
		return errorData

	default:
		// Handle string errors or other types
		errorData.Details = logtoError
		return errorData
	}
}

// ValidationFailed creates a standardized validation error response
func ValidationFailed(message string, errors []ValidationError) Response {
	return BadRequest(message, ErrorData{
		Type:   "validation_error",
		Errors: errors,
	})
}

// ExternalAPIError creates a standardized external API error response
func ExternalAPIError(statusCode int, message string, logtoError interface{}) Response {
	return Error(statusCode, message, NormalizeLogtoError(logtoError))
}
