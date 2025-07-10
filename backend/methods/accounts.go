/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// sanitizeUsernameForLogto sanitizes username to match Logto's regex: /^[A-Z_a-z]\w*$/
func sanitizeUsernameForLogto(username string) string {
	// Replace dots and other special characters with underscores
	sanitized := regexp.MustCompile(`[^A-Za-z0-9_]`).ReplaceAllString(username, "_")

	// Ensure it starts with a letter or underscore
	if len(sanitized) > 0 && !regexp.MustCompile(`^[A-Za-z_]`).MatchString(sanitized) {
		sanitized = "user_" + sanitized
	}

	return sanitized
}

// CanOperateOnAccount validates if a user can operate (read/update/delete) on a specific account
func CanOperateOnAccount(currentUserOrgRole, currentUserOrgID, currentUserRole string, targetAccount *models.LogtoUser, targetOrg *models.LogtoOrganization) (bool, string) {
	// Extract target account's organization data
	var targetAccountOrgID, targetAccountOrgRole string

	if targetAccount.CustomData != nil {
		if orgID, ok := targetAccount.CustomData["organizationId"].(string); ok {
			targetAccountOrgID = orgID
		}
		if orgRole, ok := targetAccount.CustomData["organizationRole"].(string); ok {
			targetAccountOrgRole = orgRole
		}
	}

	// If we couldn't get the org data from customData, try to get it from the organization parameter
	if targetOrg != nil {
		if targetAccountOrgID == "" {
			targetAccountOrgID = targetOrg.ID
		}
		if targetAccountOrgRole == "" && targetOrg.CustomData != nil {
			if orgType, ok := targetOrg.CustomData["type"].(string); ok {
				switch orgType {
				case "distributor":
					targetAccountOrgRole = "Distributor"
				case "reseller":
					targetAccountOrgRole = "Reseller"
				case "customer":
					targetAccountOrgRole = "Customer"
				}
			}
		}
	}

	switch currentUserOrgRole {
	case "Owner":
		// Owner can operate on any account
		return true, ""

	case "Distributor":
		// Distributors can operate on accounts in:
		// - Their own organization
		// - Reseller/Customer organizations they created
		if currentUserOrgID == targetAccountOrgID {
			return true, ""
		}
		if targetAccountOrgRole == "Reseller" || targetAccountOrgRole == "Customer" {
			// Check if the target organization was created by this distributor
			if targetOrg != nil && targetOrg.CustomData != nil {
				if createdBy, ok := targetOrg.CustomData["createdBy"].(string); ok {
					if createdBy == currentUserOrgID {
						return true, ""
					}
				}
			}
			return false, "distributors can only operate on accounts in organizations they created"
		}
		return false, "distributors cannot operate on accounts in this organization"

	case "Reseller":
		// Resellers can operate on accounts in:
		// - Their own organization
		// - Customer organizations they created
		if currentUserOrgID == targetAccountOrgID {
			return true, ""
		}
		if targetAccountOrgRole == "Customer" {
			// Check if the target organization was created by this reseller
			if targetOrg != nil && targetOrg.CustomData != nil {
				if createdBy, ok := targetOrg.CustomData["createdBy"].(string); ok {
					if createdBy == currentUserOrgID {
						return true, ""
					}
				}
			}
			return false, "resellers can only operate on accounts in customer organizations they created"
		}
		return false, "resellers cannot operate on accounts in this organization"

	case "Customer":
		// Customers can only operate on accounts in their own organization
		if currentUserOrgID == targetAccountOrgID && currentUserRole == "Admin" {
			return true, ""
		}
		return false, "customers can only operate on accounts in their own organization and must be Admin"

	default:
		return false, "unknown organization role"
	}
}

