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

// LocalDistributorRepository implements DistributorRepository for local database
type LocalDistributorRepository struct {
	db *sql.DB
}

// NewLocalDistributorRepository creates a new local distributor repository
func NewLocalDistributorRepository() *LocalDistributorRepository {
	return &LocalDistributorRepository{
		db: database.DB,
	}
}

// Create creates a new distributor in local database
func (r *LocalDistributorRepository) Create(req *models.CreateLocalDistributorRequest) (*models.LocalDistributor, error) {
	id := uuid.New().String()
	now := time.Now()

	customDataJSON, err := json.Marshal(req.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	query := `
		INSERT INTO distributors (id, logto_id, name, description, custom_data, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(query, id, nil, req.Name, req.Description, customDataJSON, now, now, nil)
	if err != nil {
		// Check for VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("already exists")
		}
		return nil, fmt.Errorf("failed to create distributor: %w", err)
	}

	return &models.LocalDistributor{
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

// GetByID retrieves a distributor by ID from local database
func (r *LocalDistributorRepository) GetByID(id string) (*models.LocalDistributor, error) {
	query := `
		SELECT id, logto_id, name, description, custom_data, created_at, updated_at,
		       logto_synced_at, logto_sync_error, deleted_at
		FROM distributors
		WHERE id = $1 AND deleted_at IS NULL
	`

	distributor := &models.LocalDistributor{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&distributor.ID, &distributor.LogtoID, &distributor.Name, &distributor.Description,
		&customDataJSON, &distributor.CreatedAt, &distributor.UpdatedAt,
		&distributor.LogtoSyncedAt, &distributor.LogtoSyncError, &distributor.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("distributor not found")
		}
		return nil, fmt.Errorf("failed to get distributor: %w", err)
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &distributor.CustomData); err != nil {
			distributor.CustomData = make(map[string]interface{})
		}
	} else {
		distributor.CustomData = make(map[string]interface{})
	}

	return distributor, nil
}

// Update updates a distributor in local database
func (r *LocalDistributorRepository) Update(id string, req *models.UpdateLocalDistributorRequest) (*models.LocalDistributor, error) {
	// First get the current distributor
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
		UPDATE distributors
		SET name = $2, description = $3, custom_data = $4, updated_at = $5, logto_synced_at = NULL
		WHERE id = $1
	`

	_, err = r.db.Exec(query, id, current.Name, current.Description, customDataJSON, current.UpdatedAt)
	if err != nil {
		// Check for VAT constraint violation (from trigger function)
		if strings.Contains(err.Error(), "VAT") && strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("already exists")
		}
		return nil, fmt.Errorf("failed to update distributor: %w", err)
	}

	return current, nil
}

// Delete soft-deletes a distributor in local database
func (r *LocalDistributorRepository) Delete(id string) error {
	query := `UPDATE distributors SET deleted_at = $2, updated_at = $2 WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete distributor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("distributor not found")
	}

	return nil
}

// List returns paginated list of distributors visible to the user
func (r *LocalDistributorRepository) List(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalDistributor, int, error) {
	// Only Owner can see distributors
	if userOrgRole != "owner" {
		return []*models.LocalDistributor{}, 0, nil
	}

	offset := (page - 1) * pageSize

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

	// Build queries with optional search
	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = `SELECT COUNT(*) FROM distributors WHERE deleted_at IS NULL AND (LOWER(name) LIKE LOWER('%' || $1 || '%') OR LOWER(description) LIKE LOWER('%' || $1 || '%'))`
		countArgs = []interface{}{search}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description, custom_data, created_at, updated_at,
			       logto_synced_at, logto_sync_error, deleted_at
			FROM distributors
			WHERE deleted_at IS NULL AND (LOWER(name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(description) LIKE LOWER('%%' || $1 || '%%'))
			%s
			LIMIT $2 OFFSET $3
		`, orderClause)
		queryArgs = []interface{}{search, pageSize, offset}
	} else {
		// Without search
		countQuery = `SELECT COUNT(*) FROM distributors WHERE deleted_at IS NULL`
		countArgs = []interface{}{}

		query = fmt.Sprintf(`
			SELECT id, logto_id, name, description, custom_data, created_at, updated_at,
			       logto_synced_at, logto_sync_error, deleted_at
			FROM distributors
			WHERE deleted_at IS NULL
			%s
			LIMIT $1 OFFSET $2
		`, orderClause)
		queryArgs = []interface{}{pageSize, offset}
	}

	// Get total count
	var totalCount int
	if len(countArgs) > 0 {
		err := r.db.QueryRow(countQuery, countArgs...).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get distributors count: %w", err)
		}
	} else {
		err := r.db.QueryRow(countQuery).Scan(&totalCount)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get distributors count: %w", err)
		}
	}

	// Get paginated results
	rows, err := r.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query distributors: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var distributors []*models.LocalDistributor
	for rows.Next() {
		distributor := &models.LocalDistributor{}
		var customDataJSON []byte

		err := rows.Scan(
			&distributor.ID, &distributor.LogtoID, &distributor.Name, &distributor.Description,
			&customDataJSON, &distributor.CreatedAt, &distributor.UpdatedAt,
			&distributor.LogtoSyncedAt, &distributor.LogtoSyncError, &distributor.DeletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan distributor: %w", err)
		}

		// Parse custom_data JSON
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &distributor.CustomData); err != nil {
				distributor.CustomData = make(map[string]interface{})
			}
		} else {
			distributor.CustomData = make(map[string]interface{})
		}

		distributors = append(distributors, distributor)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating distributors: %w", err)
	}

	return distributors, totalCount, nil
}

// GetTotals returns total count of distributors visible to the user
func (r *LocalDistributorRepository) GetTotals(userOrgRole, userOrgID string) (int, error) {
	// Only Owner can see distributors
	if userOrgRole != "owner" {
		return 0, nil
	}

	var count int
	query := `SELECT COUNT(*) FROM distributors WHERE deleted_at IS NULL`

	err := r.db.QueryRow(query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get distributors count: %w", err)
	}

	return count, nil
}
