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
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version,
		       s.system_key, s.organization_id, s.custom_data, s.notes, s.created_at, s.updated_at, s.created_by, h.last_heartbeat,
		       COALESCE(d.name, r.name, c.name, 'Owner') as organization_name,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		LEFT JOIN distributors d ON s.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON s.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON s.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE s.id = $1 AND s.deleted_at IS NULL
	`

	system := &models.System{}
	var customDataJSON []byte
	var createdByJSON []byte
	var fqdn, ipv4Address, ipv6Address, version sql.NullString
	var lastHeartbeat sql.NullTime
	var organizationName, organizationType sql.NullString

	err := r.db.QueryRow(query, id).Scan(
		&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
		&ipv4Address, &ipv6Address, &version, &system.SystemKey, &system.Organization.ID,
		&customDataJSON, &system.Notes, &system.CreatedAt, &system.UpdatedAt, &createdByJSON, &lastHeartbeat,
		&organizationName, &organizationType,
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
	system.Organization.Name = organizationName.String
	system.Organization.Type = organizationType.String

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

// ListByCreatedByOrganizations returns paginated list of systems created by users in specified organizations with filters
func (r *LocalSystemRepository) ListByCreatedByOrganizations(allowedOrgIDs []string, page, pageSize int, search, sortBy, sortDirection, filterName, filterSystemKey string, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses []string) ([]*models.System, int, error) {
	if len(allowedOrgIDs) == 0 {
		return []*models.System{}, 0, nil
	}

	offset := (page - 1) * pageSize

	// Build placeholders for IN clause (allowed organizations)
	placeholders := make([]string, len(allowedOrgIDs))
	baseArgs := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		baseArgs[i] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Check if status filter includes "deleted"
	hasDeletedFilter := false
	for _, status := range filterStatuses {
		if status == "deleted" {
			hasDeletedFilter = true
			break
		}
	}

	// Build WHERE clause for search and filters
	// By default exclude deleted systems unless explicitly requested via status filter
	var whereClause string
	if hasDeletedFilter {
		// Include deleted systems when status filter contains "deleted"
		whereClause = fmt.Sprintf("s.created_by ->> 'organization_id' IN (%s)", placeholdersStr)
	} else {
		// Normal case: exclude deleted systems
		whereClause = fmt.Sprintf("s.deleted_at IS NULL AND s.created_by ->> 'organization_id' IN (%s)", placeholdersStr)
	}
	args := make([]interface{}, len(baseArgs))
	copy(args, baseArgs)

	// Add search condition (handle nullable fields with COALESCE)
	if search != "" {
		searchPattern := "%" + search + "%"
		whereClause += fmt.Sprintf(" AND (s.name ILIKE $%d OR COALESCE(s.type, '') ILIKE $%d OR COALESCE(s.status, '') ILIKE $%d OR s.fqdn ILIKE $%d OR s.version ILIKE $%d OR s.created_by ->> 'name' ILIKE $%d OR s.created_by ->> 'email' ILIKE $%d OR s.system_key ILIKE $%d)",
			len(args)+1, len(args)+2, len(args)+3, len(args)+4, len(args)+5, len(args)+6, len(args)+7, len(args)+8)
		args = append(args, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Add filter conditions
	if filterName != "" {
		whereClause += fmt.Sprintf(" AND s.name ILIKE $%d", len(args)+1)
		args = append(args, "%"+filterName+"%")
	}

	if filterSystemKey != "" {
		whereClause += fmt.Sprintf(" AND s.system_key = $%d", len(args)+1)
		args = append(args, filterSystemKey)
	}

	// Add multiple value filters with IN clauses
	if len(filterTypes) > 0 {
		typePlaceholders := make([]string, len(filterTypes))
		baseIndex := len(args)
		for i, t := range filterTypes {
			typePlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, t)
		}
		whereClause += fmt.Sprintf(" AND s.type IN (%s)", strings.Join(typePlaceholders, ","))
	}

	if len(filterCreatedBy) > 0 {
		// Build separate conditions for each ID to check both user_id and organization_id
		// Each ID gets checked against both fields with OR logic
		conditions := make([]string, len(filterCreatedBy))
		for i, id := range filterCreatedBy {
			placeholder := fmt.Sprintf("$%d", len(args)+1)
			args = append(args, id)
			// Match either user_id OR organization_id for this specific ID
			conditions[i] = fmt.Sprintf("(s.created_by ->> 'user_id' = %s OR s.created_by ->> 'organization_id' = %s)", placeholder, placeholder)
		}
		// Combine all conditions with OR (match any of the provided IDs)
		whereClause += fmt.Sprintf(" AND (%s)", strings.Join(conditions, " OR "))
	}

	if len(filterVersions) > 0 {
		// Version filter now uses prefixed format "product:version" (e.g., "nsec:1.2.3")
		// to avoid ambiguity when same version exists for multiple products
		versionConditions := make([]string, len(filterVersions))
		baseIndex := len(args)

		for i, prefixedVersion := range filterVersions {
			// Split prefixed version into product and version parts
			parts := strings.SplitN(prefixedVersion, ":", 2)
			if len(parts) == 2 {
				// Prefixed format: match both type and version
				productPlaceholder := fmt.Sprintf("$%d", baseIndex+1)
				versionPlaceholder := fmt.Sprintf("$%d", baseIndex+2)
				versionConditions[i] = fmt.Sprintf("(s.type = %s AND s.version = %s)", productPlaceholder, versionPlaceholder)
				args = append(args, parts[0], parts[1])
				baseIndex += 2
			} else {
				// Fallback for non-prefixed format: match version only (backward compatibility)
				versionPlaceholder := fmt.Sprintf("$%d", baseIndex+1)
				versionConditions[i] = fmt.Sprintf("(s.version = %s)", versionPlaceholder)
				args = append(args, prefixedVersion)
				baseIndex += 1
			}
		}

		whereClause += fmt.Sprintf(" AND (%s)", strings.Join(versionConditions, " OR "))
	}

	if len(filterOrgIDs) > 0 {
		// Filter by logto_id (systems.organization_id stores logto_id)
		orgPlaceholders := make([]string, len(filterOrgIDs))
		baseIndex := len(args)
		for i, orgID := range filterOrgIDs {
			orgPlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, orgID)
		}
		whereClause += fmt.Sprintf(" AND s.organization_id IN (%s)", strings.Join(orgPlaceholders, ","))
	}

	// Handle status filter (treat "deleted" as a normal status value)
	if len(filterStatuses) > 0 {
		statusPlaceholders := make([]string, len(filterStatuses))
		baseIndex := len(args)
		for i, s := range filterStatuses {
			statusPlaceholders[i] = fmt.Sprintf("$%d", baseIndex+i+1)
			args = append(args, s)
		}
		whereClause += fmt.Sprintf(" AND s.status IN (%s)", strings.Join(statusPlaceholders, ","))
	}

	// Get total count
	var totalCount int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM systems s
		LEFT JOIN distributors d ON s.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON s.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON s.organization_id = c.logto_id AND c.deleted_at IS NULL
		WHERE %s`, whereClause)

	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get systems count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "s.created_at DESC"
	if sortBy != "" {
		// Map sortBy values to actual column names (use LOWER() for case-insensitive sorting on text fields)
		columnMap := map[string]string{
			"name":              "LOWER(s.name)",
			"type":              "LOWER(s.type)",
			"status":            "LOWER(s.status)",
			"fqdn":              "LOWER(s.fqdn)",
			"version":           "s.version",
			"system_key":        "LOWER(s.system_key)",
			"created_at":        "s.created_at",
			"updated_at":        "s.updated_at",
			"creator_name":      "LOWER(s.created_by ->> 'name')",
			"organization_name": "LOWER(COALESCE(d.name, r.name, c.name))",
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
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version,
		       s.system_key, s.organization_id, s.custom_data, s.notes, s.created_at, s.updated_at, s.deleted_at, s.created_by,
		       COALESCE(d.name, r.name, c.name, 'Owner') as organization_name,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type
		FROM systems s
		LEFT JOIN distributors d ON s.organization_id = d.logto_id AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON s.organization_id = r.logto_id AND r.deleted_at IS NULL
		LEFT JOIN customers c ON s.organization_id = c.logto_id AND c.deleted_at IS NULL
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
		var deletedAt sql.NullTime
		var organizationName, organizationType sql.NullString

		err := rows.Scan(
			&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
			&ipv4Address, &ipv6Address, &version, &system.SystemKey, &system.Organization.ID,
			&customDataJSON, &system.Notes, &system.CreatedAt, &system.UpdatedAt, &deletedAt, &createdByJSON,
			&organizationName, &organizationType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan system: %w", err)
		}

		// Convert NullString to string
		system.FQDN = fqdn.String
		system.IPv4Address = ipv4Address.String
		system.IPv6Address = ipv6Address.String
		system.Version = version.String
		system.Organization.Name = organizationName.String
		system.Organization.Type = organizationType.String

		// Set deleted_at if present
		if deletedAt.Valid {
			system.DeletedAt = &deletedAt.Time
		}

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
			Active:         0,
			Inactive:       0,
			Unknown:        0,
			TimeoutMinutes: timeoutMinutes,
		}, nil
	}

	// Calculate cutoff time for active/inactive determination
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
			COALESCE(SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat > $1 THEN 1 ELSE 0 END), 0) as active,
			COALESCE(SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat <= $1 THEN 1 ELSE 0 END), 0) as inactive,
			COALESCE(SUM(CASE WHEN h.last_heartbeat IS NULL THEN 1 ELSE 0 END), 0) as unknown
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.deleted_at IS NULL AND s.created_by ->> 'organization_id' IN (%s)
	`, placeholdersStr)

	var total, active, inactive, unknown int
	err := r.db.QueryRow(query, args...).Scan(&total, &active, &inactive, &unknown)
	if err != nil {
		return nil, fmt.Errorf("failed to get systems totals: %w", err)
	}

	return &models.SystemTotals{
		Total:          total,
		Active:         active,
		Inactive:       inactive,
		Unknown:        unknown,
		TimeoutMinutes: timeoutMinutes,
	}, nil
}
