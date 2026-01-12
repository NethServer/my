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
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalUserRepository implements UserRepository for local database
type LocalUserRepository struct {
	db *sql.DB
}

// NewLocalUserRepository creates a new local user repository
func NewLocalUserRepository() *LocalUserRepository {
	return &LocalUserRepository{
		db: database.DB,
	}
}

// Create creates a new user in local database
func (r *LocalUserRepository) Create(req *models.CreateLocalUserRequest) (*models.LocalUser, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO users (id, logto_id, username, email, name, phone, organization_id, user_role_ids, custom_data, created_at, updated_at, deleted_at, suspended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	// Serialize JSON fields
	customDataJSON, err := json.Marshal(req.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	userRoleIDsJSON, err := json.Marshal(req.UserRoleIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user_role_ids: %w", err)
	}

	_, err = r.db.Exec(query,
		id,
		nil, // logto_id
		req.Username,
		req.Email,
		req.Name,
		req.Phone,
		req.OrganizationID,
		userRoleIDsJSON,
		customDataJSON,
		now,
		now,
		nil, // deleted_at
		nil, // suspended_at
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &models.LocalUser{
		ID:             id,
		LogtoID:        nil,
		Username:       req.Username,
		Email:          req.Email,
		Name:           req.Name,
		Phone:          req.Phone,
		UserRoleIDs:    req.UserRoleIDs,
		OrganizationID: req.OrganizationID,
		CustomData:     req.CustomData,
		CreatedAt:      now,
		UpdatedAt:      now,
		DeletedAt:      nil,
		SuspendedAt:    nil,
	}, nil
}

// GetByID retrieves a user by ID from local database
func (r *LocalUserRepository) GetByID(id string) (*models.LocalUser, error) {
	query := `
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at,
		       COALESCE(d.name, r.name, c.name) as organization_name,
		       COALESCE(d.id, r.id, c.id) as organization_local_id,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM users u
		LEFT JOIN distributors d ON u.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON u.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON u.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`

	user := &models.LocalUser{}
	var customDataJSON []byte
	var userRoleIDsJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name, &user.Phone,
		&user.OrganizationID, &userRoleIDsJSON, &customDataJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt,
		&user.OrganizationName, &user.OrganizationLocalID, &user.OrganizationType,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Parse user_role_ids JSON
	if len(userRoleIDsJSON) > 0 {
		if err := json.Unmarshal(userRoleIDsJSON, &user.UserRoleIDs); err != nil {
			user.UserRoleIDs = []string{}
		}
	} else {
		user.UserRoleIDs = []string{}
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &user.CustomData); err != nil {
			user.CustomData = make(map[string]interface{})
		}
	} else {
		user.CustomData = make(map[string]interface{})
	}

	// Enrich with organization and role data
	_ = r.enrichUserWithRelations(user) // Relations are nice-to-have, don't fail request on error

	return user, nil
}

// GetByLogtoID retrieves a user by Logto ID from local database
func (r *LocalUserRepository) GetByLogtoID(logtoID string) (*models.LocalUser, error) {
	query := `
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at,
		       COALESCE(d.name, r.name, c.name) as organization_name,
		       COALESCE(d.id, r.id, c.id) as organization_local_id,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM users u
		LEFT JOIN distributors d ON u.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON u.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON u.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE u.logto_id = $1 AND u.deleted_at IS NULL
	`

	user := &models.LocalUser{}
	var customDataJSON []byte
	var userRoleIDsJSON []byte

	err := r.db.QueryRow(query, logtoID).Scan(
		&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name, &user.Phone,
		&user.OrganizationID, &userRoleIDsJSON, &customDataJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt,
		&user.OrganizationName, &user.OrganizationLocalID, &user.OrganizationType,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Parse user_role_ids JSON
	if len(userRoleIDsJSON) > 0 {
		if err := json.Unmarshal(userRoleIDsJSON, &user.UserRoleIDs); err != nil {
			user.UserRoleIDs = []string{}
		}
	} else {
		user.UserRoleIDs = []string{}
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &user.CustomData); err != nil {
			user.CustomData = make(map[string]interface{})
		}
	} else {
		user.CustomData = make(map[string]interface{})
	}

	// Enrich with organization and role data
	_ = r.enrichUserWithRelations(user) // Relations are nice-to-have, don't fail request on error

	return user, nil
}

// Update updates a user in local database
func (r *LocalUserRepository) Update(id string, req *models.UpdateLocalUserRequest) (*models.LocalUser, error) {
	// First get the current user
	current, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Username != nil {
		current.Username = *req.Username
	}
	if req.Email != nil {
		current.Email = *req.Email
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Phone != nil {
		current.Phone = req.Phone
	}
	if req.UserRoleIDs != nil {
		current.UserRoleIDs = *req.UserRoleIDs
	}
	if req.OrganizationID != nil {
		current.OrganizationID = req.OrganizationID
	}
	if req.CustomData != nil {
		current.CustomData = *req.CustomData
	}

	current.UpdatedAt = time.Now()
	current.LogtoSyncedAt = nil // Mark as needing sync

	// Serialize JSON fields
	customDataJSON, err := json.Marshal(current.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	userRoleIDsJSON, err := json.Marshal(current.UserRoleIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user_role_ids: %w", err)
	}

	query := `
		UPDATE users
		SET username = $2, email = $3, name = $4, phone = $5, organization_id = $6,
		    user_role_ids = $7, custom_data = $8, updated_at = $9, logto_synced_at = NULL
		WHERE id = $1
	`

	_, err = r.db.Exec(query,
		id,
		current.Username,
		current.Email,
		current.Name,
		current.Phone,
		current.OrganizationID,
		userRoleIDsJSON,
		customDataJSON,
		current.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return current, nil
}

// Delete soft-deletes a user in local database
func (r *LocalUserRepository) Delete(id string) error {
	now := time.Now()
	query := `UPDATE users SET deleted_at = $2, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(query, id, now)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// SuspendUser suspends a user by setting suspended_at timestamp
func (r *LocalUserRepository) SuspendUser(id string) error {
	now := time.Now()
	query := `UPDATE users SET suspended_at = $2, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

	result, err := r.db.Exec(query, id, now)
	if err != nil {
		return fmt.Errorf("failed to suspend user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already suspended/deleted")
	}

	return nil
}

// ReactivateUser reactivates a suspended user by clearing suspended_at timestamp
func (r *LocalUserRepository) ReactivateUser(id string) error {
	now := time.Now()
	query := `UPDATE users SET suspended_at = NULL, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL`

	result, err := r.db.Exec(query, id, now)
	if err != nil {
		return fmt.Errorf("failed to reactivate user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or not suspended")
	}

	return nil
}

// UpdateLatestLogin updates the latest_login_at field for a user
func (r *LocalUserRepository) UpdateLatestLogin(userID string) error {
	query := `UPDATE users SET latest_login_at = $2, updated_at = $2 WHERE id = $1`

	now := time.Now()
	result, err := r.db.Exec(query, userID, now)
	if err != nil {
		return fmt.Errorf("failed to update latest login: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// List returns paginated list of users based on hierarchical RBAC (matches other repository patterns)
func (r *LocalUserRepository) List(userOrgRole, userOrgID, excludeUserID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalUser, int, error) {
	// Get all organization IDs the user can access hierarchically
	allowedOrgIDs, err := r.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
	}

	return r.ListByOrganizations(allowedOrgIDs, excludeUserID, page, pageSize, search, sortBy, sortDirection)
}

// ListByOrganizations returns paginated list of users in specified organizations (excluding specified user)
func (r *LocalUserRepository) ListByOrganizations(allowedOrgIDs []string, excludeUserID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalUser, int, error) {
	if len(allowedOrgIDs) == 0 {
		return []*models.LocalUser{}, 0, nil
	}

	offset := (page - 1) * pageSize

	if search != "" {
		return r.listUsersWithSearch(allowedOrgIDs, excludeUserID, pageSize, offset, search, sortBy, sortDirection)
	} else {
		return r.listUsersWithoutSearch(allowedOrgIDs, excludeUserID, pageSize, offset, sortBy, sortDirection)
	}
}

// listUsersWithSearch handles user listing with search functionality
func (r *LocalUserRepository) listUsersWithSearch(allowedOrgIDs []string, excludeUserID string, pageSize, offset int, search, sortBy, sortDirection string) ([]*models.LocalUser, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY u.created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":            "LOWER(u.name)",
			"email":           "LOWER(u.email)",
			"username":        "LOWER(u.username)",
			"created_at":      "u.created_at",
			"updated_at":      "u.updated_at",
			"latest_login_at": "u.latest_login_at",
			"organization":    "LOWER(COALESCE(d.name, r.name, c.name))",
			"status":          "u.suspended_at",
		}

		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	// Build placeholders for organization IDs
	placeholders := make([]string, len(allowedOrgIDs))
	for i := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	orgPlaceholders := strings.Join(placeholders, ",")

	// Build count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM users u
		WHERE u.deleted_at IS NULL
		  AND u.organization_id IN (%s)
		  AND u.id != $%d
		  AND (LOWER(u.name) LIKE LOWER('%%' || $%d || '%%') OR LOWER(u.email) LIKE LOWER('%%' || $%d || '%%'))
	`, orgPlaceholders, len(allowedOrgIDs)+1, len(allowedOrgIDs)+2, len(allowedOrgIDs)+3)

	// Build main query
	mainQuery := fmt.Sprintf(`
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at,
		       COALESCE(d.name, r.name, c.name) as organization_name,
		       COALESCE(d.id, r.id, c.id) as organization_local_id,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM users u
		LEFT JOIN distributors d ON u.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON u.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON u.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE u.deleted_at IS NULL
		  AND u.organization_id IN (%s)
		  AND u.id != $%d
		  AND (LOWER(u.name) LIKE LOWER('%%' || $%d || '%%') OR LOWER(u.email) LIKE LOWER('%%' || $%d || '%%'))
		%s
		LIMIT $%d OFFSET $%d
	`, orgPlaceholders, len(allowedOrgIDs)+1, len(allowedOrgIDs)+2, len(allowedOrgIDs)+3, orderClause, len(allowedOrgIDs)+4, len(allowedOrgIDs)+5)

	// Prepare arguments for count query
	countArgs := make([]interface{}, len(allowedOrgIDs)+3)
	for i, orgID := range allowedOrgIDs {
		countArgs[i] = orgID
	}
	countArgs[len(allowedOrgIDs)] = excludeUserID
	countArgs[len(allowedOrgIDs)+1] = search
	countArgs[len(allowedOrgIDs)+2] = search

	// Prepare arguments for main query
	mainArgs := make([]interface{}, len(allowedOrgIDs)+5)
	for i, orgID := range allowedOrgIDs {
		mainArgs[i] = orgID
	}
	mainArgs[len(allowedOrgIDs)] = excludeUserID
	mainArgs[len(allowedOrgIDs)+1] = search
	mainArgs[len(allowedOrgIDs)+2] = search
	mainArgs[len(allowedOrgIDs)+3] = pageSize
	mainArgs[len(allowedOrgIDs)+4] = offset

	return r.executeUserQuery(countQuery, countArgs, mainQuery, mainArgs)
}

// listUsersWithoutSearch handles user listing without search functionality
func (r *LocalUserRepository) listUsersWithoutSearch(allowedOrgIDs []string, excludeUserID string, pageSize, offset int, sortBy, sortDirection string) ([]*models.LocalUser, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY u.created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":            "LOWER(u.name)",
			"email":           "LOWER(u.email)",
			"username":        "LOWER(u.username)",
			"created_at":      "u.created_at",
			"updated_at":      "u.updated_at",
			"latest_login_at": "u.latest_login_at",
			"organization":    "LOWER(COALESCE(d.name, r.name, c.name))",
			"status":          "u.suspended_at",
		}

		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	// Build placeholders for organization IDs
	placeholders := make([]string, len(allowedOrgIDs))
	for i := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	orgPlaceholders := strings.Join(placeholders, ",")

	// Build count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM users u
		WHERE u.deleted_at IS NULL
		  AND u.organization_id IN (%s)
		  AND u.id != $%d
	`, orgPlaceholders, len(allowedOrgIDs)+1)

	// Build main query
	mainQuery := fmt.Sprintf(`
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at,
		       COALESCE(d.name, r.name, c.name) as organization_name,
		       COALESCE(d.id, r.id, c.id) as organization_local_id,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM users u
		LEFT JOIN distributors d ON u.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON u.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON u.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE u.deleted_at IS NULL
		  AND u.organization_id IN (%s)
		  AND u.id != $%d
		%s
		LIMIT $%d OFFSET $%d
	`, orgPlaceholders, len(allowedOrgIDs)+1, orderClause, len(allowedOrgIDs)+2, len(allowedOrgIDs)+3)

	// Prepare arguments for count query
	countArgs := make([]interface{}, len(allowedOrgIDs)+1)
	for i, orgID := range allowedOrgIDs {
		countArgs[i] = orgID
	}
	countArgs[len(allowedOrgIDs)] = excludeUserID

	// Prepare arguments for main query
	mainArgs := make([]interface{}, len(allowedOrgIDs)+3)
	for i, orgID := range allowedOrgIDs {
		mainArgs[i] = orgID
	}
	mainArgs[len(allowedOrgIDs)] = excludeUserID
	mainArgs[len(allowedOrgIDs)+1] = pageSize
	mainArgs[len(allowedOrgIDs)+2] = offset

	return r.executeUserQuery(countQuery, countArgs, mainQuery, mainArgs)
}

// executeUserQuery executes the count and main queries for users
func (r *LocalUserRepository) executeUserQuery(countQuery string, countArgs []interface{}, mainQuery string, mainArgs []interface{}) ([]*models.LocalUser, int, error) {
	// Get total count
	var totalCount int
	err := r.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users count: %w", err)
	}

	// Get paginated results
	rows, err := r.db.Query(mainQuery, mainArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []*models.LocalUser
	for rows.Next() {
		user := &models.LocalUser{}
		var userRoleIDsJSON, customDataJSON []byte

		err := rows.Scan(
			&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name,
			&user.Phone, &user.OrganizationID, &userRoleIDsJSON, &customDataJSON,
			&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt,
			&user.OrganizationName, &user.OrganizationLocalID, &user.OrganizationType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}

		// Parse user_role_ids JSON
		if len(userRoleIDsJSON) > 0 {
			if err := json.Unmarshal(userRoleIDsJSON, &user.UserRoleIDs); err != nil {
				user.UserRoleIDs = []string{}
			}
		} else {
			user.UserRoleIDs = []string{}
		}

		// Parse custom_data JSON
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &user.CustomData); err != nil {
				user.CustomData = make(map[string]interface{})
			}
		} else {
			user.CustomData = make(map[string]interface{})
		}

		// Enrich with organization and role data
		_ = r.enrichUserWithRelations(user) // Relations are nice-to-have, don't fail request on error

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}

	return users, totalCount, nil
}

// GetTotals returns total count of users based on hierarchical RBAC (matches other repository patterns)
func (r *LocalUserRepository) GetTotals(userOrgRole, userOrgID string) (int, error) {
	// Get all organization IDs the user can access hierarchically
	allowedOrgIDs, err := r.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return 0, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
	}

	return r.GetTotalsByOrganizations(allowedOrgIDs)
}

// GetTotalsByOrganizations returns total count of users in specified organizations
func (r *LocalUserRepository) GetTotalsByOrganizations(allowedOrgIDs []string) (int, error) {
	if len(allowedOrgIDs) == 0 {
		return 0, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	var count int
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM users
		WHERE deleted_at IS NULL AND organization_id IN (%s)
	`, placeholdersStr)

	err := r.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// GetHierarchicalOrganizationIDs returns all organization IDs that the user can manage
// This mirrors the logic from UserService.GetHierarchicalOrganizationIDs
func (r *LocalUserRepository) GetHierarchicalOrganizationIDs(userOrgRole, userOrgID string) ([]string, error) {
	orgIDs := []string{userOrgID} // Always include own organization

	// Normalize role to lowercase for case-insensitive comparison
	normalizedRole := strings.ToLower(userOrgRole)

	switch normalizedRole {
	case "owner":
		// Owner can manage all organizations
		var allOrgIDs []string

		// Get all distributors
		rows, err := r.db.Query("SELECT logto_id FROM distributors WHERE logto_id IS NOT NULL AND deleted_at IS NULL")
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					allOrgIDs = append(allOrgIDs, orgID)
				}
			}
		}

		// Get all resellers
		rows, err = r.db.Query("SELECT logto_id FROM resellers WHERE logto_id IS NOT NULL AND deleted_at IS NULL")
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					allOrgIDs = append(allOrgIDs, orgID)
				}
			}
		}

		// Get all customers
		rows, err = r.db.Query("SELECT logto_id FROM customers WHERE logto_id IS NOT NULL AND deleted_at IS NULL")
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					allOrgIDs = append(allOrgIDs, orgID)
				}
			}
		}

		return append(orgIDs, allOrgIDs...), nil

	case "distributor":
		// Get resellers created by this distributor
		rows, err := r.db.Query("SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $1 AND logto_id IS NOT NULL AND deleted_at IS NULL", userOrgID)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					orgIDs = append(orgIDs, orgID)
				}
			}
		}

		// Get customers created by this distributor
		rows, err = r.db.Query("SELECT logto_id FROM customers WHERE custom_data->>'createdBy' = $1 AND logto_id IS NOT NULL AND deleted_at IS NULL", userOrgID)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					orgIDs = append(orgIDs, orgID)
				}
			}
		}

		// Get customers created by resellers created by this distributor
		query := `
			SELECT c.logto_id FROM customers c
			JOIN resellers r ON c.custom_data->>'createdBy' = r.logto_id
			WHERE r.custom_data->>'createdBy' = $1 AND c.logto_id IS NOT NULL AND c.deleted_at IS NULL AND r.deleted_at IS NULL
		`
		rows, err = r.db.Query(query, userOrgID)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					orgIDs = append(orgIDs, orgID)
				}
			}
		}

	case "reseller":
		// Get customers created by this reseller
		rows, err := r.db.Query("SELECT logto_id FROM customers WHERE custom_data->>'createdBy' = $1 AND logto_id IS NOT NULL AND deleted_at IS NULL", userOrgID)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					orgIDs = append(orgIDs, orgID)
				}
			}
		}
	}

	return orgIDs, nil
}

// enrichUserWithRelations populates Organization and Roles objects from database and cache data
func (r *LocalUserRepository) enrichUserWithRelations(user *models.LocalUser) error {
	// Build Organization object
	if user.OrganizationID != nil && *user.OrganizationID != "" {
		user.Organization = &models.UserOrganization{
			LogtoID: *user.OrganizationID,
			Name:    "",
			Type:    "owner", // default
		}
		// Set the local database ID
		if user.OrganizationLocalID != nil {
			user.Organization.ID = *user.OrganizationLocalID
		}
		// Set the organization name
		if user.OrganizationName != nil {
			user.Organization.Name = *user.OrganizationName
		}
		// Set the organization type
		if user.OrganizationType != nil {
			user.Organization.Type = *user.OrganizationType
		}
	}

	// Build Roles array
	if len(user.UserRoleIDs) == 0 {
		user.Roles = []models.UserRole{}
		return nil
	}

	roleNames := cache.GetRoleNames()
	roleNamesSlice := roleNames.GetNames(user.UserRoleIDs)

	user.Roles = make([]models.UserRole, len(user.UserRoleIDs))
	for i, roleID := range user.UserRoleIDs {
		user.Roles[i] = models.UserRole{
			ID: roleID,
		}
		// Set name if available
		if i < len(roleNamesSlice) {
			user.Roles[i].Name = roleNamesSlice[i]
		}
	}

	return nil
}

// GetTrend returns trend data for users over a specified period
func (r *LocalUserRepository) GetTrend(userOrgRole, userOrgID string, period int) ([]struct {
	Date  string
	Count int
}, int, int, error) {
	// Get all organization IDs the user can access hierarchically
	allowedOrgIDs, err := r.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
	}

	if len(allowedOrgIDs) == 0 {
		return []struct {
			Date  string
			Count int
		}{}, 0, 0, nil
	}

	// Determine interval for date series based on period
	var interval string
	switch period {
	case 7, 30:
		interval = "1 day"
	case 180:
		interval = "1 week"
	case 365:
		interval = "1 month"
	default:
		return nil, 0, 0, fmt.Errorf("invalid period: %d", period)
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Query to get cumulative count for each date in the period
	query := fmt.Sprintf(`
		WITH date_series AS (
			SELECT generate_series(
				CURRENT_DATE - INTERVAL '%d days',
				CURRENT_DATE,
				INTERVAL '%s'
			)::date AS date
		)
		SELECT
			ds.date::text,
			COALESCE((
				SELECT COUNT(*)
				FROM users
				WHERE deleted_at IS NULL
				  AND organization_id IN (%s)
				  AND created_at::date <= ds.date
			), 0) AS count
		FROM date_series ds
		ORDER BY ds.date
	`, period, interval, placeholdersStr)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to query trend data: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var dataPoints []struct {
		Date  string
		Count int
	}

	for rows.Next() {
		var date string
		var count int
		if err := rows.Scan(&date, &count); err != nil {
			return nil, 0, 0, fmt.Errorf("failed to scan trend data: %w", err)
		}
		dataPoints = append(dataPoints, struct {
			Date  string
			Count int
		}{Date: date, Count: count})
	}

	if err := rows.Err(); err != nil {
		return nil, 0, 0, fmt.Errorf("error iterating trend data: %w", err)
	}

	// Calculate current and previous totals
	var currentTotal, previousTotal int
	if len(dataPoints) > 0 {
		currentTotal = dataPoints[len(dataPoints)-1].Count
		previousTotal = dataPoints[0].Count
	}

	return dataPoints, currentTotal, previousTotal, nil
}
