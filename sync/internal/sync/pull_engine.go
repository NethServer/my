/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/database"
	"github.com/nethesis/my/sync/internal/logger"
	"github.com/nethesis/my/sync/internal/models"
)

// PullEngine handles the reverse synchronization process (from Logto to local)
type PullEngine struct {
	client  *client.LogtoClient
	options *PullOptions
}

// PullOptions contains pull operation options
type PullOptions struct {
	DryRun            bool
	Verbose           bool
	OrganizationsOnly bool
	UsersOnly         bool
	ResourcesOnly     bool
	ConflictStrategy  string
	OverwriteAll      bool
	Force             bool
	PurgeLocal        bool
	BackupBefore      bool
	DatabaseURL       string
	APIBaseURL        string
}

// PullResult contains the results of a pull operation
type PullResult struct {
	StartTime  time.Time       `json:"start_time" yaml:"start_time"`
	EndTime    time.Time       `json:"end_time" yaml:"end_time"`
	Duration   time.Duration   `json:"duration" yaml:"duration"`
	DryRun     bool            `json:"dry_run" yaml:"dry_run"`
	Success    bool            `json:"success" yaml:"success"`
	Summary    *PullSummary    `json:"summary" yaml:"summary"`
	Operations []PullOperation `json:"operations" yaml:"operations"`
	Conflicts  []PullConflict  `json:"conflicts,omitempty" yaml:"conflicts,omitempty"`
	Errors     []string        `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// PullSummary contains a summary of pull changes
type PullSummary struct {
	OrganizationsCreated int `json:"organizations_created" yaml:"organizations_created"`
	OrganizationsUpdated int `json:"organizations_updated" yaml:"organizations_updated"`
	OrganizationsSkipped int `json:"organizations_skipped" yaml:"organizations_skipped"`
	UsersCreated         int `json:"users_created" yaml:"users_created"`
	UsersUpdated         int `json:"users_updated" yaml:"users_updated"`
	UsersSkipped         int `json:"users_skipped" yaml:"users_skipped"`
	ResourcesCreated     int `json:"resources_created" yaml:"resources_created"`
	ResourcesUpdated     int `json:"resources_updated" yaml:"resources_updated"`
	ResourcesSkipped     int `json:"resources_skipped" yaml:"resources_skipped"`
	RolesCreated         int `json:"roles_created" yaml:"roles_created"`
	RolesUpdated         int `json:"roles_updated" yaml:"roles_updated"`
	RolesSkipped         int `json:"roles_skipped" yaml:"roles_skipped"`
	PermissionsCreated   int `json:"permissions_created" yaml:"permissions_created"`
	PermissionsUpdated   int `json:"permissions_updated" yaml:"permissions_updated"`
	PermissionsSkipped   int `json:"permissions_skipped" yaml:"permissions_skipped"`
	ConflictsDetected    int `json:"conflicts_detected" yaml:"conflicts_detected"`
	ConflictsResolved    int `json:"conflicts_resolved" yaml:"conflicts_resolved"`
}

// PullOperation represents a single pull operation performed
type PullOperation struct {
	Type        string    `json:"type" yaml:"type"`
	Action      string    `json:"action" yaml:"action"`
	Resource    string    `json:"resource" yaml:"resource"`
	Description string    `json:"description" yaml:"description"`
	Success     bool      `json:"success" yaml:"success"`
	Conflict    bool      `json:"conflict,omitempty" yaml:"conflict,omitempty"`
	Error       string    `json:"error,omitempty" yaml:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp" yaml:"timestamp"`
}

// PullConflict represents a conflict detected during pull
type PullConflict struct {
	Type        string      `json:"type" yaml:"type"`
	Resource    string      `json:"resource" yaml:"resource"`
	Description string      `json:"description" yaml:"description"`
	LocalValue  interface{} `json:"local_value" yaml:"local_value"`
	LogtoValue  interface{} `json:"logto_value" yaml:"logto_value"`
	Resolution  string      `json:"resolution" yaml:"resolution"`
	Timestamp   time.Time   `json:"timestamp" yaml:"timestamp"`
}

