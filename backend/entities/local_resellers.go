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
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalResellerRepository implements ResellerRepository for local database
type LocalResellerRepository struct {
	db *sql.DB
}

// NewLocalResellerRepository creates a new local reseller repository
func NewLocalResellerRepository() *LocalResellerRepository {
	return &LocalResellerRepository{
		db: database.DB,
	}
}

// Create creates a new reseller in local database
func (r *LocalResellerRepository) Create(req *models.CreateLocalResellerRequest) (*models.LocalReseller, error) {
	id := uuid.New().String()
	now := time.Now()

	customDataJSON, err := json.Marshal(req.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	query := `
		INSERT INTO resellers (id, logto_id, name, description, custom_data, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(query, id, nil, req.Name, req.Description, customDataJSON, now, now, nil)
	if err != nil {
		// Check for VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("already exists")
		}
		return nil, fmt.Errorf("failed to create reseller: %w", err)
	}

	return &models.LocalReseller{
		ID:          id,
		LogtoID:     nil,
		Name:        req.Name,
		Description: req.Description,
		CustomData:  req.CustomData,
		CreatedAt:   now,
		UpdatedAt:   now,
		DeletedAt:   nil,
	}, nil
}

// GetByID retrieves a reseller by ID from local database
func (r *LocalResellerRepository) GetByID(id string) (*models.LocalReseller, error) {
	query := `
		SELECT id, logto_id, name, description, custom_data, created_at, updated_at,
		       logto_synced_at, logto_sync_error, deleted_at, suspended_at, suspended_by_org_id
		FROM resellers
		WHERE logto_id = $1 AND deleted_at IS NULL
	`

	reseller := &models.LocalReseller{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&reseller.ID, &reseller.LogtoID, &reseller.Name, &reseller.Description,
		&customDataJSON, &reseller.CreatedAt, &reseller.UpdatedAt,
		&reseller.LogtoSyncedAt, &reseller.LogtoSyncError, &reseller.DeletedAt,
		&reseller.SuspendedAt, &reseller.SuspendedByOrgID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reseller not found")
		}
		return nil, fmt.Errorf("failed to get reseller: %w", err)
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &reseller.CustomData); err != nil {
			reseller.CustomData = make(map[string]interface{})
		}
	} else {
		reseller.CustomData = make(map[string]interface{})
	}

	return reseller, nil
}

// Update updates a reseller in local database
func (r *LocalResellerRepository) Update(id string, req *models.UpdateLocalResellerRequest) (*models.LocalReseller, error) {
	// First get the current reseller
	current, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Description != nil {
		current.Description = *req.Description
	}
	if req.CustomData != nil {
		current.CustomData = *req.CustomData
	}

	current.UpdatedAt = time.Now()
	current.LogtoSyncedAt = nil // Mark as needing sync

	customDataJSON, err := json.Marshal(current.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	query := `
		UPDATE resellers
		SET name = $2, description = $3, custom_data = $4, updated_at = $5, logto_synced_at = NULL
		WHERE logto_id = $1
	`

	_, err = r.db.Exec(query, id, current.Name, current.Description, customDataJSON, current.UpdatedAt)
	if err != nil {
		// Check for VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("already exists")
		}
		return nil, fmt.Errorf("failed to update reseller: %w", err)
	}

	return current, nil
}

// Delete soft-deletes a reseller in local database
func (r *LocalResellerRepository) Delete(id string) error {
	query := `UPDATE resellers SET deleted_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete reseller: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reseller not found")
	}

	return nil
}

// Suspend suspends a reseller in local database
func (r *LocalResellerRepository) Suspend(id string) error {
	query := `UPDATE resellers SET suspended_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to suspend reseller: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reseller not found or already suspended")
	}

	return nil
}

// Reactivate reactivates a suspended reseller in local database
func (r *LocalResellerRepository) Reactivate(id string) error {
	query := `UPDATE resellers SET suspended_at = NULL, suspended_by_org_id = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reactivate reseller: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reseller not found or not suspended")
	}

	return nil
}

