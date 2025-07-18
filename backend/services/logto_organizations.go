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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// GetUserOrganizations fetches organizations the user belongs to
func (c *LogtoManagementClient) GetUserOrganizations(userID string) ([]models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s/organizations", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode user organizations: %w", err)
	}

	return orgs, nil
}

// GetAllOrganizations fetches all organizations from Logto
func (c *LogtoManagementClient) GetAllOrganizations() ([]models.LogtoOrganization, error) {
	resp, err := c.makeRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	return orgs, nil
}

// GetOrganizationsPaginated fetches organizations with pagination and filters using Logto native API
func (c *LogtoManagementClient) GetOrganizationsPaginated(page, pageSize int, filters models.OrganizationFilters) (*models.PaginatedOrganizations, error) {
	// Build URL with Logto's native pagination parameters
	url := fmt.Sprintf("/organizations?page=%d&page_size=%d", page, pageSize)

	// Add Logto's native search parameter 'q' for name/ID search
	if filters.Search != "" {
		url += fmt.Sprintf("&q=%s", filters.Search)
	}

	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organizations: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organizations, status %d: %s", resp.StatusCode, string(body))
	}

	var orgs []models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&orgs); err != nil {
		return nil, fmt.Errorf("failed to decode organizations: %w", err)
	}

	// Apply client-side filtering for custom data fields that Logto doesn't support
	filteredOrgs := c.applyClientSideFilters(orgs, filters)

	// Note: Logto doesn't provide total count in response, so we estimate
	// This is a limitation - for accurate pagination we'd need to call without pagination first
	totalCount := len(filteredOrgs)

	// If we got a full page, there might be more
	if len(orgs) == pageSize {
		// Estimate based on current page
		totalCount = page*pageSize + 1 // At least one more
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	paginationInfo := models.PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    len(orgs) == pageSize, // If we got full page, assume there might be more
		HasPrev:    page > 1,
	}

	if paginationInfo.HasNext {
		nextPage := page + 1
		paginationInfo.NextPage = &nextPage
	}

	if paginationInfo.HasPrev {
		prevPage := page - 1
		paginationInfo.PrevPage = &prevPage
	}

	return &models.PaginatedOrganizations{
		Data:       filteredOrgs,
		Pagination: paginationInfo,
	}, nil
}

// applyClientSideFilters applies filters that can't be done server-side
func (c *LogtoManagementClient) applyClientSideFilters(orgs []models.LogtoOrganization, filters models.OrganizationFilters) []models.LogtoOrganization {
	if filters.Name == "" && filters.Description == "" && filters.Type == "" && filters.CreatedBy == "" {
		return orgs
	}

	var filtered []models.LogtoOrganization
	for _, org := range orgs {
		// Name filter (exact match - search is handled by Logto's 'q' parameter)
		if filters.Name != "" && org.Name != filters.Name {
			continue
		}

		// Description filter (exact match)
		if filters.Description != "" && org.Description != filters.Description {
			continue
		}

		// Custom data filters (these can't be done server-side)
		if filters.Type != "" || filters.CreatedBy != "" {
			if org.CustomData == nil {
				continue
			}

			// Type filter
			if filters.Type != "" {
				if orgType, ok := org.CustomData["type"].(string); !ok || orgType != filters.Type {
					continue
				}
			}

			// CreatedBy filter
			if filters.CreatedBy != "" {
				if createdBy, ok := org.CustomData["createdBy"].(string); !ok || createdBy != filters.CreatedBy {
					continue
				}
			}
		}

		filtered = append(filtered, org)
	}

	return filtered
}

// GetOrganizationByID fetches a specific organization by ID
func (c *LogtoManagementClient) GetOrganizationByID(orgID string) (*models.LogtoOrganization, error) {
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

	var org models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode organization: %w", err)
	}

	return &org, nil
}

// CreateOrganization creates a new organization in Logto with customData
func (c *LogtoManagementClient) CreateOrganization(request models.CreateOrganizationRequest) (*models.LogtoOrganization, error) {
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

	var org models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode created organization: %w", err)
	}

	// Invalidate organization names cache since we added a new organization
	c.InvalidateOrganizationNamesCache()

	return &org, nil
}

// UpdateOrganization updates an existing organization in Logto
func (c *LogtoManagementClient) UpdateOrganization(orgID string, request models.UpdateOrganizationRequest) (*models.LogtoOrganization, error) {
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

	var org models.LogtoOrganization
	if err := json.NewDecoder(resp.Body).Decode(&org); err != nil {
		return nil, fmt.Errorf("failed to decode updated organization: %w", err)
	}

	// Invalidate organization names cache since we may have changed the name
	c.InvalidateOrganizationNamesCache()

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

	// Invalidate organization names cache since we removed an organization
	c.InvalidateOrganizationNamesCache()

	return nil
}