// CanCreateAccountForOrganization validates if a user can create accounts for a target organization
func CanCreateAccountForOrganization(userOrgRole, userOrgID, userRole, targetOrgID, targetOrgRole string, targetOrg *models.LogtoOrganization) (bool, string) {
	// Only Admin users can create accounts for colleagues in the same organization
	if userOrgID == targetOrgID && userRole != "Admin" {
		return false, "only Admin users can create accounts for colleagues in the same organization"
	}

	switch userOrgRole {
	case "Owner":
		// Owner can create accounts for any organization
		return true, ""
	case "Distributor":
		// Distributors can create accounts for:
		// - Their own organization (if Admin)
		// - Reseller/Customer organizations that they created or are under their hierarchy
		if userOrgID == targetOrgID {
			return userRole == "Admin", "only Admin users can create accounts for colleagues in the same organization"
		}
		if targetOrgRole == "Reseller" || targetOrgRole == "Customer" {
			// Check if the target organization was created by this distributor or is under their hierarchy
			if targetOrg != nil && targetOrg.CustomData != nil {
				if createdBy, ok := targetOrg.CustomData["createdBy"].(string); ok {
					if createdBy == userOrgID {
						return true, ""
					}
				}
			}
			return false, "distributors can only create accounts for reseller or customer organizations they created"
		}
		return false, "distributors can only create accounts for reseller or customer organizations"
	case "Reseller":
		// Resellers can create accounts for:
		// - Their own organization (if Admin)
		// - Customer organizations that they created
		if userOrgID == targetOrgID {
			return userRole == "Admin", "only Admin users can create accounts for colleagues in the same organization"
		}
		if targetOrgRole == "Customer" {
			// Check if the target organization was created by this reseller
			if targetOrg != nil && targetOrg.CustomData != nil {
				if createdBy, ok := targetOrg.CustomData["createdBy"].(string); ok {
					if createdBy == userOrgID {
						return true, ""
					}
				}
			}
			return false, "resellers can only create accounts for customer organizations they created"
		}
		return false, "resellers can only create accounts for customer organizations"
	case "Customer":
		// Customers can only create accounts for their own organization (if Admin)
		if userOrgID == targetOrgID && userRole == "Admin" {
			return true, ""
		}
		return false, "customers can only create accounts for their own organization and must be Admin"
	default:
		return false, "unknown organization role"
	}
}