// List returns paginated list of resellers visible to the user
func (r *LocalResellerRepository) List(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalReseller, int, error) {
	offset := (page - 1) * pageSize

	switch userOrgRole {
	case "owner":
		return r.listForOwner(page, pageSize, offset, search, sortBy, sortDirection, statuses)
	case "distributor":
		return r.listForDistributor(userOrgID, page, pageSize, offset, search, sortBy, sortDirection, statuses)
	default:
		// Resellers and customers can't see other resellers
		return []*models.LocalReseller{}, 0, nil
	}
}

// listForOwner handles reseller listing for owner role
func (r *LocalResellerRepository) listForOwner(page, pageSize, offset int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalReseller, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":         "name",
			"description":  "description",
			"created_at":   "created_at",
			"updated_at":   "updated_at",
			"suspended_at": "suspended_at",
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
			statusConditions = append(statusConditions, "(deleted_at IS NULL AND suspended_at IS NULL)")
		case "suspended":
			statusConditions = append(statusConditions, "(deleted_at IS NULL AND suspended_at IS NOT NULL)")
		case "deleted":
			hasDeletedFilter = true
			statusConditions = append(statusConditions, "(deleted_at IS NOT NULL)")
		}
	}

	deletedClause := " AND deleted_at IS NULL"
	if hasDeletedFilter {
		deletedClause = ""
	}

	statusClause := ""
	if len(statusConditions) > 0 {
		statusClause = " AND (" + strings.Join(statusConditions, " OR ") + ")"
	}

	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM resellers WHERE 1=1%s%s AND (LOWER(name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(description) LIKE LOWER('%%' || $1 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $1 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{search}

		query = fmt.Sprintf(`
			SELECT r.id, r.logto_id, r.name, r.description, r.custom_data, r.created_at, r.updated_at,
			       r.logto_synced_at, r.logto_sync_error, r.deleted_at, r.suspended_at, r.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = r.logto_id AND s.deleted_at IS NULL) as systems_count,
			       (SELECT COUNT(*) FROM customers c WHERE c.custom_data->>'createdBy' = r.logto_id AND c.deleted_at IS NULL) as customers_count
			FROM resellers r
			WHERE 1=1%s%s AND (LOWER(r.name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(r.description) LIKE LOWER('%%' || $1 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(r.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $1 || '%%')))
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM resellers WHERE 1=1%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{}

		query = fmt.Sprintf(`
			SELECT r.id, r.logto_id, r.name, r.description, r.custom_data, r.created_at, r.updated_at,
			       r.logto_synced_at, r.logto_sync_error, r.deleted_at, r.suspended_at, r.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = r.logto_id AND s.deleted_at IS NULL) as systems_count,
			       (SELECT COUNT(*) FROM customers c WHERE c.custom_data->>'createdBy' = r.logto_id AND c.deleted_at IS NULL) as customers_count
			FROM resellers r
			WHERE 1=1%s%s
			%s
			LIMIT $1 OFFSET $2
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{pageSize, offset}
	}

	return r.executeResellerQuery(countQuery, countArgs, query, queryArgs)
}

// listForDistributor handles reseller listing for distributor role
func (r *LocalResellerRepository) listForDistributor(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalReseller, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":         "name",
			"description":  "description",
			"created_at":   "created_at",
			"updated_at":   "updated_at",
			"suspended_at": "suspended_at",
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
			statusConditions = append(statusConditions, "(deleted_at IS NULL AND suspended_at IS NULL)")
		case "suspended":
			statusConditions = append(statusConditions, "(deleted_at IS NULL AND suspended_at IS NOT NULL)")
		case "deleted":
			hasDeletedFilter = true
			statusConditions = append(statusConditions, "(deleted_at IS NOT NULL)")
		}
	}

	deletedClause := " AND deleted_at IS NULL"
	if hasDeletedFilter {
		deletedClause = ""
	}

	statusClause := ""
	if len(statusConditions) > 0 {
		statusClause = " AND (" + strings.Join(statusConditions, " OR ") + ")"
	}

	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM resellers WHERE custom_data->>'createdBy' = $1%s%s AND (LOWER(name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT r.id, r.logto_id, r.name, r.description, r.custom_data, r.created_at, r.updated_at,
			       r.logto_synced_at, r.logto_sync_error, r.deleted_at, r.suspended_at, r.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = r.logto_id AND s.deleted_at IS NULL) as systems_count,
			       (SELECT COUNT(*) FROM customers c WHERE c.custom_data->>'createdBy' = r.logto_id AND c.deleted_at IS NULL) as customers_count
			FROM resellers r
			WHERE r.custom_data->>'createdBy' = $1%s%s AND (LOWER(r.name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(r.description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(r.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))
			%s
			LIMIT $3 OFFSET $4
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM resellers WHERE custom_data->>'createdBy' = $1%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT r.id, r.logto_id, r.name, r.description, r.custom_data, r.created_at, r.updated_at,
			       r.logto_synced_at, r.logto_sync_error, r.deleted_at, r.suspended_at, r.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = r.logto_id AND s.deleted_at IS NULL) as systems_count,
			       (SELECT COUNT(*) FROM customers c WHERE c.custom_data->>'createdBy' = r.logto_id AND c.deleted_at IS NULL) as customers_count
			FROM resellers r
			WHERE r.custom_data->>'createdBy' = $1%s%s
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, pageSize, offset}
	}

	return r.executeResellerQuery(countQuery, countArgs, query, queryArgs)
}