// GetOrganizationJitRoles fetches default organization roles (just-in-time provisioning)
func (c *LogtoManagementClient) GetOrganizationJitRoles(orgID string) ([]models.LogtoOrganizationRole, error) {
	// Check cache first
	cacheManager := cache.GetJitRolesCacheManager()
	if cachedRoles, found := cacheManager.Get(orgID); found {
		return cachedRoles, nil
	}

	// Cache miss, fetch from API
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/jit/roles", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization JIT roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization JIT roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []models.LogtoOrganizationRole
	if err := json.NewDecoder(resp.Body).Decode(&roles); err != nil {
		return nil, fmt.Errorf("failed to decode organization JIT roles: %w", err)
	}

	// Store in cache
	cacheManager.Set(orgID, roles)

	return roles, nil
}

// GetOrganizationJitRolesParallel fetches JIT roles for multiple organizations in parallel
func (c *LogtoManagementClient) GetOrganizationJitRolesParallel(orgIDs []string) map[string]models.JitRolesResult {
	if len(orgIDs) == 0 {
		return make(map[string]models.JitRolesResult)
	}

	// Limit concurrent requests to respect rate limits (Logto: ~200 req/10s)
	maxConcurrent := 10
	if len(orgIDs) < maxConcurrent {
		maxConcurrent = len(orgIDs)
	}

	results := make(map[string]models.JitRolesResult)
	resultsMutex := sync.Mutex{}

	// Create semaphore for rate limiting
	semaphore := make(chan struct{}, maxConcurrent)

	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cacheManager := cache.GetJitRolesCacheManager()

	logger.ComponentLogger("logto").Info().
		Int("org_count", len(orgIDs)).
		Int("max_concurrent", maxConcurrent).
		Str("operation", "parallel_jit_fetch_start").
		Msg("Starting parallel JIT roles fetch")

	startTime := time.Now()

	for _, orgID := range orgIDs {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				resultsMutex.Lock()
				results[id] = models.JitRolesResult{
					OrgID: id,
					Error: fmt.Errorf("context timeout"),
				}
				resultsMutex.Unlock()
				return
			}

			// Check cache first
			if cachedRoles, found := cacheManager.Get(id); found {
				resultsMutex.Lock()
				results[id] = models.JitRolesResult{
					OrgID: id,
					Roles: cachedRoles,
				}
				resultsMutex.Unlock()
				return
			}

			// Fetch from API
			fetchStart := time.Now()
			roles, err := c.fetchJitRolesFromAPI(id)
			fetchDuration := time.Since(fetchStart)

			if err != nil {
				logger.ComponentLogger("logto").Warn().
					Err(err).
					Str("operation", "parallel_jit_fetch_error").
					Str("org_id", id).
					Dur("duration", fetchDuration).
					Msg("Failed to fetch JIT roles in parallel")

				resultsMutex.Lock()
				results[id] = models.JitRolesResult{
					OrgID: id,
					Error: err,
				}
				resultsMutex.Unlock()
				return
			}

			// Store in cache
			cacheManager.Set(id, roles)

			resultsMutex.Lock()
			results[id] = models.JitRolesResult{
				OrgID: id,
				Roles: roles,
			}
			resultsMutex.Unlock()

			logger.ComponentLogger("logto").Debug().
				Str("operation", "parallel_jit_fetch_success").
				Str("org_id", id).
				Int("roles_count", len(roles)).
				Dur("duration", fetchDuration).
				Msg("Successfully fetched JIT roles in parallel")

		}(orgID)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	totalDuration := time.Since(startTime)
	successCount := 0
	errorCount := 0

	for _, result := range results {
		if result.Error != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	logger.ComponentLogger("logto").Info().
		Int("total_orgs", len(orgIDs)).
		Int("success_count", successCount).
		Int("error_count", errorCount).
		Dur("total_duration", totalDuration).
		Float64("avg_duration_ms", float64(totalDuration.Nanoseconds())/float64(len(orgIDs))/1000000).
		Str("operation", "parallel_jit_fetch_complete").
		Msg("Completed parallel JIT roles fetch")

	return results
}

