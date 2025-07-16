/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLogtoErrorMappings(t *testing.T) {
	mappings := GetLogtoErrorMappings()

	// Test that mappings are not empty
	assert.NotEmpty(t, mappings.CodeToField)
	assert.NotEmpty(t, mappings.CodeToMessage)
	assert.NotEmpty(t, mappings.FieldMapping)

	// Test specific mappings
	assert.Equal(t, "username", mappings.CodeToField["user.username_already_in_use"])
	assert.Equal(t, "email", mappings.CodeToField["user.email_already_in_use"])
	assert.Equal(t, "phone", mappings.CodeToField["user.phone_already_in_use"])

	// Test message mappings
	assert.Equal(t, "already_exists", mappings.CodeToMessage["user.username_already_in_use"])
	assert.Equal(t, "invalid_format", mappings.CodeToMessage["user.invalid_email"])

	// Test field mappings
	assert.Equal(t, "email", mappings.FieldMapping["primaryEmail"])
	assert.Equal(t, "phone", mappings.FieldMapping["primaryPhone"])
}

func TestMapLogtoCodeToField(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "Username already in use",
			code:     "user.username_already_in_use",
			expected: "username",
		},
		{
			name:     "Email already in use",
			code:     "user.email_already_in_use",
			expected: "email",
		},
		{
			name:     "Phone already in use",
			code:     "user.phone_already_in_use",
			expected: "phone",
		},
		{
			name:     "Invalid email",
			code:     "user.invalid_email",
			expected: "email",
		},
		{
			name:     "Invalid phone",
			code:     "user.invalid_phone",
			expected: "phone",
		},
		{
			name:     "Organization require membership",
			code:     "organization.require_membership",
			expected: "organizationId",
		},
		{
			name:     "Organization role names not found",
			code:     "organization.role_names_not_found",
			expected: "userRoleIds",
		},
		{
			name:     "Unknown code returns general",
			code:     "unknown.error.code",
			expected: "general",
		},
		{
			name:     "Empty code returns general",
			code:     "",
			expected: "general",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapLogtoCodeToField(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapLogtoCodeToMessage(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "Username already in use",
			code:     "user.username_already_in_use",
			expected: "already_exists",
		},
		{
			name:     "Email already in use",
			code:     "user.email_already_in_use",
			expected: "already_exists",
		},
		{
			name:     "Phone already in use",
			code:     "user.phone_already_in_use",
			expected: "already_exists",
		},
		{
			name:     "Invalid email",
			code:     "user.invalid_email",
			expected: "invalid_format",
		},
		{
			name:     "Invalid phone",
			code:     "user.invalid_phone",
			expected: "invalid_format",
		},
		{
			name:     "Email not exist",
			code:     "user.email_not_exist",
			expected: "not_found",
		},
		{
			name:     "Phone not exist",
			code:     "user.phone_not_exist",
			expected: "not_found",
		},
		{
			name:     "Identity not exist",
			code:     "user.identity_not_exist",
			expected: "not_found",
		},
		{
			name:     "Identity already in use",
			code:     "user.identity_already_in_use",
			expected: "already_exists",
		},
		{
			name:     "Social account exists in profile",
			code:     "user.social_account_exists_in_profile",
			expected: "already_exists",
		},
		{
			name:     "Organization require membership",
			code:     "organization.require_membership",
			expected: "access_denied",
		},
		{
			name:     "Organization role names not found",
			code:     "organization.role_names_not_found",
			expected: "not_found",
		},
		{
			name:     "Unknown code returns error",
			code:     "unknown.error.code",
			expected: "error",
		},
		{
			name:     "Empty code returns error",
			code:     "",
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapLogtoCodeToMessage(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapLogtoFieldToOurs(t *testing.T) {
	tests := []struct {
		name       string
		logtoField string
		expected   string
	}{
		{
			name:       "Primary email maps to email",
			logtoField: "primaryEmail",
			expected:   "email",
		},
		{
			name:       "Primary phone maps to phone",
			logtoField: "primaryPhone",
			expected:   "phone",
		},
		{
			name:       "Username maps to username",
			logtoField: "username",
			expected:   "username",
		},
		{
			name:       "Password maps to password",
			logtoField: "password",
			expected:   "password",
		},
		{
			name:       "Name maps to name",
			logtoField: "name",
			expected:   "name",
		},
		{
			name:       "Unknown field returns as-is",
			logtoField: "unknownField",
			expected:   "unknownField",
		},
		{
			name:       "Empty field returns as-is",
			logtoField: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapLogtoFieldToOurs(tt.logtoField)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNormalizeLogtoError(t *testing.T) {
	tests := []struct {
		name           string
		logtoError     interface{}
		expectedType   string
		expectedErrors []ValidationError
		hasDetails     bool
	}{
		{
			name: "Simple Logto error code",
			logtoError: map[string]interface{}{
				"code": "user.username_already_in_use",
			},
			expectedType: "validation_error",
			expectedErrors: []ValidationError{
				{Key: "username", Message: "already_exists", Value: ""},
			},
			hasDetails: false,
		},
		{
			name: "Logto Zod validation error",
			logtoError: map[string]interface{}{
				"data": map[string]interface{}{
					"issues": []interface{}{
						map[string]interface{}{
							"path": []interface{}{"email"},
							"code": "invalid_email",
						},
					},
				},
			},
			expectedType: "validation_error",
			expectedErrors: []ValidationError{
				{Key: "email", Message: "invalid_email", Value: ""},
			},
			hasDetails: false,
		},
		{
			name:         "Unknown error structure",
			logtoError:   map[string]interface{}{"unknown": "format"},
			expectedType: "external_api_error",
			hasDetails:   true,
		},
		{
			name:         "String error",
			logtoError:   "simple string error",
			expectedType: "external_api_error",
			hasDetails:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeLogtoError(tt.logtoError)

			assert.Equal(t, tt.expectedType, result.Type)

			if tt.expectedErrors != nil {
				assert.Equal(t, tt.expectedErrors, result.Errors)
			}

			if tt.hasDetails {
				assert.NotNil(t, result.Details)
			} else {
				assert.Nil(t, result.Details)
			}
		})
	}
}

func TestNormalizeLogtoErrorZodMultipleIssues(t *testing.T) {
	logtoError := map[string]interface{}{
		"data": map[string]interface{}{
			"issues": []interface{}{
				map[string]interface{}{
					"path": []interface{}{"email"},
					"code": "invalid_email",
				},
				map[string]interface{}{
					"path": []interface{}{"username"},
					"code": "required",
				},
			},
		},
	}

	result := NormalizeLogtoError(logtoError)

	assert.Equal(t, "validation_error", result.Type)
	assert.Len(t, result.Errors, 2)

	// Check that both errors are present
	var emailError, usernameError *ValidationError
	for i := range result.Errors {
		if result.Errors[i].Key == "email" {
			emailError = &result.Errors[i]
		}
		if result.Errors[i].Key == "username" {
			usernameError = &result.Errors[i]
		}
	}

	assert.NotNil(t, emailError)
	assert.Equal(t, "invalid_email", emailError.Message)

	assert.NotNil(t, usernameError)
	assert.Equal(t, "required", usernameError.Message)
}

func TestLogtoErrorMappingsStructure(t *testing.T) {
	mappings := GetLogtoErrorMappings()

	// Test that the struct has the expected structure
	assert.IsType(t, map[string]string{}, mappings.CodeToField)
	assert.IsType(t, map[string]string{}, mappings.CodeToMessage)
	assert.IsType(t, map[string]string{}, mappings.FieldMapping)

	// Test that user error codes exist
	userCodes := []string{
		"user.username_already_in_use",
		"user.email_already_in_use",
		"user.phone_already_in_use",
		"user.invalid_email",
		"user.invalid_phone",
		"user.email_not_exist",
		"user.phone_not_exist",
		"user.identity_not_exist",
		"user.identity_already_in_use",
		"user.social_account_exists_in_profile",
	}

	for _, code := range userCodes {
		_, existsInField := mappings.CodeToField[code]
		_, existsInMessage := mappings.CodeToMessage[code]
		assert.True(t, existsInField, "Code %s should exist in CodeToField", code)
		assert.True(t, existsInMessage, "Code %s should exist in CodeToMessage", code)
	}

	// Test that organization error codes exist
	orgCodes := []string{
		"organization.require_membership",
		"organization.role_names_not_found",
	}

	for _, code := range orgCodes {
		_, existsInField := mappings.CodeToField[code]
		_, existsInMessage := mappings.CodeToMessage[code]
		assert.True(t, existsInField, "Code %s should exist in CodeToField", code)
		assert.True(t, existsInMessage, "Code %s should exist in CodeToMessage", code)
	}

	// Test that field mappings exist
	expectedFields := []string{"primaryEmail", "primaryPhone", "username", "password", "name"}
	for _, field := range expectedFields {
		_, exists := mappings.FieldMapping[field]
		assert.True(t, exists, "Field %s should exist in FieldMapping", field)
	}
}
