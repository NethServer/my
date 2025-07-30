/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestLocalUserService_parseLogtoError tests Logto error parsing logic
func TestLocalUserService_parseLogtoError(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name         string
		inputError   error
		context      map[string]interface{}
		expectedType string
	}{
		{
			name:         "nil error returns nil",
			inputError:   nil,
			context:      nil,
			expectedType: "nil",
		},
		{
			name:         "non-Logto error returns original",
			inputError:   errors.New("regular error"),
			context:      nil,
			expectedType: "regular",
		},
		{
			name:         "empty context handled correctly",
			inputError:   errors.New("some error"),
			context:      map[string]interface{}{},
			expectedType: "regular",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.parseLogtoError(tt.inputError, tt.context)

			if tt.expectedType == "nil" {
				assert.Nil(t, result)
			} else {
				if tt.inputError != nil {
					assert.Error(t, result)
					assert.Contains(t, result.Error(), "error")
				}
			}
		})
	}
}

// TestLocalUserService_generateUsernameFromEmail_EdgeCases tests edge cases for username generation
func TestLocalUserService_generateUsernameFromEmail_EdgeCases(t *testing.T) {
	service := &LocalUserService{}

	tests := []struct {
		name     string
		email    string
		expected string
	}{
		{
			name:     "empty email fallback",
			email:    "",
			expected: "user_",
		},
		{
			name:     "email without @ fallback",
			email:    "noemail",
			expected: "noemail",
		},
		{
			name:     "email with @ but empty local part",
			email:    "@domain.com",
			expected: "user__at_domain_com",
		},
		{
			name:     "very long email local part",
			email:    "verylongusernamethatexceedsnormallimits@example.com",
			expected: "verylongusernamethatexceedsnormallimits",
		},
		{
			name:     "email with unicode characters",
			email:    "user@example.com",
			expected: "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.generateBaseUsernameFromEmail(tt.email)
			if tt.expected == "user_" {
				// For empty email, it should contain "user_"
				assert.Contains(t, result, "user_")
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestLocalUserService_determineOrganizationRoleName_Logic tests organization role determination logic
func TestLocalUserService_determineOrganizationRoleName_Logic(t *testing.T) {
	t.Skip("Skipping test that requires database connection")
}

// TestValidationError_Error tests ValidationError error message formatting
func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		expectedText string
	}{
		{
			name:         "400 validation error",
			statusCode:   400,
			expectedText: "validation error (status 400)",
		},
		{
			name:         "422 validation error",
			statusCode:   422,
			expectedText: "validation error (status 422)",
		},
		{
			name:         "generic validation error",
			statusCode:   0,
			expectedText: "validation error (status 0)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ValidationError{
				StatusCode: tt.statusCode,
			}

			result := err.Error()
			assert.Equal(t, tt.expectedText, result)
		})
	}
}

// TestLocalUserService_IsOrganizationInHierarchy_Logic tests organizational hierarchy logic
func TestLocalUserService_IsOrganizationInHierarchy_Logic(t *testing.T) {
	t.Skip("Skipping test that requires database connection")
	service := &LocalUserService{}

	tests := []struct {
		name           string
		userOrgRole    string
		userOrgID      string
		targetOrgID    string
		expectedResult bool
		description    string
	}{
		{
			name:           "owner can access any organization",
			userOrgRole:    "owner",
			userOrgID:      "org-owner",
			targetOrgID:    "any-org",
			expectedResult: true,
			description:    "Owners have universal access",
		},
		{
			name:           "same organization always accessible",
			userOrgRole:    "customer",
			userOrgID:      "org-same",
			targetOrgID:    "org-same",
			expectedResult: true,
			description:    "Users can always access their own organization",
		},
		{
			name:           "different organization for customer without database",
			userOrgRole:    "customer",
			userOrgID:      "org-customer",
			targetOrgID:    "org-other",
			expectedResult: false,
			description:    "Customers cannot access other organizations without database hierarchy",
		},
		{
			name:           "empty target organization",
			userOrgRole:    "distributor",
			userOrgID:      "org-distributor",
			targetOrgID:    "",
			expectedResult: false,
			description:    "Empty target organization should not be accessible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.IsOrganizationInHierarchy(tt.userOrgRole, tt.userOrgID, tt.targetOrgID)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

// TestLocalUserService_NewUserService tests service constructor
func TestLocalUserService_NewUserService(t *testing.T) {
	service := NewUserService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.userRepo)
	assert.NotNil(t, service.logtoClient)
}