// CreateAccount handles POST /api/accounts - creates a new account in Logto with role and organization assignment
func CreateAccount(c *gin.Context) {
	var request models.CreateAccountRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("missing user context"), "validate_user_context", http.StatusUnauthorized, "Missing required user context in JWT token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("incomplete user context in token", nil))
		return
	}

	// Extract user role from array (Admin role is required for account creation)
	userRolesSlice := currentUserRoles.([]string)
	var currentUserRole string
	for _, role := range userRolesSlice {
		if role == "Admin" {
			currentUserRole = "Admin"
			break
		}
	}
	if currentUserRole == "" {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("insufficient permissions"), "check_admin_role", http.StatusForbidden, "User does not have Admin role required for account creation")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to create accounts", nil))
		return
	}

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// Verify the target organization exists
	targetOrg, err := client.GetOrganizationByID(request.OrganizationID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "find_target_organization", http.StatusBadRequest, "Invalid organization ID in request")
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid organization ID", map[string]interface{}{
			"field": "organizationId",
			"error": "organization not found",
			"value": request.OrganizationID,
		}))
		return
	}

	// Get target organization's JIT roles to determine the default organization role
	jitRoles, err := client.GetOrganizationJitRoles(request.OrganizationID)
	if err != nil {
		// Parse error to determine if this is a client or server error
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "404") || strings.Contains(errorMsg, "not found") {
			logger.RequestLogger(c, "accounts").Warn().
				Err(err).
				Str("operation", "get_organization_jit_roles").
				Str("organization_id", request.OrganizationID).
				Msg("Organization JIT roles not found")
			c.JSON(http.StatusNotFound, response.NotFound("organization JIT roles not configured", err.Error()))
			return
		} else if strings.Contains(errorMsg, "403") || strings.Contains(errorMsg, "forbidden") {
			logger.RequestLogger(c, "accounts").Warn().
				Err(err).
				Str("operation", "get_organization_jit_roles").
				Msg("Insufficient permissions to access organization JIT roles")
			c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to access organization configuration", nil))
			return
		} else if strings.Contains(errorMsg, "503") || strings.Contains(errorMsg, "502") || strings.Contains(errorMsg, "timeout") {
			logger.RequestLogger(c, "accounts").Warn().
				Err(err).
				Str("operation", "get_organization_jit_roles").
				Msg("Logto service temporarily unavailable")
			c.JSON(http.StatusServiceUnavailable, response.Error(http.StatusServiceUnavailable, "identity provider temporarily unavailable", nil))
			return
		} else {
			// For genuine server errors (500, database issues, etc.)
			logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "get_organization_jit_roles", http.StatusInternalServerError, "Failed to get JIT roles for organization")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get organization JIT roles", err.Error()))
			return
		}
	}

	// Determine target organization role from JIT configuration
	var targetOrgRole string
	if len(jitRoles) > 0 {
		// Use the first JIT role as the default organization role
		targetOrgRole = jitRoles[0].Name
		logger.RequestLogger(c, "accounts").Info().
			Str("operation", "assign_jit_role").
			Str("role", targetOrgRole).
			Str("organization_id", request.OrganizationID).
			Msg("Using JIT role for organization")
	} else {
		c.JSON(http.StatusBadRequest, response.BadRequest("target organization has no JIT roles configured", nil))
		return
	}

	// Validate hierarchical permissions
	canCreate, reason := CanCreateAccountForOrganization(
		currentUserOrgRole.(string),
		currentUserOrgID.(string),
		currentUserRole,
		request.OrganizationID,
		targetOrgRole,
		targetOrg,
	)

	if !canCreate {
		logger.LogAccountOperation(c, "create_denied", request.OrganizationID, request.OrganizationID, currentUserID.(string), currentUserOrgID.(string), false, fmt.Errorf("insufficient permissions"))
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to create account for this organization", reason))
		return
	}

	// Prepare custom data for the account
	customData := map[string]interface{}{
		"userRoleIds":      request.UserRoleIDs,
		"organizationId":   request.OrganizationID,
		"organizationRole": targetOrgRole, // Derived from JIT configuration
		"createdBy":        currentUserOrgID,
		"createdAt":        time.Now().Format(time.RFC3339),
	}

	// Add phone to custom data if provided
	if request.Phone != "" {
		customData["phone"] = request.Phone
	}

	// Add custom data if provided
	if request.CustomData != nil {
		for k, v := range request.CustomData {
			customData[k] = v
		}
	}

	// Sanitize fields for Logto compliance
	sanitizedUsername := sanitizeUsernameForLogto(request.Username)

	// Create account request for Logto
	accountRequest := models.CreateUserRequest{
		Username:     sanitizedUsername,
		PrimaryEmail: request.Email,
		Name:         request.Name,
		Password:     request.Password,
		CustomData:   customData,
	}

	// Set PrimaryPhone if provided - Logto validates this field with a regex
	if request.Phone != "" {
		accountRequest.PrimaryPhone = &request.Phone
	}

	// Only set avatar if it's not empty
	if request.Avatar != "" {
		accountRequest.Avatar = &request.Avatar
	}

	// Debug: log the request being sent to Logto
	if reqJSON, err := json.Marshal(accountRequest); err == nil {
		logger.RequestLogger(c, "accounts").Debug().
			Str("operation", "send_to_logto").
			Str("payload", logger.SanitizeString(string(reqJSON))).
			Msg("Sending account creation request to Logto")
	}

	// Create the account in Logto
	account, err := client.CreateUser(accountRequest)
	if err != nil {
		// Parse error to determine appropriate status code and logging level
		errorMsg := err.Error()
		var detailedError interface{}
		var statusCode int
		var logLevel string

		// Check for different status codes and extract JSON
		statusMappings := map[string]struct {
			code  int
			level string
		}{
			"status 400: ": {http.StatusBadRequest, "warn"},
			"status 422: ": {http.StatusUnprocessableEntity, "warn"},
			"status 409: ": {http.StatusConflict, "warn"},
			"status 500: ": {http.StatusInternalServerError, "error"},
		}

		for prefix, mapping := range statusMappings {
			if strings.Contains(errorMsg, prefix) {
				statusCode = mapping.code
				logLevel = mapping.level
				parts := strings.Split(errorMsg, prefix)
				if len(parts) > 1 {
					// Try to parse the JSON error for better formatting
					var logtoError map[string]interface{}
					if json.Unmarshal([]byte(parts[1]), &logtoError) == nil {
						detailedError = logtoError
					} else {
						detailedError = parts[1]
					}
				}
				break
			}
		}

		// If no status prefix found, default to internal server error
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
			logLevel = "error"
			detailedError = errorMsg
		}

		// Log with appropriate level and status code
		if logLevel == "error" {
			logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "create_account_logto", statusCode, "Failed to create account in Logto")
		} else {
			logger.RequestLogger(c, "accounts").Warn().
				Err(err).
				Str("operation", "create_account_logto").
				Int("logto_status_code", statusCode).
				Str("user_message", "Failed to create account in Logto").
				Msg("Logto API returned client error")
		}

		// Use standardized external API error response
		c.JSON(statusCode, response.ExternalAPIError(statusCode, "failed to create account", detailedError))
		return
	}

	// Assign user roles using IDs directly (more secure)
	if len(request.UserRoleIDs) > 0 {
		if err := client.AssignUserRoles(account.ID, request.UserRoleIDs); err != nil {
			logger.RequestLogger(c, "accounts").Error().
				Err(err).
				Str("operation", "assign_user_roles").
				Interface("role_ids", request.UserRoleIDs).
				Msg("Failed to assign user roles")
		}
	}

	// Step 1: Assign user to organization (without roles)
	if err := client.AssignUserToOrganization(request.OrganizationID, account.ID); err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "assign_to_organization", http.StatusInternalServerError, "Failed to assign account to organization")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to assign account to organization", err.Error()))
		return
	}

	// Step 2: Assign organization role using the specific API endpoint
	if len(jitRoles) > 0 {
		// Use the first JIT role for the organization
		roleIDs := []string{jitRoles[0].ID}
		roleNames := []string{jitRoles[0].Name}

		logger.RequestLogger(c, "accounts").Info().
			Str("operation", "assign_organization_role").
			Str("role_name", jitRoles[0].Name).
			Str("role_id", jitRoles[0].ID).
			Str("user_id", account.ID).
			Str("organization_id", request.OrganizationID).
			Msg("Assigning organization role to user")

		if err := client.AssignOrganizationRolesToUser(request.OrganizationID, account.ID, roleIDs, roleNames); err != nil {
			logger.RequestLogger(c, "accounts").Error().
				Err(err).
				Str("operation", "assign_organization_role").
				Msg("Failed to assign organization role to user")
		} else {
			logger.RequestLogger(c, "accounts").Info().
				Str("operation", "assign_organization_role_success").
				Str("role_name", jitRoles[0].Name).
				Str("user_id", account.ID).
				Msg("Successfully assigned organization role")
		}
	}

	// Invalidate organization users cache to ensure fresh data on next request
	cacheManager := cache.GetOrgUsersCacheManager()
	cacheManager.Clear(request.OrganizationID)

	logger.LogAccountOperation(c, "create", account.ID, request.OrganizationID, currentUserID.(string), currentUserOrgID.(string), true, nil)

	// Convert to response format
	var lastSignInAt *time.Time
	if account.LastSignInAt != nil && *account.LastSignInAt != 0 {
		t := time.Unix(*account.LastSignInAt/1000, 0)
		lastSignInAt = &t
	}

	accountResponse := models.AccountResponse{
		ID:               account.ID,
		Username:         account.Username,
		Email:            account.PrimaryEmail,
		Name:             account.Name,
		Phone:            account.PrimaryPhone,
		Avatar:           account.Avatar,
		UserRoleIDs:      request.UserRoleIDs,
		OrganizationID:   request.OrganizationID,
		OrganizationName: targetOrg.Name,
		OrganizationRole: targetOrgRole, // Derived from JIT configuration
		IsSuspended:      account.IsSuspended,
		LastSignInAt:     lastSignInAt,
		CreatedAt:        time.Unix(account.CreatedAt/1000, 0),
		UpdatedAt:        time.Unix(account.UpdatedAt/1000, 0),
		CustomData:       request.CustomData,
	}

	c.JSON(http.StatusCreated, response.Created("account created successfully", accountResponse))
}

