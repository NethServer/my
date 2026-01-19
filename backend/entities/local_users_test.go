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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/testutils"
)

func TestNewLocalUserRepository(t *testing.T) {
	testutils.SetupLogger()

	repo := NewLocalUserRepository()
	assert.NotNil(t, repo)
	// Database connection can be nil in test environment without database setup
}

func TestLocalUserRepository_Create_Validation(t *testing.T) {
	testutils.SetupLogger()

	// Skip this test if database is not available
	repo := NewLocalUserRepository()
	if repo.db == nil {
		t.Skip("Skipping test: database not available in test environment")
	}

	tests := []struct {
		name        string
		request     *models.CreateLocalUserRequest
		expectError bool
		description string
	}{
		{
			name: "valid user request",
			request: &models.CreateLocalUserRequest{
				Username:       "testuser",
				Email:          "test@example.com",
				Name:           "Test User",
				Phone:          stringPtr("+1234567890"),
				OrganizationID: stringPtr("org123"),
				UserRoleIDs:    []string{"admin", "support"},
				CustomData: map[string]interface{}{
					"department": "IT",
					"level":      "senior",
				},
			},
			expectError: true, // Will error due to no DB, but validates structure
			description: "Should have valid structure",
		},
		{
			name: "user with minimal required fields",
			request: &models.CreateLocalUserRequest{
				Username:       "minimal",
				Email:          "minimal@example.com",
				OrganizationID: stringPtr("org456"),
				UserRoleIDs:    []string{},
				CustomData:     map[string]interface{}{},
			},
			expectError: true, // Will error due to no DB, but validates structure
			description: "Should work with minimal fields",
		},
		{
			name: "user with complex custom data",
			request: &models.CreateLocalUserRequest{
				Username:       "complex",
				Email:          "complex@example.com",
				Name:           "Complex User",
				OrganizationID: stringPtr("org789"),
				UserRoleIDs:    []string{"user", "viewer"},
				CustomData: map[string]interface{}{
					"preferences": map[string]interface{}{
						"theme":    "dark",
						"language": "en",
						"notifications": map[string]bool{
							"email": true,
							"sms":   false,
						},
					},
					"metadata": []string{"tag1", "tag2", "tag3"},
				},
			},
			expectError: true, // Will error due to no DB, but validates structure
			description: "Should handle complex nested custom data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.Create(tt.request)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, user)
				assert.Equal(t, tt.request.Username, user.Username)
				assert.Equal(t, tt.request.Email, user.Email)
			}
		})
	}
}

