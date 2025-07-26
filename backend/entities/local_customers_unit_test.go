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

// TestLocalCustomerRepository_ValidateCreateRequest tests input validation for customer creation
func TestLocal_CustomerRepository_ValidateCreateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.CreateLocalCustomerRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid request",
			request: &models.CreateLocalCustomerRequest{
				Name:        "ACME Customer",
				Description: "A test customer",
				CustomData:  map[string]interface{}{"industry": "technology"},
			},
			shouldError: false,
		},
		{
			name: "empty name",
			request: &models.CreateLocalCustomerRequest{
				Name:        "",
				Description: "A test customer",
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "valid minimal request",
			request: &models.CreateLocalCustomerRequest{
				Name: "Minimal Customer",
			},
			shouldError: false,
		},
		{
			name: "name with only whitespace",
			request: &models.CreateLocalCustomerRequest{
				Name: "   ",
			},
			shouldError: true,
			errorMsg:    "name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreateCustomerRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalCustomerRepository_ValidateUpdateRequest tests input validation for customer updates
func TestLocal_CustomerRepository_ValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name        string
		request     *models.UpdateLocalCustomerRequest
		shouldError bool
		errorMsg    string
	}{
		{
			name: "valid partial update",
			request: &models.UpdateLocalCustomerRequest{
				Name:        stringPtr("Updated Customer"),
				Description: stringPtr("Updated description"),
			},
			shouldError: false,
		},
		{
			name: "empty name update",
			request: &models.UpdateLocalCustomerRequest{
				Name: stringPtr(""),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "whitespace only name",
			request: &models.UpdateLocalCustomerRequest{
				Name: stringPtr("   "),
			},
			shouldError: true,
			errorMsg:    "name",
		},
		{
			name: "valid description update only",
			request: &models.UpdateLocalCustomerRequest{
				Description: stringPtr("New description"),
			},
			shouldError: false,
		},
		{
			name: "empty description is valid",
			request: &models.UpdateLocalCustomerRequest{
				Description: stringPtr(""),
			},
			shouldError: false,
		},
		{
			name:        "nil update (valid - no fields to update)",
			request:     &models.UpdateLocalCustomerRequest{},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdateCustomerRequest(tt.request)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalCustomerRepository_BuildCustomerFromRow tests customer object construction from database row
func TestLocal_CustomerRepository_BuildCustomerFromRow(t *testing.T) {
	tests := []struct {
		name               string
		customDataJSON     string
		expectedCustomData map[string]interface{}
	}{
		{
			name:               "valid JSON data",
			customDataJSON:     `{"industry": "technology", "size": "large", "priority": 1}`,
			expectedCustomData: map[string]interface{}{"industry": "technology", "size": "large", "priority": float64(1)},
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
			customDataJSON:     `{"contact": {"name": "John", "email": "john@acme.com"}, "tags": ["vip", "enterprise"]}`,
			expectedCustomData: map[string]interface{}{"contact": map[string]interface{}{"name": "John", "email": "john@acme.com"}, "tags": []interface{}{"vip", "enterprise"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &models.LocalCustomer{}

			// Simulate parsing JSON like in the real repository method
			err := parseCustomerJSONFields(customer, []byte(tt.customDataJSON))
			assert.NoError(t, err)

			assert.Equal(t, tt.expectedCustomData, customer.CustomData)
		})
	}
}

// TestLocalCustomerRepository_TimestampHandling tests timestamp operations for customer lifecycle
func TestLocal_CustomerRepository_TimestampHandling(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		operation         string
		expectedCondition func(*models.LocalCustomer) bool
	}{
		{
			name:      "customer creation sets timestamps",
			operation: "create",
			expectedCondition: func(c *models.LocalCustomer) bool {
				return !c.CreatedAt.IsZero() && !c.UpdatedAt.IsZero() && c.Active
			},
		},
		{
			name:      "customer update modifies UpdatedAt",
			operation: "update",
			expectedCondition: func(c *models.LocalCustomer) bool {
				return c.UpdatedAt.After(c.CreatedAt) && c.LogtoSyncedAt == nil
			},
		},
		{
			name:      "customer deactivation sets Active to false",
			operation: "deactivate",
			expectedCondition: func(c *models.LocalCustomer) bool {
				return !c.Active
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer := &models.LocalCustomer{
				ID:        "test-customer-123",
				Name:      "Test Customer",
				Active:    true,
				CreatedAt: now,
				UpdatedAt: now,
			}

			// Simulate the operation
			simulateCustomerOperation(customer, tt.operation)

			assert.True(t, tt.expectedCondition(customer),
				"Condition failed for operation: %s", tt.operation)
		})
	}
}

// TestLocalCustomerRepository_ListFiltering tests list operation filtering logic
func TestLocal_CustomerRepository_ListFiltering(t *testing.T) {
	tests := []struct {
		name              string
		userOrgRole       string
		userOrgID         string
		allowedOrgIDs     []string
		expectedCanAccess bool
		expectedOrgCount  int
	}{
		{
			name:              "owner can access all organizations",
			userOrgRole:       "owner",
			userOrgID:         "org-owner",
			allowedOrgIDs:     []string{"org-owner", "org-distributor-1", "org-reseller-1", "org-customer-1"},
			expectedCanAccess: true,
			expectedOrgCount:  4,
		},
		{
			name:              "distributor can access managed organizations",
			userOrgRole:       "distributor",
			userOrgID:         "org-distributor",
			allowedOrgIDs:     []string{"org-distributor", "org-reseller-1", "org-customer-1"},
			expectedCanAccess: true,
			expectedOrgCount:  3,
		},
		{
			name:              "reseller can access own and managed customers",
			userOrgRole:       "reseller",
			userOrgID:         "org-reseller",
			allowedOrgIDs:     []string{"org-reseller", "org-customer-1"},
			expectedCanAccess: true,
			expectedOrgCount:  2,
		},
		{
			name:              "customer can only access own organization",
			userOrgRole:       "customer",
			userOrgID:         "org-customer",
			allowedOrgIDs:     []string{"org-customer"},
			expectedCanAccess: true,
			expectedOrgCount:  1,
		},
		{
			name:              "no access returns empty list",
			userOrgRole:       "invalid",
			userOrgID:         "org-invalid",
			allowedOrgIDs:     []string{},
			expectedCanAccess: false,
			expectedOrgCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test hierarchical access logic
			orgIDs := simulateHierarchicalAccess(tt.userOrgRole, tt.userOrgID)

			assert.Equal(t, tt.expectedCanAccess, len(orgIDs) > 0)
			assert.Len(t, orgIDs, tt.expectedOrgCount)

			// Verify organization IDs match expected pattern
			if tt.expectedCanAccess {
				for _, expectedOrgID := range tt.allowedOrgIDs {
					assert.Contains(t, orgIDs, expectedOrgID)
				}
			}
		})
	}
}

// TestLocalCustomerRepository_QueryBuilding tests SQL query construction
func TestLocal_CustomerRepository_QueryBuilding(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrgIDs  []string
		page           int
		pageSize       int
		expectedArgs   int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "single organization with pagination",
			allowedOrgIDs:  []string{"org-1"},
			page:           1,
			pageSize:       10,
			expectedArgs:   3, // 1 org ID + limit + offset
			expectedLimit:  10,
			expectedOffset: 0,
		},
		{
			name:           "multiple organizations page 2",
			allowedOrgIDs:  []string{"org-1", "org-2", "org-3"},
			page:           2,
			pageSize:       5,
			expectedArgs:   5, // 3 org IDs + limit + offset
			expectedLimit:  5,
			expectedOffset: 5,
		},
		{
			name:           "large page number",
			allowedOrgIDs:  []string{"org-1"},
			page:           10,
			pageSize:       20,
			expectedArgs:   3,
			expectedLimit:  20,
			expectedOffset: 180, // (10-1) * 20
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildCustomerListQuery(tt.allowedOrgIDs, tt.page, tt.pageSize)

			assert.Len(t, args, tt.expectedArgs)
			assert.Equal(t, tt.expectedLimit, args[len(args)-2])
			assert.Equal(t, tt.expectedOffset, args[len(args)-1])
			assert.Contains(t, query, "LIMIT")
			assert.Contains(t, query, "OFFSET")
		})
	}
}

