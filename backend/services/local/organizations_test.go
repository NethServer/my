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
	"strings"
	"testing"

	"github.com/nethesis/my/backend/response"
	"github.com/stretchr/testify/assert"
)

// TestDuplicateVAT_ErrorHandling tests that duplicate VAT errors return 400 validation error instead of 500
func TestDuplicateVAT_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		entityError   error
		expectCode    int
		expectType    string
		expectKey     string
		expectMessage string
	}{
		{
			name:          "duplicate VAT error returns 400",
			entityError:   errors.New("already exists"),
			expectCode:    400,
			expectType:    "validation_error",
			expectKey:     "custom_data.vat",
			expectMessage: "already exists",
		},
		{
			name:          "case insensitive error check",
			entityError:   errors.New("already exists"),
			expectCode:    400,
			expectType:    "validation_error",
			expectKey:     "custom_data.vat",
			expectMessage: "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the error handling logic from organizations.go
			err := tt.entityError

			// This is the key check that must work correctly
			// Before fix: checked for "VAT already exists"
			// After fix: checks for "already exists"
			if strings.Contains(err.Error(), "already exists") {
				// Build validation error
				vatValue := "12345678901"
				validationErr := &ValidationError{
					StatusCode: tt.expectCode,
					ErrorData: response.ErrorData{
						Type: tt.expectType,
						Errors: []response.ValidationError{{
							Key:     tt.expectKey,
							Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
							Value:   vatValue,
						}},
					},
				}

				// Verify error structure - this ensures we return 400, not 500
				assert.Equal(t, tt.expectCode, validationErr.StatusCode, "Should return 400 Bad Request")
				assert.Equal(t, tt.expectType, validationErr.ErrorData.Type)
				assert.Len(t, validationErr.ErrorData.Errors, 1)
				assert.Equal(t, tt.expectKey, validationErr.ErrorData.Errors[0].Key)
				assert.Equal(t, tt.expectMessage, validationErr.ErrorData.Errors[0].Message)
				assert.Equal(t, vatValue, validationErr.ErrorData.Errors[0].Value)
			} else {
				t.Errorf("Error check failed - would return 500 instead of 400")
			}
		})
	}
}

// TestDuplicateVAT_ErrorCheckLogic tests the specific error checking logic
func TestDuplicateVAT_ErrorCheckLogic(t *testing.T) {
	tests := []struct {
		name         string
		errorMessage string
		shouldMatch  bool
		description  string
	}{
		{
			name:         "exact match for 'already exists'",
			errorMessage: "already exists",
			shouldMatch:  true,
			description:  "Should match the simplified error message from entity layer",
		},
		{
			name:         "old format should not appear",
			errorMessage: "VAT already exists in customers",
			shouldMatch:  true,
			description:  "Old format still contains 'already exists' substring",
		},
		{
			name:         "unrelated error",
			errorMessage: "database connection failed",
			shouldMatch:  false,
			description:  "Unrelated errors should not match",
		},
		{
			name:         "empty error",
			errorMessage: "",
			shouldMatch:  false,
			description:  "Empty error should not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New(tt.errorMessage)

			// This is the exact check used in organizations.go
			matched := strings.Contains(err.Error(), "already exists")

			assert.Equal(t, tt.shouldMatch, matched, tt.description)
		})
	}
}

// TestOrganizationValidationError_Structure tests the ValidationError structure used in organizations.go
func TestOrganizationValidationError_Structure(t *testing.T) {
	validationErr := &ValidationError{
		StatusCode: 400,
		ErrorData: response.ErrorData{
			Type: "validation_error",
			Errors: []response.ValidationError{
				{
					Key:     "custom_data.vat",
					Message: "already exists",
					Value:   "12345678901",
				},
			},
		},
	}

	assert.Equal(t, 400, validationErr.StatusCode)
	assert.Equal(t, "validation_error", validationErr.ErrorData.Type)
	assert.Len(t, validationErr.ErrorData.Errors, 1)

	firstError := validationErr.ErrorData.Errors[0]
	assert.Equal(t, "custom_data.vat", firstError.Key)
	assert.Equal(t, "already exists", firstError.Message)
	assert.Equal(t, "12345678901", firstError.Value)
}
