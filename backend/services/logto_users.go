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
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// GetUserByID fetches a specific user by ID
func (c *LogtoManagementClient) GetUserByID(userID string) (*LogtoUser, error) {
	resp, err := c.makeRequest("GET", fmt.Sprintf("/users/%s", userID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("user not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user, status %d: %s", resp.StatusCode, string(body))
	}

	var user LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user: %w", err)
	}

	return &user, nil
}

// CreateUser creates a new account in Logto
func (c *LogtoManagementClient) CreateUser(request CreateUserRequest) (*LogtoUser, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user request: %w", err)
	}

	resp, err := c.makeRequest("POST", "/users", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create user, status %d: %s", resp.StatusCode, string(body))
	}

	var user LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode created user: %w", err)
	}

	return &user, nil
}

// UpdateUser updates an existing user in Logto
func (c *LogtoManagementClient) UpdateUser(userID string, request UpdateUserRequest) (*LogtoUser, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user update request: %w", err)
	}

	resp, err := c.makeRequest("PATCH", fmt.Sprintf("/users/%s", userID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update user, status %d: %s", resp.StatusCode, string(body))
	}

	var user LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode updated user: %w", err)
	}

	return &user, nil
}

// DeleteUser deletes a user from Logto
func (c *LogtoManagementClient) DeleteUser(userID string) error {
	resp, err := c.makeRequest("DELETE", fmt.Sprintf("/users/%s", userID), nil)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// AssignUserToOrganization assigns a user to an organization without any roles
// Roles should be assigned separately using AssignOrganizationRolesToUser
func (c *LogtoManagementClient) AssignUserToOrganization(orgID, userID string) error {
	requestBody := map[string]interface{}{
		"userIds": []string{userID},
		// No organizationRoleIds - roles assigned separately
	}

	reqBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal organization assignment request: %w", err)
	}

	resp, err := c.makeRequest("POST", fmt.Sprintf("/organizations/%s/users", orgID), bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to assign user to organization: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to assign user to organization, status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetAllUsers fetches all users from Logto
func (c *LogtoManagementClient) GetAllUsers() ([]LogtoUser, error) {
	resp, err := c.makeRequest("GET", "/users", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch users, status %d: %s", resp.StatusCode, string(body))
	}

	var users []LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	return users, nil
}

// GetUsersPaginated fetches users with pagination and filters using Logto native API
func (c *LogtoManagementClient) GetUsersPaginated(page, pageSize int, filters UserFilters) (*PaginatedUsers, error) {
	// Build URL with Logto's native pagination parameters
	url := fmt.Sprintf("/users?page=%d&page_size=%d", page, pageSize)

	// Add Logto's native search parameter if provided
	if filters.Search != "" {
		url += fmt.Sprintf("&search=%s", filters.Search)
	}

	resp, err := c.makeRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch users, status %d: %s", resp.StatusCode, string(body))
	}

	var users []LogtoUser
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	// Apply client-side filtering for fields that Logto doesn't support server-side
	filteredUsers := c.applyUserClientSideFilters(users, filters)

	// Calculate pagination info (similar limitation as organizations)
	totalCount := len(filteredUsers)

	// If we got a full page, there might be more
	if len(users) == pageSize {
		totalCount = page*pageSize + 1 // At least one more
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	paginationInfo := PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    len(users) == pageSize,
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

	return &PaginatedUsers{
		Data:       filteredUsers,
		Pagination: paginationInfo,
	}, nil
}

// applyUserClientSideFilters applies filters that can't be done server-side
func (c *LogtoManagementClient) applyUserClientSideFilters(users []LogtoUser, filters UserFilters) []LogtoUser {
	if filters.Username == "" && filters.Email == "" && filters.Role == "" && filters.OrganizationID == "" {
		return users
	}

	var filtered []LogtoUser
	for _, user := range users {
		// Username filter (exact match)
		if filters.Username != "" && user.Username != filters.Username {
			continue
		}

		// Email filter (exact match)
		if filters.Email != "" && user.PrimaryEmail != filters.Email {
			continue
		}

		// Custom data filters (these can't be done server-side)
		if filters.Role != "" || filters.OrganizationID != "" {
			if user.CustomData == nil {
				continue
			}

			// Role filter
			if filters.Role != "" {
				if userRole, ok := user.CustomData["role"].(string); !ok || userRole != filters.Role {
					continue
				}
			}

			// OrganizationID filter
			if filters.OrganizationID != "" {
				if orgID, ok := user.CustomData["organizationId"].(string); !ok || orgID != filters.OrganizationID {
					continue
				}
			}
		}

		filtered = append(filtered, user)
	}

	return filtered
}

// GetOrganizationUsersParallel fetches users for multiple organizations in parallel
func (c *LogtoManagementClient) GetOrganizationUsersParallel(orgIDs []string) map[string]OrgUsersResult {
	if len(orgIDs) == 0 {
		return make(map[string]OrgUsersResult)
	}

	// Limit concurrent requests to respect rate limits
	maxConcurrent := 10
	if len(orgIDs) < maxConcurrent {
		maxConcurrent = len(orgIDs)
	}

	results := make(map[string]OrgUsersResult)
	resultsMutex := sync.Mutex{}

	// Create semaphore for rate limiting
	semaphore := make(chan struct{}, maxConcurrent)

	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// Context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.ComponentLogger("logto").Info().
		Int("org_count", len(orgIDs)).
		Int("max_concurrent", maxConcurrent).
		Str("operation", "parallel_org_users_fetch_start").
		Msg("Starting parallel organization users fetch")

	startTime := time.Now()

	for _, orgID := range orgIDs {
		wg.Add(1)

		go func(id string) {
			defer wg.Done()

			// Check cache first (avoid semaphore if cached)
			cache := GetOrgUsersCacheManager()
			if cachedUsers, found := cache.Get(id); found {
				resultsMutex.Lock()
				results[id] = OrgUsersResult{
					OrgID: id,
					Users: cachedUsers,
				}
				resultsMutex.Unlock()
				return
			}

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				resultsMutex.Lock()
				results[id] = OrgUsersResult{
					OrgID: id,
					Error: fmt.Errorf("context timeout"),
				}
				resultsMutex.Unlock()
				return
			}

			// Fetch from API
			fetchStart := time.Now()
			users, err := c.GetOrganizationUsers(id)
			fetchDuration := time.Since(fetchStart)

			if err != nil {
				logger.ComponentLogger("logto").Warn().
					Err(err).
					Str("operation", "parallel_org_users_fetch_error").
					Str("org_id", id).
					Dur("duration", fetchDuration).
					Msg("Failed to fetch organization users in parallel")

				resultsMutex.Lock()
				results[id] = OrgUsersResult{
					OrgID: id,
					Error: err,
				}
				resultsMutex.Unlock()
				return
			}

			resultsMutex.Lock()
			results[id] = OrgUsersResult{
				OrgID: id,
				Users: users,
			}
			resultsMutex.Unlock()

			logger.ComponentLogger("logto").Debug().
				Str("operation", "parallel_org_users_fetch_success").
				Str("org_id", id).
				Int("users_count", len(users)).
				Dur("duration", fetchDuration).
				Msg("Successfully fetched organization users in parallel")

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
		Str("operation", "parallel_org_users_fetch_complete").
		Msg("Completed parallel organization users fetch")

	return results
}
