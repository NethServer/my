/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

// LocalSystemsService handles business logic for systems management
type LocalSystemsService struct {
	logtoClient *logto.LogtoManagementClient
}

// NewSystemsService creates a new systems service
func NewSystemsService() *LocalSystemsService {
	return &LocalSystemsService{
		logtoClient: logto.NewManagementClient(),
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// CreateSystem creates a new system with automatic secret generation
func (s *LocalSystemsService) CreateSystem(request *models.CreateSystemRequest, creatorInfo *models.SystemCreator, userOrgRole, userOrgID string) (*models.System, error) {
	// Validate system type is allowed
	if err := s.validateSystemType(request.Type); err != nil {
		return nil, err
	}

	// Systems can now be created by any user - access will be based on created_by field

	// Generate unique system ID
	systemID := uuid.New().String()

	// Generate secure system secret (64 characters)
	secret, err := s.generateSystemSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system secret: %w", err)
	}

	now := time.Now()

	// Hash the secret for storage
	hash := sha256.Sum256([]byte(secret))
	hashedSecret := hex.EncodeToString(hash[:])

	// Convert custom_data to JSON for storage
	customDataJSON, err := json.Marshal(request.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	// Convert created_by to JSON for storage
	createdByJSON, err := json.Marshal(creatorInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal created_by: %w", err)
	}

	// Insert system into database
	query := `
		INSERT INTO systems (id, name, type, status, custom_data, secret_hash, secret_hint, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = database.DB.Exec(query, systemID, request.Name, request.Type, "offline",
		customDataJSON, hashedSecret, secret[len(secret)-4:], now, now, createdByJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create system: %w", err)
	}

	// Create system object
	system := &models.System{
		ID:     systemID,
		Name:   request.Name,
		Type:   request.Type,
		Status: "offline",
		// FQDN, IPv4Address, IPv6Address will be populated by collect service
		CustomData: request.CustomData,
		Secret:     secret,
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  *creatorInfo,
	}

	logger.Info().
		Str("system_id", systemID).
		Str("system_name", system.Name).
		Str("created_by_user", creatorInfo.UserID).
		Str("created_by_org", creatorInfo.OrganizationID).
		Msg("System created successfully")

	return system, nil
}

// GetSystemsByOrganization retrieves systems filtered by organization access
func (s *LocalSystemsService) GetSystemsByOrganization(userID string, userOrgRole, userRole string) ([]*models.System, error) {
	query := `
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version, s.last_seen,
		       s.custom_data, s.created_at, s.updated_at, s.created_by, h.last_heartbeat
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.deleted_at IS NULL
		ORDER BY s.created_at DESC
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query systems: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var systems []*models.System
	for rows.Next() {
		system := &models.System{}
		var customDataJSON []byte
		var createdByJSON []byte
		var fqdn, ipv4Address, ipv6Address, version sql.NullString
		var lastHeartbeat sql.NullTime

		err := rows.Scan(
			&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
			&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON,
			&system.CreatedAt, &system.UpdatedAt, &createdByJSON, &lastHeartbeat,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan system: %w", err)
		}

		// Convert NullString to string
		system.FQDN = fqdn.String
		system.IPv4Address = ipv4Address.String
		system.IPv6Address = ipv6Address.String
		system.Version = version.String

		// Parse custom_data JSON
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &system.CustomData); err != nil {
				logger.Warn().Err(err).Str("system_id", system.ID).Msg("Failed to parse system custom_data")
				system.CustomData = make(map[string]string)
			}
		} else {
			system.CustomData = make(map[string]string)
		}

		// Parse created_by JSON
		if len(createdByJSON) > 0 {
			if err := json.Unmarshal(createdByJSON, &system.CreatedBy); err != nil {
				logger.Warn().Err(err).Str("system_id", system.ID).Msg("Failed to parse created_by")
			}
		}

		// Calculate heartbeat status (15 minutes timeout)
		var heartbeatTime *time.Time
		if lastHeartbeat.Valid {
			heartbeatTime = &lastHeartbeat.Time
		}
		system.HeartbeatStatus, system.HeartbeatMinutes = s.calculateHeartbeatStatus(heartbeatTime, 15)
		system.LastHeartbeat = heartbeatTime

		systems = append(systems, system)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating systems: %w", err)
	}

	logger.Debug().
		Str("user_id", userID).
		Int("count", len(systems)).
		Msg("Retrieved systems for user")

	return systems, nil
}

// GetSystemsByOrganizationPaginated retrieves systems filtered by organization with pagination, search, sorting and RBAC
func (s *LocalSystemsService) GetSystemsByOrganizationPaginated(userID, userOrgID, userOrgRole string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.System, int, error) {
	// Get hierarchical organization IDs using existing user service logic
	userService := NewUserService()
	allowedOrgIDs, err := userService.GetHierarchicalOrganizationIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
	}

	// Use repository layer for pagination, search and sorting
	systemRepo := entities.NewLocalSystemRepository()
	return systemRepo.ListByCreatedByOrganizations(allowedOrgIDs, page, pageSize, search, sortBy, sortDirection)
}

