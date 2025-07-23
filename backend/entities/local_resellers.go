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
		INSERT INTO resellers (id, logto_id, name, description, custom_data, created_at, updated_at, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = r.db.Exec(query, id, nil, req.Name, req.Description, customDataJSON, now, now, true)
	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Message, "uk_resellers_name_created_by") {
				return nil, fmt.Errorf("reseller name already exists for this creator")
			}
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
		Active:      true,
	}, nil
}

// GetByID retrieves a reseller by ID from local database
func (r *LocalResellerRepository) GetByID(id string) (*models.LocalReseller, error) {
	query := `
		SELECT id, logto_id, name, description,  custom_data, created_at, updated_at, 
		       logto_synced_at, logto_sync_error, active
		FROM resellers 
		WHERE id = $1 AND active = TRUE
	`

	reseller := &models.LocalReseller{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&reseller.ID, &reseller.LogtoID, &reseller.Name, &reseller.Description,
		&customDataJSON, &reseller.CreatedAt, &reseller.UpdatedAt,
		&reseller.LogtoSyncedAt, &reseller.LogtoSyncError, &reseller.Active,
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
		WHERE id = $1
	`

	_, err = r.db.Exec(query, id, current.Name, current.Description, customDataJSON, current.UpdatedAt)
	if err != nil {
		// Check for unique constraint violation
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Message, "uk_resellers_name_created_by") {
				return nil, fmt.Errorf("reseller name already exists for this creator")
			}
		}
		return nil, fmt.Errorf("failed to update reseller: %w", err)
	}

	return current, nil
}

// Delete soft-deletes a reseller in local database
func (r *LocalResellerRepository) Delete(id string) error {
	query := `UPDATE resellers SET active = FALSE, updated_at = $2 WHERE id = $1`

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

// List returns paginated list of resellers visible to the user
func (r *LocalResellerRepository) List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.LocalReseller, int, error) {
	offset := (page - 1) * pageSize
	var baseQuery, countQuery string
	var args []interface{}

	switch userOrgRole {
	case "owner":
		// Owner sees all resellers
		countQuery = `SELECT COUNT(*) FROM resellers WHERE active = TRUE`
		baseQuery = `
			SELECT id, logto_id, name, description, custom_data, created_at, updated_at, 
			       logto_synced_at, logto_sync_error, active
			FROM resellers 
			WHERE active = TRUE
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{pageSize, offset}

	case "distributor":
		// Distributor sees resellers they created (hierarchy via custom_data)
		countQuery = `SELECT COUNT(*) FROM resellers WHERE active = TRUE AND custom_data->>'createdBy' = $1`
		baseQuery = `
			SELECT id, logto_id, name, description, custom_data, created_at, updated_at, 
			       logto_synced_at, logto_sync_error, active
			FROM resellers 
			WHERE active = TRUE AND custom_data->>'createdBy' = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userOrgID, pageSize, offset}

	default:
		// Resellers and customers can't see other resellers
		return []*models.LocalReseller{}, 0, nil
	}

	// Get total count
	var totalCount int
	if userOrgRole == "distributor" {
		err := r.db.QueryRow(countQuery, userOrgID).Scan(&totalCount)
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
	rows, err := r.db.Query(baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query resellers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var resellers []*models.LocalReseller
	for rows.Next() {
		reseller := &models.LocalReseller{}
		var customDataJSON []byte

		err := rows.Scan(
			&reseller.ID, &reseller.LogtoID, &reseller.Name, &reseller.Description,
			&customDataJSON, &reseller.CreatedAt, &reseller.UpdatedAt,
			&reseller.LogtoSyncedAt, &reseller.LogtoSyncError, &reseller.Active,
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
		query = `SELECT COUNT(*) FROM resellers WHERE active = TRUE`
		err := r.db.QueryRow(query).Scan(&count)
		return count, err

	case "distributor":
		// Distributor sees resellers they created (hierarchy via custom_data)
		query = `SELECT COUNT(*) FROM resellers WHERE active = TRUE AND custom_data->>'createdBy' = $1`
		err := r.db.QueryRow(query, userOrgID).Scan(&count)
		return count, err

	default:
		// Resellers and customers can't see resellers
		return 0, nil
	}
}