// executeResellerQuery executes the count and query operations
func (r *LocalResellerRepository) executeResellerQuery(countQuery string, countArgs []interface{}, query string, queryArgs []interface{}) ([]*models.LocalReseller, int, error) {
	// Get total count
	var totalCount int
	if len(countArgs) > 0 {
		err := r.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get resellers count: %w", err)
		}
	} else {
		err := r.db.QueryRow(countQuery).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get resellers count: %w", err)
		}
	}

	// Get paginated results
	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query resellers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var resellers []*models.LocalReseller
	for rows.Next() {
		reseller := &models.LocalReseller{}
		var customDataJSON []byte
		var systemsCount, customersCount int

		err := rows.Scan(
			&reseller.ID, &reseller.LogtoID, &reseller.Name, &reseller.Description,
			&customDataJSON, &reseller.CreatedAt, &reseller.UpdatedAt,
			&reseller.LogtoSyncedAt, &reseller.LogtoSyncError, &reseller.DeletedAt,
			&reseller.SuspendedAt, &reseller.SuspendedByOrgID,
			&systemsCount, &customersCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan reseller: %w", err)
		}

		// Parse custom_data JSON
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &reseller.CustomData); err != nil {
				reseller.CustomData = make(map[string]interface{})
			}
		} else {
			reseller.CustomData = make(map[string]interface{})
		}

		reseller.SystemsCount = &systemsCount
		reseller.CustomersCount = &customersCount

		resellers = append(resellers, reseller)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating resellers: %w", err)
	}

	return resellers, totalCount, nil
}

// GetTotals returns total count of resellers visible to the user
func (r *LocalResellerRepository) GetTotals(userOrgRole, userOrgID string) (int, error) {
	var count int
	var query string

	switch userOrgRole {
	case "owner":
		// Owner sees all resellers
		query = `SELECT COUNT(*) FROM resellers WHERE deleted_at IS NULL`
		err := r.db.QueryRow(query).Scan(&count)
		return count, err

	case "distributor":
		// Distributor sees resellers they created (hierarchy via custom_data)
		query = `SELECT COUNT(*) FROM resellers WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1`
		err := r.db.QueryRow(query, userOrgID).Scan(&count)
		return count, err

	default:
		// Resellers and customers can't see resellers
		return 0, nil
	}
}

