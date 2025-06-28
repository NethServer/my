/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: GPL-2.0-only
*/

package methods

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/fatih/structs"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/logs"
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
func CanOperateOnAccount(currentUserOrgRole, currentUserOrgID, currentUserRole string, targetAccount *services.LogtoUser, targetOrg *services.LogtoOrganization) (bool, string) {
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
	case "God":
		// God can operate on any account
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
func CanCreateAccountForOrganization(userOrgRole, userOrgID, userRole, targetOrgID, targetOrgRole string, targetOrg *services.LogtoOrganization) (bool, string) {
	// Only Admin users can create accounts for colleagues in the same organization
	if userOrgID == targetOrgID && userRole != "Admin" {
		return false, "only Admin users can create accounts for colleagues in the same organization"
	}

	switch userOrgRole {
	case "God":
		// God can create accounts for any organization
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
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Missing required user context in JWT token")
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
		logs.Logs.Printf("[ERROR][ACCOUNTS] User does not have Admin role required for account creation")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to create accounts", nil))
		return
	}

	// Connect to Logto Management API
	client := services.NewLogtoManagementClient()

	// Verify the target organization exists
	targetOrg, err := client.GetOrganizationByID(request.OrganizationID)
	if err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Target organization not found: %v", err)
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "target organization not found",
			Data:    err.Error(),
		}))
		return
	}

	// Get target organization's JIT roles to determine the default organization role
	jitRoles, err := client.GetOrganizationJitRoles(request.OrganizationID)
	if err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to get JIT roles for organization %s: %v", request.OrganizationID, err)
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "failed to get organization JIT roles",
			Data:    err.Error(),
		}))
		return
	}

	// Determine target organization role from JIT configuration
	var targetOrgRole string
	if len(jitRoles) > 0 {
		// Use the first JIT role as the default organization role
		targetOrgRole = jitRoles[0].Name
		logs.Logs.Printf("[INFO][ACCOUNTS] Using JIT role '%s' for organization %s", targetOrgRole, request.OrganizationID)
	} else {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "target organization has no JIT roles configured",
			Data:    nil,
		}))
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
		logs.Logs.Printf("[WARN][ACCOUNTS] User %s (role: %s, org: %s) denied creating account for org %s (role: %s): %s",
			currentUserID, currentUserOrgRole, currentUserOrgID, request.OrganizationID, targetOrgRole, reason)
		c.JSON(http.StatusForbidden, structs.Map(response.StatusNotFound{
			Code:    403,
			Message: "insufficient permissions to create account for this organization",
			Data:    reason,
		}))
		return
	}

	// Prepare custom data for the account
	customData := map[string]interface{}{
		"userRoleId":       request.UserRoleID,
		"organizationId":   request.OrganizationID,
		"organizationRole": targetOrgRole, // Derived from JIT configuration
		"createdBy":        currentUserOrgID,
		"createdAt":        time.Now().Format(time.RFC3339),
	}

	// Add phone to custom data if provided
	if request.Phone != "" {
		customData["phone"] = request.Phone
	}

	// Add metadata if provided
	if request.Metadata != nil {
		for k, v := range request.Metadata {
			customData[k] = v
		}
	}

	// Sanitize fields for Logto compliance
	sanitizedUsername := sanitizeUsernameForLogto(request.Username)

	// Create account request for Logto
	accountRequest := services.CreateUserRequest{
		Username:     sanitizedUsername,
		PrimaryEmail: request.Email,
		Name:         request.Name,
		Password:     request.Password,
		CustomData:   customData,
	}

	// Phone is stored in customData, not as primaryPhone

	// Only set avatar if it's not empty
	if request.Avatar != "" {
		accountRequest.Avatar = &request.Avatar
	}

	// Debug: log the request being sent to Logto
	if reqJSON, err := json.Marshal(accountRequest); err == nil {
		logs.Logs.Printf("[DEBUG][ACCOUNTS] Sending to Logto: %s", string(reqJSON))
	}

	// Create the account in Logto
	account, err := client.CreateUser(accountRequest)
	if err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to create account in Logto: %v", err)

		// Try to parse and format the error message better
		errorMsg := err.Error()
		var detailedError interface{}

		// Check for different status codes and extract JSON
		statusPrefixes := []string{"status 400: ", "status 422: ", "status 409: ", "status 500: "}
		for _, prefix := range statusPrefixes {
			if strings.Contains(errorMsg, prefix) {
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

		// If no status prefix found, use original error
		if detailedError == nil {
			detailedError = errorMsg
		}

		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "failed to create account",
			Data:    detailedError,
		}))
		return
	}

	// Assign user role using ID directly (more secure)
	if request.UserRoleID != "" {
		if err := client.AssignUserRoles(account.ID, []string{request.UserRoleID}); err != nil {
			logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to assign user role %s: %v", request.UserRoleID, err)
		}
	}

	// Step 1: Assign user to organization (without roles)
	if err := client.AssignUserToOrganization(request.OrganizationID, account.ID); err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to assign account to organization: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to assign account to organization",
			Data:    err.Error(),
		}))
		return
	}

	// Step 2: Assign organization role using the specific API endpoint
	if len(jitRoles) > 0 {
		// Use the first JIT role for the organization
		roleIDs := []string{jitRoles[0].ID}
		roleNames := []string{jitRoles[0].Name}

		logs.Logs.Printf("[INFO][ACCOUNTS] Assigning organization role '%s' (ID: %s) to user %s in organization %s",
			jitRoles[0].Name, jitRoles[0].ID, account.ID, request.OrganizationID)

		if err := client.AssignOrganizationRolesToUser(request.OrganizationID, account.ID, roleIDs, roleNames); err != nil {
			logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to assign organization role to user: %v", err)
		} else {
			logs.Logs.Printf("[SUCCESS][ACCOUNTS] Successfully assigned organization role '%s' to user %s", jitRoles[0].Name, account.ID)
		}
	}

	logs.Logs.Printf("[INFO][ACCOUNTS] Account created in Logto: %s (ID: %s) by user %s", account.Name, account.ID, currentUserID)

	// Convert to response format
	var lastSignInAt *time.Time
	if account.LastSignInAt != nil {
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
		UserRole:         request.UserRoleID, // Note: This should be resolved to name in response
		OrganizationID:   request.OrganizationID,
		OrganizationName: targetOrg.Name,
		OrganizationRole: targetOrgRole, // Derived from JIT configuration
		IsSuspended:      account.IsSuspended,
		LastSignInAt:     lastSignInAt,
		CreatedAt:        time.Unix(account.CreatedAt/1000, 0),
		UpdatedAt:        time.Unix(account.UpdatedAt/1000, 0),
		Metadata:         request.Metadata,
	}

	c.JSON(http.StatusCreated, structs.Map(response.StatusOK{
		Code:    201,
		Message: "account created successfully",
		Data:    accountResponse,
	}))
}