// GetAccount handles GET /api/accounts/:id - retrieves a single account
func GetAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("missing user context"), "validate_user_context", http.StatusUnauthorized, "Missing required user context in JWT token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("incomplete user context in token", nil))
		return
	}

	// Extract user role from array (Admin role is required for account operations)
	userRolesSlice := currentUserRoles.([]string)
	var currentUserRole string
	for _, role := range userRolesSlice {
		if role == "Admin" {
			currentUserRole = "Admin"
			break
		}
	}
	if currentUserRole == "" {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("insufficient permissions"), "check_admin_role", http.StatusForbidden, "User does not have Admin role required for account operations")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to view accounts", nil))
		return
	}

	logger.RequestLogger(c, "accounts").Info().
		Str("operation", "get_account").
		Str("account_id", accountID).
		Msg("Single account requested")

	client := services.NewLogtoManagementClient()

	// Get the specific account
	account, err := client.GetUserByID(accountID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "fetch_account", http.StatusInternalServerError, "Failed to fetch account")
		c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
		return
	}

	// Get target account's organization to validate permissions
	var targetOrg *models.LogtoOrganization
	if account.CustomData != nil {
		if orgID, ok := account.CustomData["organizationId"].(string); ok {
			targetOrg, err = client.GetOrganizationByID(orgID)
			if err != nil {
				logger.RequestLogger(c, "accounts").Warn().
					Err(err).
					Str("operation", "fetch_target_organization").
					Msg("Failed to fetch target organization")
			}
		}
	}

	// Validate hierarchical permissions
	canOperate, reason := CanOperateOnAccount(
		currentUserOrgRole.(string),
		currentUserOrgID.(string),
		currentUserRole,
		account,
		targetOrg,
	)

	if !canOperate {
		logger.LogAccountOperation(c, "get_denied", accountID, "", currentUserID.(string), currentUserOrgID.(string), false, fmt.Errorf("insufficient permissions: %s", reason))
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to view this account", reason))
		return
	}

	// Convert to response format
	accountResponse := convertLogtoUserToAccountResponse(*account, targetOrg)

	logger.RequestLogger(c, "accounts").Info().
		Str("operation", "get_account_result").
		Str("account_id", accountID).
		Msg("Retrieved account")

	c.JSON(http.StatusOK, response.OK("account retrieved successfully", accountResponse))
}

