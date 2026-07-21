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
	return r.create(r.db, req)
}

// CreateWithTx creates a new distributor inside the provided transaction so the row
// participates in the caller's atomic create-and-sync flow; a later failure rolls
// the insert back instead of leaving an orphaned org (logto_id IS NULL).
func (r *LocalDistributorRepository) CreateWithTx(tx *sql.Tx, req *models.CreateLocalDistributorRequest) (*models.LocalDistributor, error) {
	return r.create(tx, req)
}

func (r *LocalDistributorRepository) create(exec dbExecer, req *models.CreateLocalDistributorRequest) (*models.LocalDistributor, error) {
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

	_, err = exec.Exec(query, id, nil, req.Name, req.Description, customDataJSON, now, now, nil)
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
		       logto_synced_at, logto_sync_error, deleted_at, suspended_at
		FROM distributors
		WHERE logto_id = $1 AND deleted_at IS NULL
	`

	distributor := &models.LocalDistributor{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&distributor.ID, &distributor.LogtoID, &distributor.Name, &distributor.Description,
		&customDataJSON, &distributor.CreatedAt, &distributor.UpdatedAt,
		&distributor.LogtoSyncedAt, &distributor.LogtoSyncError, &distributor.DeletedAt,
		&distributor.SuspendedAt,
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

	distributor.CreatedBy = models.ExtractOrgCreator(distributor.CustomData)

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
		WHERE logto_id = $1
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
	query := `UPDATE distributors SET deleted_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL`

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

// Suspend suspends a distributor in local database
func (r *LocalDistributorRepository) Suspend(id string) error {
	query := `UPDATE distributors SET suspended_at = $2, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to suspend distributor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("distributor not found or already suspended")
	}

	return nil
}

