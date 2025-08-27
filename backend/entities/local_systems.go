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

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalSystemRepository implements SystemRepository for local database
type LocalSystemRepository struct {
	db *sql.DB
}

// NewLocalSystemRepository creates a new local system repository
func NewLocalSystemRepository() *LocalSystemRepository {
	return &LocalSystemRepository{
		db: database.DB,
	}
}

// Create creates a new system in the local database
func (r *LocalSystemRepository) Create(req *models.CreateSystemRequest) (*models.System, error) {
	// Import the service dynamically to avoid circular imports
	// For now, return error indicating method needs implementation
	return nil, fmt.Errorf("Create method not yet implemented - use SystemsService directly")
}

// GetByID retrieves a specific system by ID without access validation (validation is done at service level)
func (r *LocalSystemRepository) GetByID(id string) (*models.System, error) {
	query := `
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version, s.last_seen,
		       s.custom_data, s.created_at, s.updated_at, s.created_by, h.last_heartbeat
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.id = $1 AND s.deleted_at IS NULL
	`

	system := &models.System{}
	var customDataJSON []byte
	var createdByJSON []byte
	var fqdn, ipv4Address, ipv6Address, version sql.NullString
	var lastHeartbeat sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
		&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON,
		&system.CreatedAt, &system.UpdatedAt, &createdByJSON, &lastHeartbeat,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("system not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query system: %w", err)
	}

	// Convert NullString to string
	system.FQDN = fqdn.String
	system.IPv4Address = ipv4Address.String
	system.IPv6Address = ipv6Address.String
	system.Version = version.String

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &system.CustomData); err != nil {
			system.CustomData = make(map[string]string)
		}
	} else {
		system.CustomData = make(map[string]string)
	}

	// Parse created_by JSON
	if len(createdByJSON) > 0 {
		_ = json.Unmarshal(createdByJSON, &system.CreatedBy) // Ignore JSON unmarshal errors - keep default zero value
	}

	// Set heartbeat time for service layer calculation
	var heartbeatTime *time.Time
	if lastHeartbeat.Valid {
		heartbeatTime = &lastHeartbeat.Time
	}
	system.LastHeartbeat = heartbeatTime

	return system, nil
}

// Update updates an existing system with access validation
func (r *LocalSystemRepository) Update(id string, req *models.UpdateSystemRequest) (*models.System, error) {
	// Import the service dynamically to avoid circular imports
	// For now, return error indicating method needs implementation
	return nil, fmt.Errorf("Update method not yet implemented - use SystemsService directly")
}

// Delete deletes a system with access validation
func (r *LocalSystemRepository) Delete(id string) error {
	// Import the service dynamically to avoid circular imports
	// For now, return error indicating method needs implementation
	return fmt.Errorf("Delete method not yet implemented - use SystemsService directly")
}