// GetTrend returns trend data for resellers over a specified period
func (r *LocalResellerRepository) GetTrend(userOrgRole, userOrgID string, period int) ([]struct {
	Date  string
	Count int
}, int, int, error) {
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

	var query string

	switch userOrgRole {
	case "owner":
		// Owner sees all resellers
		query = fmt.Sprintf(`
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
					FROM resellers
					WHERE deleted_at IS NULL
					  AND created_at::date <= ds.date
				), 0) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period, interval)

	case "distributor":
		// Distributor sees resellers they created
		query = fmt.Sprintf(`
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
					FROM resellers
					WHERE deleted_at IS NULL
					  AND custom_data->>'createdBy' = '%s'
					  AND created_at::date <= ds.date
				), 0) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period, interval, userOrgID)

	default:
		// Resellers and customers can't see resellers
		return []struct {
			Date  string
			Count int
		}{}, 0, 0, nil
	}

	rows, err := r.db.Query(query)
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

// GetByIDIncludeDeleted retrieves a reseller by logto_id including soft-deleted ones
func (r *LocalResellerRepository) GetByIDIncludeDeleted(id string) (*models.LocalReseller, error) {
	query := `
		SELECT id, logto_id, name, description, custom_data, created_at, updated_at,
		       logto_synced_at, logto_sync_error, deleted_at, suspended_at, suspended_by_org_id
		FROM resellers
		WHERE logto_id = $1
	`

	reseller := &models.LocalReseller{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&reseller.ID, &reseller.LogtoID, &reseller.Name, &reseller.Description,
		&customDataJSON, &reseller.CreatedAt, &reseller.UpdatedAt,
		&reseller.LogtoSyncedAt, &reseller.LogtoSyncError, &reseller.DeletedAt,
		&reseller.SuspendedAt, &reseller.SuspendedByOrgID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("reseller not found")
		}
		return nil, fmt.Errorf("failed to get reseller: %w", err)
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &reseller.CustomData); err != nil {
			reseller.CustomData = make(map[string]interface{})
		}
	} else {
		reseller.CustomData = make(map[string]interface{})
	}

	return reseller, nil
}

// Restore restores a soft-deleted reseller
func (r *LocalResellerRepository) Restore(logtoID string) error {
	query := `UPDATE resellers SET deleted_at = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NOT NULL`

	result, err := r.db.Exec(query, logtoID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to restore reseller: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reseller not found or not deleted")
	}

	return nil
}

// HardDelete permanently removes a reseller from the database
func (r *LocalResellerRepository) HardDelete(logtoID string) error {
	query := `DELETE FROM resellers WHERE logto_id = $1`

	result, err := r.db.Exec(query, logtoID)
	if err != nil {
		return fmt.Errorf("failed to hard-delete reseller: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reseller not found")
	}

	return nil
}

// GetStats returns users, systems, customers and applications count for a specific reseller
func (r *LocalResellerRepository) GetStats(id string) (*models.ResellerStats, error) {
	// First get the reseller to obtain its logto_id
	reseller, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	// If reseller has no logto_id, return zero counts
	if reseller.LogtoID == nil {
		return &models.ResellerStats{
			UsersCount:                 0,
			SystemsCount:               0,
			CustomersCount:             0,
			ApplicationsCount:          0,
			ApplicationsHierarchyCount: 0,
		}, nil
	}

	var stats models.ResellerStats
	query := `
		SELECT
			(SELECT COUNT(*) FROM users WHERE organization_id = $1 AND deleted_at IS NULL) as users_count,
			(SELECT COUNT(*) FROM systems WHERE organization_id = $1 AND deleted_at IS NULL) as systems_count,
			(SELECT COUNT(*) FROM customers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL) as customers_count,
			(SELECT COUNT(*) FROM applications WHERE organization_id = $1 AND deleted_at IS NULL) as applications_count,
			(SELECT COUNT(*) FROM applications a WHERE a.deleted_at IS NULL AND (
				a.organization_id = $1
				OR a.organization_id IN (SELECT logto_id FROM customers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL)
			)) as applications_hierarchy_count
	`

	err = r.db.QueryRow(query, *reseller.LogtoID).Scan(
		&stats.UsersCount, &stats.SystemsCount, &stats.CustomersCount,
		&stats.ApplicationsCount, &stats.ApplicationsHierarchyCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get reseller stats: %w", err)
	}

	return &stats, nil
}

