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
		       s.custom_data, s.customer_id, s.created_at, s.updated_at, s.created_by, h.last_heartbeat
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.id = $1
	`

	system := &models.System{}
	var customDataJSON []byte
	var createdByJSON []byte
	var fqdn, ipv4Address, ipv6Address, version sql.NullString
	var lastHeartbeat sql.NullTime

	err := r.db.QueryRow(query, id).Scan(
		&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
		&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON, &system.CustomerID,
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

// List retrieves systems filtered by organization with pagination and RBAC (matches other repository patterns)
func (r *LocalSystemRepository) List(userOrgRole, userOrgID string, page, pageSize int) ([]*models.System, int, error) {
	// Get all customer organization IDs the user can access hierarchically
	allowedOrgIDs, err := r.GetHierarchicalCustomerIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get hierarchical customer IDs: %w", err)
	}

	return r.ListByCustomerIDs(allowedOrgIDs, page, pageSize)
}

// ListByCustomerIDs returns paginated list of systems in specified customer organizations
func (r *LocalSystemRepository) ListByCustomerIDs(allowedOrgIDs []string, page, pageSize int) ([]*models.System, int, error) {
	if len(allowedOrgIDs) == 0 {
		return []*models.System{}, 0, nil
	}

	offset := (page - 1) * pageSize

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Get total count
	var totalCount int
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM systems
		WHERE active = TRUE AND customer_id IN (%s)
	`, placeholdersStr)

	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get systems count: %w", err)
	}

	// Get paginated results
	listArgs := make([]interface{}, len(args)+2)
	copy(listArgs, args)
	listArgs[len(args)] = pageSize
	listArgs[len(args)+1] = offset

	query := fmt.Sprintf(`
		SELECT id, name, type, status, fqdn, ipv4_address, ipv6_address, version, last_seen,
		       custom_data, customer_id, created_at, updated_at, created_by
		FROM systems
		WHERE active = TRUE AND customer_id IN (%s)
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, placeholdersStr, len(args)+1, len(args)+2)

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
			&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON, &system.CustomerID,
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
			_ = json.Unmarshal(createdByJSON, &system.CreatedBy) // Ignore JSON unmarshal errors - keep default zero value
		}

		systems = append(systems, system)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating systems: %w", err)
	}

	return systems, totalCount, nil
}

// GetTotals returns total counts and status for systems visible to the user (matches other repository patterns)
func (r *LocalSystemRepository) GetTotals(userOrgRole, userOrgID string) (*models.SystemTotals, error) {
	// Get all customer organization IDs the user can access hierarchically
	allowedOrgIDs, err := r.GetHierarchicalCustomerIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchical customer IDs: %w", err)
	}

	// Use default timeout of 15 minutes for heartbeat status calculation (matching service logic)
	return r.GetTotalsByCustomerIDs(allowedOrgIDs, 15)
}

// GetTotalsByCustomerIDs returns total counts and status for systems in specified customer organizations
func (r *LocalSystemRepository) GetTotalsByCustomerIDs(allowedOrgIDs []string, timeoutMinutes int) (*models.SystemTotals, error) {
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
		WHERE s.active = TRUE AND s.customer_id IN (%s)
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

// GetHierarchicalCustomerIDs returns all customer organization IDs that the user can access
// This mirrors the logic from UserService.GetHierarchicalOrganizationIDs but filtered for customers only
func (r *LocalSystemRepository) GetHierarchicalCustomerIDs(userOrgRole, userOrgID string) ([]string, error) {
	var customerIDs []string

	switch userOrgRole {
	case "owner":
		// Owner can see systems from all customers
		rows, err := r.db.Query("SELECT logto_id FROM customers WHERE logto_id IS NOT NULL AND active = TRUE")
		if err != nil {
			return nil, fmt.Errorf("failed to get all customers: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var orgID string
			if err := rows.Scan(&orgID); err != nil {
				return nil, fmt.Errorf("failed to scan customer ID: %w", err)
			}
			customerIDs = append(customerIDs, orgID)
		}

	case "distributor":
		// Distributor can see systems from customers created by them or by their resellers
		// Get customers created directly by this distributor
		rows, err := r.db.Query("SELECT logto_id FROM customers WHERE custom_data->>'createdBy' = $1 AND logto_id IS NOT NULL AND active = TRUE", userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get distributor customers: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var orgID string
			if err := rows.Scan(&orgID); err != nil {
				return nil, fmt.Errorf("failed to scan customer ID: %w", err)
			}
			customerIDs = append(customerIDs, orgID)
		}

		// Get customers created by resellers managed by this distributor
		rows2, err := r.db.Query(`
			SELECT c.logto_id FROM customers c
			WHERE c.custom_data->>'createdBy' IN (
				SELECT logto_id FROM resellers
				WHERE custom_data->>'createdBy' = $1 AND logto_id IS NOT NULL AND active = TRUE
			) AND c.logto_id IS NOT NULL AND c.active = TRUE
		`, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get reseller customers: %w", err)
		}
		defer func() { _ = rows2.Close() }()

		for rows2.Next() {
			var orgID string
			if err := rows2.Scan(&orgID); err != nil {
				return nil, fmt.Errorf("failed to scan reseller customer ID: %w", err)
			}
			customerIDs = append(customerIDs, orgID)
		}

	case "reseller":
		// Reseller can see systems from customers they created
		rows, err := r.db.Query("SELECT logto_id FROM customers WHERE custom_data->>'createdBy' = $1 AND logto_id IS NOT NULL AND active = TRUE", userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get reseller customers: %w", err)
		}
		defer func() { _ = rows.Close() }()

		for rows.Next() {
			var orgID string
			if err := rows.Scan(&orgID); err != nil {
				return nil, fmt.Errorf("failed to scan customer ID: %w", err)
			}
			customerIDs = append(customerIDs, orgID)
		}

	case "customer":
		// Customer can only see systems in their own organization
		customerIDs = append(customerIDs, userOrgID)

	default:
		// Unknown role - no access
		return []string{}, nil
	}

	return customerIDs, nil
}