// GetAccounts handles GET /api/accounts - retrieves accounts with organization filtering
func GetAccounts(c *gin.Context) {
	currentUserID, _ := c.Get("user_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserOrgID, _ := c.Get("organization_id")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Missing required user context in JWT token")
		c.JSON(http.StatusUnauthorized, structs.Map(response.StatusUnauthorized{
			Code:    401,
			Message: "incomplete user context in token",
			Data:    nil,
		}))
		return
	}

	logs.Logs.Printf("[INFO][ACCOUNTS] Accounts list requested by user %s (role: %s, org: %s)", currentUserID, currentUserOrgRole, currentUserOrgID)

	client := services.NewLogtoManagementClient()

	// Get organization filter from query parameter
	orgFilter := c.Query("organizationId")

	var accounts []models.AccountResponse

	if orgFilter != "" {
		// Get accounts from specific organization
		orgAccounts, err := client.GetOrganizationUsers(orgFilter)
		if err != nil {
			logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to fetch organization accounts: %v", err)
			c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
				Code:    500,
				Message: "failed to fetch organization accounts",
				Data:    err.Error(),
			}))
			return
		}

		// Get organization details
		org, err := client.GetOrganizationByID(orgFilter)
		if err != nil {
			logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to fetch organization details: %v", err)
		}

		// Convert to response format
		for _, account := range orgAccounts {
			accountResponse := convertLogtoUserToAccountResponse(account, org)
			accounts = append(accounts, accountResponse)
		}
	} else {
		// Get all organizations visible to current user and their accounts
		allOrgs, err := services.GetAllVisibleOrganizations(currentUserOrgRole.(string), currentUserOrgID.(string))
		if err != nil {
			logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to fetch visible organizations: %v", err)
			c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
				Code:    500,
				Message: "failed to fetch organizations",
				Data:    err.Error(),
			}))
			return
		}

		// Get accounts from each visible organization
		for _, org := range allOrgs {
			orgAccounts, err := client.GetOrganizationUsers(org.ID)
			if err != nil {
				logs.Logs.Printf("[WARN][ACCOUNTS] Failed to fetch accounts for org %s: %v", org.ID, err)
				continue
			}

			for _, account := range orgAccounts {
				accountResponse := convertLogtoUserToAccountResponse(account, &org)
				accounts = append(accounts, accountResponse)
			}
		}
	}

	logs.Logs.Printf("[INFO][ACCOUNTS] Retrieved %d accounts", len(accounts))

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "accounts retrieved successfully",
		Data:    gin.H{"accounts": accounts, "count": len(accounts)},
	}))
}

