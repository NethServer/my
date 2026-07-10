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
	"github.com/lib/pq"
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

// dbExecer is satisfied by *sql.DB and *sql.Tx so an insert can run either
// standalone or inside a caller's transaction.
type dbExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// Create creates a new user in local database
func (r *LocalUserRepository) Create(req *models.CreateLocalUserRequest) (*models.LocalUser, error) {
	return r.create(r.db, req)
}

// CreateWithTx creates a new user inside the provided transaction so the row
// participates in the caller's atomic create-and-sync flow; a later failure
// rolls the insert back instead of leaving an orphaned user (logto_id IS NULL).
func (r *LocalUserRepository) CreateWithTx(tx *sql.Tx, req *models.CreateLocalUserRequest) (*models.LocalUser, error) {
	return r.create(tx, req)
}

func (r *LocalUserRepository) create(exec dbExecer, req *models.CreateLocalUserRequest) (*models.LocalUser, error) {
	id := uuid.New().String()
	now := time.Now()

	query := `
		INSERT INTO users (id, logto_id, username, email, name, phone, organization_id, user_role_ids, custom_data, created_by, created_at, updated_at, deleted_at, suspended_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
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

	var createdByJSON []byte
	if req.CreatedBy != nil {
		createdByJSON, err = json.Marshal(req.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal created_by: %w", err)
		}
	}

	_, err = exec.Exec(query,
		id,
		nil, // logto_id
		req.Username,
		req.Email,
		req.Name,
		req.Phone,
		req.OrganizationID,
		userRoleIDsJSON,
		customDataJSON,
		createdByJSON,
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
		CreatedBy:      req.CreatedBy,
		CreatedAt:      now,
		UpdatedAt:      now,
		DeletedAt:      nil,
		SuspendedAt:    nil,
	}, nil
}

// GetByID retrieves a user by ID from local database
func (r *LocalUserRepository) GetByID(id string) (*models.LocalUser, error) {
	query := `
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data, u.created_by,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at, u.suspended_by_org_id,
		       uo.name as organization_name,
		       COALESCE(uo.db_id, '') as organization_local_id,
		       COALESCE(uo.org_type, 'owner') as organization_type
		FROM users u
		LEFT JOIN unified_organizations uo ON u.organization_id = uo.logto_id
		WHERE u.logto_id = $1 AND u.deleted_at IS NULL
	`

	user := &models.LocalUser{}
	var customDataJSON []byte
	var userRoleIDsJSON []byte
	var createdByJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name, &user.Phone,
		&user.OrganizationID, &userRoleIDsJSON, &customDataJSON, &createdByJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt, &user.SuspendedByOrgID,
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

	// Parse created_by JSON (creator snapshot; may be NULL on pre-backfill rows)
	if len(createdByJSON) > 0 {
		var creator models.OrgCreator
		if err := json.Unmarshal(createdByJSON, &creator); err == nil {
			user.CreatedBy = &creator
		}
	}

	// Enrich with organization and role data
	_ = r.enrichUserWithRelations(user) // Relations are nice-to-have, don't fail request on error

	return user, nil
}

// GetByLogtoID retrieves a user by Logto ID from local database
func (r *LocalUserRepository) GetByLogtoID(logtoID string) (*models.LocalUser, error) {
	query := `
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data, u.created_by,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at, u.suspended_by_org_id,
		       uo.name as organization_name,
		       COALESCE(uo.db_id, '') as organization_local_id,
		       COALESCE(uo.org_type, 'owner') as organization_type
		FROM users u
		LEFT JOIN unified_organizations uo ON u.organization_id = uo.logto_id
		WHERE u.logto_id = $1 AND u.deleted_at IS NULL
	`

	user := &models.LocalUser{}
	var customDataJSON []byte
	var userRoleIDsJSON []byte
	var createdByJSON []byte

	err := r.db.QueryRow(query, logtoID).Scan(
		&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name, &user.Phone,
		&user.OrganizationID, &userRoleIDsJSON, &customDataJSON, &createdByJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt, &user.SuspendedByOrgID,
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

	// Parse created_by JSON (creator snapshot; may be NULL on pre-backfill rows)
	if len(createdByJSON) > 0 {
		var creator models.OrgCreator
		if err := json.Unmarshal(createdByJSON, &creator); err == nil {
			user.CreatedBy = &creator
		}
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
		WHERE logto_id = $1
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

// UpdateProfileInfo syncs the locally-cached self-service profile fields
// (name/email/phone) into the users table after the caller has already pushed
// the same change to Logto. Unlike Update it marks the row as synced
// (logto_synced_at = now()) — Logto is now in agreement — and touches only the
// columns editable from account settings, leaving roles, organization and
// custom_data untouched.
//
// A nil pointer leaves the column unchanged; an empty phone pointer means the
// user cleared their phone, so it is stored as NULL. Keyed by logto_id, the
// stable identifier for a self-service user.
func (r *LocalUserRepository) UpdateProfileInfo(logtoID string, name, email, phone *string) error {
	setClauses := make([]string, 0, 5)
	args := []any{logtoID}

	add := func(expr string, val any) {
		args = append(args, val)
		setClauses = append(setClauses, fmt.Sprintf(expr, len(args)))
	}

	if name != nil {
		add("name = $%d", *name)
	}
	if email != nil {
		add("email = $%d", *email)
	}
	if phone != nil {
		if *phone == "" {
			setClauses = append(setClauses, "phone = NULL")
		} else {
			add("phone = $%d", *phone)
		}
	}

	// Nothing user-facing changed; skip the write entirely.
	if len(setClauses) == 0 {
		return nil
	}

	now := time.Now()
	add("updated_at = $%d", now)
	add("logto_synced_at = $%d", now)

	query := fmt.Sprintf(
		"UPDATE users SET %s WHERE logto_id = $1 AND deleted_at IS NULL",
		strings.Join(setClauses, ", "),
	)

	if _, err := r.db.Exec(query, args...); err != nil {
		return fmt.Errorf("failed to update local profile info: %w", err)
	}
	return nil
}

// Delete soft-deletes a user in local database
func (r *LocalUserRepository) Delete(id string) error {
	now := time.Now()
	query := `UPDATE users SET deleted_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL`

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
	query := `UPDATE users SET suspended_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

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

// ReactivateUser reactivates a suspended user by clearing suspended_at and suspended_by_org_id
func (r *LocalUserRepository) ReactivateUser(id string) error {
	now := time.Now()
	query := `UPDATE users SET suspended_at = NULL, suspended_by_org_id = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL`

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

// SuspendUsersByOrgID suspends all active users belonging to an organization (cascade suspension)
// Returns the list of suspended users (for Logto sync) and count
func (r *LocalUserRepository) SuspendUsersByOrgID(orgID string) ([]*models.LocalUser, int, error) {
	now := time.Now()

	// First, get all active users that will be suspended (for Logto sync)
	selectQuery := `
		SELECT id, logto_id, username, email, name
		FROM users
		WHERE organization_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL
	`

	rows, err := r.db.Query(selectQuery, orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users for cascade suspension: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []*models.LocalUser
	for rows.Next() {
		user := &models.LocalUser{}
		if err := rows.Scan(&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}

	if len(users) == 0 {
		return users, 0, nil
	}

	// Now suspend all these users
	updateQuery := `
		UPDATE users
		SET suspended_at = $2, suspended_by_org_id = $1, updated_at = $2
		WHERE organization_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL
	`

	result, err := r.db.Exec(updateQuery, orgID, now)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to cascade suspend users: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return users, int(rowsAffected), nil
}

// ReactivateUsersByOrgID reactivates users that were cascade-suspended by this organization
// Returns the list of reactivated users (for Logto sync) and count
func (r *LocalUserRepository) ReactivateUsersByOrgID(orgID string) ([]*models.LocalUser, int, error) {
	now := time.Now()

	// First, get all users that were cascade-suspended by this org (for Logto sync)
	selectQuery := `
		SELECT id, logto_id, username, email, name
		FROM users
		WHERE suspended_by_org_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL
	`

	rows, err := r.db.Query(selectQuery, orgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users for cascade reactivation: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []*models.LocalUser
	for rows.Next() {
		user := &models.LocalUser{}
		if err := rows.Scan(&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}

	if len(users) == 0 {
		return users, 0, nil
	}

	// Now reactivate all these users
	updateQuery := `
		UPDATE users
		SET suspended_at = NULL, suspended_by_org_id = NULL, updated_at = $2
		WHERE suspended_by_org_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL
	`

	result, err := r.db.Exec(updateQuery, orgID, now)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to cascade reactivate users: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return users, int(rowsAffected), nil
}

// SuspendUsersByMultipleOrgIDs suspends all active users belonging to any of the given organizations
// The suspendedByOrgID is the org that initiated the cascade (not necessarily the user's org)
// Returns the list of suspended users (for Logto sync) and count
func (r *LocalUserRepository) SuspendUsersByMultipleOrgIDs(orgIDs []string, suspendedByOrgID string) ([]*models.LocalUser, int, error) {
	if len(orgIDs) == 0 {
		return nil, 0, nil
	}

	now := time.Now()

	// Build placeholders for IN clause
	placeholders := make([]string, len(orgIDs))
	args := make([]interface{}, len(orgIDs))
	for i, id := range orgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	// Get all active users that will be suspended (for Logto sync)
	selectQuery := fmt.Sprintf(`
		SELECT id, logto_id, username, email, name
		FROM users
		WHERE organization_id IN (%s) AND deleted_at IS NULL AND suspended_at IS NULL
	`, inClause)

	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users for cascade suspension: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var users []*models.LocalUser
	for rows.Next() {
		user := &models.LocalUser{}
		if err := rows.Scan(&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating users: %w", err)
	}

	if len(users) == 0 {
		return users, 0, nil
	}

	// Suspend all these users
	suspendedByIdx := len(orgIDs) + 1
	nowIdx := len(orgIDs) + 2
	updateQuery := fmt.Sprintf(`
		UPDATE users
		SET suspended_at = $%d, suspended_by_org_id = $%d, updated_at = $%d
		WHERE organization_id IN (%s) AND deleted_at IS NULL AND suspended_at IS NULL
	`, nowIdx, suspendedByIdx, nowIdx, inClause)

	updateArgs := make([]interface{}, 0, len(orgIDs)+2)
	updateArgs = append(updateArgs, args...)
	updateArgs = append(updateArgs, suspendedByOrgID, now)

	result, err := r.db.Exec(updateQuery, updateArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to cascade suspend users: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return users, int(rowsAffected), nil
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
func (r *LocalUserRepository) List(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, organizationFilter, statuses, roleFilter, createdByFilter []string) ([]*models.LocalUser, int, error) {
	// Owner can access all users - pass nil to skip RBAC filtering in query
	var allowedOrgIDs []string
	if strings.ToLower(userOrgRole) != "owner" {
		var err error
		allowedOrgIDs, err = r.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
		}
	}

	return r.ListByOrganizations(allowedOrgIDs, page, pageSize, search, sortBy, sortDirection, organizationFilter, statuses, roleFilter, createdByFilter)
}

// ListByOrganizations returns paginated list of users in specified organizations
// nil allowedOrgIDs = owner (no RBAC filter), empty = no access
func (r *LocalUserRepository) ListByOrganizations(allowedOrgIDs []string, page, pageSize int, search, sortBy, sortDirection string, organizationFilter, statuses, roleFilter, createdByFilter []string) ([]*models.LocalUser, int, error) {
	// nil = owner (no RBAC filter), empty = no access
	if allowedOrgIDs != nil && len(allowedOrgIDs) == 0 {
		return []*models.LocalUser{}, 0, nil
	}

	// If organization filter is specified, verify each is in allowed orgs (skip for owner)
	if len(organizationFilter) > 0 && allowedOrgIDs != nil {
		var filteredOrgIDs []string
		allowedSet := make(map[string]bool, len(allowedOrgIDs))
		for _, orgID := range allowedOrgIDs {
			allowedSet[orgID] = true
		}
		for _, filterOrgID := range organizationFilter {
			if allowedSet[filterOrgID] {
				filteredOrgIDs = append(filteredOrgIDs, filterOrgID)
			}
		}
		if len(filteredOrgIDs) == 0 {
			return []*models.LocalUser{}, 0, nil
		}
		allowedOrgIDs = filteredOrgIDs
	} else if len(organizationFilter) > 0 && allowedOrgIDs == nil {
		// Owner with org filter: use the filter directly as the allowed list
		allowedOrgIDs = organizationFilter
	}

	offset := (page - 1) * pageSize

	if search != "" {
		return r.listUsersWithSearch(allowedOrgIDs, pageSize, offset, search, sortBy, sortDirection, statuses, roleFilter, createdByFilter)
	} else {
		return r.listUsersWithoutSearch(allowedOrgIDs, pageSize, offset, sortBy, sortDirection, statuses, roleFilter, createdByFilter)
	}
}

// buildUsersCreatedByClause builds the created_by filter fragment: each value
// matches either the creating user or their organization, mirroring the
// systems created_by filter. Placeholders are appended to args.
func buildUsersCreatedByClause(createdByFilter []string, args *[]interface{}) string {
	if len(createdByFilter) == 0 {
		return ""
	}
	var conditions []string
	for _, cb := range createdByFilter {
		*args = append(*args, cb)
		placeholder := fmt.Sprintf("$%d", len(*args))
		conditions = append(conditions, fmt.Sprintf("(u.created_by ->> 'user_id' = %s OR u.created_by ->> 'organization_id' = %s)", placeholder, placeholder))
	}
	return " AND (" + strings.Join(conditions, " OR ") + ")"
}

// listUsersWithSearch handles user listing with search functionality
func (r *LocalUserRepository) listUsersWithSearch(allowedOrgIDs []string, pageSize, offset int, search, sortBy, sortDirection string, statuses, roleFilter, createdByFilter []string) ([]*models.LocalUser, int, error) {
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
			"organization":    "LOWER(uo.name)",
			"creator_name":    "LOWER(u.created_by ->> 'name')",
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

	// Build status filter clauses
	hasDeletedFilter := false
	var statusConditions []string
	for _, s := range statuses {
		switch strings.ToLower(s) {
		case "enabled":
			statusConditions = append(statusConditions, "(u.deleted_at IS NULL AND u.suspended_at IS NULL)")
		case "suspended":
			statusConditions = append(statusConditions, "(u.deleted_at IS NULL AND u.suspended_at IS NOT NULL)")
		case "deleted":
			hasDeletedFilter = true
			statusConditions = append(statusConditions, "(u.deleted_at IS NOT NULL)")
		}
	}

	deletedClause := " AND u.deleted_at IS NULL"
	if hasDeletedFilter {
		deletedClause = ""
	}

	statusClause := ""
	if len(statusConditions) > 0 {
		statusClause = " AND (" + strings.Join(statusConditions, " OR ") + ")"
	}

	// Build WHERE clause and args: nil allowedOrgIDs = owner (no org filter)
	var args []interface{}
	orgClause := ""
	if allowedOrgIDs != nil {
		args = append(args, pq.Array(allowedOrgIDs))
		orgClause = fmt.Sprintf(" AND u.organization_id = ANY($%d::text[])", len(args))
	}

	args = append(args, search, search)
	searchClause := fmt.Sprintf(" AND (LOWER(u.name) LIKE LOWER('%%%%' || $%d || '%%%%') OR LOWER(u.email) LIKE LOWER('%%%%' || $%d || '%%%%'))", len(args)-1, len(args))

	// Build role filter clause
	roleClause := ""
	if len(roleFilter) > 0 {
		var roleConditions []string
		for _, role := range roleFilter {
			args = append(args, role)
			roleConditions = append(roleConditions, fmt.Sprintf("u.user_role_ids @> jsonb_build_array($%d::text)", len(args)))
		}
		roleClause = " AND (" + strings.Join(roleConditions, " OR ") + ")"
	}

	createdByClause := buildUsersCreatedByClause(createdByFilter, &args)

	whereClause := fmt.Sprintf("1=1%s%s%s%s%s%s", deletedClause, orgClause, searchClause, statusClause, roleClause, createdByClause)

	// Single query with COUNT(*) OVER() for total count + paginated results
	mainQuery := fmt.Sprintf(`
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data, u.created_by,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at, u.suspended_by_org_id,
		       uo.name as organization_name,
		       uo.db_id as organization_local_id,
		       COALESCE(uo.org_type, 'owner') as organization_type,
		       COUNT(*) OVER() as total_count
		FROM users u
		LEFT JOIN unified_organizations uo ON u.organization_id = uo.logto_id
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, len(args)+1, len(args)+2)

	mainArgs := make([]interface{}, len(args)+2)
	copy(mainArgs, args)
	mainArgs[len(args)] = pageSize
	mainArgs[len(args)+1] = offset

	return r.executeUserQuery("", nil, mainQuery, mainArgs)
}

// listUsersWithoutSearch handles user listing without search functionality
func (r *LocalUserRepository) listUsersWithoutSearch(allowedOrgIDs []string, pageSize, offset int, sortBy, sortDirection string, statuses, roleFilter, createdByFilter []string) ([]*models.LocalUser, int, error) {
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
			"organization":    "LOWER(uo.name)",
			"creator_name":    "LOWER(u.created_by ->> 'name')",
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

	// Build status filter clauses
	hasDeletedFilter := false
	var statusConditions []string
	for _, s := range statuses {
		switch strings.ToLower(s) {
		case "enabled":
			statusConditions = append(statusConditions, "(u.deleted_at IS NULL AND u.suspended_at IS NULL)")
		case "suspended":
			statusConditions = append(statusConditions, "(u.deleted_at IS NULL AND u.suspended_at IS NOT NULL)")
		case "deleted":
			hasDeletedFilter = true
			statusConditions = append(statusConditions, "(u.deleted_at IS NOT NULL)")
		}
	}

	deletedClause := " AND u.deleted_at IS NULL"
	if hasDeletedFilter {
		deletedClause = ""
	}

	statusClause := ""
	if len(statusConditions) > 0 {
		statusClause = " AND (" + strings.Join(statusConditions, " OR ") + ")"
	}

	// Build WHERE clause and args: nil allowedOrgIDs = owner (no org filter)
	var args []interface{}
	orgClause := ""
	if allowedOrgIDs != nil {
		args = append(args, pq.Array(allowedOrgIDs))
		orgClause = fmt.Sprintf(" AND u.organization_id = ANY($%d::text[])", len(args))
	}

	// Build role filter clause
	roleClause := ""
	if len(roleFilter) > 0 {
		var roleConditions []string
		for _, role := range roleFilter {
			args = append(args, role)
			roleConditions = append(roleConditions, fmt.Sprintf("u.user_role_ids @> jsonb_build_array($%d::text)", len(args)))
		}
		roleClause = " AND (" + strings.Join(roleConditions, " OR ") + ")"
	}

	createdByClause := buildUsersCreatedByClause(createdByFilter, &args)

	whereClause := fmt.Sprintf("1=1%s%s%s%s%s", deletedClause, orgClause, statusClause, roleClause, createdByClause)

	// Single query with COUNT(*) OVER() for total count + paginated results
	mainQuery := fmt.Sprintf(`
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data, u.created_by,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at, u.suspended_by_org_id,
		       uo.name as organization_name,
		       uo.db_id as organization_local_id,
		       COALESCE(uo.org_type, 'owner') as organization_type,
		       COUNT(*) OVER() as total_count
		FROM users u
		LEFT JOIN unified_organizations uo ON u.organization_id = uo.logto_id
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, len(args)+1, len(args)+2)

	mainArgs := make([]interface{}, len(args)+2)
	copy(mainArgs, args)
	mainArgs[len(args)] = pageSize
	mainArgs[len(args)+1] = offset

	return r.executeUserQuery("", nil, mainQuery, mainArgs)
}

// executeUserQuery executes the main query for users (count is embedded via COUNT(*) OVER())
func (r *LocalUserRepository) executeUserQuery(_ string, _ []interface{}, mainQuery string, mainArgs []interface{}) ([]*models.LocalUser, int, error) {
	// Single query with COUNT(*) OVER() for total count + paginated results
	rows, err := r.db.Query(mainQuery, mainArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query users: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var totalCount int
	var users []*models.LocalUser
	for rows.Next() {
		user := &models.LocalUser{}
		var userRoleIDsJSON, customDataJSON, createdByJSON []byte

		err := rows.Scan(
			&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name,
			&user.Phone, &user.OrganizationID, &userRoleIDsJSON, &customDataJSON, &createdByJSON,
			&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt, &user.SuspendedByOrgID,
			&user.OrganizationName, &user.OrganizationLocalID, &user.OrganizationType,
			&totalCount,
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

		// Parse created_by JSON (creator snapshot; may be NULL on pre-backfill rows)
		if len(createdByJSON) > 0 {
			var creator models.OrgCreator
			if err := json.Unmarshal(createdByJSON, &creator); err == nil {
				user.CreatedBy = &creator
			}
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

// ListCreators returns the distinct creator snapshots of the users visible to
// the caller, for the created_by filter on GET /api/users. One entry per
// user_id (first row seen wins), sorted by name.
func (r *LocalUserRepository) ListCreators(userOrgRole, userOrgID string) ([]models.OrgCreator, error) {
	// Owner sees all users - skip RBAC filtering
	var allowedOrgIDs []string
	if strings.ToLower(userOrgRole) != "owner" {
		var err error
		allowedOrgIDs, err = r.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
		}
		if len(allowedOrgIDs) == 0 {
			return []models.OrgCreator{}, nil
		}
	}

	query := `
		SELECT DISTINCT
			created_by->>'user_id' as user_id,
			created_by->>'name' as name,
			COALESCE(created_by->>'email', '') as email,
			COALESCE(created_by->>'organization_id', '') as organization_id,
			COALESCE(created_by->>'organization_name', '') as organization_name
		FROM users
		WHERE deleted_at IS NULL
			AND created_by->>'user_id' IS NOT NULL
			AND created_by->>'user_id' != ''
			AND created_by->>'name' IS NOT NULL
			AND created_by->>'name' != ''
	`
	var args []interface{}
	if allowedOrgIDs != nil {
		args = append(args, pq.Array(allowedOrgIDs))
		query += ` AND organization_id = ANY($1::text[])`
	}
	query += ` ORDER BY name ASC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query user creators: %w", err)
	}
	defer func() { _ = rows.Close() }()

	creators := make([]models.OrgCreator, 0)
	seen := make(map[string]bool)
	for rows.Next() {
		var c models.OrgCreator
		if err := rows.Scan(&c.UserID, &c.Name, &c.Email, &c.OrganizationID, &c.OrganizationName); err != nil {
			continue
		}
		if seen[c.UserID] {
			continue
		}
		seen[c.UserID] = true
		creators = append(creators, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user creators: %w", err)
	}

	return creators, nil
}

// GetTotals returns user totals (with enabled/suspended breakdown) based on hierarchical RBAC
func (r *LocalUserRepository) GetTotals(userOrgRole, userOrgID string) (*models.UserTotals, error) {
	// Owner can access all users - pass nil to skip RBAC filtering
	var allowedOrgIDs []string
	if strings.ToLower(userOrgRole) != "owner" {
		var err error
		allowedOrgIDs, err = r.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
		}
	}

	return r.GetTotalsByOrganizations(allowedOrgIDs)
}

// GetTotalsByOrganizations returns user totals in specified organizations: the
// total of non-deleted users plus an enabled/suspended breakdown.
// nil allowedOrgIDs = owner (no RBAC filter), empty = no access
func (r *LocalUserRepository) GetTotalsByOrganizations(allowedOrgIDs []string) (*models.UserTotals, error) {
	if allowedOrgIDs != nil && len(allowedOrgIDs) == 0 {
		return &models.UserTotals{}, nil
	}

	// total counts all non-deleted users; enabled/suspended split them by suspended_at.
	query := `SELECT COUNT(*),
	                 COUNT(*) FILTER (WHERE suspended_at IS NULL),
	                 COUNT(*) FILTER (WHERE suspended_at IS NOT NULL)
	          FROM users WHERE deleted_at IS NULL`
	var args []interface{}
	if allowedOrgIDs != nil {
		query += ` AND organization_id = ANY($1::text[])`
		args = []interface{}{pq.Array(allowedOrgIDs)}
	}

	var totals models.UserTotals
	if err := r.db.QueryRow(query, args...).Scan(&totals.Total, &totals.Enabled, &totals.Suspended); err != nil {
		return nil, err
	}
	return &totals, nil
}

// GetHierarchicalOrganizationIDs returns all organization IDs that the user can manage
// This mirrors the logic from UserService.GetHierarchicalOrganizationIDs
func (r *LocalUserRepository) GetHierarchicalOrganizationIDs(userOrgRole, userOrgID string) ([]string, error) {
	orgIDs := []string{userOrgID} // Always include own organization

	// Normalize role to lowercase for case-insensitive comparison
	normalizedRole := strings.ToLower(userOrgRole)

	switch normalizedRole {
	case "owner":
		// Owner can manage all organizations - single UNION query
		query := `
			SELECT logto_id FROM distributors WHERE logto_id IS NOT NULL AND deleted_at IS NULL
			UNION ALL
			SELECT logto_id FROM resellers WHERE logto_id IS NOT NULL AND deleted_at IS NULL
			UNION ALL
			SELECT logto_id FROM customers WHERE logto_id IS NOT NULL AND deleted_at IS NULL
		`
		rows, err := r.db.Query(query)
		if err == nil {
			defer func() { _ = rows.Close() }()
			for rows.Next() {
				var orgID string
				if rows.Scan(&orgID) == nil {
					orgIDs = append(orgIDs, orgID)
				}
			}
		}

		return orgIDs, nil

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

// SoftDeleteByMultipleOrgIDs cascade soft-deletes users belonging to any of the given organizations
func (r *LocalUserRepository) SoftDeleteByMultipleOrgIDs(orgIDs []string, deletedByOrgID string) (int, error) {
	if len(orgIDs) == 0 {
		return 0, nil
	}

	now := time.Now()

	placeholders := make([]string, len(orgIDs))
	args := make([]interface{}, len(orgIDs))
	for i, id := range orgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	deletedByIdx := len(orgIDs) + 1
	nowIdx := len(orgIDs) + 2
	query := fmt.Sprintf(`
		UPDATE users
		SET deleted_at = $%d, deleted_by_org_id = $%d, updated_at = $%d
		WHERE organization_id IN (%s) AND deleted_at IS NULL
	`, nowIdx, deletedByIdx, nowIdx, inClause)

	updateArgs := make([]interface{}, 0, len(orgIDs)+2)
	updateArgs = append(updateArgs, args...)
	updateArgs = append(updateArgs, deletedByOrgID, now)

	result, err := r.db.Exec(query, updateArgs...)
	if err != nil {
		return 0, fmt.Errorf("failed to cascade soft-delete users: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// RestoreByDeletedByOrgID cascade restores users that were soft-deleted by a specific organization
func (r *LocalUserRepository) RestoreByDeletedByOrgID(deletedByOrgID string) (int, error) {
	now := time.Now()

	query := `
		UPDATE users
		SET deleted_at = NULL, deleted_by_org_id = NULL, updated_at = $2
		WHERE deleted_by_org_id = $1 AND deleted_at IS NOT NULL
	`

	result, err := r.db.Exec(query, deletedByOrgID, now)
	if err != nil {
		return 0, fmt.Errorf("failed to cascade restore users: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// Restore restores a single soft-deleted user by logto_id
func (r *LocalUserRepository) Restore(logtoID string) error {
	now := time.Now()
	query := `UPDATE users SET deleted_at = NULL, deleted_by_org_id = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NOT NULL`

	result, err := r.db.Exec(query, logtoID, now)
	if err != nil {
		return fmt.Errorf("failed to restore user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or not deleted")
	}

	return nil
}

// HardDelete permanently deletes a user by logto_id
func (r *LocalUserRepository) HardDelete(logtoID string) error {
	query := `DELETE FROM users WHERE logto_id = $1`

	result, err := r.db.Exec(query, logtoID)
	if err != nil {
		return fmt.Errorf("failed to hard-delete user: %w", err)
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

// HardDeleteByOrgID permanently deletes all users belonging to an organization
// Returns the logto_ids of deleted users for Logto cleanup
func (r *LocalUserRepository) HardDeleteByOrgID(orgLogtoID string) ([]string, error) {
	// First get the logto_ids of users that will be deleted (for Logto cleanup)
	selectQuery := `SELECT logto_id FROM users WHERE organization_id = $1 AND logto_id IS NOT NULL`
	rows, err := r.db.Query(selectQuery, orgLogtoID)
	if err != nil {
		return nil, fmt.Errorf("failed to query users for hard-delete: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var logtoIDs []string
	for rows.Next() {
		var logtoID string
		if err := rows.Scan(&logtoID); err != nil {
			return nil, fmt.Errorf("failed to scan user logto_id: %w", err)
		}
		logtoIDs = append(logtoIDs, logtoID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	// Hard-delete all users for this org
	deleteQuery := `DELETE FROM users WHERE organization_id = $1`
	_, err = r.db.Exec(deleteQuery, orgLogtoID)
	if err != nil {
		return nil, fmt.Errorf("failed to hard-delete users by org: %w", err)
	}

	return logtoIDs, nil
}

// GetByIDIncludeDeleted retrieves a user by logto_id including soft-deleted users
func (r *LocalUserRepository) GetByIDIncludeDeleted(id string) (*models.LocalUser, error) {
	query := `
		SELECT u.id, u.logto_id, u.username, u.email, u.name, u.phone, u.organization_id, u.user_role_ids, u.custom_data, u.created_by,
		       u.created_at, u.updated_at, u.logto_synced_at, u.latest_login_at, u.deleted_at, u.suspended_at, u.suspended_by_org_id,
		       COALESCE(d.name, r.name, c.name) as organization_name,
		       COALESCE(d.id, r.id, c.id) as organization_local_id,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM users u
		LEFT JOIN distributors d ON (u.organization_id = d.logto_id OR u.organization_id = d.id::text)
		LEFT JOIN resellers r ON (u.organization_id = r.logto_id OR u.organization_id = r.id::text)
		LEFT JOIN customers c ON (u.organization_id = c.logto_id OR u.organization_id = c.id::text)
		WHERE u.logto_id = $1
	`

	user := &models.LocalUser{}
	var customDataJSON []byte
	var userRoleIDsJSON []byte
	var createdByJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name, &user.Phone,
		&user.OrganizationID, &userRoleIDsJSON, &customDataJSON, &createdByJSON,
		&user.CreatedAt, &user.UpdatedAt, &user.LogtoSyncedAt, &user.LatestLoginAt, &user.DeletedAt, &user.SuspendedAt, &user.SuspendedByOrgID,
		&user.OrganizationName, &user.OrganizationLocalID, &user.OrganizationType,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if len(userRoleIDsJSON) > 0 {
		if err := json.Unmarshal(userRoleIDsJSON, &user.UserRoleIDs); err != nil {
			user.UserRoleIDs = []string{}
		}
	} else {
		user.UserRoleIDs = []string{}
	}

	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &user.CustomData); err != nil {
			user.CustomData = make(map[string]interface{})
		}
	} else {
		user.CustomData = make(map[string]interface{})
	}

	// Parse created_by JSON (creator snapshot; may be NULL on pre-backfill rows)
	if len(createdByJSON) > 0 {
		var creator models.OrgCreator
		if err := json.Unmarshal(createdByJSON, &creator); err == nil {
			user.CreatedBy = &creator
		}
	}

	_ = r.enrichUserWithRelations(user)

	return user, nil
}

// GetAvatar retrieves the avatar binary and MIME type for a user by logto_id.
func (r *LocalUserRepository) GetAvatar(logtoID string) ([]byte, string, error) {
	query := `SELECT avatar, avatar_mime FROM users WHERE logto_id = $1 AND deleted_at IS NULL LIMIT 1`

	var avatar []byte
	var mime sql.NullString
	err := r.db.QueryRow(query, logtoID).Scan(&avatar, &mime)
	if err != nil {
		return nil, "", err
	}
	if avatar == nil {
		return nil, "", nil
	}

	return avatar, mime.String, nil
}

// SetAvatar stores the avatar binary and MIME type for a user by logto ID.
func (r *LocalUserRepository) SetAvatar(logtoID string, data []byte, mime string) error {
	query := `UPDATE users SET avatar = $2, avatar_mime = $3, updated_at = NOW() WHERE logto_id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(query, logtoID, data, mime)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// DeleteAvatar removes the avatar for a user by logto ID.
func (r *LocalUserRepository) DeleteAvatar(logtoID string) error {
	query := `UPDATE users SET avatar = NULL, avatar_mime = NULL, updated_at = NOW() WHERE logto_id = $1 AND deleted_at IS NULL`
	result, err := r.db.Exec(query, logtoID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