// GetLogtoIDsByCreatedBy returns logto_ids of active resellers created by a specific organization
func (r *LocalResellerRepository) GetLogtoIDsByCreatedBy(createdByOrgID string) ([]string, error) {
	query := `
		SELECT logto_id FROM resellers
		WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL AND logto_id IS NOT NULL
	`

	rows, err := r.db.Query(query, createdByOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query reseller logto_ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var logtoIDs []string
	for rows.Next() {
		var logtoID string
		if err := rows.Scan(&logtoID); err != nil {
			return nil, fmt.Errorf("failed to scan reseller logto_id: %w", err)
		}
		logtoIDs = append(logtoIDs, logtoID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating reseller logto_ids: %w", err)
	}

	return logtoIDs, nil
}

// SuspendWithCascadeOrigin suspends a reseller and records the originating org for cascade tracking
func (r *LocalResellerRepository) SuspendWithCascadeOrigin(id, suspendedByOrgID string) error {
	query := `UPDATE resellers SET suspended_at = $2, suspended_by_org_id = $3, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now(), suspendedByOrgID)
	if err != nil {
		return fmt.Errorf("failed to suspend reseller: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("reseller not found or already suspended")
	}

	return nil
}

// SuspendByCreatedBy suspends all resellers created by a specific org, setting suspended_by_org_id
// Returns the logto_ids of suspended resellers for cascade propagation
func (r *LocalResellerRepository) SuspendByCreatedBy(createdByOrgID, suspendedByOrgID string) ([]string, int, error) {
	now := time.Now()

	// Get logto_ids of resellers that will be suspended
	selectQuery := `
		SELECT logto_id FROM resellers
		WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL AND suspended_at IS NULL AND logto_id IS NOT NULL
	`
	rows, err := r.db.Query(selectQuery, createdByOrgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query resellers for cascade suspension: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var logtoIDs []string
	for rows.Next() {
		var logtoID string
		if err := rows.Scan(&logtoID); err != nil {
			return nil, 0, fmt.Errorf("failed to scan reseller logto_id: %w", err)
		}
		logtoIDs = append(logtoIDs, logtoID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating resellers: %w", err)
	}

	// Suspend all matching resellers
	updateQuery := `
		UPDATE resellers
		SET suspended_at = $3, suspended_by_org_id = $2, updated_at = $3
		WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL AND suspended_at IS NULL
	`
	result, err := r.db.Exec(updateQuery, createdByOrgID, suspendedByOrgID, now)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to cascade suspend resellers: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return logtoIDs, int(rowsAffected), nil
}

// ReactivateBySuspendedByOrgID reactivates all resellers that were cascade-suspended by a specific org
func (r *LocalResellerRepository) ReactivateBySuspendedByOrgID(suspendedByOrgID string) (int, error) {
	now := time.Now()

	query := `
		UPDATE resellers
		SET suspended_at = NULL, suspended_by_org_id = NULL, updated_at = $2
		WHERE suspended_by_org_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL
	`

	result, err := r.db.Exec(query, suspendedByOrgID, now)
	if err != nil {
		return 0, fmt.Errorf("failed to cascade reactivate resellers: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}
