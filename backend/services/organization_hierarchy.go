/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"fmt"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// OrganizationHierarchyService manages the organization hierarchy cache table
type OrganizationHierarchyService struct {
	logtoClient *LogtoManagementClient
}

// NewOrganizationHierarchyService creates a new organization hierarchy service
func NewOrganizationHierarchyService() *OrganizationHierarchyService {
	return &OrganizationHierarchyService{
		logtoClient: NewLogtoManagementClient(),
	}
}

// SyncOrganizationHierarchy rebuilds the complete organization hierarchy cache
func (s *OrganizationHierarchyService) SyncOrganizationHierarchy() error {
	logger.Info().
		Str("operation", "sync_organization_hierarchy").
		Msg("Starting organization hierarchy synchronization")

	startTime := time.Now()

	// Get all organizations from Logto
	allOrgs, err := s.logtoClient.GetAllOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get organizations from Logto: %w", err)
	}

	// Begin transaction
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// Clear existing hierarchy
	_, err = tx.Exec("DELETE FROM organization_hierarchy")
	if err != nil {
		return fmt.Errorf("failed to clear existing hierarchy: %w", err)
	}

	// Build hierarchy for each organization
	hierarchyCount := 0
	for _, org := range allOrgs {
		orgType := s.getOrganizationType(org)
		if orgType == "" {
			continue // Skip organizations without type
		}

		accessibleCustomers := s.calculateAccessibleCustomers(org, orgType, allOrgs)

		// Insert accessible customers for this organization
		for _, customerID := range accessibleCustomers {
			_, err = tx.Exec(`
				INSERT INTO organization_hierarchy (user_org_id, user_org_role, accessible_customer_id, created_at, updated_at)
				VALUES ($1, $2, $3, NOW(), NOW())
			`, org.ID, orgType, customerID)

			if err != nil {
				return fmt.Errorf("failed to insert hierarchy entry for org %s: %w", org.ID, err)
			}
			hierarchyCount++
		}
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	duration := time.Since(startTime)
	logger.Info().
		Str("operation", "sync_organization_hierarchy").
		Int("organizations_processed", len(allOrgs)).
		Int("hierarchy_entries", hierarchyCount).
		Dur("duration", duration).
		Msg("Organization hierarchy synchronization completed successfully")

	return nil
}

// UpdateOrganizationHierarchy updates hierarchy for specific organization changes
func (s *OrganizationHierarchyService) UpdateOrganizationHierarchy(orgID string) error {
	logger.Info().
		Str("operation", "update_organization_hierarchy").
		Str("org_id", orgID).
		Msg("Updating organization hierarchy for specific organization")

	// For simplicity, do a full sync to ensure consistency
	return s.SyncOrganizationHierarchy()
}

// GetAccessibleCustomerIDs retrieves accessible customer IDs from the hierarchy cache
func (s *OrganizationHierarchyService) GetAccessibleCustomerIDs(userOrgID, userOrgRole string) ([]string, error) {
	query := `
		SELECT accessible_customer_id
		FROM organization_hierarchy
		WHERE user_org_id = $1 AND user_org_role = $2
		ORDER BY accessible_customer_id
	`

	rows, err := database.DB.Query(query, userOrgID, userOrgRole)
	if err != nil {
		return nil, fmt.Errorf("failed to query accessible customer IDs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var customerIDs []string
	for rows.Next() {
		var customerID string
		if err := rows.Scan(&customerID); err != nil {
			return nil, fmt.Errorf("failed to scan customer ID: %w", err)
		}
		customerIDs = append(customerIDs, customerID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer IDs: %w", err)
	}

	logger.Debug().
		Str("operation", "get_accessible_customer_ids").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("customer_count", len(customerIDs)).
		Msg("Retrieved accessible customer IDs from hierarchy cache")

	return customerIDs, nil
}

// calculateAccessibleCustomers calculates which customer organizations are accessible by a given organization
func (s *OrganizationHierarchyService) calculateAccessibleCustomers(org models.LogtoOrganization, orgType string, allOrgs []models.LogtoOrganization) []string {
	var accessibleCustomers []string

	switch orgType {
	case "Owner":
		// Owner can access all customer organizations
		for _, otherOrg := range allOrgs {
			if s.getOrganizationType(otherOrg) == "customer" {
				accessibleCustomers = append(accessibleCustomers, otherOrg.ID)
			}
		}

	case "Distributor":
		// Distributors can access:
		// 1. Customers they created directly
		// 2. Customers created by their resellers

		// First, find all resellers created by this distributor
		var distributorResellers []string
		for _, otherOrg := range allOrgs {
			if s.getOrganizationType(otherOrg) == "reseller" && s.isOrganizationCreatedBy(otherOrg, org.ID) {
				distributorResellers = append(distributorResellers, otherOrg.ID)
			}
		}

		// Now find customers created by this distributor or their resellers
		for _, otherOrg := range allOrgs {
			if s.getOrganizationType(otherOrg) == "customer" {
				if s.isOrganizationCreatedBy(otherOrg, org.ID) {
					// Customer created directly by distributor
					accessibleCustomers = append(accessibleCustomers, otherOrg.ID)
				} else {
					// Check if created by one of distributor's resellers
					for _, resellerID := range distributorResellers {
						if s.isOrganizationCreatedBy(otherOrg, resellerID) {
							accessibleCustomers = append(accessibleCustomers, otherOrg.ID)
							break
						}
					}
				}
			}
		}

	case "Reseller":
		// Resellers can only access customers they created directly
		for _, otherOrg := range allOrgs {
			if s.getOrganizationType(otherOrg) == "customer" && s.isOrganizationCreatedBy(otherOrg, org.ID) {
				accessibleCustomers = append(accessibleCustomers, otherOrg.ID)
			}
		}

	case "Customer":
		// Customers can only access their own organization
		accessibleCustomers = append(accessibleCustomers, org.ID)
	}

	return accessibleCustomers
}

// getOrganizationType extracts the organization type from custom data
func (s *OrganizationHierarchyService) getOrganizationType(org models.LogtoOrganization) string {
	if org.CustomData != nil {
		if orgType, exists := org.CustomData["type"]; exists {
			if typeStr, ok := orgType.(string); ok {
				// Normalize role names to match RBAC expectations
				switch typeStr {
				case "owner":
					return "Owner"
				case "distributor":
					return "Distributor"
				case "reseller":
					return "Reseller"
				case "customer":
					return "Customer"
				}
			}
		}
	}
	return ""
}

// isOrganizationCreatedBy checks if an organization was created by a specific organization
func (s *OrganizationHierarchyService) isOrganizationCreatedBy(org models.LogtoOrganization, creatorOrgID string) bool {
	if org.CustomData != nil {
		if createdBy, exists := org.CustomData["createdBy"]; exists {
			if createdByStr, ok := createdBy.(string); ok {
				return createdByStr == creatorOrgID
			}
		}
	}
	return false
}
