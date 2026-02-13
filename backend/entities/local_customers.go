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

// LocalCustomerRepository implements CustomerRepository for local database
type LocalCustomerRepository struct {
	db *sql.DB
}

// NewLocalCustomerRepository creates a new local customer repository
func NewLocalCustomerRepository() *LocalCustomerRepository {
	return &LocalCustomerRepository{
		db: database.DB,
	}
}

// Create creates a new customer in local database
func (r *LocalCustomerRepository) Create(req *models.CreateLocalCustomerRequest) (*models.LocalCustomer, error) {
	id := uuid.New().String()
	now := time.Now()

	customDataJSON, err := json.Marshal(req.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	query := `
		INSERT INTO customers (id, logto_id, name, description, custom_data, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(query, id, nil, req.Name, req.Description, customDataJSON, now, now, nil)
	if err != nil {
		// Check for global VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("already exists")
		}
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return &models.LocalCustomer{
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

// GetByID retrieves a customer by ID from local database
func (r *LocalCustomerRepository) GetByID(id string) (*models.LocalCustomer, error) {
	query := `
		SELECT id, logto_id, name, description, custom_data,
		       created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at, suspended_at, suspended_by_org_id
		FROM customers
		WHERE logto_id = $1 AND deleted_at IS NULL
	`

	customer := &models.LocalCustomer{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&customer.ID, &customer.LogtoID, &customer.Name, &customer.Description,
		&customDataJSON, &customer.CreatedAt, &customer.UpdatedAt,
		&customer.LogtoSyncedAt, &customer.LogtoSyncError, &customer.DeletedAt,
		&customer.SuspendedAt, &customer.SuspendedByOrgID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &customer.CustomData); err != nil {
			customer.CustomData = make(map[string]interface{})
		}
	} else {
		customer.CustomData = make(map[string]interface{})
	}

	return customer, nil
}

// Update updates a customer in local database
func (r *LocalCustomerRepository) Update(id string, req *models.UpdateLocalCustomerRequest) (*models.LocalCustomer, error) {
	// First get the current customer
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
		UPDATE customers
		SET name = $2, description = $3, custom_data = $4, updated_at = $5, logto_synced_at = NULL
		WHERE logto_id = $1
	`

	_, err = r.db.Exec(query, id, current.Name, current.Description, customDataJSON, current.UpdatedAt)
	if err != nil {
		// Check for VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("already exists")
		}
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return current, nil
}

// Delete soft-deletes a customer in local database
func (r *LocalCustomerRepository) Delete(id string) error {
	query := `UPDATE customers SET deleted_at = NOW(), updated_at = $2 WHERE logto_id = $1`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

// Suspend suspends a customer in local database
func (r *LocalCustomerRepository) Suspend(id string) error {
	query := `UPDATE customers SET suspended_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to suspend customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found or already suspended")
	}

	return nil
}

// Reactivate reactivates a suspended customer in local database
func (r *LocalCustomerRepository) Reactivate(id string) error {
	query := `UPDATE customers SET suspended_at = NULL, suspended_by_org_id = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reactivate customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found or not suspended")
	}

	return nil
}

// List returns paginated list of customers visible to the user
func (r *LocalCustomerRepository) List(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalCustomer, int, error) {
	offset := (page - 1) * pageSize

	switch userOrgRole {
	case "owner":
		return r.listForOwner(page, pageSize, offset, search, sortBy, sortDirection, statuses)
	case "distributor":
		return r.listForDistributor(userOrgID, page, pageSize, offset, search, sortBy, sortDirection, statuses)
	case "reseller":
		return r.listForReseller(userOrgID, page, pageSize, offset, search, sortBy, sortDirection, statuses)
	case "customer":
		return r.listForCustomer(userOrgID, page, pageSize, offset, search, sortBy, sortDirection, statuses)
	default:
		return []*models.LocalCustomer{}, 0, nil
	}
}

// listForOwner handles customer listing for owner role
func (r *LocalCustomerRepository) listForOwner(page, pageSize, offset int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalCustomer, int, error) {
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
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE 1=1%s%s AND (LOWER(name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(description) LIKE LOWER('%%' || $1 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $1 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{search}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE 1=1%s%s AND (LOWER(c.name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(c.description) LIKE LOWER('%%' || $1 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(c.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $1 || '%%')))
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE 1=1%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE 1=1%s%s
			%s
			LIMIT $1 OFFSET $2
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// listForDistributor handles customer listing for distributor role
func (r *LocalCustomerRepository) listForDistributor(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalCustomer, int, error) {
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
			statusConditions = append(statusConditions, "(c.deleted_at IS NULL AND c.suspended_at IS NULL)")
		case "suspended":
			statusConditions = append(statusConditions, "(c.deleted_at IS NULL AND c.suspended_at IS NOT NULL)")
		case "deleted":
			hasDeletedFilter = true
			statusConditions = append(statusConditions, "(c.deleted_at IS NOT NULL)")
		}
	}

	deletedClause := " AND c.deleted_at IS NULL"
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
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*) FROM customers c
			WHERE (
				c.custom_data->>'createdBy' = $1 OR
				c.custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)%s%s AND (LOWER(c.name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(c.description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(c.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE (
				c.custom_data->>'createdBy' = $1 OR
				c.custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)%s%s AND (LOWER(c.name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(c.description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(c.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))
			%s
			LIMIT $3 OFFSET $4
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`
			SELECT COUNT(*) FROM customers c
			WHERE (
				c.custom_data->>'createdBy' = $1 OR
				c.custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE (
				c.custom_data->>'createdBy' = $1 OR
				c.custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)%s%s
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// listForReseller handles customer listing for reseller role
func (r *LocalCustomerRepository) listForReseller(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalCustomer, int, error) {
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
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE custom_data->>'createdBy' = $1%s%s AND (LOWER(name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE c.custom_data->>'createdBy' = $1%s%s AND (LOWER(c.name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(c.description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(c.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))
			%s
			LIMIT $3 OFFSET $4
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE custom_data->>'createdBy' = $1%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE c.custom_data->>'createdBy' = $1%s%s
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// listForCustomer handles customer listing for customer role
func (r *LocalCustomerRepository) listForCustomer(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalCustomer, int, error) {
	if userOrgID == "" {
		return []*models.LocalCustomer{}, 0, nil
	}

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
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE id = $1%s%s AND (LOWER(name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE c.id = $1%s%s AND (LOWER(c.name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(c.description) LIKE LOWER('%%' || $2 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(c.custom_data) AS kv(key, value) WHERE kv.key != 'createdBy' AND LOWER(kv.value) LIKE LOWER('%%' || $2 || '%%')))
			%s
			LIMIT $3 OFFSET $4
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM customers WHERE id = $1%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT c.id, c.logto_id, c.name, c.description,
			       c.custom_data, c.created_at, c.updated_at, c.logto_synced_at, c.logto_sync_error, c.deleted_at, c.suspended_at, c.suspended_by_org_id,
			       (SELECT COUNT(*) FROM systems s WHERE s.organization_id = c.logto_id AND s.deleted_at IS NULL) as systems_count
			FROM customers c
			WHERE c.id = $1%s%s
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{userOrgID, pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// executeCustomerQuery executes the count and query operations
func (r *LocalCustomerRepository) executeCustomerQuery(countQuery string, countArgs []interface{}, query string, queryArgs []interface{}) ([]*models.LocalCustomer, int, error) {
	// Get total count
	var totalCount int
	if len(countArgs) > 0 {
		err := r.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get customers count: %w", err)
		}
	} else {
		err := r.db.QueryRow(countQuery).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get customers count: %w", err)
		}
	}

	// Get paginated results
	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query customers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var customers []*models.LocalCustomer
	for rows.Next() {
		customer := &models.LocalCustomer{}
		var customDataJSON []byte
		var systemsCount int

		err := rows.Scan(
			&customer.ID, &customer.LogtoID, &customer.Name, &customer.Description,
			&customDataJSON, &customer.CreatedAt, &customer.UpdatedAt,
			&customer.LogtoSyncedAt, &customer.LogtoSyncError, &customer.DeletedAt,
			&customer.SuspendedAt, &customer.SuspendedByOrgID,
			&systemsCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan customer: %w", err)
		}

		// Parse custom_data JSON
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &customer.CustomData); err != nil {
				customer.CustomData = make(map[string]interface{})
			}
		} else {
			customer.CustomData = make(map[string]interface{})
		}

		customer.SystemsCount = &systemsCount

		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating customers: %w", err)
	}

	return customers, totalCount, nil
}

// GetTotals returns total count of customers visible to the user
func (r *LocalCustomerRepository) GetTotals(userOrgRole, userOrgID string) (int, error) {
	var count int
	var query string

	switch userOrgRole {
	case "owner":
		// Owner sees all customers
		query = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL`
		err := r.db.QueryRow(query).Scan(&count)
		return count, err

	case "distributor":
		// Distributor sees customers they created (hierarchy via custom_data)
		query = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1`
		err := r.db.QueryRow(query, userOrgID).Scan(&count)
		return count, err

	case "reseller":
		// Reseller sees customers they created (hierarchy via custom_data)
		query = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1`
		err := r.db.QueryRow(query, userOrgID).Scan(&count)
		return count, err

	case "customer":
		// Customers see only themselves
		if userOrgID != "" {
			query = `SELECT COUNT(*) FROM customers WHERE id = $1 AND deleted_at IS NULL`
			err := r.db.QueryRow(query, userOrgID).Scan(&count)
			return count, err
		}
		return 0, nil

	default:
		return 0, nil
	}
}

// GetTrend returns trend data for customers over a specified period
func (r *LocalCustomerRepository) GetTrend(userOrgRole, userOrgID string, period int) ([]struct {
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
		// Owner sees all customers
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
					FROM customers
					WHERE deleted_at IS NULL
					  AND created_at::date <= ds.date
				), 0) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period, interval)

	case "distributor":
		// Distributor sees customers they or their resellers created
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
					FROM customers
					WHERE deleted_at IS NULL
					  AND (
					    custom_data->>'createdBy' = '%s'
					    OR custom_data->>'createdBy' IN (
					      SELECT logto_id FROM resellers
					      WHERE custom_data->>'createdBy' = '%s' AND deleted_at IS NULL
					    )
					  )
					  AND created_at::date <= ds.date
				), 0) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period, interval, userOrgID, userOrgID)

	case "reseller":
		// Reseller sees customers they created
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
					FROM customers
					WHERE deleted_at IS NULL
					  AND custom_data->>'createdBy' = '%s'
					  AND created_at::date <= ds.date
				), 0) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period, interval, userOrgID)

	case "customer":
		// Customer sees only themselves (no trend, just 0 or 1)
		if userOrgID == "" {
			return []struct {
				Date  string
				Count int
			}{}, 0, 0, nil
		}
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
					FROM customers
					WHERE deleted_at IS NULL
					  AND id = '%s'
					  AND created_at::date <= ds.date
				), 0) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period, interval, userOrgID)

	default:
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