// GetAccounts handles GET /api/accounts - retrieves accounts with organization filtering
func GetAccounts(c *gin.Context) {
	_, _ = c.Get("user_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserOrgID, _ := c.Get("organization_id")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("missing user context"), "validate_user_context", http.StatusUnauthorized, "Missing required user context in JWT token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("incomplete user context in token", nil))
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if parsedPage, err := strconv.Atoi(p); err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}
	if ps := c.Query("page_size"); ps != "" {
		if parsedPageSize, err := strconv.Atoi(ps); err == nil && parsedPageSize > 0 && parsedPageSize <= 100 {
			pageSize = parsedPageSize
		}
	}

	// Parse filters
	filters := models.UserFilters{
		Search:         c.Query("search"),
		OrganizationID: c.Query("organization_id"),
		Role:           c.Query("role"),
		Username:       c.Query("username"),
		Email:          c.Query("email"),
	}

	logger.RequestLogger(c, "accounts").Info().
		Str("operation", "list_accounts").
		Int("page", page).
		Int("page_size", pageSize).
		Str("organization_filter", filters.OrganizationID).
		Str("search", filters.Search).
		Msg("Accounts list requested")

	client := services.NewLogtoManagementClient()
	var accounts []models.AccountResponse
	var paginationInfo models.PaginationInfo

	if filters.OrganizationID != "" {
		// Single organization mode - get accounts from specific organization
		orgAccounts, err := client.GetOrganizationUsers(c.Request.Context(), filters.OrganizationID)
		if err != nil {
			logger.RequestLogger(c, "accounts").Error().
				Err(err).
				Str("operation", "fetch_organization_accounts").
				Str("org_id", filters.OrganizationID).
				Msg("Failed to fetch organization accounts")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch organization accounts", err.Error()))
			return
		}

		// Get organization details
		org, err := client.GetOrganizationByID(filters.OrganizationID)
		if err != nil {
			logger.RequestLogger(c, "accounts").Error().
				Err(err).
				Str("operation", "fetch_organization_details").
				Str("org_id", filters.OrganizationID).
				Msg("Failed to fetch organization details")
		}

		// Apply client-side pagination and filtering
		filteredAccounts := applyAccountFilters(orgAccounts, filters)
		totalCount := len(filteredAccounts)
		totalPages := (totalCount + pageSize - 1) / pageSize

		// Calculate slice bounds for current page
		start := (page - 1) * pageSize
		end := start + pageSize
		if start > totalCount {
			start = totalCount
		}
		if end > totalCount {
			end = totalCount
		}

		var pageAccounts []models.LogtoUser
		if start < totalCount {
			pageAccounts = filteredAccounts[start:end]
		}

		// Convert to response format
		for _, account := range pageAccounts {
			accountResponse := convertLogtoUserToAccountResponse(account, org)
			accounts = append(accounts, accountResponse)
		}

		paginationInfo = models.PaginationInfo{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
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

	} else {
		// Multi-organization mode - get accounts from all visible organizations with parallel processing
		allOrgs, err := services.GetAllVisibleOrganizations(currentUserOrgRole.(string), currentUserOrgID.(string))
		if err != nil {
			logger.RequestLogger(c, "accounts").Error().
				Err(err).
				Str("operation", "fetch_visible_organizations").
				Msg("Failed to fetch visible organizations")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to fetch organizations", err.Error()))
			return
		}

		// Extract organization IDs for parallel processing
		orgIDs := make([]string, len(allOrgs))
		orgMap := make(map[string]models.LogtoOrganization)

		for i, org := range allOrgs {
			orgIDs[i] = org.ID
			orgMap[org.ID] = org
		}

		// Fetch users from all organizations in parallel
		usersResults, err := client.GetOrganizationUsersParallel(c.Request.Context(), orgIDs)
		if err != nil {
			logger.RequestLogger(c, "accounts").Error().
				Err(err).
				Str("operation", "fetch_parallel_organization_accounts").
				Msg("Failed to fetch accounts from multiple organizations")

			c.JSON(http.StatusInternalServerError, response.InternalServerError("Failed to fetch accounts", err.Error()))
			return
		}

		var allAccounts []models.LogtoUser

		// Process results and combine accounts
		for orgID, users := range usersResults {
			// Add organization context to each user for filtering
			org := orgMap[orgID]
			for _, user := range users {
				// Add organization context to custom data for filtering
				if user.CustomData == nil {
					user.CustomData = make(map[string]interface{})
				}
				user.CustomData["__org_id"] = org.ID
				user.CustomData["__org_name"] = org.Name
				allAccounts = append(allAccounts, user)
			}
		}

		// Apply filters to all accounts
		filteredAccounts := applyAccountFilters(allAccounts, filters)
		totalCount := len(filteredAccounts)
		totalPages := (totalCount + pageSize - 1) / pageSize

		// Calculate slice bounds for current page
		start := (page - 1) * pageSize
		end := start + pageSize
		if start > totalCount {
			start = totalCount
		}
		if end > totalCount {
			end = totalCount
		}

		var pageAccounts []models.LogtoUser
		if start < totalCount {
			pageAccounts = filteredAccounts[start:end]
		}

		// Convert to response format
		for _, account := range pageAccounts {
			// Get organization details for this account
			var org *models.LogtoOrganization
			if orgID, ok := account.CustomData["__org_id"].(string); ok {
				if orgData, exists := orgMap[orgID]; exists {
					org = &orgData
				}
			}

			accountResponse := convertLogtoUserToAccountResponse(account, org)
			accounts = append(accounts, accountResponse)
		}

		paginationInfo = models.PaginationInfo{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: totalCount,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
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
	}

	logger.RequestLogger(c, "accounts").Info().
		Int("account_count", len(accounts)).
		Int("total_count", paginationInfo.TotalCount).
		Int("page", page).
		Str("operation", "list_accounts_result").
		Msg("Retrieved accounts")

	c.JSON(http.StatusOK, response.OK("accounts retrieved successfully", gin.H{
		"accounts":   accounts,
		"pagination": paginationInfo,
	}))
}

// applyAccountFilters applies client-side filters to user accounts
func applyAccountFilters(users []models.LogtoUser, filters models.UserFilters) []models.LogtoUser {
	if filters.Username == "" && filters.Email == "" && filters.Role == "" && filters.Search == "" {
		return users
	}

	var filtered []models.LogtoUser
	for _, user := range users {
		// Username filter (exact match)
		if filters.Username != "" && user.Username != filters.Username {
			continue
		}

		// Email filter (exact match)
		if filters.Email != "" && user.PrimaryEmail != filters.Email {
			continue
		}

		// Search filter (partial match in username, email, or name)
		if filters.Search != "" {
			searchTerm := strings.ToLower(filters.Search)
			match := false

			if strings.Contains(strings.ToLower(user.Username), searchTerm) ||
				strings.Contains(strings.ToLower(user.PrimaryEmail), searchTerm) ||
				strings.Contains(strings.ToLower(user.Name), searchTerm) {
				match = true
			}

			if !match {
				continue
			}
		}

		// Role filter (from custom data)
		if filters.Role != "" {
			if user.CustomData == nil {
				continue
			}
			if userRole, ok := user.CustomData["role"].(string); !ok || userRole != filters.Role {
				continue
			}
		}

		filtered = append(filtered, user)
	}

	return filtered
}

// UpdateAccount handles PUT /api/accounts/:id - updates an existing account
func UpdateAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	var request models.UpdateAccountRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("missing user context"), "validate_user_context", http.StatusUnauthorized, "Missing required user context in JWT token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("incomplete user context in token", nil))
		return
	}

	// Extract user role from array (Admin role is required for account operations)
	userRolesSlice := currentUserRoles.([]string)
	var currentUserRole string
	for _, role := range userRolesSlice {
		if role == "Admin" {
			currentUserRole = "Admin"
			break
		}
	}
	if currentUserRole == "" {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("insufficient permissions"), "check_admin_role", http.StatusForbidden, "User does not have Admin role required for account operations")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to modify accounts", nil))
		return
	}

	client := services.NewLogtoManagementClient()

	// Get current account data
	currentAccount, err := client.GetUserByID(accountID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "fetch_account", http.StatusInternalServerError, "Failed to fetch account")
		c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
		return
	}

	// Get target account's organization to validate permissions
	var targetOrg *models.LogtoOrganization
	if currentAccount.CustomData != nil {
		if orgID, ok := currentAccount.CustomData["organizationId"].(string); ok {
			targetOrg, err = client.GetOrganizationByID(orgID)
			if err != nil {
				logger.RequestLogger(c, "accounts").Warn().
					Err(err).
					Str("operation", "fetch_target_organization").
					Msg("Failed to fetch target organization")
			}
		}
	}

	// Validate hierarchical permissions
	canOperate, reason := CanOperateOnAccount(
		currentUserOrgRole.(string),
		currentUserOrgID.(string),
		currentUserRole,
		currentAccount,
		targetOrg,
	)

	if !canOperate {
		logger.LogAccountOperation(c, "update_denied", accountID, "", currentUserID.(string), currentUserOrgID.(string), false, fmt.Errorf("insufficient permissions: %s", reason))
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to update this account", reason))
		return
	}

	// Prepare update request
	updateRequest := models.UpdateUserRequest{}

	if request.Username != "" {
		updateRequest.Username = &request.Username
	}
	if request.Email != "" {
		updateRequest.PrimaryEmail = &request.Email
	}
	if request.Name != "" {
		updateRequest.Name = &request.Name
	}
	if request.Phone != "" {
		updateRequest.PrimaryPhone = &request.Phone
	}
	if request.Avatar != "" {
		updateRequest.Avatar = &request.Avatar
	}

	// Merge custom data with existing data
	if currentAccount.CustomData != nil {
		updateRequest.CustomData = make(map[string]interface{})
		// Copy existing custom data
		for k, v := range currentAccount.CustomData {
			updateRequest.CustomData[k] = v
		}

		// Update with new values
		if len(request.UserRoleIDs) > 0 {
			updateRequest.CustomData["userRoleIds"] = request.UserRoleIDs
		}
		if request.OrganizationID != "" {
			updateRequest.CustomData["organizationId"] = request.OrganizationID
		}
		// OrganizationRole is now managed via JIT provisioning
		if request.CustomData != nil {
			for k, v := range request.CustomData {
				updateRequest.CustomData[k] = v
			}
		}

		// Update modification tracking
		updateRequest.CustomData["updatedBy"] = currentUserOrgID
		updateRequest.CustomData["updatedAt"] = time.Now().Format(time.RFC3339)
	}

	// Update the account in Logto
	updatedAccount, err := client.UpdateUser(accountID, updateRequest)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "update_account_logto", http.StatusInternalServerError, "Failed to update account in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update account", err.Error()))
		return
	}

	// Invalidate organization users cache to ensure fresh data on next request
	cacheManager := cache.GetOrgUsersCacheManager()
	if targetOrg != nil {
		cacheManager.Clear(targetOrg.ID)
	}
	// Also clear cache for the updated organization if it changed
	if request.OrganizationID != "" && (targetOrg == nil || targetOrg.ID != request.OrganizationID) {
		cacheManager.Clear(request.OrganizationID)
	}

	logger.LogAccountOperation(c, "update", accountID, "", currentUserID.(string), currentUserOrgID.(string), true, nil)

	// Convert to response format
	accountResponse := convertLogtoUserToAccountResponse(*updatedAccount, nil)

	c.JSON(http.StatusOK, response.OK("account updated successfully", accountResponse))
}