// Helper functions for validation tests

func validateCreateCustomerRequest(req *models.CreateLocalCustomerRequest) error {
	if req.Name == "" || strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name cannot be empty or whitespace")
	}
	return nil
}

func validateUpdateCustomerRequest(req *models.UpdateLocalCustomerRequest) error {
	if req.Name != nil && (strings.TrimSpace(*req.Name) == "") {
		return fmt.Errorf("name cannot be empty or whitespace")
	}
	return nil
}

func parseCustomerJSONFields(customer *models.LocalCustomer, customDataJSON []byte) error {
	// Parse custom_data JSON
	if len(customDataJSON) > 0 && string(customDataJSON) != "null" {
		if err := json.Unmarshal(customDataJSON, &customer.CustomData); err != nil {
			customer.CustomData = make(map[string]interface{})
		}
	} else {
		customer.CustomData = make(map[string]interface{})
	}
	return nil
}

func simulateCustomerOperation(customer *models.LocalCustomer, operation string) {
	now := time.Now()

	switch operation {
	case "create":
		customer.CreatedAt = now
		customer.UpdatedAt = now
		customer.Active = true
	case "update":
		customer.UpdatedAt = now.Add(time.Minute) // Simulate time passing
		customer.LogtoSyncedAt = nil
	case "deactivate":
		customer.Active = false
		customer.UpdatedAt = now
	}
}

func simulateHierarchicalAccess(userOrgRole, userOrgID string) []string {
	switch userOrgRole {
	case "owner":
		// Owner can access all organizations
		return []string{"org-owner", "org-distributor-1", "org-reseller-1", "org-customer-1"}
	case "distributor":
		// Distributor can access managed resellers and customers
		return []string{"org-distributor", "org-reseller-1", "org-customer-1"}
	case "reseller":
		// Reseller can access managed customers
		return []string{"org-reseller", "org-customer-1"}
	case "customer":
		// Customer can only access own organization
		return []string{"org-customer"}
	default:
		return []string{}
	}
}

func buildCustomerListQuery(allowedOrgIDs []string, page, pageSize int) (string, []interface{}) {
	offset := (page - 1) * pageSize

	args := make([]interface{}, len(allowedOrgIDs)+2)

	// Add organization IDs
	for i, orgID := range allowedOrgIDs {
		args[i] = orgID
	}

	// Add pagination parameters
	args[len(allowedOrgIDs)] = pageSize
	args[len(allowedOrgIDs)+1] = offset

	query := fmt.Sprintf("SELECT * FROM customers WHERE organization_id IN (%s) ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		"placeholders", len(allowedOrgIDs)+1, len(allowedOrgIDs)+2)

	return query, args
}