// fetchJitRolesFromAPI is a helper function that only does the API call without caching
func (c *LogtoManagementClient) fetchJitRolesFromAPI(orgID string) ([]models.LogtoOrganizationRole, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/jit/roles", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization JIT roles: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization JIT roles, status %d: %s", resp.StatusCode, string(body))
	}

	var roles []models.LogtoOrganizationRole
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

	// Invalidate cache for this organization
	cacheManager := cache.GetJitRolesCacheManager()
	cacheManager.Clear(orgID)

	return nil
}

// GetOrganizationUsers fetches users belonging to an organization
func (c *LogtoManagementClient) GetOrganizationUsers(ctx context.Context, orgID string) ([]models.LogtoUser, error) {
	// Check cache first
	cacheManager := cache.GetOrgUsersCacheManager()
	if cachedUsers, found := cacheManager.Get(orgID); found {
		return cachedUsers, nil
	}

	// Cache miss, fetch from API
	resp, err := c.makeRequest("GET", fmt.Sprintf("/organizations/%s/users", orgID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch organization users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch organization users, status %d: %s", resp.StatusCode, string(body))
	}

	var users []models.LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode organization users: %w", err)
	}

	// Store in cache
	cacheManager.Set(orgID, users)

	return users, nil
}

// GetOrganizationsByRole fetches organizations that have specific default organization roles (JIT)
// This is used to filter distributors, resellers, customers based on their JIT role configuration
func GetOrganizationsByRole(roleType string) ([]models.LogtoOrganization, error) {
	client := NewLogtoManagementClient()

	// Get all organizations
	allOrgs, err := client.GetAllOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	var filteredOrgs []models.LogtoOrganization

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

// GetOrganizationsByRolePaginated fetches organizations with pagination and filters
func GetOrganizationsByRolePaginated(roleType string, page, pageSize int, filters models.OrganizationFilters) (*models.PaginatedOrganizations, error) {
	client := NewLogtoManagementClient()

	// Don't apply Type filter here - we'll check JIT roles instead
	tempFilters := filters
	tempFilters.Type = "" // Remove type filter to get all orgs first

	// Get organizations with pagination and filters (without type filter)
	result, err := client.GetOrganizationsPaginated(page, pageSize, tempFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	// Extract organization IDs for parallel processing
	orgIDs := make([]string, len(result.Data))
	orgMap := make(map[string]models.LogtoOrganization)

	for i, org := range result.Data {
		orgIDs[i] = org.ID
		orgMap[org.ID] = org
	}

	// Fetch JIT roles in parallel
	jitResults := client.GetOrganizationJitRolesParallel(orgIDs)

	var filteredOrgs []models.LogtoOrganization

	// Process results and filter organizations
	for orgID, jitResult := range jitResults {
		if jitResult.Error != nil {
			logger.ComponentLogger("logto").Warn().
				Err(jitResult.Error).
				Str("operation", "get_jit_roles_parallel").
				Str("org_id", orgID).
				Msg("Failed to get JIT roles for organization in parallel")
			continue
		}

		// Check if this organization has the target role as default
		hasRole := false
		for _, role := range jitResult.Roles {
			if role.Name == roleType {
				hasRole = true
				break
			}
		}

		if hasRole {
			// Get the organization from our map
			if org, exists := orgMap[orgID]; exists {
				filteredOrgs = append(filteredOrgs, org)
			}
		}
	}

	// Update result with filtered organizations
	result.Data = filteredOrgs
	result.Pagination.TotalCount = len(filteredOrgs)
	result.Pagination.TotalPages = (result.Pagination.TotalCount + pageSize - 1) / pageSize

	logger.ComponentLogger("logto").Info().
		Int("org_count", len(filteredOrgs)).
		Str("operation", "filter_by_role_paginated").
		Str("role_type", roleType).
		Int("page", page).
		Int("page_size", pageSize).
		Msg("Found organizations with JIT role (paginated)")

	return result, nil
}

// FilterOrganizationsByVisibility filters organizations based on user's visibility permissions
func FilterOrganizationsByVisibility(orgs []models.LogtoOrganization, userOrgRole, userOrgID string, targetRole string) []models.LogtoOrganization {
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

	var filteredOrgs []models.LogtoOrganization

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
			// Distributors see customers created by themselves OR their resellers
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

			// Filter customers created by this distributor OR their resellers
			for _, org := range orgs {
				if org.CustomData != nil {
					if createdBy, ok := org.CustomData["createdBy"].(string); ok {
						// Check if created directly by this distributor
						if createdBy == userOrgID {
							filteredOrgs = append(filteredOrgs, org)
						} else {
							// Check if created by one of this distributor's resellers
							for _, resellerID := range resellerIDs {
								if createdBy == resellerID {
									filteredOrgs = append(filteredOrgs, org)
									break
								}
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
				Msg("Distributor filtered customers (direct + via resellers)")

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
func GetAllVisibleOrganizations(userOrgRole, userOrgID string) ([]models.LogtoOrganization, error) {
	client := NewLogtoManagementClient()

	// Get all organizations first
	allOrgs, err := client.GetAllOrganizations()
	if err != nil {
		return nil, fmt.Errorf("failed to get all organizations: %w", err)
	}

	var visibleOrgs []models.LogtoOrganization

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

// CheckOrganizationNameUniqueness verifies if an organization name is already in use
func (c *LogtoManagementClient) CheckOrganizationNameUniqueness(name string) (bool, error) {
	cacheManager := cache.GetOrganizationNamesCacheManager()

	// Try to get from cache first
	isNameTaken, existingName := cacheManager.IsNameTaken(name)
	if isNameTaken {
		logger.ComponentLogger("logto").Debug().
			Str("operation", "name_uniqueness_check").
			Str("requested_name", name).
			Str("existing_name", existingName).
			Msg("Organization name found in cache - not unique")
		return false, nil // Not unique
	}

	// If cache is available but name not found, it's unique
	names, cacheHit := cacheManager.Get()
	if cacheHit {
		logger.ComponentLogger("logto").Debug().
			Str("operation", "name_uniqueness_check").
			Str("requested_name", name).
			Int("cached_names_count", len(names)).
			Msg("Organization name not found in cache - unique")
		return true, nil // Unique
	}

	// Cache miss - need to populate cache first
	logger.ComponentLogger("logto").Info().
		Str("operation", "name_uniqueness_check").
		Str("requested_name", name).
		Msg("Cache miss - populating organization names cache")

	err := c.PopulateOrganizationNamesCache()
	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "populate_names_cache").
			Msg("Failed to populate organization names cache")
		// Fall back to direct API call
		return c.checkOrganizationNameUniquenessDirect(name)
	}

	// Now check cache again
	isNameTaken, existingName = cacheManager.IsNameTaken(name)
	if isNameTaken {
		logger.ComponentLogger("logto").Debug().
			Str("operation", "name_uniqueness_check").
			Str("requested_name", name).
			Str("existing_name", existingName).
			Msg("Organization name found after cache population - not unique")
		return false, nil // Not unique
	}

	logger.ComponentLogger("logto").Debug().
		Str("operation", "name_uniqueness_check").
		Str("requested_name", name).
		Msg("Organization name not found after cache population - unique")
	return true, nil // Unique
}

// checkOrganizationNameUniquenessDirect is a fallback method that checks uniqueness directly via API
func (c *LogtoManagementClient) checkOrganizationNameUniquenessDirect(name string) (bool, error) {
	logger.ComponentLogger("logto").Warn().
		Str("operation", "name_uniqueness_check_direct").
		Str("requested_name", name).
		Msg("Falling back to direct API call for name uniqueness check")

	// Get all organizations and check manually
	allOrgs, err := c.GetAllOrganizations()
	if err != nil {
		return false, fmt.Errorf("failed to get all organizations for name check: %w", err)
	}

	// Check if any organization has the exact same name (case-insensitive)
	for _, org := range allOrgs {
		if strings.EqualFold(org.Name, name) {
			return false, nil // Not unique
		}
	}

	return true, nil // Unique
}

// PopulateOrganizationNamesCache populates the organization names cache
func (c *LogtoManagementClient) PopulateOrganizationNamesCache() error {
	logger.ComponentLogger("logto").Info().
		Str("operation", "populate_names_cache").
		Msg("Starting organization names cache population")

	startTime := time.Now()

	// Get all organizations
	allOrgs, err := c.GetAllOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get all organizations: %w", err)
	}

	// Build names map
	names := make(map[string]string)
	for _, org := range allOrgs {
		names[strings.ToLower(org.Name)] = org.Name
	}

	// Store in cache
	cacheManager := cache.GetOrganizationNamesCacheManager()
	cacheManager.Set(names)

	duration := time.Since(startTime)
	logger.ComponentLogger("logto").Info().
		Str("operation", "populate_names_cache").
		Int("organizations_count", len(allOrgs)).
		Dur("duration", duration).
		Msg("Organization names cache populated successfully")

	return nil
}

// InvalidateOrganizationNamesCache clears the organization names cache
func (c *LogtoManagementClient) InvalidateOrganizationNamesCache() {
	cacheManager := cache.GetOrganizationNamesCacheManager()
	cacheManager.Clear()

	logger.ComponentLogger("logto").Info().
		Str("operation", "invalidate_names_cache").
		Msg("Organization names cache invalidated")
}