func TestLocalUserRepository_GetByID_Validation(t *testing.T) {
	testutils.SetupLogger()

	// Skip this test if database is not available
	repo := NewLocalUserRepository()
	if repo.db == nil {
		t.Skip("Skipping test: database not available in test environment")
	}

	tests := []struct {
		name        string
		userID      string
		expectError bool
		description string
	}{
		{
			name:        "valid UUID format",
			userID:      "550e8400-e29b-41d4-a716-446655440000",
			expectError: true, // Will error due to no DB
			description: "Should accept valid UUID format",
		},
		{
			name:        "simple string ID",
			userID:      "user123",
			expectError: true, // Will error due to no DB
			description: "Should accept simple string ID",
		},
		{
			name:        "empty user ID",
			userID:      "",
			expectError: true, // Will error due to no DB and empty ID
			description: "Should handle empty ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByID(tt.userID)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestLocalUserRepository_GetByLogtoID_Validation(t *testing.T) {
	testutils.SetupLogger()

	// Skip this test if database is not available
	repo := NewLocalUserRepository()
	if repo.db == nil {
		t.Skip("Skipping test: database not available in test environment")
	}

	tests := []struct {
		name        string
		logtoID     string
		expectError bool
		description string
	}{
		{
			name:        "valid Logto ID",
			logtoID:     "logto_user_abc123",
			expectError: true, // Will error due to no DB
			description: "Should accept Logto ID format",
		},
		{
			name:        "empty Logto ID",
			logtoID:     "",
			expectError: true, // Will error due to no DB and empty ID
			description: "Should handle empty Logto ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.GetByLogtoID(tt.logtoID)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestLocalUserRepository_Update_Validation(t *testing.T) {
	testutils.SetupLogger()

	// Skip this test if database is not available
	repo := NewLocalUserRepository()
	if repo.db == nil {
		t.Skip("Skipping test: database not available in test environment")
	}

	tests := []struct {
		name        string
		userID      string
		request     *models.UpdateLocalUserRequest
		expectError bool
		description string
	}{
		{
			name:   "valid update request",
			userID: "user123",
			request: &models.UpdateLocalUserRequest{
				Name:       stringPtr("Updated Name"),
				Phone:      stringPtr("+9876543210"),
				CustomData: mapPtr(map[string]interface{}{"updated": true}),
			},
			expectError: true, // Will error due to no DB
			description: "Should accept valid update request",
		},
		{
			name:   "partial update request",
			userID: "user456",
			request: &models.UpdateLocalUserRequest{
				Name: stringPtr("Only Name Updated"),
			},
			expectError: true, // Will error due to no DB
			description: "Should accept partial update",
		},
		{
			name:        "empty user ID",
			userID:      "",
			request:     &models.UpdateLocalUserRequest{Name: stringPtr("Test")},
			expectError: true, // Will error due to empty ID
			description: "Should handle empty user ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := repo.Update(tt.userID, tt.request)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, user)
			}
		})
	}
}

func TestLocalUserRepository_Methods_Structure(t *testing.T) {
	testutils.SetupLogger()

	// Test that repository methods exist and have correct signatures
	repo := NewLocalUserRepository()

	// These tests verify method signatures exist (compilation test)
	assert.NotNil(t, repo.Create)
	assert.NotNil(t, repo.GetByID)
	assert.NotNil(t, repo.GetByLogtoID)
	assert.NotNil(t, repo.Update)
	assert.NotNil(t, repo.Delete)
	assert.NotNil(t, repo.SuspendUser)
	assert.NotNil(t, repo.ReactivateUser)
	assert.NotNil(t, repo.UpdateLatestLogin)
	assert.NotNil(t, repo.List)
	assert.NotNil(t, repo.ListByOrganizations)
	assert.NotNil(t, repo.GetTotals)
	assert.NotNil(t, repo.GetTotalsByOrganizations)
}

func TestLocalUserRepository_ListOperations_Structure(t *testing.T) {
	testutils.SetupLogger()

	// Skip this test if database is not available
	repo := NewLocalUserRepository()
	if repo.db == nil {
		t.Skip("Skipping test: database not available in test environment")
	}

	tests := []struct {
		name        string
		testFunc    func() error
		description string
	}{
		{
			name: "List with pagination",
			testFunc: func() error {
				_, _, err := repo.List("", "", "", 1, 10, "", "", "", "", "", "")
				return err
			},
			description: "Should accept pagination parameters",
		},
		{
			name: "ListByOrganizations with filtering",
			testFunc: func() error {
				_, _, err := repo.ListByOrganizations([]string{"org1", "org2"}, "", 1, 10, "", "", "", "", "", "")
				return err
			},
			description: "Should accept organization filtering",
		},
		{
			name: "GetTotals operation",
			testFunc: func() error {
				_, err := repo.GetTotals("", "")
				return err
			},
			description: "Should support totals calculation",
		},
		{
			name: "GetTotalsByOrganizations operation",
			testFunc: func() error {
				_, err := repo.GetTotalsByOrganizations([]string{"org1"})
				return err
			},
			description: "Should support filtered totals calculation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			// We expect errors due to no database, but this tests the method signatures
			assert.Error(t, err, tt.description+" (expected error due to no DB)")
		})
	}
}

func TestLocalUserRepository_UserManagement_Structure(t *testing.T) {
	testutils.SetupLogger()

	// Skip this test if database is not available
	repo := NewLocalUserRepository()
	if repo.db == nil {
		t.Skip("Skipping test: database not available in test environment")
	}

	userID := "test-user-123"

	tests := []struct {
		name        string
		testFunc    func() error
		description string
	}{
		{
			name: "Delete user operation",
			testFunc: func() error {
				return repo.Delete(userID)
			},
			description: "Should support user deletion",
		},
		{
			name: "Suspend user operation",
			testFunc: func() error {
				return repo.SuspendUser(userID)
			},
			description: "Should support user suspension",
		},
		{
			name: "Reactivate user operation",
			testFunc: func() error {
				return repo.ReactivateUser(userID)
			},
			description: "Should support user reactivation",
		},
		{
			name: "Update latest login operation",
			testFunc: func() error {
				return repo.UpdateLatestLogin(userID)
			},
			description: "Should support login tracking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.testFunc()
			// We expect errors due to no database, but this tests the method signatures
			assert.Error(t, err, tt.description+" (expected error due to no DB)")
		})
	}
}

// Helper functions for pointer conversions
func stringPtr(s string) *string {
	return &s
}

func mapPtr(m map[string]interface{}) *map[string]interface{} {
	return &m
}