// DeleteAccount handles DELETE /api/accounts/:id - deletes an account
func DeleteAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("account ID required", nil))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("missing user context"), "validate_user_context", http.StatusUnauthorized, "Missing required user context in JWT token")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("incomplete user context in token", nil))
		return
	}

	// Extract user role from array (Admin role is required for account operations)
	userRolesSlice := currentUserRoles.([]string)
	var currentUserRole string
	for _, role := range userRolesSlice {
		if role == "Admin" {
			currentUserRole = "Admin"
			break
		}
	}
	if currentUserRole == "" {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(fmt.Errorf("insufficient permissions"), "check_admin_role", http.StatusForbidden, "User does not have Admin role required for account operations")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to delete accounts", nil))
		return
	}

	client := services.NewLogtoManagementClient()

	// Get account data for logging before deletion
	currentAccount, err := client.GetUserByID(accountID)
	if err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "fetch_account_for_deletion", http.StatusInternalServerError, "Failed to fetch account for deletion")
		c.JSON(http.StatusNotFound, response.NotFound("account not found", nil))
		return
	}

	// Get target account's organization to validate permissions
	var targetOrg *models.LogtoOrganization
	if currentAccount.CustomData != nil {
		if orgID, ok := currentAccount.CustomData["organizationId"].(string); ok {
			targetOrg, err = client.GetOrganizationByID(orgID)
			if err != nil {
				logger.RequestLogger(c, "accounts").Warn().
					Err(err).
					Str("operation", "fetch_target_organization").
					Msg("Failed to fetch target organization")
			}
		}
	}

	// Prevent self-deletion - critical security check
	if currentUserID.(string) == accountID {
		logger.LogAccountOperation(c, "delete_denied_self", accountID, "", currentUserID.(string), currentUserOrgID.(string), false, fmt.Errorf("attempted self-deletion"))
		c.JSON(http.StatusForbidden, response.Forbidden("cannot delete your own account", "self-deletion is not allowed for security reasons"))
		return
	}

	// Validate hierarchical permissions
	canOperate, reason := CanOperateOnAccount(
		currentUserOrgRole.(string),
		currentUserOrgID.(string),
		currentUserRole,
		currentAccount,
		targetOrg,
	)

	if !canOperate {
		logger.LogAccountOperation(c, "delete_denied", accountID, "", currentUserID.(string), currentUserOrgID.(string), false, fmt.Errorf("insufficient permissions: %s", reason))
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to delete this account", reason))
		return
	}

	// Delete the account from Logto
	if err := client.DeleteUser(accountID); err != nil {
		logger.NewHTTPErrorLogger(c, "accounts").LogError(err, "delete_account_logto", http.StatusInternalServerError, "Failed to delete account from Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to delete account", err.Error()))
		return
	}

	// Invalidate organization users cache to ensure fresh data on next request
	cacheManager := cache.GetOrgUsersCacheManager()
	if targetOrg != nil {
		cacheManager.Clear(targetOrg.ID)
	}

	logger.LogAccountOperation(c, "delete", accountID, "", currentUserID.(string), currentUserOrgID.(string), true, nil)
	c.JSON(http.StatusOK, response.OK("account deleted successfully", gin.H{
		"id":        accountID,
		"name":      currentAccount.Name,
		"deletedAt": time.Now(),
	}))
}