// GetSystem retrieves a specific system with RBAC validation
func (s *LocalSystemsService) GetSystem(systemID, userOrgRole, userOrgID string) (*models.System, error) {
	// Get the system first
	systemRepo := entities.NewLocalSystemRepository()
	system, err := systemRepo.GetByID(systemID)
	if err != nil {
		return nil, err
	}

	// Validate hierarchical access
	if canAccess, reason := s.CanAccessSystem(system, userOrgRole, userOrgID); !canAccess {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

	// Calculate heartbeat status (15 minutes timeout)
	system.HeartbeatStatus, system.HeartbeatMinutes = s.calculateHeartbeatStatus(system.LastHeartbeat, 15)

	logger.Debug().
		Str("system_id", systemID).
		Str("heartbeat_status", system.HeartbeatStatus).
		Msg("Retrieved system by ID")

	return system, nil
}

// UpdateSystem updates an existing system with access validation
func (s *LocalSystemsService) UpdateSystem(systemID string, request *models.UpdateSystemRequest, userID, userOrgID, userOrgRole string) (*models.System, error) {
	// Get the system first to check permissions
	system, err := s.GetSystem(systemID, userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	// Validate update permissions based on created_by
	if canUpdate, reason := s.CanUpdateSystemByCreator(userOrgRole, userOrgID, &system.CreatedBy); !canUpdate {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

	// Update system fields
	now := time.Now()

	if request.Name != "" {
		system.Name = request.Name
	}
	if request.Type != "" {
		// Validate system type is allowed
		if err := s.validateSystemType(request.Type); err != nil {
			return nil, err
		}
		system.Type = request.Type
	}
	// FQDN, IPv4Address, IPv6Address are managed by collect service, not via API updates
	if request.CustomData != nil {
		system.CustomData = request.CustomData
	}

	system.UpdatedAt = now

	// Convert custom_data to JSON for storage
	customDataJSON, err := json.Marshal(system.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	// Update system in database (FQDN and IP addresses are managed by collect service)
	query := `
		UPDATE systems
		SET name = $2, type = $3, custom_data = $4, updated_at = $5
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err = database.DB.Exec(query, systemID, system.Name, system.Type,
		customDataJSON, now)
	if err != nil {
		return nil, fmt.Errorf("failed to update system: %w", err)
	}

	logger.Info().
		Str("system_id", systemID).
		Str("updated_by", userID).
		Msg("System updated successfully")

	return system, nil
}

// DeleteSystem deletes a system with access validation
func (s *LocalSystemsService) DeleteSystem(systemID, userID, userOrgID, userOrgRole string) error {
	// Get the system first to check permissions
	system, err := s.GetSystem(systemID, userOrgRole, userOrgID)
	if err != nil {
		return err
	}

	// Validate delete permissions based on created_by
	if canDelete, reason := s.CanDeleteSystemByCreator(userOrgRole, userOrgID, &system.CreatedBy); !canDelete {
		return fmt.Errorf("access denied: %s", reason)
	}

	// Soft delete system from database (set deleted_at timestamp)
	query := `UPDATE systems SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`

	result, err := database.DB.Exec(query, systemID)
	if err != nil {
		return fmt.Errorf("failed to delete system: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("system not found")
	}

	logger.Info().
		Str("system_id", systemID).
		Str("deleted_by", userID).
		Msg("System deleted successfully")

	return nil
}

// RegenerateSystemSecret generates a new secret for an existing system
func (s *LocalSystemsService) RegenerateSystemSecret(systemID, userID, userOrgID, userOrgRole string) (*models.System, error) {
	// Get the system first to check permissions
	system, err := s.GetSystem(systemID, userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}

	// Validate update permissions (regenerating secret is an update operation) based on created_by
	if canUpdate, reason := s.CanUpdateSystemByCreator(userOrgRole, userOrgID, &system.CreatedBy); !canUpdate {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

	// Generate new secret
	secret, err := s.generateSystemSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system secret: %w", err)
	}

	now := time.Now()

	// Hash the secret for storage
	hash := sha256.Sum256([]byte(secret))
	hashedSecret := hex.EncodeToString(hash[:])

	// Update system secret
	query := `
		UPDATE systems
		SET secret_hash = $2, secret_hint = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err = database.DB.Exec(query, systemID, hashedSecret, secret[len(secret)-4:], now)
	if err != nil {
		return nil, fmt.Errorf("failed to update system credentials: %w", err)
	}

	// Get updated system
	system, err = s.GetSystem(systemID, userOrgRole, userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated system: %w", err)
	}

	// Set the secret for this response only
	system.Secret = secret

	logger.Info().
		Str("system_id", systemID).
		Str("regenerated_by", userID).
		Msg("System secret regenerated successfully")

	return system, nil
}

// GetTotals returns total counts and status for systems based on hierarchical RBAC
func (s *LocalSystemsService) GetTotals(userOrgRole, userOrgID string, timeoutMinutes int) (*models.SystemTotals, error) {
	// First validate that the user can access systems (basic RBAC check)
	switch userOrgRole {
	case "owner", "distributor", "reseller", "customer":
		// These roles can access system totals
	default:
		return nil, fmt.Errorf("insufficient permissions to access system totals")
	}

	// Get all organization IDs the user can access hierarchically
	userService := NewUserService()
	allowedOrgIDs, err := userService.GetHierarchicalOrganizationIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
	}

	// Get totals with the specified timeout based on created_by organizations
	return s.GetTotalsByCreatedByOrganizations(allowedOrgIDs, timeoutMinutes)
}

// Systems can now be created by any authenticated user
// Access control is based on the created_by field which is set to the creator's organization

// CanUpdateSystemByCreator validates if a user can update a system based on created_by organization
func (s *LocalSystemsService) CanUpdateSystemByCreator(userOrgRole, userOrgID string, creator *models.SystemCreator) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can update systems created by organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, creator.OrganizationID) {
			return true, ""
		}
		return false, "distributors can only update systems created by organizations they manage"
	case "reseller":
		// Reseller can update systems created by their own organization
		if creator.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "resellers can only update systems created by their own organization"
	case "customer":
		// Customers can only update systems they created themselves
		if creator.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "customers can only update systems created by their own organization"
	default:
		return false, "insufficient permissions to update systems"
	}
}