// UpdateAccount handles PUT /api/accounts/:id - updates an existing account
func UpdateAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "account ID required",
			Data:    nil,
		}))
		return
	}

	var request models.UpdateAccountRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "request fields malformed",
			Data:    err.Error(),
		}))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Missing required user context in JWT token")
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
		logs.Logs.Printf("[ERROR][ACCOUNTS] User does not have Admin role required for account operations")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to modify accounts", nil))
		return
	}

	client := services.NewLogtoManagementClient()

	// Get current account data
	currentAccount, err := client.GetUserByID(accountID)
	if err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to fetch account: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "account not found",
			Data:    nil,
		}))
		return
	}

	// Get target account's organization to validate permissions
	var targetOrg *services.LogtoOrganization
	if currentAccount.CustomData != nil {
		if orgID, ok := currentAccount.CustomData["organizationId"].(string); ok {
			targetOrg, err = client.GetOrganizationByID(orgID)
			if err != nil {
				logs.Logs.Printf("[WARN][ACCOUNTS] Failed to fetch target organization: %v", err)
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
		logs.Logs.Printf("[WARN][ACCOUNTS] User %s (role: %s, org: %s) denied updating account %s: %s",
			currentUserID, currentUserOrgRole, currentUserOrgID, accountID, reason)
		c.JSON(http.StatusForbidden, structs.Map(response.StatusNotFound{
			Code:    403,
			Message: "insufficient permissions to update this account",
			Data:    reason,
		}))
		return
	}

	// Prepare update request
	updateRequest := services.UpdateUserRequest{}

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
		if request.UserRoleID != "" {
			updateRequest.CustomData["userRoleId"] = request.UserRoleID
		}
		if request.OrganizationID != "" {
			updateRequest.CustomData["organizationId"] = request.OrganizationID
		}
		// OrganizationRole is now managed via JIT provisioning
		if request.Metadata != nil {
			for k, v := range request.Metadata {
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
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to update account in Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to update account",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][ACCOUNTS] Account updated in Logto: %s (ID: %s) by user %s", updatedAccount.Name, updatedAccount.ID, currentUserID)

	// Convert to response format
	accountResponse := convertLogtoUserToAccountResponse(*updatedAccount, nil)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "account updated successfully",
		Data:    accountResponse,
	}))
}