// ListByCreatedByOrganizations returns paginated list of systems created by users in specified organizations
func (r *LocalSystemRepository) ListByCreatedByOrganizations(allowedOrgIDs []string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.System, int, error) {
	if len(allowedOrgIDs) == 0 {
		return []*models.System{}, 0, nil
	}

	offset := (page - 1) * pageSize

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	baseArgs := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		baseArgs[i] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Build WHERE clause for search
	whereClause := fmt.Sprintf("deleted_at IS NULL AND JSON_EXTRACT(created_by, '$.organization_id') IN (%s)", placeholdersStr)
	args := make([]interface{}, len(baseArgs))
	copy(args, baseArgs)

	if search != "" {
		searchPattern := "%" + search + "%"
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR type ILIKE $%d OR status ILIKE $%d OR fqdn ILIKE $%d OR version ILIKE $%d OR JSON_EXTRACT(created_by, '$.user_name') ILIKE $%d)",
			len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5, len(args)+6)
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	var totalCount int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM systems WHERE %s", whereClause)

	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get systems count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "created_at DESC"
	if sortBy != "" {
		// Map sortBy values to actual column names
		columnMap := map[string]string{
			"name":         "name",
			"type":         "type",
			"status":       "status",
			"fqdn":         "fqdn",
			"version":      "version",
			"created_at":   "created_at",
			"updated_at":   "updated_at",
			"last_seen":    "last_seen",
			"creator_name": "JSON_EXTRACT(created_by, '$.user_name')",
		}

		if column, exists := columnMap[sortBy]; exists {
			direction := "ASC"
			if sortDirection == "desc" {
				direction = "DESC"
			}
			orderBy = fmt.Sprintf("%s %s", column, direction)
		}
	}

	// Build main query
	query := fmt.Sprintf(`
		SELECT id, name, type, status, fqdn, ipv4_address, ipv6_address, version, last_seen,
		       custom_data, created_at, updated_at, created_by
		FROM systems
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, len(args)+1, len(args)+2)

	// Add pagination parameters
	listArgs := make([]interface{}, len(args)+2)
	copy(listArgs, args)
	listArgs[len(args)] = pageSize
	listArgs[len(args)+1] = offset

	rows, err := r.db.Query(query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query systems: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var systems []*models.System
	for rows.Next() {
		system := &models.System{}
		var customDataJSON, createdByJSON []byte
		var fqdn, ipv4Address, ipv6Address, version sql.NullString

		err := rows.Scan(
			&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
			&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON,
			&system.CreatedAt, &system.UpdatedAt, &createdByJSON,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan system: %w", err)
		}

		// Convert NullString to string
		system.FQDN = fqdn.String
		system.IPv4Address = ipv4Address.String
		system.IPv6Address = ipv6Address.String
		system.Version = version.String

		// Parse custom_data JSON
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &system.CustomData); err != nil {
				system.CustomData = make(map[string]string)
			}
		} else {
			system.CustomData = make(map[string]string)
		}

		// Parse created_by JSON
		if len(createdByJSON) > 0 {
			_ = json.Unmarshal(createdByJSON, &system.CreatedBy)
		}

		systems = append(systems, system)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating systems: %w", err)
	}

	return systems, totalCount, nil
}

// GetTotalsByCreatedByOrganizations returns total counts and status for systems created by users in specified organizations
func (r *LocalSystemRepository) GetTotalsByCreatedByOrganizations(allowedOrgIDs []string, timeoutMinutes int) (*models.SystemTotals, error) {
	if len(allowedOrgIDs) == 0 {
		return &models.SystemTotals{
			Total:          0,
			Alive:          0,
			Dead:           0,
			Zombie:         0,
			TimeoutMinutes: timeoutMinutes,
		}, nil
	}

	// Calculate cutoff time for alive/dead determination
	timeout := time.Duration(timeoutMinutes) * time.Minute
	cutoff := time.Now().Add(-timeout)

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, 1+len(allowedOrgIDs)) // +1 for cutoff time
	args[0] = cutoff

	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2) // +2 because $1 is cutoff
		args[i+1] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Base query with heartbeat status calculation and hierarchical filtering
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat > $1 THEN 1 ELSE 0 END) as alive,
			SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat <= $1 THEN 1 ELSE 0 END) as dead,
			SUM(CASE WHEN h.last_heartbeat IS NULL THEN 1 ELSE 0 END) as zombie
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.deleted_at IS NULL AND JSON_EXTRACT(s.created_by, '$.organization_id') IN (%s)
	`, placeholdersStr)

	var total, alive, dead, zombie int
	err := r.db.QueryRow(query, args...).Scan(&total, &alive, &dead, &zombie)
	if err != nil {
		return nil, fmt.Errorf("failed to get systems totals: %w", err)
	}

	return &models.SystemTotals{
		Total:          total,
		Alive:          alive,
		Dead:           dead,
		Zombie:         zombie,
		TimeoutMinutes: timeoutMinutes,
	}, nil
}