// Helper function to convert LogtoUser to AccountResponse
func convertLogtoUserToAccountResponse(account models.LogtoUser, org *models.LogtoOrganization) models.AccountResponse {
	var lastSignInAt *time.Time
	if account.LastSignInAt != nil && *account.LastSignInAt != 0 {
		t := time.Unix(*account.LastSignInAt/1000, 0)
		lastSignInAt = &t
	}

	accountResponse := models.AccountResponse{
		ID:           account.ID,
		Username:     account.Username,
		Email:        account.PrimaryEmail,
		Name:         account.Name,
		Phone:        "", // Will be set from customData
		Avatar:       account.Avatar,
		IsSuspended:  account.IsSuspended,
		LastSignInAt: lastSignInAt,
		CreatedAt:    time.Unix(account.CreatedAt/1000, 0),
		UpdatedAt:    time.Unix(account.UpdatedAt/1000, 0),
	}

	// Extract data from custom data
	if account.CustomData != nil {
		if userRoleIds, ok := account.CustomData["userRoleIds"].([]interface{}); ok {
			// Convert []interface{} to []string
			var roleIDs []string
			for _, roleID := range userRoleIds {
				if roleIDStr, ok := roleID.(string); ok {
					roleIDs = append(roleIDs, roleIDStr)
				}
			}
			accountResponse.UserRoleIDs = roleIDs
		}
		if orgID, ok := account.CustomData["organizationId"].(string); ok {
			accountResponse.OrganizationID = orgID
		}
		if orgRole, ok := account.CustomData["organizationRole"].(string); ok {
			accountResponse.OrganizationRole = orgRole
		}
		if phone, ok := account.CustomData["phone"].(string); ok {
			accountResponse.Phone = phone
		}

		// Extract custom data (excluding reserved fields)
		customData := make(map[string]interface{})
		for k, v := range account.CustomData {
			if k != "userRoleIds" && k != "userRoleId" && k != "organizationId" && k != "organizationRole" && k != "phone" && k != "createdBy" && k != "createdAt" && k != "updatedBy" && k != "updatedAt" {
				customData[k] = v
			}
		}
		accountResponse.CustomData = customData
	}

	// Set organization name if provided
	if org != nil {
		accountResponse.OrganizationName = org.Name
	}

	return accountResponse
}