// NewPullEngine creates a new pull synchronization engine
func NewPullEngine(client *client.LogtoClient, options *PullOptions) *PullEngine {
	if options == nil {
		options = &PullOptions{}
	}

	return &PullEngine{
		client:  client,
		options: options,
	}
}

// Pull performs the reverse synchronization (from Logto to local)
func (e *PullEngine) Pull() (*PullResult, error) {
	result := &PullResult{
		StartTime:  time.Now(),
		DryRun:     e.options.DryRun,
		Summary:    &PullSummary{},
		Operations: []PullOperation{},
		Conflicts:  []PullConflict{},
		Errors:     []string{},
	}

	logger.Info("Starting pull from Logto to local database")

	if e.options.DryRun {
		logger.Info("Running in dry-run mode - no changes will be made")
	}

	// Pull organizations and organization roles
	if !e.options.UsersOnly && !e.options.ResourcesOnly {
		if err := e.pullOrganizations(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Organizations pull failed: %v", err))
		}

		if err := e.pullOrganizationRoles(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Organization roles pull failed: %v", err))
		}
	}

	// Pull users and user roles
	if !e.options.OrganizationsOnly && !e.options.ResourcesOnly {
		if err := e.pullUsers(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Users pull failed: %v", err))
		}

		if err := e.pullUserRoles(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("User roles pull failed: %v", err))
		}
	}

	// Pull resources and permissions
	if !e.options.OrganizationsOnly && !e.options.UsersOnly {
		if err := e.pullResources(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Resources pull failed: %v", err))
		}

		if err := e.pullPermissions(result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Permissions pull failed: %v", err))
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0

	if result.Success {
		logger.Info("Pull completed successfully in %v", result.Duration)
	} else {
		logger.Error("Pull completed with %d errors in %v", len(result.Errors), result.Duration)
	}

	return result, nil
}

// addPullOperation adds an operation to the result
func (e *PullEngine) addPullOperation(result *PullResult, opType, action, resource, description string, conflict bool, err error) {
	op := PullOperation{
		Type:        opType,
		Action:      action,
		Resource:    resource,
		Description: description,
		Success:     err == nil,
		Conflict:    conflict,
		Timestamp:   time.Now(),
	}

	if err != nil {
		op.Error = err.Error()
		logger.LogSyncOperation(opType, resource, action, false, err)
	} else {
		logger.LogSyncOperation(opType, resource, action, true, nil)
	}

	result.Operations = append(result.Operations, op)
}

// Placeholder methods for actual pull operations
// These would be implemented to fetch data from Logto and update local database

func (e *PullEngine) pullOrganizations(result *PullResult) error {
	logger.Info("Pulling organizations from Logto...")

	// Fetch organizations from Logto
	logtoOrgs, err := e.client.GetOrganizations()
	if err != nil {
		return fmt.Errorf("failed to fetch organizations from Logto: %w", err)
	}

	logger.Info("Found %d organizations in Logto", len(logtoOrgs))

	if e.options.DryRun {
		processableOrgs := 0
		for _, org := range logtoOrgs {
			if org.Name == "Owner" {
				logger.Info("DRY RUN: Would skip organization 'Owner' (Logto-only)")
				continue
			}
			logger.Info("DRY RUN: Would process organization '%s' (ID: %s)", org.Name, org.ID)
			processableOrgs++
		}
		logger.Info("DRY RUN: Would process %d organizations from Logto (skipped %d)", processableOrgs, len(logtoOrgs)-processableOrgs)
		e.addPullOperation(result, "organization", "pull", "organizations", fmt.Sprintf("%d organizations from Logto", processableOrgs), false, nil)
		return nil
	}

	// Process each organization
	for _, logtoOrg := range logtoOrgs {
		// Skip Owner organization - it should only exist in Logto, not in local database
		if logtoOrg.Name == "Owner" {
			logger.Debug("Skipping Owner organization - exists only in Logto")
			result.Summary.OrganizationsSkipped++
			e.addPullOperation(result, "organization", "skip", logtoOrg.Name, "Owner organization skipped (Logto-only)", false, nil)
			continue
		}

		if err := e.processOrganization(logtoOrg, result); err != nil {
			logger.Error("Failed to process organization '%s': %v", logtoOrg.Name, err)
			e.addPullOperation(result, "organization", "create/update", logtoOrg.Name, fmt.Sprintf("Process organization %s", logtoOrg.Name), false, err)
			continue
		}
	}

	return nil
}

// processOrganization processes a single organization from Logto
func (e *PullEngine) processOrganization(logtoOrg client.LogtoOrganization, result *PullResult) error {
	// Determine organization type based on custom_data.type
	orgType := e.determineOrganizationType(logtoOrg.CustomData)

	logger.Debug("Processing organization '%s' as type '%s'", logtoOrg.Name, orgType)

	switch orgType {
	case "distributor":
		return e.processDistributor(logtoOrg, result)
	case "reseller":
		return e.processReseller(logtoOrg, result)
	case "customer":
		return e.processCustomer(logtoOrg, result)
	default:
		logger.Warn("Unknown organization type for '%s', skipping", logtoOrg.Name)
		result.Summary.OrganizationsSkipped++
		return nil
	}
}

// determineOrganizationType determines the type of organization based on custom_data.type
func (e *PullEngine) determineOrganizationType(customData map[string]interface{}) string {
	// Check custom_data.type field for organization type
	if customData != nil {
		if orgType, ok := customData["type"].(string); ok && orgType != "" {
			switch orgType {
			case "distributor", "reseller", "customer":
				return orgType
			default:
				logger.Warn("Unknown organization type in custom_data: '%s', defaulting to customer", orgType)
				return "customer"
			}
		}
	}

	// If custom_data.type is not available, default to customer
	logger.Warn("Organization missing custom_data.type, defaulting to customer")
	return "customer"
}

// processDistributor processes a distributor organization
func (e *PullEngine) processDistributor(logtoOrg client.LogtoOrganization, result *PullResult) error {
	// Check if distributor already exists
	var existingID string
	err := database.DB.QueryRow(
		"SELECT id FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL",
		logtoOrg.ID,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new distributor
		distributor := models.LocalDistributor{
			ID:            uuid.New().String(),
			LogtoID:       &logtoOrg.ID,
			Name:          logtoOrg.Name,
			Description:   logtoOrg.Description,
			CustomData:    logtoOrg.CustomData,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			LogtoSyncedAt: &time.Time{},
		}
		*distributor.LogtoSyncedAt = time.Now()

		// Serialize custom data
		customDataJSON, err := json.Marshal(distributor.CustomData)
		if err != nil {
			return fmt.Errorf("failed to marshal custom data: %w", err)
		}

		_, err = database.DB.Exec(`
			INSERT INTO distributors (id, logto_id, name, description, custom_data, created_at, updated_at, logto_synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			distributor.ID, distributor.LogtoID, distributor.Name, distributor.Description,
			customDataJSON, distributor.CreatedAt, distributor.UpdatedAt, distributor.LogtoSyncedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to create distributor: %w", err)
		}

		logger.Info("Created distributor '%s' with ID %s", distributor.Name, distributor.ID)
		e.addPullOperation(result, "distributor", "create", logtoOrg.Name, fmt.Sprintf("Created distributor %s", logtoOrg.Name), false, nil)
		result.Summary.OrganizationsCreated++

	} else if err != nil {
		return fmt.Errorf("failed to check existing distributor: %w", err)
	} else {
		// Update existing distributor
		customDataJSON, err := json.Marshal(logtoOrg.CustomData)
		if err != nil {
			return fmt.Errorf("failed to marshal custom data: %w", err)
		}

		_, err = database.DB.Exec(`
			UPDATE distributors
			SET name = $1, description = $2, custom_data = $3, updated_at = $4, logto_synced_at = $5
			WHERE id = $6`,
			logtoOrg.Name, logtoOrg.Description, customDataJSON, time.Now(), time.Now(), existingID,
		)

		if err != nil {
			return fmt.Errorf("failed to update distributor: %w", err)
		}

		logger.Info("Updated distributor '%s' with ID %s", logtoOrg.Name, existingID)
		e.addPullOperation(result, "distributor", "update", logtoOrg.Name, fmt.Sprintf("Updated distributor %s", logtoOrg.Name), false, nil)
		result.Summary.OrganizationsUpdated++
	}

	return nil
}

// processReseller processes a reseller organization
func (e *PullEngine) processReseller(logtoOrg client.LogtoOrganization, result *PullResult) error {
	// Check if reseller already exists
	var existingID string
	err := database.DB.QueryRow(
		"SELECT id FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL",
		logtoOrg.ID,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new reseller
		reseller := models.LocalReseller{
			ID:            uuid.New().String(),
			LogtoID:       &logtoOrg.ID,
			Name:          logtoOrg.Name,
			Description:   logtoOrg.Description,
			CustomData:    logtoOrg.CustomData,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			LogtoSyncedAt: &time.Time{},
		}
		*reseller.LogtoSyncedAt = time.Now()

		// Serialize custom data
		customDataJSON, err := json.Marshal(reseller.CustomData)
		if err != nil {
			return fmt.Errorf("failed to marshal custom data: %w", err)
		}

		_, err = database.DB.Exec(`
			INSERT INTO resellers (id, logto_id, name, description, custom_data, created_at, updated_at, logto_synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			reseller.ID, reseller.LogtoID, reseller.Name, reseller.Description,
			customDataJSON, reseller.CreatedAt, reseller.UpdatedAt, reseller.LogtoSyncedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to create reseller: %w", err)
		}

		logger.Info("Created reseller '%s' with ID %s", reseller.Name, reseller.ID)
		e.addPullOperation(result, "reseller", "create", logtoOrg.Name, fmt.Sprintf("Created reseller %s", logtoOrg.Name), false, nil)
		result.Summary.OrganizationsCreated++

	} else if err != nil {
		return fmt.Errorf("failed to check existing reseller: %w", err)
	} else {
		// Update existing reseller
		customDataJSON, err := json.Marshal(logtoOrg.CustomData)
		if err != nil {
			return fmt.Errorf("failed to marshal custom data: %w", err)
		}

		_, err = database.DB.Exec(`
			UPDATE resellers
			SET name = $1, description = $2, custom_data = $3, updated_at = $4, logto_synced_at = $5
			WHERE id = $6`,
			logtoOrg.Name, logtoOrg.Description, customDataJSON, time.Now(), time.Now(), existingID,
		)

		if err != nil {
			return fmt.Errorf("failed to update reseller: %w", err)
		}

		logger.Info("Updated reseller '%s' with ID %s", logtoOrg.Name, existingID)
		e.addPullOperation(result, "reseller", "update", logtoOrg.Name, fmt.Sprintf("Updated reseller %s", logtoOrg.Name), false, nil)
		result.Summary.OrganizationsUpdated++
	}

	return nil
}

// processCustomer processes a customer organization
func (e *PullEngine) processCustomer(logtoOrg client.LogtoOrganization, result *PullResult) error {
	// Check if customer already exists
	var existingID string
	err := database.DB.QueryRow(
		"SELECT id FROM customers WHERE logto_id = $1 AND deleted_at IS NULL",
		logtoOrg.ID,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Create new customer
		customer := models.LocalCustomer{
			ID:            uuid.New().String(),
			LogtoID:       &logtoOrg.ID,
			Name:          logtoOrg.Name,
			Description:   logtoOrg.Description,
			CustomData:    logtoOrg.CustomData,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
			LogtoSyncedAt: &time.Time{},
		}
		*customer.LogtoSyncedAt = time.Now()

		// Serialize custom data
		customDataJSON, err := json.Marshal(customer.CustomData)
		if err != nil {
			return fmt.Errorf("failed to marshal custom data: %w", err)
		}

		_, err = database.DB.Exec(`
			INSERT INTO customers (id, logto_id, name, description, custom_data, created_at, updated_at, logto_synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			customer.ID, customer.LogtoID, customer.Name, customer.Description,
			customDataJSON, customer.CreatedAt, customer.UpdatedAt, customer.LogtoSyncedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to create customer: %w", err)
		}

		logger.Info("Created customer '%s' with ID %s", customer.Name, customer.ID)
		e.addPullOperation(result, "customer", "create", logtoOrg.Name, fmt.Sprintf("Created customer %s", logtoOrg.Name), false, nil)
		result.Summary.OrganizationsCreated++

	} else if err != nil {
		return fmt.Errorf("failed to check existing customer: %w", err)
	} else {
		// Update existing customer
		customDataJSON, err := json.Marshal(logtoOrg.CustomData)
		if err != nil {
			return fmt.Errorf("failed to marshal custom data: %w", err)
		}

		_, err = database.DB.Exec(`
			UPDATE customers
			SET name = $1, description = $2, custom_data = $3, updated_at = $4, logto_synced_at = $5
			WHERE id = $6`,
			logtoOrg.Name, logtoOrg.Description, customDataJSON, time.Now(), time.Now(), existingID,
		)

		if err != nil {
			return fmt.Errorf("failed to update customer: %w", err)
		}

		logger.Info("Updated customer '%s' with ID %s", logtoOrg.Name, existingID)
		e.addPullOperation(result, "customer", "update", logtoOrg.Name, fmt.Sprintf("Updated customer %s", logtoOrg.Name), false, nil)
		result.Summary.OrganizationsUpdated++
	}

	return nil
}

func (e *PullEngine) pullOrganizationRoles(result *PullResult) error {
	logger.Info("Pulling organization roles from Logto...")

	// Organization roles define the business hierarchy roles (Owner, Distributor, Reseller, Customer)
	// These are managed in Logto and used for organization-level permissions
	// For the current architecture, these roles are primarily used for access control
	// rather than stored as local entities

	if e.options.DryRun {
		logger.Info("DRY RUN: Would acknowledge organization roles from Logto (RBAC hierarchy roles)")
		e.addPullOperation(result, "role", "acknowledge", "organization-roles", "Organization roles acknowledged from Logto", false, nil)
		return nil
	}

	// Organization roles are part of the RBAC system managed in Logto
	logger.Info("Organization roles acknowledged - RBAC hierarchy managed in Logto")
	e.addPullOperation(result, "role", "acknowledge", "organization-roles", "Organization roles acknowledged from Logto", false, nil)

	return nil
}

func (e *PullEngine) pullUsers(result *PullResult) error {
	logger.Info("Pulling users from Logto...")

	// Fetch users from Logto
	logtoUsers, err := e.client.GetUsers()
	if err != nil {
		return fmt.Errorf("failed to fetch users from Logto: %w", err)
	}

	logger.Info("Found %d users in Logto", len(logtoUsers))

	if e.options.DryRun {
		processableUsers := 0
		for _, user := range logtoUsers {
			userName := getUserName(user)
			userEmail := getUserEmail(user)

			// Skip Owner users (similar to Owner organization)
			if isOwnerUser(userName) {
				logger.Info("DRY RUN: Would skip user '%s' (Owner user - Logto-only)", userName)
				continue
			}
			logger.Info("DRY RUN: Would process user '%s' (%s)", userName, userEmail)
			processableUsers++
		}
		logger.Info("DRY RUN: Would process %d users from Logto (skipped %d)", processableUsers, len(logtoUsers)-processableUsers)
		e.addPullOperation(result, "user", "pull", "users", fmt.Sprintf("%d users from Logto", processableUsers), false, nil)
		return nil
	}

	// Process each user
	for _, logtoUser := range logtoUsers {
		userName := getUserName(logtoUser)

		// Skip Owner users - they should only exist in Logto, not in local database
		if isOwnerUser(userName) {
			logger.Debug("Skipping Owner user '%s' - exists only in Logto", userName)
			result.Summary.UsersSkipped++
			e.addPullOperation(result, "user", "skip", userName, "Owner user skipped (Logto-only)", false, nil)
			continue
		}

		if err := e.processUser(logtoUser, result); err != nil {
			logger.Error("Failed to process user '%s': %v", userName, err)
			e.addPullOperation(result, "user", "create/update", userName, fmt.Sprintf("Process user %s", userName), false, err)
			continue
		}
	}

	return nil
}

// getUserName extracts user name from Logto user data
func getUserName(user map[string]interface{}) string {
	if name, ok := user["name"].(string); ok && name != "" {
		return name
	}
	if username, ok := user["username"].(string); ok && username != "" {
		return username
	}
	if email, ok := user["primaryEmail"].(string); ok && email != "" {
		return email
	}
	return "Unknown User"
}

// getUserEmail extracts user email from Logto user data
func getUserEmail(user map[string]interface{}) string {
	if email, ok := user["primaryEmail"].(string); ok && email != "" {
		return email
	}
	if emails, ok := user["identities"].(map[string]interface{}); ok {
		if email, ok := emails["email"].(string); ok && email != "" {
			return email
		}
	}
	return ""
}

// isOwnerUser determines if a user is an Owner user that should be skipped
func isOwnerUser(name string) bool {
	// Check if username is "owner"
	return name == "owner"
}

// processUser processes a single user from Logto
func (e *PullEngine) processUser(logtoUser map[string]interface{}, result *PullResult) error {
	userName := getUserName(logtoUser)
	userEmail := getUserEmail(logtoUser)
	userID := ""
	if id, ok := logtoUser["id"].(string); ok {
		userID = id
	}

	logger.Debug("Processing user '%s' (%s)", userName, userEmail)

	// For reverse sync, we'll assign users to the Owner organization so they're visible in the API
	// This is a simplified approach - in a full implementation, we'd fetch actual organization memberships from Logto
	ownerOrgID, err := e.getOwnerOrganizationID()
	if err != nil {
		logger.Warn("Could not get Owner organization ID, user may not be visible in API: %v", err)
	}

	// Check if user already exists
	var existingID string
	var existingLogtoID *string
	err = database.DB.QueryRow(
		"SELECT id, logto_id FROM users WHERE logto_id = $1 AND deleted_at IS NULL",
		userID,
	).Scan(&existingID, &existingLogtoID)

	if err == sql.ErrNoRows {
		// Get user roles from Logto
		userRoles, err := e.client.GetUserRoles(userID)
		if err != nil {
			logger.Warn("Failed to get roles for user '%s': %v", userName, err)
			userRoles = []string{} // Default to empty roles
		}

		// Create new user
		user := models.LocalUser{
			ID:             uuid.New().String(),
			LogtoID:        &userID,
			Username:       userName,
			Email:          userEmail,
			Name:           userName,
			OrganizationID: ownerOrgID, // Assign to Owner org for visibility
			UserRoleIDs:    userRoles,  // Use actual roles from Logto
			CustomData:     make(map[string]interface{}),
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			LogtoSyncedAt:  &time.Time{},
		}
		*user.LogtoSyncedAt = time.Now()

		// Extract phone if available
		if phone, ok := logtoUser["primaryPhone"].(string); ok && phone != "" {
			user.Phone = &phone
		}

		// Serialize user roles to JSON
		userRolesJSON, err := json.Marshal(user.UserRoleIDs)
		if err != nil {
			return fmt.Errorf("failed to marshal user roles: %w", err)
		}

		_, err = database.DB.Exec(`
			INSERT INTO users (id, logto_id, username, email, name, phone, organization_id, user_role_ids, custom_data, created_at, updated_at, logto_synced_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			user.ID, user.LogtoID, user.Username, user.Email, user.Name, user.Phone,
			user.OrganizationID, userRolesJSON, "{}", user.CreatedAt, user.UpdatedAt, user.LogtoSyncedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		logger.Info("Created user '%s' (%s) with ID %s", user.Name, user.Email, user.ID)
		e.addPullOperation(result, "user", "create", userName, fmt.Sprintf("Created user %s", userName), false, nil)
		result.Summary.UsersCreated++

	} else if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	} else {
		// Update existing user
		var phone *string
		if phoneVal, ok := logtoUser["primaryPhone"].(string); ok && phoneVal != "" {
			phone = &phoneVal
		}

		// Get user roles from Logto
		userRoles, err := e.client.GetUserRoles(userID)
		if err != nil {
			logger.Warn("Failed to get roles for user '%s': %v", userName, err)
			userRoles = []string{} // Default to empty roles
		}

		// Serialize roles to JSON
		userRolesJSON, err := json.Marshal(userRoles)
		if err != nil {
			return fmt.Errorf("failed to marshal user roles: %w", err)
		}

		_, err = database.DB.Exec(`
			UPDATE users
			SET username = $1, email = $2, name = $3, phone = $4, organization_id = $5, user_role_ids = $6, updated_at = $7, logto_synced_at = $8
			WHERE id = $9`,
			userName, userEmail, userName, phone, ownerOrgID, userRolesJSON, time.Now(), time.Now(), existingID,
		)

		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		logger.Info("Updated user '%s' (%s) with ID %s", userName, userEmail, existingID)
		e.addPullOperation(result, "user", "update", userName, fmt.Sprintf("Updated user %s", userName), false, nil)
		result.Summary.UsersUpdated++
	}

	return nil
}

// getOwnerOrganizationID gets the Owner organization ID from the database
// Since Owner org is not synced to local database, we'll use the first distributor or nil
func (e *PullEngine) getOwnerOrganizationID() (*string, error) {
	var orgID string
	// Try to find any distributor organization - users will be assigned there for visibility
	// IMPORTANT: Use logto_id (not local id) because the API filters by Logto organization IDs
	err := database.DB.QueryRow("SELECT logto_id FROM distributors WHERE logto_id IS NOT NULL AND deleted_at IS NULL LIMIT 1").Scan(&orgID)
	if err == sql.ErrNoRows {
		// No distributors found, return nil (users won't have organization)
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query distributors: %w", err)
	}
	return &orgID, nil
}

func (e *PullEngine) pullUserRoles(result *PullResult) error {
	logger.Info("Pulling user roles from Logto...")

	// For now, user role assignments are handled differently in the system
	// The actual role assignments happen through organization memberships
	// and direct role assignments via the Management API

	// This is a placeholder that acknowledges the user roles exist in Logto
	// but doesn't sync them to local database as they're managed differently

	if e.options.DryRun {
		logger.Info("DRY RUN: Would acknowledge user roles from Logto (managed via organization memberships)")
		e.addPullOperation(result, "role", "acknowledge", "user-roles", "User roles acknowledged from Logto", false, nil)
		return nil
	}

	// User roles are managed through organization memberships and direct assignments
	// in Logto, so we just acknowledge them here
	logger.Info("User roles acknowledged - managed via organization memberships in Logto")
	e.addPullOperation(result, "role", "acknowledge", "user-roles", "User roles acknowledged from Logto", false, nil)

	return nil
}

func (e *PullEngine) pullResources(result *PullResult) error {
	logger.Info("Pulling resources from Logto...")

	// Resources in Logto are API resources that define the scopes and permissions
	// For the current system architecture, resources are primarily managed in Logto
	// and referenced by the applications rather than stored locally

	if e.options.DryRun {
		logger.Info("DRY RUN: Would acknowledge resources from Logto (API resources managed in Logto)")
		e.addPullOperation(result, "resource", "acknowledge", "resources", "Resources acknowledged from Logto", false, nil)
		return nil
	}

	// Resources are API definitions managed in Logto
	logger.Info("Resources acknowledged - API resources managed in Logto")
	e.addPullOperation(result, "resource", "acknowledge", "resources", "Resources acknowledged from Logto", false, nil)

	return nil
}

func (e *PullEngine) pullPermissions(result *PullResult) error {
	logger.Info("Pulling permissions from Logto...")

	// Permissions in this system are managed through:
	// 1. Organization role scopes (managed in Logto)
	// 2. User role permissions (managed in Logto)
	// 3. Direct role assignments (managed in Logto)
	// The local database focuses on the business entities (orgs, users, systems)

	if e.options.DryRun {
		logger.Info("DRY RUN: Would acknowledge permissions from Logto (managed via roles and scopes)")
		e.addPullOperation(result, "permission", "acknowledge", "permissions", "Permissions acknowledged from Logto", false, nil)
		return nil
	}

	// Permissions are managed through role-based access control in Logto
	logger.Info("Permissions acknowledged - managed via RBAC in Logto")
	e.addPullOperation(result, "permission", "acknowledge", "permissions", "Permissions acknowledged from Logto", false, nil)

	return nil
}

// OutputText outputs the result in text format
func (r *PullResult) OutputText(w io.Writer) error {
	_, _ = fmt.Fprintf(w, "Pull Operation Results\n")
	_, _ = fmt.Fprintf(w, "=====================\n\n")
	_, _ = fmt.Fprintf(w, "Status: %s\n", map[bool]string{true: "SUCCESS", false: "FAILED"}[r.Success])
	_, _ = fmt.Fprintf(w, "Duration: %v\n", r.Duration)
	_, _ = fmt.Fprintf(w, "Dry Run: %v\n\n", r.DryRun)

	_, _ = fmt.Fprintf(w, "Summary:\n")
	_, _ = fmt.Fprintf(w, "  Organizations: %d created, %d updated, %d skipped\n",
		r.Summary.OrganizationsCreated, r.Summary.OrganizationsUpdated, r.Summary.OrganizationsSkipped)
	_, _ = fmt.Fprintf(w, "  Users: %d created, %d updated, %d skipped\n",
		r.Summary.UsersCreated, r.Summary.UsersUpdated, r.Summary.UsersSkipped)
	_, _ = fmt.Fprintf(w, "  Resources: %d created, %d updated, %d skipped\n",
		r.Summary.ResourcesCreated, r.Summary.ResourcesUpdated, r.Summary.ResourcesSkipped)
	_, _ = fmt.Fprintf(w, "  Roles: %d created, %d updated, %d skipped\n",
		r.Summary.RolesCreated, r.Summary.RolesUpdated, r.Summary.RolesSkipped)
	_, _ = fmt.Fprintf(w, "  Permissions: %d created, %d updated, %d skipped\n",
		r.Summary.PermissionsCreated, r.Summary.PermissionsUpdated, r.Summary.PermissionsSkipped)
	_, _ = fmt.Fprintf(w, "  Conflicts: %d detected, %d resolved\n\n",
		r.Summary.ConflictsDetected, r.Summary.ConflictsResolved)

	if len(r.Conflicts) > 0 {
		_, _ = fmt.Fprintf(w, "Conflicts:\n")
		for _, conflict := range r.Conflicts {
			_, _ = fmt.Fprintf(w, "  ⚠️ %s in %s - %s (Resolution: %s)\n",
				conflict.Type, conflict.Resource, conflict.Description, conflict.Resolution)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	if len(r.Errors) > 0 {
		_, _ = fmt.Fprintf(w, "Errors:\n")
		for _, err := range r.Errors {
			_, _ = fmt.Fprintf(w, "  - %s\n", err)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	if len(r.Operations) > 0 {
		_, _ = fmt.Fprintf(w, "Operations:\n")
		for _, op := range r.Operations {
			status := "✓"
			if !op.Success {
				status = "✗"
			}
			if op.Conflict {
				status += " ⚠️"
			}
			_, _ = fmt.Fprintf(w, "  %s %s %s %s - %s\n", status, op.Type, op.Action, op.Resource, op.Description)
		}
	}

	return nil
}

// OutputJSON outputs the result in JSON format
func (r *PullResult) OutputJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// OutputYAML outputs the result in YAML format
func (r *PullResult) OutputYAML(w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(r)
}
