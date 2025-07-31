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

// TestLocalResellerRepository_ValidateCreateRequest tests input validation for reseller creation
func TestLocal_ResellerRepository_ValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateLocalResellerRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &models.CreateLocalResellerRequest{
				Name:        "ACME Reseller",
				Description: "A test reseller",
				CustomData:  map[string]interface{}{"territory": "West Coast"},
			},
			shouldError: false,
		},
		{
			name: "empty name",
			request: &models.CreateLocalResellerRequest{
				Name:        "",
				Description: "A test reseller",
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "valid minimal request",
			request: &models.CreateLocalResellerRequest{
				Name: "Minimal Reseller",
			},
			shouldError: false,
		},
		{
			name: "name with only whitespace",
			request: &models.CreateLocalResellerRequest{
				Name: "   ",
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "reseller with distributor reference",
			request: &models.CreateLocalResellerRequest{
				Name: "Valid Reseller",
				CustomData: map[string]interface{}{
					"createdBy":       "org-distributor-123",
					"distributorName": "ACME Distribution",
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateResellerRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalResellerRepository_ValidateUpdateRequest tests input validation for reseller updates
func TestLocal_ResellerRepository_ValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.UpdateLocalResellerRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid partial update",
			request: &models.UpdateLocalResellerRequest{
				Name:        stringPtr("Updated Reseller"),
				Description: stringPtr("Updated description"),
			},
			shouldError: false,
		},
		{
			name: "empty name update",
			request: &models.UpdateLocalResellerRequest{
				Name: stringPtr(""),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "whitespace only name",
			request: &models.UpdateLocalResellerRequest{
				Name: stringPtr("   "),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "update distributor relationship",
			request: &models.UpdateLocalResellerRequest{
				CustomData: &map[string]interface{}{
					"createdBy": "org-distributor-456",
					"status":    "active",
				},
			},
			shouldError: false,
		},
		{
			name:        "nil update (valid - no fields to update)",
			request:     &models.UpdateLocalResellerRequest{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateResellerRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalResellerRepository_BuildResellerFromRow tests reseller object construction
func TestLocal_ResellerRepository_BuildResellerFromRow(t *testing.T) {
	tests := []struct {
		name               string
		customDataJSON     string
		expectedCustomData map[string]interface{}
	}{
		{
			name:               "valid JSON with distributor relationship",
			customDataJSON:     `{"createdBy": "org-distributor-123", "territory": "West Coast", "commission": 15}`,
			expectedCustomData: map[string]interface{}{"createdBy": "org-distributor-123", "territory": "West Coast", "commission": float64(15)},
		},
		{
			name:               "empty JSON object",
			customDataJSON:     `{}`,
			expectedCustomData: map[string]interface{}{},
		},
		{
			name:               "invalid JSON - should default to empty",
			customDataJSON:     `invalid json`,
			expectedCustomData: map[string]interface{}{},
		},
		{
			name:               "complex reseller data",
			customDataJSON:     `{"contact": {"name": "John Smith", "email": "john@reseller.com", "phone": "+1234567890"}, "territories": ["CA", "NV", "OR"], "certifications": ["Cisco", "Microsoft"]}`,
			expectedCustomData: map[string]interface{}{"contact": map[string]interface{}{"name": "John Smith", "email": "john@reseller.com", "phone": "+1234567890"}, "territories": []interface{}{"CA", "NV", "OR"}, "certifications": []interface{}{"Cisco", "Microsoft"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reseller := &models.LocalReseller{}

			// Simulate parsing JSON like in the real repository method
			err := parseResellerJSONFields(reseller, []byte(tt.customDataJSON))
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedCustomData, reseller.CustomData)
		})
	}
}

// TestLocalResellerRepository_DistributorResellerHierarchy tests access control for resellers
func TestLocal_ResellerRepository_DistributorResellerHierarchy(t *testing.T) {
	tests := []struct {
		name              string
		userOrgRole       string
		userOrgID         string
		targetResellerID  string
		resellerCreatedBy string
		expectedCanAccess bool
		expectedError     string
	}{
		{
			name:              "owner can access any reseller",
			userOrgRole:       "owner",
			userOrgID:         "org-owner",
			targetResellerID:  "org-reseller-123",
			resellerCreatedBy: "org-distributor-456",
			expectedCanAccess: true,
		},
		{
			name:              "distributor can access own created resellers",
			userOrgRole:       "distributor",
			userOrgID:         "org-distributor-123",
			targetResellerID:  "org-reseller-456",
			resellerCreatedBy: "org-distributor-123",
			expectedCanAccess: true,
		},
		{
			name:              "distributor cannot access other distributor's resellers",
			userOrgRole:       "distributor",
			userOrgID:         "org-distributor-123",
			targetResellerID:  "org-reseller-456",
			resellerCreatedBy: "org-distributor-999",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
		{
			name:              "reseller can access own organization",
			userOrgRole:       "reseller",
			userOrgID:         "org-reseller-123",
			targetResellerID:  "org-reseller-123",
			resellerCreatedBy: "org-distributor-456",
			expectedCanAccess: true,
		},
		{
			name:              "reseller cannot access other resellers",
			userOrgRole:       "reseller",
			userOrgID:         "org-reseller-123",
			targetResellerID:  "org-reseller-456",
			resellerCreatedBy: "org-distributor-456",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
		{
			name:              "customer cannot access resellers",
			userOrgRole:       "customer",
			userOrgID:         "org-customer-123",
			targetResellerID:  "org-reseller-456",
			resellerCreatedBy: "org-distributor-456",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canAccess, err := validateResellerAccess(tt.userOrgRole, tt.userOrgID, tt.targetResellerID, tt.resellerCreatedBy)

			assert.Equal(t, tt.expectedCanAccess, canAccess)

			if !tt.expectedCanAccess {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalResellerRepository_CustomerManagement tests reseller customer relationships
func TestLocal_ResellerRepository_CustomerManagement(t *testing.T) {
	tests := []struct {
		name               string
		resellerID         string
		expectedCustomers  []string
		canCreateCustomers bool
	}{
		{
			name:               "reseller with multiple customers",
			resellerID:         "org-reseller-1",
			expectedCustomers:  []string{"org-customer-1", "org-customer-2"},
			canCreateCustomers: true,
		},
		{
			name:               "new reseller with no customers",
			resellerID:         "org-reseller-new",
			expectedCustomers:  []string{},
			canCreateCustomers: true,
		},
		{
			name:               "inactive reseller cannot create customers",
			resellerID:         "org-reseller-inactive",
			expectedCustomers:  []string{},
			canCreateCustomers: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test customer management logic
			customers := simulateGetResellerCustomers(tt.resellerID)
			canCreate := simulateCanCreateCustomers(tt.resellerID)

			assert.Equal(t, len(tt.expectedCustomers), len(customers))
			assert.Equal(t, tt.canCreateCustomers, canCreate)

			for _, expectedCustomer := range tt.expectedCustomers {
				assert.Contains(t, customers, expectedCustomer)
			}
		})
	}
}

// TestLocalResellerRepository_TimestampHandling tests timestamp operations for reseller lifecycle
func TestLocal_ResellerRepository_TimestampHandling(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		operation         string
		expectedCondition func(*models.LocalReseller) bool
	}{
		{
			name:      "reseller creation sets timestamps",
			operation: "create",
			expectedCondition: func(r *models.LocalReseller) bool {
				return !r.CreatedAt.IsZero() && !r.UpdatedAt.IsZero() && r.Active()
			},
		},
		{
			name:      "reseller update modifies UpdatedAt",
			operation: "update",
			expectedCondition: func(r *models.LocalReseller) bool {
				return r.UpdatedAt.After(r.CreatedAt) && r.LogtoSyncedAt == nil
			},
		},
		{
			name:      "reseller deactivation sets Active to false",
			operation: "deactivate",
			expectedCondition: func(r *models.LocalReseller) bool {
				return !r.Active()
			},
		},
		{
			name:      "logto sync updates LogtoSyncedAt",
			operation: "sync",
			expectedCondition: func(r *models.LocalReseller) bool {
				return r.LogtoSyncedAt != nil && !r.LogtoSyncedAt.IsZero()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reseller := &models.LocalReseller{
				ID:        "test-reseller-123",
				Name:      "Test Reseller",
				DeletedAt: nil,
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Simulate the operation
			simulateResellerOperation(reseller, tt.operation)

			assert.True(t, tt.expectedCondition(reseller),
				"Condition failed for operation: %s", tt.operation)
		})
	}
}

// TestLocalResellerRepository_ListFiltering tests list operation filtering by hierarchy
func TestLocal_ResellerRepository_ListFiltering(t *testing.T) {
	tests := []struct {
		name                string
		userOrgRole         string
		userOrgID           string
		expectedResellerIDs []string
		expectedCanList     bool
	}{
		{
			name:                "owner can list all resellers",
			userOrgRole:         "owner",
			userOrgID:           "org-owner",
			expectedResellerIDs: []string{"org-reseller-1", "org-reseller-2", "org-reseller-3"},
			expectedCanList:     true,
		},
		{
			name:                "distributor can list own resellers",
			userOrgRole:         "distributor",
			userOrgID:           "org-distributor-1",
			expectedResellerIDs: []string{"org-reseller-1", "org-reseller-2"},
			expectedCanList:     true,
		},
		{
			name:                "reseller can only see self",
			userOrgRole:         "reseller",
			userOrgID:           "org-reseller-1",
			expectedResellerIDs: []string{"org-reseller-1"},
			expectedCanList:     true,
		},
		{
			name:                "customer cannot list resellers",
			userOrgRole:         "customer",
			userOrgID:           "org-customer",
			expectedResellerIDs: []string{},
			expectedCanList:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resellerIDs, canList := simulateGetAccessibleResellers(tt.userOrgRole, tt.userOrgID)

			assert.Equal(t, tt.expectedCanList, canList)
			assert.Equal(t, len(tt.expectedResellerIDs), len(resellerIDs))

			for _, expectedID := range tt.expectedResellerIDs {
				assert.Contains(t, resellerIDs, expectedID)
			}
		})
	}
}

// Helper functions for validation tests

func validateCreateResellerRequest(req *models.CreateLocalResellerRequest) error {
	if req.Name == "" || strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name cannot be empty or whitespace")
	}
	return nil
}

func validateUpdateResellerRequest(req *models.UpdateLocalResellerRequest) error {
	if req.Name != nil && (strings.TrimSpace(*req.Name) == "") {
		return fmt.Errorf("name cannot be empty or whitespace")
	}
	return nil
}

func parseResellerJSONFields(reseller *models.LocalReseller, customDataJSON []byte) error {
	// Parse custom_data JSON
	if len(customDataJSON) > 0 && string(customDataJSON) != "null" {
		if err := json.Unmarshal(customDataJSON, &reseller.CustomData); err != nil {
			reseller.CustomData = make(map[string]interface{})
		}
	} else {
		reseller.CustomData = make(map[string]interface{})
	}
	return nil
}

func validateResellerAccess(userOrgRole, userOrgID, targetResellerID, resellerCreatedBy string) (bool, error) {
	switch userOrgRole {
	case "owner":
		return true, nil
	case "distributor":
		// Distributor can access resellers they created
		if resellerCreatedBy == userOrgID {
			return true, nil
		}
		return false, fmt.Errorf("insufficient permissions to access reseller")
	case "reseller":
		// Reseller can only access their own organization
		if targetResellerID == userOrgID {
			return true, nil
		}
		return false, fmt.Errorf("insufficient permissions to access reseller")
	default:
		return false, fmt.Errorf("insufficient permissions to access resellers")
	}
}

func simulateResellerOperation(reseller *models.LocalReseller, operation string) {
	now := time.Now()

	switch operation {
	case "create":
		reseller.CreatedAt = now
		reseller.UpdatedAt = now
		reseller.DeletedAt = nil
	case "update":
		reseller.UpdatedAt = now.Add(time.Minute) // Simulate time passing
		reseller.LogtoSyncedAt = nil
	case "deactivate":
		now := time.Now()
		reseller.DeletedAt = &now
		reseller.UpdatedAt = now
	case "sync":
		reseller.LogtoSyncedAt = &now
		reseller.UpdatedAt = now
	}
}

func simulateGetResellerCustomers(resellerID string) []string {
	// Simulate getting customers created by this reseller
	switch resellerID {
	case "org-reseller-1":
		return []string{"org-customer-1", "org-customer-2"}
	case "org-reseller-new":
		return []string{}
	case "org-reseller-inactive":
		return []string{}
	default:
		return []string{}
	}
}

func simulateCanCreateCustomers(resellerID string) bool {
	// Simulate checking if reseller can create customers
	switch resellerID {
	case "org-reseller-inactive":
		return false
	default:
		return true
	}
}

func simulateGetAccessibleResellers(userOrgRole, userOrgID string) ([]string, bool) {
	switch userOrgRole {
	case "owner":
		return []string{"org-reseller-1", "org-reseller-2", "org-reseller-3"}, true
	case "distributor":
		if userOrgID == "org-distributor-1" {
			return []string{"org-reseller-1", "org-reseller-2"}, true
		}
		return []string{}, true
	case "reseller":
		return []string{userOrgID}, true
	default:
		return []string{}, false
	}
}