// DeleteAccount handles DELETE /api/accounts/:id - deletes an account
func DeleteAccount(c *gin.Context) {
	accountID := c.Param("id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, structs.Map(response.StatusNotFound{
			Code:    400,
			Message: "account ID required",
			Data:    nil,
		}))
		return
	}

	currentUserID, _ := c.Get("user_id")
	currentUserOrgID, _ := c.Get("organization_id")
	currentUserOrgRole, _ := c.Get("org_role")
	currentUserRoles, _ := c.Get("user_roles")

	// Validate required user context
	if currentUserOrgRole == nil || currentUserOrgID == nil || currentUserRoles == nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Missing required user context in JWT token")
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
		logs.Logs.Printf("[ERROR][ACCOUNTS] User does not have Admin role required for account operations")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to delete accounts", nil))
		return
	}

	client := services.NewLogtoManagementClient()

	// Get account data for logging before deletion
	currentAccount, err := client.GetUserByID(accountID)
	if err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to fetch account for deletion: %v", err)
		c.JSON(http.StatusNotFound, structs.Map(response.StatusNotFound{
			Code:    404,
			Message: "account not found",
			Data:    nil,
		}))
		return
	}

	// Get target account's organization to validate permissions
	var targetOrg *services.LogtoOrganization
	if currentAccount.CustomData != nil {
		if orgID, ok := currentAccount.CustomData["organizationId"].(string); ok {
			targetOrg, err = client.GetOrganizationByID(orgID)
			if err != nil {
				logs.Logs.Printf("[WARN][ACCOUNTS] Failed to fetch target organization: %v", err)
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
		logs.Logs.Printf("[WARN][ACCOUNTS] User %s (role: %s, org: %s) denied deleting account %s: %s",
			currentUserID, currentUserOrgRole, currentUserOrgID, accountID, reason)
		c.JSON(http.StatusForbidden, structs.Map(response.StatusNotFound{
			Code:    403,
			Message: "insufficient permissions to delete this account",
			Data:    reason,
		}))
		return
	}

	// Delete the account from Logto
	if err := client.DeleteUser(accountID); err != nil {
		logs.Logs.Printf("[ERROR][ACCOUNTS] Failed to delete account from Logto: %v", err)
		c.JSON(http.StatusInternalServerError, structs.Map(response.StatusInternalServerError{
			Code:    500,
			Message: "failed to delete account",
			Data:    err.Error(),
		}))
		return
	}

	logs.Logs.Printf("[INFO][ACCOUNTS] Account deleted from Logto: %s (ID: %s) by user %s", currentAccount.Name, accountID, currentUserID)

	c.JSON(http.StatusOK, structs.Map(response.StatusOK{
		Code:    200,
		Message: "account deleted successfully",
		Data: gin.H{
			"id":        accountID,
			"name":      currentAccount.Name,
			"deletedAt": time.Now(),
		},
	}))
}

// Helper function to convert LogtoUser to AccountResponse
func convertLogtoUserToAccountResponse(account services.LogtoUser, org *services.LogtoOrganization) models.AccountResponse {
	var lastSignInAt *time.Time
	if account.LastSignInAt != nil {
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
		if userRoleId, ok := account.CustomData["userRoleId"].(string); ok {
			accountResponse.UserRole = userRoleId // Note: Should resolve to role name for display
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

		// Extract metadata
		metadata := make(map[string]string)
		for k, v := range account.CustomData {
			if k != "userRoleId" && k != "organizationId" && k != "organizationRole" && k != "phone" && k != "createdBy" && k != "createdAt" && k != "updatedBy" && k != "updatedAt" {
				if str, ok := v.(string); ok {
					metadata[k] = str
				}
			}
		}
		accountResponse.Metadata = metadata
	}

	// Set organization name if provided
	if org != nil {
		accountResponse.OrganizationName = org.Name
	}

	return accountResponse
}
