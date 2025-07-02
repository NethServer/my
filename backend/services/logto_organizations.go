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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nethesis/my/backend/logger"
)

// GetUserOrganizations fetches organizations the user belongs to
func (c *LogtoManagementClient) GetUserOrganizations(userID string) ([]LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/organizations", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode user organizations: %w", err)
	}

	return orgs, nil
}

// GetAllOrganizations fetches all organizations from Logto
func (c *LogtoManagementClient) GetAllOrganizations() ([]LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	return orgs, nil
}

// GetOrganizationByID fetches a specific organization by ID
func (c *LogtoManagementClient) GetOrganizationByID(orgID string) (*LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("organization not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization, status %d: %s", resp.StatusCode, string(body))
	}

	var org LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode organization: %w", err)
	}

	return &org, nil
}

// CreateOrganization creates a new organization in Logto with customData
func (c *LogtoManagementClient) CreateOrganization(request CreateOrganizationRequest) (*LogtoOrganization, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/organizations", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create organization, status %d: %s", resp.StatusCode, string(body))
	}

	var org LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode created organization: %w", err)
	}

	return &org, nil
}

// UpdateOrganization updates an existing organization in Logto
func (c *LogtoManagementClient) UpdateOrganization(orgID string, request UpdateOrganizationRequest) (*LogtoOrganization, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/organizations/%s", orgID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update organization, status %d: %s", resp.StatusCode, string(body))
	}

	var org LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode updated organization: %w", err)
	}

	return &org, nil
}

// DeleteOrganization deletes an organization from Logto
func (c *LogtoManagementClient) DeleteOrganization(orgID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/organizations/%s", orgID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete organization, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetOrganizationJitRoles fetches default organization roles (just-in-time provisioning)
func (c *LogtoManagementClient) GetOrganizationJitRoles(orgID string) ([]LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/jit/roles", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization JIT roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization JIT roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode organization JIT roles: %w", err)
	}

	return roles, nil
}

// AssignOrganizationJitRoles assigns default organization roles to an organization
func (c *LogtoManagementClient) AssignOrganizationJitRoles(orgID string, roleIDs []string) error {
	requestBody := map[string]interface{}{
		"organizationRoleIds": roleIDs,
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal JIT roles request: %w", err)
	}

	resp, err := c.makeRequest("PUT", fmt.Sprintf("/organizations/%s/jit/roles", orgID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to assign JIT roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign JIT roles, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetOrganizationUsers fetches users belonging to an organization
func (c *LogtoManagementClient) GetOrganizationUsers(orgID string) ([]LogtoUser, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/users", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization users, status %d: %s", resp.StatusCode, string(body))
	}

	var users []LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode organization users: %w", err)
	}

	return users, nil
}

// GetOrganizationsByRole fetches organizations that have specific default organization roles (JIT)
// This is used to filter distributors, resellers, customers based on their JIT role configuration
func GetOrganizationsByRole(roleType string) ([]LogtoOrganization, error) {
	client := NewLogtoManagementClient()

	// Get all organizations
	allOrgs, err := client.GetAllOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	var filteredOrgs []LogtoOrganization

	// For each organization, check if it has the specified default role (JIT)
	for _, org := range allOrgs {
		jitRoles, err := client.GetOrganizationJitRoles(org.ID)
		if err != nil {
			logger.ComponentLogger("logto").Warn().
				Err(err).
				Str("operation", "get_jit_roles").
				Str("org_id", org.ID).
				Msg("Failed to get JIT roles for organization")
			continue
		}

		// Check if this organization has the target role as default
		hasRole := false
		for _, role := range jitRoles {
			logger.ComponentLogger("logto").Debug().
				Str("operation", "check_jit_role").
				Str("org_id", org.ID).
				Str("org_name", org.Name).
				Str("role_name", role.Name).
				Msg("Organization has JIT role")
			if role.Name == roleType {
				hasRole = true
				break
			}
		}

		if hasRole {
			logger.ComponentLogger("logto").Info().
				Str("operation", "role_match").
				Str("org_id", org.ID).
				Str("org_name", org.Name).
				Str("role_type", roleType).
				Msg("Organization matches role")
			filteredOrgs = append(filteredOrgs, org)
		}
	}

	logger.ComponentLogger("logto").Info().
		Int("org_count", len(filteredOrgs)).
		Str("operation", "filter_by_role").
		Str("role_type", roleType).
		Msg("Found organizations with JIT role")
	return filteredOrgs, nil
}