// GetByIDIncludeDeleted retrieves a customer by logto_id including soft-deleted ones
func (r *LocalCustomerRepository) GetByIDIncludeDeleted(id string) (*models.LocalCustomer, error) {
	query := `
		SELECT id, logto_id, name, description, custom_data,
		       created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at, suspended_at, suspended_by_org_id
		FROM customers
		WHERE logto_id = $1
	`

	customer := &models.LocalCustomer{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&customer.ID, &customer.LogtoID, &customer.Name, &customer.Description,
		&customDataJSON, &customer.CreatedAt, &customer.UpdatedAt,
		&customer.LogtoSyncedAt, &customer.LogtoSyncError, &customer.DeletedAt,
		&customer.SuspendedAt, &customer.SuspendedByOrgID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer not found")
		}
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &customer.CustomData); err != nil {
			customer.CustomData = make(map[string]interface{})
		}
	} else {
		customer.CustomData = make(map[string]interface{})
	}

	return customer, nil
}

// Restore restores a soft-deleted customer
func (r *LocalCustomerRepository) Restore(logtoID string) error {
	query := `UPDATE customers SET deleted_at = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NOT NULL`

	result, err := r.db.Exec(query, logtoID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to restore customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found or not deleted")
	}

	return nil
}

// HardDelete permanently removes a customer from the database
func (r *LocalCustomerRepository) HardDelete(logtoID string) error {
	query := `DELETE FROM customers WHERE logto_id = $1`

	result, err := r.db.Exec(query, logtoID)
	if err != nil {
		return fmt.Errorf("failed to hard-delete customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

// GetStats returns users, systems and applications count for a specific customer
func (r *LocalCustomerRepository) GetStats(id string) (*models.CustomerStats, error) {
	// First get the customer to obtain its logto_id
	customer, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	// If customer has no logto_id, return zero counts
	if customer.LogtoID == nil {
		return &models.CustomerStats{
			UsersCount:        0,
			SystemsCount:      0,
			ApplicationsCount: 0,
		}, nil
	}

	var stats models.CustomerStats
	query := `
		SELECT
			(SELECT COUNT(*) FROM users WHERE organization_id = $1 AND deleted_at IS NULL) as users_count,
			(SELECT COUNT(*) FROM systems WHERE organization_id = $1 AND deleted_at IS NULL) as systems_count,
			(SELECT COUNT(*) FROM applications WHERE organization_id = $1 AND deleted_at IS NULL) as applications_count
	`

	err = r.db.QueryRow(query, *customer.LogtoID).Scan(&stats.UsersCount, &stats.SystemsCount, &stats.ApplicationsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer stats: %w", err)
	}

	return &stats, nil
}

// GetLogtoIDsByCreatedByMultiple returns logto_ids of active customers created by any of the given organizations
func (r *LocalCustomerRepository) GetLogtoIDsByCreatedByMultiple(createdByOrgIDs []string) ([]string, error) {
	if len(createdByOrgIDs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(createdByOrgIDs))
	args := make([]interface{}, len(createdByOrgIDs))
	for i, id := range createdByOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	query := fmt.Sprintf(`
		SELECT logto_id FROM customers
		WHERE custom_data->>'createdBy' IN (%s) AND deleted_at IS NULL AND logto_id IS NOT NULL
	`, inClause)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query customer logto_ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var logtoIDs []string
	for rows.Next() {
		var logtoID string
		if err := rows.Scan(&logtoID); err != nil {
			return nil, fmt.Errorf("failed to scan customer logto_id: %w", err)
		}
		logtoIDs = append(logtoIDs, logtoID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer logto_ids: %w", err)
	}

	return logtoIDs, nil
}

// SuspendWithCascadeOrigin suspends a customer and records the originating org for cascade tracking
func (r *LocalCustomerRepository) SuspendWithCascadeOrigin(id, suspendedByOrgID string) error {
	query := `UPDATE customers SET suspended_at = $2, suspended_by_org_id = $3, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now(), suspendedByOrgID)
	if err != nil {
		return fmt.Errorf("failed to suspend customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found or already suspended")
	}

	return nil
}

// SuspendByCreatedByMultiple suspends all customers created by any of the given org IDs, setting suspended_by_org_id
// Returns the logto_ids of suspended customers for cascade propagation
func (r *LocalCustomerRepository) SuspendByCreatedByMultiple(createdByOrgIDs []string, suspendedByOrgID string) ([]string, int, error) {
	if len(createdByOrgIDs) == 0 {
		return nil, 0, nil
	}

	now := time.Now()

	// Build placeholders for IN clause
	placeholders := make([]string, len(createdByOrgIDs))
	args := make([]interface{}, len(createdByOrgIDs))
	for i, id := range createdByOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	inClause := strings.Join(placeholders, ",")

	// Get logto_ids of customers that will be suspended
	selectQuery := fmt.Sprintf(`
		SELECT logto_id FROM customers
		WHERE custom_data->>'createdBy' IN (%s) AND deleted_at IS NULL AND suspended_at IS NULL AND logto_id IS NOT NULL
	`, inClause)

	rows, err := r.db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query customers for cascade suspension: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var logtoIDs []string
	for rows.Next() {
		var logtoID string
		if err := rows.Scan(&logtoID); err != nil {
			return nil, 0, fmt.Errorf("failed to scan customer logto_id: %w", err)
		}
		logtoIDs = append(logtoIDs, logtoID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating customers: %w", err)
	}

	// Suspend all matching customers
	suspendedByIdx := len(createdByOrgIDs) + 1
	nowIdx := len(createdByOrgIDs) + 2
	updateQuery := fmt.Sprintf(`
		UPDATE customers
		SET suspended_at = $%d, suspended_by_org_id = $%d, updated_at = $%d
		WHERE custom_data->>'createdBy' IN (%s) AND deleted_at IS NULL AND suspended_at IS NULL
	`, nowIdx, suspendedByIdx, nowIdx, inClause)

	updateArgs := make([]interface{}, 0, len(createdByOrgIDs)+2)
	updateArgs = append(updateArgs, args...)
	updateArgs = append(updateArgs, suspendedByOrgID, now)

	result, err := r.db.Exec(updateQuery, updateArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to cascade suspend customers: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return logtoIDs, int(rowsAffected), nil
}

// ReactivateBySuspendedByOrgID reactivates all customers that were cascade-suspended by a specific org
func (r *LocalCustomerRepository) ReactivateBySuspendedByOrgID(suspendedByOrgID string) (int, error) {
	now := time.Now()

	query := `
		UPDATE customers
		SET suspended_at = NULL, suspended_by_org_id = NULL, updated_at = $2
		WHERE suspended_by_org_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL
	`

	result, err := r.db.Exec(query, suspendedByOrgID, now)
	if err != nil {
		return 0, fmt.Errorf("failed to cascade reactivate customers: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}