// Reactivate reactivates a suspended distributor in local database
func (r *LocalDistributorRepository) Reactivate(id string) error {
	query := `UPDATE distributors SET suspended_at = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NULL AND suspended_at IS NOT NULL`

	result, err := r.db.Exec(query, id, time.Now())
	if err != nil {
		return fmt.Errorf("failed to reactivate distributor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("distributor not found or not suspended")
	}

	return nil
}

// List returns paginated list of distributors visible to the user
func (r *LocalDistributorRepository) List(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses, createdBy []string) ([]*models.LocalDistributor, int, error) {
	// Only Owner can see distributors
	if userOrgRole != "owner" {
		return []*models.LocalDistributor{}, 0, nil
	}

	offset := (page - 1) * pageSize

	// Validate and build sorting clause
	orderClause := "ORDER BY created_at DESC" // default sorting
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":         "LOWER(name)",
			"description":  "LOWER(description)",
			"created_at":   "created_at",
			"updated_at":   "updated_at",
			"suspended_at": "suspended_at",
			"creator_name": "LOWER(custom_data->'createdByUser'->>'name')",
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

	// Restrict to the requested creators (matches the systems created_by filter:
	// either the creating user or their organization). Flows into both queries
	// via statusClause, so no positional-arg reshuffling is needed.
	statusClause += createdByFilterClause(createdBy)

	// Build queries with optional search and status filter
	var countQuery, query string
	var countArgs, queryArgs []interface{}

	if search != "" {
		// With search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM distributors WHERE 1=1%s%s AND (LOWER(name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(description) LIKE LOWER('%%' || $1 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(custom_data) AS kv(key, value) WHERE kv.key NOT IN ('createdBy', 'createdByUser') AND LOWER(kv.value) LIKE LOWER('%%' || $1 || '%%')))`, deletedClause, statusClause)
		countArgs = []interface{}{search}

		query = fmt.Sprintf(`
			SELECT d.id, d.logto_id, d.name, d.description, d.custom_data, d.created_at, d.updated_at,
			       d.logto_synced_at, d.logto_sync_error, d.deleted_at, d.suspended_at
			FROM distributors d
			WHERE 1=1%s%s AND (LOWER(d.name) LIKE LOWER('%%' || $1 || '%%') OR LOWER(d.description) LIKE LOWER('%%' || $1 || '%%') OR EXISTS (SELECT 1 FROM jsonb_each_text(d.custom_data) AS kv(key, value) WHERE kv.key NOT IN ('createdBy', 'createdByUser') AND LOWER(kv.value) LIKE LOWER('%%' || $1 || '%%')))
			%s
			LIMIT $2 OFFSET $3
		`, deletedClause, statusClause, orderClause)
		queryArgs = []interface{}{search, pageSize, offset}
	} else {
		// Without search
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM distributors WHERE 1=1%s%s`, deletedClause, statusClause)
		countArgs = []interface{}{}

		query = fmt.Sprintf(`
			SELECT d.id, d.logto_id, d.name, d.description, d.custom_data, d.created_at, d.updated_at,
			       d.logto_synced_at, d.logto_sync_error, d.deleted_at, d.suspended_at
			FROM distributors d
			WHERE 1=1%s%s
			%s
			LIMIT $1 OFFSET $2
		`, deletedClause, statusClause, orderClause)
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
			&distributor.SuspendedAt,
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

		distributor.CreatedBy = models.ExtractOrgCreator(distributor.CustomData)
		distributors = append(distributors, distributor)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating distributors: %w", err)
	}

	// Counts (systems/resellers/customers/applications per distributor subtree)
	// are computed in a small number of single-scan queries rather than as
	// per-row correlated subqueries: a distributor's subtree spans almost the
	// whole database, so the correlated form re-scanned systems/applications
	// once per row. See populateDistributorCounts.
	if err := r.populateDistributorCounts(distributors); err != nil {
		return nil, 0, fmt.Errorf("failed to populate distributor counts: %w", err)
	}

	return distributors, totalCount, nil
}

// populateDistributorCounts fills SystemsCount/ResellersCount/CustomersCount/
// ApplicationsCount for the given distributors in a fixed, small number of
// single-scan queries independent of the number of distributors on the page.
//
// A distributor's subtree (itself + its resellers + all their customers) spans
// almost the entire org tree, so the previous per-row correlated subqueries
// re-scanned systems and applications once for every distributor row. Instead
// we build the org->distributor map once (two indexed lookups over resellers
// and customers) and fold per-organization counts into it, so systems and
// applications are each scanned a single time.
func (r *LocalDistributorRepository) populateDistributorCounts(distributors []*models.LocalDistributor) error {
	// Initialise every distributor to zero so the API always returns the fields,
	// and index the ones that have a logto_id (unsynced rows own nothing yet).
	byLogto := make(map[string]*models.LocalDistributor, len(distributors))
	distIDs := make([]string, 0, len(distributors))
	for _, d := range distributors {
		zero := 0
		sc, rc, cc, ac := zero, zero, zero, zero
		d.SystemsCount, d.ResellersCount, d.CustomersCount, d.ApplicationsCount = &sc, &rc, &cc, &ac
		if d.LogtoID != nil && *d.LogtoID != "" {
			byLogto[*d.LogtoID] = d
			distIDs = append(distIDs, *d.LogtoID)
		}
	}
	if len(distIDs) == 0 {
		return nil
	}

	// orgToDist maps every organization in any page distributor's subtree to
	// that distributor's logto_id: the distributor itself, its resellers, and
	// the customers owned directly by it or by one of its resellers.
	orgToDist := make(map[string]string)
	for _, id := range distIDs {
		orgToDist[id] = id
	}

	// Resellers created by the page distributors.
	resToDist := make(map[string]string)
	rows, err := r.db.Query(`SELECT logto_id, custom_data->>'createdBy' FROM resellers WHERE deleted_at IS NULL AND logto_id IS NOT NULL AND custom_data->>'createdBy' = ANY($1)`, pq.Array(distIDs))
	if err != nil {
		return fmt.Errorf("failed to query resellers for counts: %w", err)
	}
	var resellerIDs []string
	for rows.Next() {
		var resLogto, distID string
		if err := rows.Scan(&resLogto, &distID); err != nil {
			_ = rows.Close()
			return fmt.Errorf("failed to scan reseller for counts: %w", err)
		}
		resToDist[resLogto] = distID
		orgToDist[resLogto] = distID
		resellerIDs = append(resellerIDs, resLogto)
		if d := byLogto[distID]; d != nil {
			*d.ResellersCount++
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("error iterating resellers for counts: %w", err)
	}
	_ = rows.Close()

	// Customers created directly by the page distributors.
	if err := r.foldCustomers(`custom_data->>'createdBy' = ANY($1)`, pq.Array(distIDs), func(custLogto, createdBy string) string {
		return createdBy // createdBy is the distributor itself
	}, orgToDist, byLogto); err != nil {
		return err
	}

	// Customers created by resellers of the page distributors.
	if len(resellerIDs) > 0 {
		if err := r.foldCustomers(`custom_data->>'createdBy' = ANY($1)`, pq.Array(resellerIDs), func(custLogto, createdBy string) string {
			return resToDist[createdBy] // map reseller -> its distributor
		}, orgToDist, byLogto); err != nil {
			return err
		}
	}

	// Fold per-organization system and certified-application counts into the map.
	// The queries scan each table once, grouped by organization, with no org
	// filter: a distributor's subtree covers almost every organization, so a
	// single sequential scan is cheaper than probing the index for thousands of
	// IDs. Organizations outside any page distributor's subtree resolve to no
	// distributor and are ignored.
	if err := r.foldOrgCounts(
		`SELECT organization_id, COUNT(*) FROM systems WHERE deleted_at IS NULL AND organization_id IS NOT NULL GROUP BY organization_id`,
		orgToDist, byLogto, func(d *models.LocalDistributor) *int { return d.SystemsCount },
	); err != nil {
		return fmt.Errorf("failed to fold system counts: %w", err)
	}
	if err := r.foldOrgCounts(
		`SELECT organization_id, COUNT(*) FROM applications WHERE deleted_at IS NULL AND organization_id IS NOT NULL AND (inventory_data->>'certification_level')::int IN (4, 5) GROUP BY organization_id`,
		orgToDist, byLogto, func(d *models.LocalDistributor) *int { return d.ApplicationsCount },
	); err != nil {
		return fmt.Errorf("failed to fold application counts: %w", err)
	}

	return nil
}

// foldCustomers runs a customers query filtered by whereExpr/arg, and for each
// row records the customer's owning distributor (resolved by distFor) into
// orgToDist while incrementing that distributor's CustomersCount.
func (r *LocalDistributorRepository) foldCustomers(whereExpr string, arg interface{}, distFor func(custLogto, createdBy string) string, orgToDist map[string]string, byLogto map[string]*models.LocalDistributor) error {
	query := fmt.Sprintf(`SELECT logto_id, custom_data->>'createdBy' FROM customers WHERE deleted_at IS NULL AND logto_id IS NOT NULL AND %s`, whereExpr)
	rows, err := r.db.Query(query, arg)
	if err != nil {
		return fmt.Errorf("failed to query customers for counts: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var custLogto, createdBy string
		if err := rows.Scan(&custLogto, &createdBy); err != nil {
			return fmt.Errorf("failed to scan customer for counts: %w", err)
		}
		distID := distFor(custLogto, createdBy)
		if distID == "" {
			continue
		}
		orgToDist[custLogto] = distID
		if d := byLogto[distID]; d != nil {
			*d.CustomersCount++
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating customers for counts: %w", err)
	}
	return nil
}

// foldOrgCounts runs a per-organization COUNT(*) query (one scan, grouped by
// organization) and adds each organization's count to its owning distributor's
// counter (selected by pick). Organizations with no owning distributor in the
// current page are ignored.
func (r *LocalDistributorRepository) foldOrgCounts(query string, orgToDist map[string]string, byLogto map[string]*models.LocalDistributor, pick func(*models.LocalDistributor) *int) error {
	rows, err := r.db.Query(query)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var orgID string
		var n int
		if err := rows.Scan(&orgID, &n); err != nil {
			return err
		}
		if d := byLogto[orgToDist[orgID]]; d != nil {
			*pick(d) += n
		}
	}
	return rows.Err()
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

// GetTrend returns trend data for distributors over a specified period
func (r *LocalDistributorRepository) GetTrend(userOrgRole, userOrgID string, period int) ([]struct {
	Date  string
	Count int
}, int, int, error) {
	// Only Owner can see distributors
	if userOrgRole != "owner" {
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
				FROM distributors
				WHERE deleted_at IS NULL
				  AND created_at::date <= ds.date
			), 0) AS count
		FROM date_series ds
		ORDER BY ds.date
	`, period, interval)

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

// GetByIDIncludeDeleted retrieves a distributor by logto_id including soft-deleted ones
func (r *LocalDistributorRepository) GetByIDIncludeDeleted(id string) (*models.LocalDistributor, error) {
	query := `
		SELECT id, logto_id, name, description, custom_data, created_at, updated_at,
		       logto_synced_at, logto_sync_error, deleted_at, suspended_at
		FROM distributors
		WHERE logto_id = $1
	`

	distributor := &models.LocalDistributor{}
	var customDataJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&distributor.ID, &distributor.LogtoID, &distributor.Name, &distributor.Description,
		&customDataJSON, &distributor.CreatedAt, &distributor.UpdatedAt,
		&distributor.LogtoSyncedAt, &distributor.LogtoSyncError, &distributor.DeletedAt,
		&distributor.SuspendedAt,
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

	distributor.CreatedBy = models.ExtractOrgCreator(distributor.CustomData)

	return distributor, nil
}

// Restore restores a soft-deleted distributor
func (r *LocalDistributorRepository) Restore(logtoID string) error {
	query := `UPDATE distributors SET deleted_at = NULL, updated_at = $2 WHERE logto_id = $1 AND deleted_at IS NOT NULL`

	result, err := r.db.Exec(query, logtoID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to restore distributor: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("distributor not found or not deleted")
	}

	return nil
}

// HardDelete permanently removes a distributor from the database
func (r *LocalDistributorRepository) HardDelete(logtoID string) error {
	query := `DELETE FROM distributors WHERE logto_id = $1`

	result, err := r.db.Exec(query, logtoID)
	if err != nil {
		return fmt.Errorf("failed to hard-delete distributor: %w", err)
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

// GetStats returns users, systems, resellers, customers and applications count for a specific distributor
func (r *LocalDistributorRepository) GetStats(id string) (*models.DistributorStats, error) {
	// First get the distributor to obtain its logto_id
	distributor, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	// If distributor has no logto_id, return zero counts
	if distributor.LogtoID == nil {
		return &models.DistributorStats{
			UsersCount:                 0,
			UsersHierarchyCount:        0,
			SystemsCount:               0,
			SystemsHierarchyCount:      0,
			ResellersCount:             0,
			CustomersCount:             0,
			ApplicationsCount:          0,
			ApplicationsHierarchyCount: 0,
		}, nil
	}

	var stats models.DistributorStats
	query := `
		SELECT
			(SELECT COUNT(*) FROM users WHERE organization_id = $1 AND deleted_at IS NULL) as users_count,
			(SELECT COUNT(*) FROM users u WHERE u.deleted_at IS NULL AND (
				u.organization_id = $1
				OR u.organization_id IN (SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL)
				OR u.organization_id IN (
					SELECT c.logto_id FROM customers c
					WHERE c.deleted_at IS NULL AND (
						c.custom_data->>'createdBy' = $1
						OR c.custom_data->>'createdBy' IN (
							SELECT logto_id FROM resellers
							WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
						)
					)
				)
			)) as users_hierarchy_count,
			(SELECT COUNT(*) FROM systems WHERE organization_id = $1 AND deleted_at IS NULL) as systems_count,
			(SELECT COUNT(*) FROM systems s WHERE s.deleted_at IS NULL AND (
				s.organization_id = $1
				OR s.organization_id IN (SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL)
				OR s.organization_id IN (
					SELECT c.logto_id FROM customers c
					WHERE c.deleted_at IS NULL AND (
						c.custom_data->>'createdBy' = $1
						OR c.custom_data->>'createdBy' IN (
							SELECT logto_id FROM resellers
							WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
						)
					)
				)
			)) as systems_hierarchy_count,
			(SELECT COUNT(*) FROM resellers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL) as resellers_count,
			(SELECT COUNT(*) FROM customers c WHERE c.deleted_at IS NULL AND (
				c.custom_data->>'createdBy' = $1
				OR c.custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
				)
			)) as customers_count,
			(SELECT COUNT(*) FROM applications WHERE organization_id = $1 AND deleted_at IS NULL AND (inventory_data->>'certification_level')::int IN (4, 5)) as applications_count,
			(SELECT COUNT(*) FROM applications a WHERE a.deleted_at IS NULL AND (a.inventory_data->>'certification_level')::int IN (4, 5) AND (
				a.organization_id = $1
				OR a.organization_id IN (SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL)
				OR a.organization_id IN (
					SELECT c.logto_id FROM customers c
					WHERE c.deleted_at IS NULL AND (
						c.custom_data->>'createdBy' = $1
						OR c.custom_data->>'createdBy' IN (
							SELECT logto_id FROM resellers
							WHERE custom_data->>'createdBy' = $1 AND deleted_at IS NULL
						)
					)
				)
			)) as applications_hierarchy_count
	`

	err = r.db.QueryRow(query, *distributor.LogtoID).Scan(
		&stats.UsersCount, &stats.UsersHierarchyCount,
		&stats.SystemsCount, &stats.SystemsHierarchyCount,
		&stats.ResellersCount, &stats.CustomersCount,
		&stats.ApplicationsCount, &stats.ApplicationsHierarchyCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get distributor stats: %w", err)
	}

	return &stats, nil
}