// FilterOrganizationsByVisibility filters organizations based on user's visibility permissions
func FilterOrganizationsByVisibility(orgs []LogtoOrganization, userOrgRole, userOrgID string, targetRole string) []LogtoOrganization {
	// Owner can see everything
	if userOrgRole == "Owner" {
		logger.ComponentLogger("logto").Info().
			Int("org_count", len(orgs)).
			Str("operation", "filter_organizations").
			Str("user_role", "Owner").
			Str("target_role", targetRole).
			Msg("Owner user - showing all organizations")
		return orgs
	}

	var filteredOrgs []LogtoOrganization

	switch targetRole {
	case "Distributor":
		// Only Owner should access distributors (already protected by middleware)
		logger.ComponentLogger("logto").Info().
			Str("operation", "filter_organizations").
			Str("target_role", "distributors").
			Msg("Non-Owner user accessing distributors - should be blocked by middleware")
		return filteredOrgs

	case "Reseller":
		// Distributors see only resellers they created
		if userOrgRole == "Distributor" {
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok && createdBy == userOrgID {
						filteredOrgs = append(filteredOrgs, org)
					}
				}
			}
			logger.ComponentLogger("logto").Info().
				Str("operation", "filter_organizations").
				Str("user_org_id", userOrgID).
				Int("visible_count", len(filteredOrgs)).
				Int("total_count", len(orgs)).
				Str("target_role", "resellers").
				Msg("Distributor filtered resellers")
		}

	case "Customer":
		if userOrgRole == "Distributor" {
			// Distributors see customers created by their resellers
			// First, get all resellers created by this distributor
			distributorResellers, err := GetOrganizationsByRole("Reseller")
			if err != nil {
				logger.ComponentLogger("logto").Error().
					Err(err).
					Str("operation", "get_distributor_resellers").
					Msg("Failed to get distributor's resellers")
				return filteredOrgs
			}

			// Get IDs of resellers created by this distributor
			var resellerIDs []string
			for _, reseller := range distributorResellers {
				if reseller.CustomData != nil {
					if createdBy, ok := reseller.CustomData["createdBy"].(string); ok && createdBy == userOrgID {
						resellerIDs = append(resellerIDs, reseller.ID)
					}
				}
			}

			// Filter customers created by these resellers
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok {
						for _, resellerID := range resellerIDs {
							if createdBy == resellerID {
								filteredOrgs = append(filteredOrgs, org)
								break
							}
						}
					}
				}
			}
			logger.ComponentLogger("logto").Info().
				Str("operation", "filter_organizations").
				Str("user_org_id", userOrgID).
				Int("visible_count", len(filteredOrgs)).
				Int("total_count", len(orgs)).
				Int("reseller_count", len(resellerIDs)).
				Str("target_role", "customers").
				Msg("Distributor filtered customers via resellers")

		} else if userOrgRole == "Reseller" {
			// Resellers see only customers they created
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok && createdBy == userOrgID {
						filteredOrgs = append(filteredOrgs, org)
					}
				}
			}
			logger.ComponentLogger("logto").Info().
				Str("operation", "filter_organizations").
				Str("user_org_id", userOrgID).
				Int("visible_count", len(filteredOrgs)).
				Int("total_count", len(orgs)).
				Str("target_role", "customers").
				Msg("Reseller filtered customers")
		}
	}

	return filteredOrgs
}

// GetAllVisibleOrganizations gets all organizations visible to a user based on their role and organization
func GetAllVisibleOrganizations(userOrgRole, userOrgID string) ([]LogtoOrganization, error) {
	client := NewLogtoManagementClient()

	// Get all organizations first
	allOrgs, err := client.GetAllOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get all organizations: %w", err)
	}

	var visibleOrgs []LogtoOrganization

	// Owner can see everything
	if userOrgRole == "Owner" {
		return allOrgs, nil
	}

	// For other roles, filter based on hierarchy and creation relationships
	for _, org := range allOrgs {
		// Determine if this organization should be visible
		shouldInclude := false

		if org.CustomData != nil {
			orgType, _ := org.CustomData["type"].(string)
			createdBy, _ := org.CustomData["createdBy"].(string)

			switch userOrgRole {
			case "Distributor":
				// Distributors can see:
				// - Their own organization
				// - Resellers they created
				// - Customers created by their resellers
				// BUT NEVER Owner organizations (higher in hierarchy)
				if orgType == "owner" {
					shouldInclude = false // Explicitly exclude Owner organizations
				} else if org.ID == userOrgID {
					shouldInclude = true
				} else if orgType == "reseller" && createdBy == userOrgID {
					shouldInclude = true
				} else if orgType == "customer" {
					// Check if customer was created by a reseller owned by this distributor
					resellers, _ := GetOrganizationsByRole("Reseller")
					for _, reseller := range resellers {
						if reseller.CustomData != nil {
							if resellerCreatedBy, ok := reseller.CustomData["createdBy"].(string); ok && resellerCreatedBy == userOrgID {
								if createdBy == reseller.ID {
									shouldInclude = true
									break
								}
							}
						}
					}
				}

			case "Reseller":
				// Resellers can see:
				// - Their own organization
				// - Customers they created
				// BUT NEVER Owner or Distributor organizations (higher in hierarchy)
				if orgType == "owner" || orgType == "distributor" {
					shouldInclude = false // Explicitly exclude higher hierarchy organizations
				} else if org.ID == userOrgID {
					shouldInclude = true
				} else if orgType == "customer" && createdBy == userOrgID {
					shouldInclude = true
				}

			case "Customer":
				// Customers can only see their own organization
				// NEVER Owner, Distributor, or Reseller organizations (higher in hierarchy)
				if orgType == "owner" || orgType == "distributor" || orgType == "reseller" {
					shouldInclude = false // Explicitly exclude higher hierarchy organizations
				} else if org.ID == userOrgID {
					shouldInclude = true
				}
			}
		}

		if shouldInclude {
			visibleOrgs = append(visibleOrgs, org)
		}
	}

	return visibleOrgs, nil
}
