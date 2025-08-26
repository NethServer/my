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

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination interface{} `json:"pagination"`
}

// Paginated creates a response with pagination info using entity-specific field names
func Paginated(message string, entityName string, data interface{}, totalCount, page, pageSize int) Response {
	totalPages := (totalCount + pageSize - 1) / pageSize

	pagination := map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total_count": totalCount,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	// Create response data with entity-specific field name
	responseData := map[string]interface{}{
		entityName:   data,
		"pagination": pagination,
	}

	return OK(message, responseData)
}

// PaginatedWithSorting creates a response with pagination and sorting info using entity-specific field names
func PaginatedWithSorting(message string, entityName string, data interface{}, totalCount, page, pageSize int, sortBy, sortDirection string) Response {
	totalPages := (totalCount + pageSize - 1) / pageSize

	pagination := map[string]interface{}{
		"page":        page,
		"page_size":   pageSize,
		"total_count": totalCount,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	// Add sorting fields if provided
	if sortBy != "" {
		pagination["sort_by"] = sortBy
		pagination["sort_direction"] = sortDirection
	}

	// Create response data with entity-specific field name
	responseData := map[string]interface{}{
		entityName:   data,
		"pagination": pagination,
	}

	return OK(message, responseData)
}

// getJSONFieldName extracts the JSON field name from validator.FieldError
func getJSONFieldName(ve validator.FieldError) string {
	// Try to get the JSON tag name from the struct field
	if ve.StructNamespace() != "" && ve.StructField() != "" {
		// Get the struct type and field
		if structType := ve.Type(); structType != nil {
			if structType.Kind() == reflect.Struct {
				if field, found := structType.FieldByName(ve.Field()); found {
					if jsonTag := field.Tag.Get("json"); jsonTag != "" {
						// Parse the JSON tag to extract the field name
						// JSON tags can be like: "fieldname" or "fieldname,omitempty"
						if jsonName := strings.Split(jsonTag, ",")[0]; jsonName != "" && jsonName != "-" {
							return jsonName
						}
					}
				}
			}
		}
	}

	// Fallback to lowercase field name if no JSON tag found
	return strings.ToLower(ve.Field())
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

// PasswordValidationBadRequest creates a 400 Bad Request response for password validation errors
func PasswordValidationBadRequest(errorCodes []string) Response {
	var validationErrors []ValidationError

	for _, code := range errorCodes {
		validationErrors = append(validationErrors, ValidationError{
			Key:     "password",
			Message: code, // Return just the error code, UI will handle translation
		})
	}

	return BadRequest("validation failed", ErrorData{
		Type:   "validation_error",
		Errors: validationErrors,
	})
}

// NormalizeLogtoErrorWithContext converts various Logto error formats to our standard format with context
func NormalizeLogtoErrorWithContext(logtoError interface{}, context map[string]interface{}) ErrorData {
	// Add context to the error for value extraction
	if errorMap, ok := logtoError.(map[string]interface{}); ok {
		for k, v := range context {
			errorMap[k] = v
		}
	}
	return NormalizeLogtoError(logtoError)
}

// NormalizeLogtoError converts various Logto error formats to our standard format
func NormalizeLogtoError(logtoError interface{}) ErrorData {
	errorData := ErrorData{
		Type: "external_api_error",
	}

	switch err := logtoError.(type) {
	case map[string]interface{}:
		// Handle Logto Zod validation errors (guard.invalid_input) - check this first
		if data, exists := err["data"].(map[string]interface{}); exists {
			if issues, exists := data["issues"].([]interface{}); exists {
				errorData.Type = "validation_error"
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

						// Normalize the message to match Gin validation format
						normalizedMessage := code
						switch code {
						case "invalid_string":
							normalizedMessage = field // Use field name as message (like "phone", "email")
						}

						// Try to extract value from the context based on field
						value := ""
						switch field {
						case "phone":
							if phoneValue, exists := err["phone"].(string); exists {
								value = phoneValue
							}
						case "email":
							if emailValue, exists := err["email"].(string); exists {
								value = emailValue
							}
						case "username":
							if usernameValue, exists := err["username"].(string); exists {
								value = usernameValue
							}
						}

						errorData.Errors = append(errorData.Errors, ValidationError{
							Key:     field,
							Message: normalizedMessage,
							Value:   value,
						})
					}
				}
				return errorData
			}
		}

		// Handle Logto simple errors (e.g., user.username_already_in_use)
		if code, exists := err["code"].(string); exists {
			errorData.Type = "validation_error"
			errorData.Errors = []ValidationError{{
				Key:     MapLogtoCodeToField(code),
				Message: MapLogtoCodeToMessage(code),
				Value:   "",
			}}
			return errorData
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

// ExternalAPIErrorWithContext creates a standardized external API error response with additional context
func ExternalAPIErrorWithContext(statusCode int, message string, logtoError interface{}, context map[string]interface{}) Response {
	return Error(statusCode, message, NormalizeLogtoErrorWithContext(logtoError, context))
}
