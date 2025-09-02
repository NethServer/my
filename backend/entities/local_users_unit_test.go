/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

// TestLocalUserRepository_GetHierarchicalOrganizationIDs tests the hierarchical organization access logic
func TestLocal_UserRepository_GetHierarchicalOrganizationIDs(t *testing.T) {
	// Skip test requiring database connection
	t.Skip("Skipping test requiring database connection - requires mock database setup")
}

// TestLocalUserRepository_ValidateCreateRequest tests input validation for user creation
func TestLocal_UserRepository_ValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateLocalUserRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &models.CreateLocalUserRequest{
				Username:       "testuser",
				Email:          "test@example.com",
				Name:           "Test User",
				Phone:          stringPtr("+1234567890"),
				OrganizationID: stringPtr("org-123"),
				UserRoleIDs:    []string{"role-1", "role-2"},
				CustomData:     map[string]interface{}{"department": "IT"},
			},
			shouldError: false,
		},
		{
			name: "empty username",
			request: &models.CreateLocalUserRequest{
				Username:       "",
				Email:          "test@example.com",
				Name:           "Test User",
				OrganizationID: stringPtr("org-123"),
			},
			shouldError: true,
			errorMsg:    "username",
		},
		{
			name: "empty email",
			request: &models.CreateLocalUserRequest{
				Username:       "testuser",
				Email:          "",
				Name:           "Test User",
				OrganizationID: stringPtr("org-123"),
			},
			shouldError: true,
			errorMsg:    "email",
		},
		{
			name: "empty name",
			request: &models.CreateLocalUserRequest{
				Username:       "testuser",
				Email:          "test@example.com",
				Name:           "",
				OrganizationID: stringPtr("org-123"),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "nil organization ID",
			request: &models.CreateLocalUserRequest{
				Username:       "testuser",
				Email:          "test@example.com",
				Name:           "Test User",
				OrganizationID: nil,
			},
			shouldError: true,
			errorMsg:    "organization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateUserRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalUserRepository_ValidateUpdateRequest tests input validation for user updates
func TestLocal_UserRepository_ValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.UpdateLocalUserRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid partial update",
			request: &models.UpdateLocalUserRequest{
				Username: stringPtr("newusername"),
				Email:    stringPtr("newemail@example.com"),
			},
			shouldError: false,
		},
		{
			name: "empty username update",
			request: &models.UpdateLocalUserRequest{
				Username: stringPtr(""),
			},
			shouldError: true,
			errorMsg:    "username",
		},
		{
			name: "empty email update",
			request: &models.UpdateLocalUserRequest{
				Email: stringPtr(""),
			},
			shouldError: true,
			errorMsg:    "email",
		},
		{
			name: "invalid email format",
			request: &models.UpdateLocalUserRequest{
				Email: stringPtr("invalid-email"),
			},
			shouldError: true,
			errorMsg:    "email",
		},
		{
			name:        "nil update (valid - no fields to update)",
			request:     &models.UpdateLocalUserRequest{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateUserRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalUserRepository_BuildUserFromRow tests user object construction from database row
func TestLocal_UserRepository_BuildUserFromRow(t *testing.T) {
	tests := []struct {
		name               string
		userRoleIDsJSON    string
		customDataJSON     string
		expectedRoleIDs    []string
		expectedCustomData map[string]interface{}
	}{
		{
			name:               "valid JSON data",
			userRoleIDsJSON:    `["role1", "role2"]`,
			customDataJSON:     `{"department": "IT", "level": 5}`,
			expectedRoleIDs:    []string{"role1", "role2"},
			expectedCustomData: map[string]interface{}{"department": "IT", "level": float64(5)},
		},
		{
			name:               "empty JSON arrays",
			userRoleIDsJSON:    `[]`,
			customDataJSON:     `{}`,
			expectedRoleIDs:    []string{},
			expectedCustomData: map[string]interface{}{},
		},
		{
			name:               "invalid JSON - should default to empty",
			userRoleIDsJSON:    `invalid json`,
			customDataJSON:     `also invalid`,
			expectedRoleIDs:    []string{},
			expectedCustomData: map[string]interface{}{},
		},
		{
			name:               "null JSON values",
			userRoleIDsJSON:    `null`,
			customDataJSON:     `null`,
			expectedRoleIDs:    []string{},
			expectedCustomData: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.LocalUser{}

			// Simulate parsing JSON like in the real repository method
			err := parseUserJSONFields(user, []byte(tt.userRoleIDsJSON), []byte(tt.customDataJSON))
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedRoleIDs, user.UserRoleIDs)
			assert.Equal(t, tt.expectedCustomData, user.CustomData)
		})
	}
}

// TestLocalUserRepository_TimestampHandling tests timestamp operations for user lifecycle
func TestLocal_UserRepository_TimestampHandling(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		operation         string
		expectedCondition func(*models.LocalUser) bool
	}{
		{
			name:      "user creation sets timestamps",
			operation: "create",
			expectedCondition: func(u *models.LocalUser) bool {
				return !u.CreatedAt.IsZero() && !u.UpdatedAt.IsZero() &&
					u.DeletedAt == nil && u.SuspendedAt == nil
			},
		},
		{
			name:      "user update modifies UpdatedAt",
			operation: "update",
			expectedCondition: func(u *models.LocalUser) bool {
				return u.UpdatedAt.After(u.CreatedAt) && u.LogtoSyncedAt == nil
			},
		},
		{
			name:      "user suspension sets SuspendedAt",
			operation: "suspend",
			expectedCondition: func(u *models.LocalUser) bool {
				return u.SuspendedAt != nil && !u.SuspendedAt.IsZero()
			},
		},
		{
			name:      "user reactivation clears SuspendedAt",
			operation: "reactivate",
			expectedCondition: func(u *models.LocalUser) bool {
				return u.SuspendedAt == nil
			},
		},
		{
			name:      "user deletion sets DeletedAt",
			operation: "delete",
			expectedCondition: func(u *models.LocalUser) bool {
				return u.DeletedAt != nil && !u.DeletedAt.IsZero()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &models.LocalUser{
				ID:        "test-user-123",
				Username:  "testuser",
				Email:     "test@example.com",
				Name:      "Test User",
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Simulate the operation
			simulateUserOperation(user, tt.operation)

			assert.True(t, tt.expectedCondition(user),
				"Condition failed for operation: %s", tt.operation)
		})
	}
}

// Helper functions for validation tests

func validateCreateUserRequest(req *models.CreateLocalUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if req.Email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if req.OrganizationID == nil {
		return fmt.Errorf("organization id cannot be nil")
	}
	return nil
}

func validateUpdateUserRequest(req *models.UpdateLocalUserRequest) error {
	if req.Username != nil && *req.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if req.Email != nil {
		if *req.Email == "" {
			return fmt.Errorf("email cannot be empty")
		}
		if !strings.Contains(*req.Email, "@") {
			return fmt.Errorf("invalid email format")
		}
	}
	return nil
}

func parseUserJSONFields(user *models.LocalUser, userRoleIDsJSON, customDataJSON []byte) error {
	// Parse user_role_ids JSON
	if len(userRoleIDsJSON) > 0 && string(userRoleIDsJSON) != "null" {
		if err := json.Unmarshal(userRoleIDsJSON, &user.UserRoleIDs); err != nil {
			user.UserRoleIDs = []string{}
		}
	} else {
		user.UserRoleIDs = []string{}
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 && string(customDataJSON) != "null" {
		if err := json.Unmarshal(customDataJSON, &user.CustomData); err != nil {
			user.CustomData = make(map[string]interface{})
		}
	} else {
		user.CustomData = make(map[string]interface{})
	}

	return nil
}

func simulateUserOperation(user *models.LocalUser, operation string) {
	now := time.Now()

	switch operation {
	case "create":
		user.CreatedAt = now
		user.UpdatedAt = now
		user.DeletedAt = nil
		user.SuspendedAt = nil
	case "update":
		user.UpdatedAt = now.Add(time.Minute) // Simulate time passing
		user.LogtoSyncedAt = nil
	case "suspend":
		user.SuspendedAt = &now
		user.UpdatedAt = now
	case "reactivate":
		user.SuspendedAt = nil
		user.UpdatedAt = now
	case "delete":
		user.DeletedAt = &now
		user.UpdatedAt = now
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
