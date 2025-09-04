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
			return nil, fmt.Errorf("VAT already exists in the system")
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
		SELECT id, logto_id, name, description,  custom_data,
		       created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
		FROM customers
		WHERE id = $1 AND deleted_at IS NULL
	`

	customer := &models.LocalCustomer{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&customer.ID, &customer.LogtoID, &customer.Name, &customer.Description,
		&customDataJSON, &customer.CreatedAt, &customer.UpdatedAt,
		&customer.LogtoSyncedAt, &customer.LogtoSyncError, &customer.DeletedAt,
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
		WHERE id = $1
	`

	_, err = r.db.Exec(query, id, current.Name, current.Description, customDataJSON, current.UpdatedAt)
	if err != nil {
		// Check for VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("VAT already exists in customers")
		}
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	return current, nil
}

// Delete soft-deletes a customer in local database
func (r *LocalCustomerRepository) Delete(id string) error {
	query := `UPDATE customers SET deleted_at = NOW(), updated_at = $2 WHERE id = $1`

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

// List returns paginated list of customers visible to the user
func (r *LocalCustomerRepository) List(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalCustomer, int, error) {
	offset := (page - 1) * pageSize

	switch userOrgRole {
	case "owner":
		return r.listForOwner(page, pageSize, offset, search, sortBy, sortDirection)
	case "distributor":
		return r.listForDistributor(userOrgID, page, pageSize, offset, search, sortBy, sortDirection)
	case "reseller":
		return r.listForReseller(userOrgID, page, pageSize, offset, search, sortBy, sortDirection)
	case "customer":
		return r.listForCustomer(userOrgID, page, pageSize, offset, search, sortBy, sortDirection)
	default:
		return []*models.LocalCustomer{}, 0, nil
	}
}

// listForOwner handles customer listing for owner role
func (r *LocalCustomerRepository) listForOwner(page, pageSize, offset int, search, sortBy, sortDirection string) ([]*models.LocalCustomer, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":        "name",
			"description": "description",
			"created_at":  "created_at",
			"updated_at":  "updated_at",
		}

		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND (LOWER(name) LIKE LOWER('%' || $1 || '%') OR LOWER(description) LIKE LOWER('%' || $1 || '%'))`
		countArgs = []interface{}{search}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE deleted_at IS NULL AND (LOWER(name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(description) LIKE LOWER('%%' || $1 || '%%'))
			%s
			LIMIT $2 OFFSET $3
		`, orderClause)
		queryArgs = []interface{}{search, pageSize, offset}
	} else {
		// Without search
		countQuery = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL`
		countArgs = []interface{}{}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE deleted_at IS NULL
			%s
			LIMIT $1 OFFSET $2
		`, orderClause)
		queryArgs = []interface{}{pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// listForDistributor handles customer listing for distributor role
func (r *LocalCustomerRepository) listForDistributor(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string) ([]*models.LocalCustomer, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":        "name",
			"description": "description",
			"created_at":  "created_at",
			"updated_at":  "updated_at",
		}

		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = `
			SELECT COUNT(*) FROM customers
			WHERE deleted_at IS NULL AND (
				custom_data->>'createdBy' = $1 OR
				custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			) AND (LOWER(name) LIKE LOWER('%' || $2 || '%') OR LOWER(description) LIKE LOWER('%' || $2 || '%'))`
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE deleted_at IS NULL AND (
				custom_data->>'createdBy' = $1 OR
				custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			) AND (LOWER(name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(description) LIKE LOWER('%%' || $2 || '%%'))
			%s
			LIMIT $3 OFFSET $4
		`, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = `
			SELECT COUNT(*) FROM customers
			WHERE deleted_at IS NULL AND (
				custom_data->>'createdBy' = $1 OR
				custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)`
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE deleted_at IS NULL AND (
				custom_data->>'createdBy' = $1 OR
				custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)
			%s
			LIMIT $2 OFFSET $3
		`, orderClause)
		queryArgs = []interface{}{userOrgID, pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// listForReseller handles customer listing for reseller role
func (r *LocalCustomerRepository) listForReseller(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string) ([]*models.LocalCustomer, int, error) {
	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":        "name",
			"description": "description",
			"created_at":  "created_at",
			"updated_at":  "updated_at",
		}

		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1 AND (LOWER(name) LIKE LOWER('%' || $2 || '%') OR LOWER(description) LIKE LOWER('%' || $2 || '%'))`
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1 AND (LOWER(name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(description) LIKE LOWER('%%' || $2 || '%%'))
			%s
			LIMIT $3 OFFSET $4
		`, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = `SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1`
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE deleted_at IS NULL AND custom_data->>'createdBy' = $1
			%s
			LIMIT $2 OFFSET $3
		`, orderClause)
		queryArgs = []interface{}{userOrgID, pageSize, offset}
	}

	return r.executeCustomerQuery(countQuery, countArgs, query, queryArgs)
}

// listForCustomer handles customer listing for customer role
func (r *LocalCustomerRepository) listForCustomer(userOrgID string, page, pageSize, offset int, search, sortBy, sortDirection string) ([]*models.LocalCustomer, int, error) {
	if userOrgID == "" {
		return []*models.LocalCustomer{}, 0, nil
	}

	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":        "name",
			"description": "description",
			"created_at":  "created_at",
			"updated_at":  "updated_at",
		}

		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = `SELECT COUNT(*) FROM customers WHERE id = $1 AND deleted_at IS NULL AND (LOWER(name) LIKE LOWER('%' || $2 || '%') OR LOWER(description) LIKE LOWER('%' || $2 || '%'))`
		countArgs = []interface{}{userOrgID, search}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE id = $1 AND deleted_at IS NULL AND (LOWER(name) LIKE LOWER('%%' || $2 || '%%') OR LOWER(description) LIKE LOWER('%%' || $2 || '%%'))
			%s
			LIMIT $3 OFFSET $4
		`, orderClause)
		queryArgs = []interface{}{userOrgID, search, pageSize, offset}
	} else {
		// Without search
		countQuery = `SELECT COUNT(*) FROM customers WHERE id = $1 AND deleted_at IS NULL`
		countArgs = []interface{}{userOrgID}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description,
			       custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
			FROM customers
			WHERE id = $1 AND deleted_at IS NULL
			%s
			LIMIT $2 OFFSET $3
		`, orderClause)
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

		err := rows.Scan(
			&customer.ID, &customer.LogtoID, &customer.Name, &customer.Description,
			&customDataJSON, &customer.CreatedAt, &customer.UpdatedAt,
			&customer.LogtoSyncedAt, &customer.LogtoSyncError, &customer.DeletedAt,
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