// CanDeleteSystemByCreator validates if a user can delete a system based on created_by organization
func (s *LocalSystemsService) CanDeleteSystemByCreator(userOrgRole, userOrgID string, creator *models.SystemCreator) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can delete systems created by organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, creator.OrganizationID) {
			return true, ""
		}
		return false, "distributors can only delete systems created by organizations they manage"
	case "reseller":
		// Reseller can delete systems created by their own organization
		if creator.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "resellers can only delete systems created by their own organization"
	case "customer":
		// Customers can only delete systems they created themselves
		if creator.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "customers can only delete systems created by their own organization"
	default:
		return false, "insufficient permissions to delete systems"
	}
}

// CanAccessSystem validates if a user can access a specific system based on created_by organization
func (s *LocalSystemsService) CanAccessSystem(system *models.System, userOrgRole, userOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can access systems created by organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, system.CreatedBy.OrganizationID) {
			return true, ""
		}
		return false, "distributors can only access systems created by organizations they manage"
	case "reseller":
		// Reseller can access systems created by their own organization
		if system.CreatedBy.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "resellers can only access systems created by their own organization"
	case "customer":
		// Customers can access systems created by their own organization
		if system.CreatedBy.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "customers can only access systems created by their own organization"
	default:
		return false, "insufficient permissions to access systems"
	}
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// calculateHeartbeatStatus calculates heartbeat status based on last_heartbeat timestamp
func (s *LocalSystemsService) calculateHeartbeatStatus(lastHeartbeat *time.Time, timeoutMinutes int) (string, *int) {
	if lastHeartbeat == nil {
		return "zombie", nil
	}

	timeout := time.Duration(timeoutMinutes) * time.Minute
	cutoff := time.Now().Add(-timeout)
	minutes := int(time.Since(*lastHeartbeat).Minutes())

	if lastHeartbeat.After(cutoff) {
		return "alive", &minutes
	}
	return "dead", &minutes
}

// generateSystemSecret generates a cryptographically secure random secret
func (s *LocalSystemsService) generateSystemSecret() (string, error) {
	// Generate 32 random bytes (will be 64 hex characters)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetTotalsByCreatedByOrganizations returns total counts and status for systems created by specified organizations
func (s *LocalSystemsService) GetTotalsByCreatedByOrganizations(allowedOrgIDs []string, timeoutMinutes int) (*models.SystemTotals, error) {
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

	// Base query with heartbeat status calculation and hierarchical filtering by created_by
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
	err := database.DB.QueryRow(query, args...).Scan(&total, &alive, &dead, &zombie)
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

// validateSystemType validates that the system type is in the allowed list from configuration
func (s *LocalSystemsService) validateSystemType(systemType string) error {
	for _, allowedType := range configuration.Config.SystemTypes {
		if systemType == allowedType {
			return nil
		}
	}

	return fmt.Errorf("invalid system type '%s', allowed types: %v", systemType, configuration.Config.SystemTypes)
}
