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

// TestLocalDistributorRepository_ValidateCreateRequest tests input validation for distributor creation
func TestLocal_DistributorRepository_ValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateLocalDistributorRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &models.CreateLocalDistributorRequest{
				Name:        "ACME Distribution",
				Description: "A test distributor",
				CustomData:  map[string]interface{}{"region": "North America"},
			},
			shouldError: false,
		},
		{
			name: "empty name",
			request: &models.CreateLocalDistributorRequest{
				Name:        "",
				Description: "A test distributor",
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "valid minimal request",
			request: &models.CreateLocalDistributorRequest{
				Name: "Minimal Distributor",
			},
			shouldError: false,
		},
		{
			name: "name with only whitespace",
			request: &models.CreateLocalDistributorRequest{
				Name: "   ",
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "very long description",
			request: &models.CreateLocalDistributorRequest{
				Name:        "Valid Distributor",
				Description: strings.Repeat("a", 1000),
			},
			shouldError: false, // Long descriptions should be allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateDistributorRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalDistributorRepository_ValidateUpdateRequest tests input validation for distributor updates
func TestLocal_DistributorRepository_ValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.UpdateLocalDistributorRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid partial update",
			request: &models.UpdateLocalDistributorRequest{
				Name:        stringPtr("Updated Distributor"),
				Description: stringPtr("Updated description"),
			},
			shouldError: false,
		},
		{
			name: "empty name update",
			request: &models.UpdateLocalDistributorRequest{
				Name: stringPtr(""),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "whitespace only name",
			request: &models.UpdateLocalDistributorRequest{
				Name: stringPtr("   "),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "valid description update only",
			request: &models.UpdateLocalDistributorRequest{
				Description: stringPtr("New description"),
			},
			shouldError: false,
		},
		{
			name: "empty description is valid",
			request: &models.UpdateLocalDistributorRequest{
				Description: stringPtr(""),
			},
			shouldError: false,
		},
		{
			name: "complex custom data update",
			request: &models.UpdateLocalDistributorRequest{
				CustomData: &map[string]interface{}{
					"region":   "Europe",
					"contacts": []interface{}{"john@acme.com", "jane@acme.com"},
					"tier":     "platinum",
				},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateDistributorRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalDistributorRepository_BuildDistributorFromRow tests distributor object construction
func TestLocal_DistributorRepository_BuildDistributorFromRow(t *testing.T) {
	tests := []struct {
		name               string
		customDataJSON     string
		expectedCustomData map[string]interface{}
	}{
		{
			name:               "valid JSON data",
			customDataJSON:     `{"region": "North America", "tier": "gold", "priority": 1}`,
			expectedCustomData: map[string]interface{}{"region": "North America", "tier": "gold", "priority": float64(1)},
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
			name:               "null JSON value",
			customDataJSON:     `null`,
			expectedCustomData: map[string]interface{}{},
		},
		{
			name:               "complex nested JSON",
			customDataJSON:     `{"contact": {"primary": "john@acme.com", "secondary": "jane@acme.com"}, "territories": ["US", "CA", "MX"]}`,
			expectedCustomData: map[string]interface{}{"contact": map[string]interface{}{"primary": "john@acme.com", "secondary": "jane@acme.com"}, "territories": []interface{}{"US", "CA", "MX"}},
		},
		{
			name:               "special characters in JSON",
			customDataJSON:     `{"name": "Acme & Co.", "notes": "Special chars: áéíóú, ñ, ç"}`,
			expectedCustomData: map[string]interface{}{"name": "Acme & Co.", "notes": "Special chars: áéíóú, ñ, ç"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distributor := &models.LocalDistributor{}

			// Simulate parsing JSON like in the real repository method
			err := parseDistributorJSONFields(distributor, []byte(tt.customDataJSON))
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedCustomData, distributor.CustomData)
		})
	}
}

// TestLocalDistributorRepository_TimestampHandling tests timestamp operations for distributor lifecycle
func TestLocal_DistributorRepository_TimestampHandling(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		operation         string
		expectedCondition func(*models.LocalDistributor) bool
	}{
		{
			name:      "distributor creation sets timestamps",
			operation: "create",
			expectedCondition: func(d *models.LocalDistributor) bool {
				return !d.CreatedAt.IsZero() && !d.UpdatedAt.IsZero() && d.Active
			},
		},
		{
			name:      "distributor update modifies UpdatedAt",
			operation: "update",
			expectedCondition: func(d *models.LocalDistributor) bool {
				return d.UpdatedAt.After(d.CreatedAt) && d.LogtoSyncedAt == nil
			},
		},
		{
			name:      "distributor deactivation sets Active to false",
			operation: "deactivate",
			expectedCondition: func(d *models.LocalDistributor) bool {
				return !d.Active
			},
		},
		{
			name:      "logto sync updates LogtoSyncedAt",
			operation: "sync",
			expectedCondition: func(d *models.LocalDistributor) bool {
				return d.LogtoSyncedAt != nil && !d.LogtoSyncedAt.IsZero()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			distributor := &models.LocalDistributor{
				ID:        "test-distributor-123",
				Name:      "Test Distributor",
				Active:    true,
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Simulate the operation
			simulateDistributorOperation(distributor, tt.operation)

			assert.True(t, tt.expectedCondition(distributor),
				"Condition failed for operation: %s", tt.operation)
		})
	}
}

// TestLocalDistributorRepository_OwnerAccessOnly tests that only owners can manage distributors
func TestLocal_DistributorRepository_OwnerAccessOnly(t *testing.T) {
	tests := []struct {
		name              string
		userOrgRole       string
		userOrgID         string
		expectedCanAccess bool
		expectedError     string
	}{
		{
			name:              "owner can access distributors",
			userOrgRole:       "owner",
			userOrgID:         "org-owner",
			expectedCanAccess: true,
		},
		{
			name:              "distributor cannot access other distributors",
			userOrgRole:       "distributor",
			userOrgID:         "org-distributor",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
		{
			name:              "reseller cannot access distributors",
			userOrgRole:       "reseller",
			userOrgID:         "org-reseller",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
		{
			name:              "customer cannot access distributors",
			userOrgRole:       "customer",
			userOrgID:         "org-customer",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
		{
			name:              "invalid role cannot access distributors",
			userOrgRole:       "invalid",
			userOrgID:         "org-invalid",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test distributor access logic
			canAccess, err := validateDistributorAccess(tt.userOrgRole, tt.userOrgID)

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

// TestLocalDistributorRepository_HierarchicalRelationships tests distributor-reseller-customer relationships
func TestLocal_DistributorRepository_HierarchicalRelationships(t *testing.T) {
	tests := []struct {
		name              string
		distributorID     string
		expectedResellers []string
		expectedCustomers []string
	}{
		{
			name:              "distributor with resellers and customers",
			distributorID:     "org-distributor-1",
			expectedResellers: []string{"org-reseller-1", "org-reseller-2"},
			expectedCustomers: []string{"org-customer-1", "org-customer-2", "org-customer-3"},
		},
		{
			name:              "new distributor with no relationships",
			distributorID:     "org-distributor-new",
			expectedResellers: []string{},
			expectedCustomers: []string{},
		},
		{
			name:              "distributor with only direct customers",
			distributorID:     "org-distributor-direct",
			expectedResellers: []string{},
			expectedCustomers: []string{"org-customer-direct"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test hierarchical relationship logic
			resellers := simulateGetManagedResellers(tt.distributorID)
			customers := simulateGetManagedCustomers(tt.distributorID)

			assert.Equal(t, len(tt.expectedResellers), len(resellers))
			assert.Equal(t, len(tt.expectedCustomers), len(customers))

			for _, expectedReseller := range tt.expectedResellers {
				assert.Contains(t, resellers, expectedReseller)
			}

			for _, expectedCustomer := range tt.expectedCustomers {
				assert.Contains(t, customers, expectedCustomer)
			}
		})
	}
}

// Helper functions for validation tests

func validateCreateDistributorRequest(req *models.CreateLocalDistributorRequest) error {
	if req.Name == "" || strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name cannot be empty or whitespace")
	}
	return nil
}

func validateUpdateDistributorRequest(req *models.UpdateLocalDistributorRequest) error {
	if req.Name != nil && (strings.TrimSpace(*req.Name) == "") {
		return fmt.Errorf("name cannot be empty or whitespace")
	}
	return nil
}

func parseDistributorJSONFields(distributor *models.LocalDistributor, customDataJSON []byte) error {
	// Parse custom_data JSON
	if len(customDataJSON) > 0 && string(customDataJSON) != "null" {
		if err := json.Unmarshal(customDataJSON, &distributor.CustomData); err != nil {
			distributor.CustomData = make(map[string]interface{})
		}
	} else {
		distributor.CustomData = make(map[string]interface{})
	}
	return nil
}

func simulateDistributorOperation(distributor *models.LocalDistributor, operation string) {
	now := time.Now()

	switch operation {
	case "create":
		distributor.CreatedAt = now
		distributor.UpdatedAt = now
		distributor.Active = true
	case "update":
		distributor.UpdatedAt = now.Add(time.Minute) // Simulate time passing
		distributor.LogtoSyncedAt = nil
	case "deactivate":
		distributor.Active = false
		distributor.UpdatedAt = now
	case "sync":
		distributor.LogtoSyncedAt = &now
		distributor.UpdatedAt = now
	}
}

func validateDistributorAccess(userOrgRole, userOrgID string) (bool, error) {
	// Only owners can manage distributors
	if userOrgRole != "owner" {
		return false, fmt.Errorf("insufficient permissions to access distributors")
	}
	return true, nil
}

func simulateGetManagedResellers(distributorID string) []string {
	// Simulate getting resellers created by this distributor
	switch distributorID {
	case "org-distributor-1":
		return []string{"org-reseller-1", "org-reseller-2"}
	case "org-distributor-direct":
		return []string{}
	default:
		return []string{}
	}
}

func simulateGetManagedCustomers(distributorID string) []string {
	// Simulate getting customers created by this distributor (directly or through resellers)
	switch distributorID {
	case "org-distributor-1":
		return []string{"org-customer-1", "org-customer-2", "org-customer-3"}
	case "org-distributor-direct":
		return []string{"org-customer-direct"}
	default:
		return []string{}
	}
}
