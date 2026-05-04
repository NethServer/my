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
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
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
	// Validate organization access: user can only create systems for organizations they can manage
	if canCreate, reason := s.CanCreateSystemForOrganization(userOrgRole, userOrgID, request.OrganizationID); !canCreate {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

	// Generate unique system ID
	systemID := uuid.New().String()

	// Generate system key (NETH-XXXX-XXXX format)
	systemKey, err := s.generateSystemKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system key: %w", err)
	}

	// Generate system secret token in format: my_<public>.<secret>
	fullToken, publicPart, secretPart, err := s.generateSystemSecretToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system secret token: %w", err)
	}

	// Hash the secret part using SHA256 (fast, negligible memory)
	hashedSecretSHA256, err := helpers.HashSystemSecretSHA256(secretPart)
	if err != nil {
		return nil, fmt.Errorf("failed to hash system secret: %w", err)
	}

	now := time.Now()

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

	// Insert system into database (type starts as NULL, status defaults to 'unknown' until first inventory)
	query := `
		INSERT INTO systems (id, name, type, status, system_key, system_secret_public, system_secret_sha256, organization_id, custom_data, notes, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = database.DB.Exec(query, systemID, request.Name, nil, "unknown", systemKey, publicPart, hashedSecretSHA256, request.OrganizationID,
		customDataJSON, request.Notes, now, now, createdByJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to create system: %w", err)
	}

	// Fetch organization info (name and type)
	organization := s.getOrganizationInfo(request.OrganizationID)

	// Create system object (type is nil until first inventory, status defaults to 'unknown')
	system := &models.System{
		ID:           systemID,
		Name:         request.Name,
		Type:         nil,
		Status:       "unknown",
		SystemKey:    systemKey,
		Organization: organization,
		// FQDN, IPv4Address, IPv6Address, Version will be populated by collect service
		CustomData:   request.CustomData,
		SystemSecret: fullToken, // Return full token only during creation (my_<public>.<secret>)
		Notes:        request.Notes,
		CreatedAt:    now,
		UpdatedAt:    now,
		CreatedBy:    *creatorInfo,
	}

	logger.Info().
		Str("system_id", systemID).
		Str("system_name", system.Name).
		Str("organization_id", request.OrganizationID).
		Str("created_by_user", creatorInfo.UserID).
		Str("created_by_org", creatorInfo.OrganizationID).
		Msg("System created successfully")

	return system, nil
}

// GetSystemsByOrganization retrieves systems filtered by organization access
func (s *LocalSystemsService) GetSystemsByOrganization(userID string, userOrgRole, userRole string) ([]*models.System, error) {
	query := `
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version,
		       s.system_key, s.organization_id, s.custom_data, s.notes, s.created_at, s.updated_at, s.created_by, s.registered_at, s.suspended_at, s.suspended_by_org_id, h.last_heartbeat, s.last_inventory_at,
		       COALESCE(d.name, r.name, c.name, 'Owner') as organization_name,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type,
		       COALESCE(d.id::text, r.id::text, c.id::text, '') as organization_db_id
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id::text) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id::text) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id::text) AND c.deleted_at IS NULL
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
		var registeredAt, suspendedAt, lastHeartbeat, lastInventory sql.NullTime
		var suspendedByOrgID sql.NullString
		var organizationName, organizationType, organizationDBID sql.NullString

		err := rows.Scan(
			&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
			&ipv4Address, &ipv6Address, &version, &system.SystemKey, &system.Organization.LogtoID,
			&customDataJSON, &system.Notes, &system.CreatedAt, &system.UpdatedAt, &createdByJSON, &registeredAt, &suspendedAt, &suspendedByOrgID, &lastHeartbeat, &lastInventory,
			&organizationName, &organizationType, &organizationDBID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan system: %w", err)
		}

		// Convert NullString to string
		system.FQDN = fqdn.String
		system.IPv4Address = ipv4Address.String
		system.IPv6Address = ipv6Address.String
		system.Version = version.String
		system.Organization.ID = organizationDBID.String
		system.Organization.Name = organizationName.String
		system.Organization.Type = organizationType.String

		// Convert registered_at
		if registeredAt.Valid {
			system.RegisteredAt = &registeredAt.Time
		}

		// Convert suspended_at
		if suspendedAt.Valid {
			system.SuspendedAt = &suspendedAt.Time
		}
		if suspendedByOrgID.Valid {
			system.SuspendedByOrgID = &suspendedByOrgID.String
		}

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

		// Set heartbeat and inventory timestamps
		if lastHeartbeat.Valid {
			system.LastHeartbeat = &lastHeartbeat.Time
		}
		if lastInventory.Valid {
			system.LastInventory = &lastInventory.Time
		}

		// Apply unified status (suspended takes priority over DB status)
		s.applyUnifiedStatus(system)

		// Hide system_key if system is not registered yet
		if system.RegisteredAt == nil {
			system.SystemKey = ""
		}

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
func (s *LocalSystemsService) GetSystemsByOrganizationPaginated(userID, userOrgID, userOrgRole string, page, pageSize int, search, sortBy, sortDirection, filterName, filterSystemKey string, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses []string) ([]*models.System, int, error) {
	// Owner can access all systems - pass nil to skip RBAC filtering in query
	var allowedOrgIDs []string
	if strings.ToLower(userOrgRole) != "owner" {
		userService := NewUserService()
		var err error
		allowedOrgIDs, err = userService.GetHierarchicalOrganizationIDs(strings.ToLower(userOrgRole), userOrgID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
		}
	}

	// Use repository layer for pagination, search, sorting and filters
	systemRepo := entities.NewLocalSystemRepository()
	systems, totalCount, err := systemRepo.ListByCreatedByOrganizations(allowedOrgIDs, page, pageSize, search, sortBy, sortDirection, filterName, filterSystemKey, filterTypes, filterCreatedBy, filterVersions, filterOrgIDs, filterStatuses)
	if err != nil {
		return nil, 0, err
	}

	for _, system := range systems {
		// Hide system_key for systems that are not registered yet
		if system.RegisteredAt == nil {
			system.SystemKey = ""
		}
		// Apply unified status (suspended takes priority over DB status)
		s.applyUnifiedStatus(system)
	}

	return systems, totalCount, nil
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

	// Apply unified status (suspended takes priority over DB status)
	s.applyUnifiedStatus(system)

	// Hide system_key if system is not registered yet
	if system.RegisteredAt == nil {
		system.SystemKey = ""
	}

	logger.Debug().
		Str("system_id", systemID).
		Str("status", system.Status).
		Msg("Retrieved system by ID")

	return system, nil
}

// GetSystemIncludingDeleted is like GetSystem but also returns a system that
// has already been soft-deleted. It is used by the destroy flow so a backup
// purge can resolve the (org_id, system_key) tuple even for a row whose
// deleted_at is set — GetSystem would otherwise return "system not found"
// and the purge would be silently skipped, leaving ciphertext in the bucket.
func (s *LocalSystemsService) GetSystemIncludingDeleted(systemID, userOrgRole, userOrgID string) (*models.System, error) {
	systemRepo := entities.NewLocalSystemRepository()
	system, err := systemRepo.GetByIDIncludingDeleted(systemID)
	if err != nil {
		return nil, err
	}

	if canAccess, reason := s.CanAccessSystem(system, userOrgRole, userOrgID); !canAccess {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

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
	// Note: Type and SystemKey are not modifiable via update API
	// Validate organization_id change if provided
	if request.OrganizationID != "" && request.OrganizationID != system.Organization.LogtoID {
		// Validate user can assign system to the new organization
		if canCreate, reason := s.CanCreateSystemForOrganization(userOrgRole, userOrgID, request.OrganizationID); !canCreate {
			return nil, fmt.Errorf("access denied for organization change: %s", reason)
		}
		// Update organization with new info
		system.Organization = s.getOrganizationInfo(request.OrganizationID)
	}
	// FQDN, IPv4Address, IPv6Address are managed by collect service, not via API updates
	if request.CustomData != nil {
		system.CustomData = request.CustomData
	}
	// Update notes if provided (empty string clears notes)
	if request.Notes != "" || request.Notes == "" {
		system.Notes = request.Notes
	}

	system.UpdatedAt = now

	// Convert custom_data to JSON for storage
	customDataJSON, err := json.Marshal(system.CustomData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	// Update system in database (FQDN and IP addresses are managed by collect service)
	// Note: type and system_key are not modifiable
	query := `
		UPDATE systems
		SET name = $2, organization_id = $3, custom_data = $4, notes = $5, updated_at = $6
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err = database.DB.Exec(query, systemID, system.Name, system.Organization.LogtoID, customDataJSON, system.Notes, now)
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

	// Soft delete system from database (set deleted_at timestamp and status to 'deleted')
	query := `UPDATE systems SET deleted_at = NOW(), updated_at = NOW(), status = 'deleted' WHERE id = $1 AND deleted_at IS NULL`

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

	cache.InvalidateSystemAuth(context.Background(), system.SystemKey)

	logger.Info().
		Str("system_id", systemID).
		Str("deleted_by", userID).
		Msg("System deleted successfully")

	return nil
}

// RestoreSystem restores a soft-deleted system
func (s *LocalSystemsService) RestoreSystem(systemID, userID, userOrgID, userOrgRole string) error {
	// First, check if system exists and is deleted (query without deleted_at filter)
	query := `
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version,
		       s.system_key, s.organization_id, s.custom_data, s.notes, s.created_at, s.updated_at, s.created_by, s.registered_at, s.deleted_at,
		       COALESCE(d.name, r.name, c.name, 'Owner') as organization_name,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type,
		       COALESCE(d.id::text, r.id::text, c.id::text, '') as organization_db_id
		FROM systems s
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id::text) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id::text) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id::text) AND c.deleted_at IS NULL
		WHERE s.id = $1
	`

	system := &models.System{}
	var customDataJSON []byte
	var createdByJSON []byte
	var fqdn, ipv4Address, ipv6Address, version sql.NullString
	var registeredAt, deletedAt sql.NullTime
	var organizationName, organizationType, organizationDBID sql.NullString
	var systemType sql.NullString

	err := database.DB.QueryRow(query, systemID).Scan(
		&system.ID, &system.Name, &systemType, &system.Status, &fqdn,
		&ipv4Address, &ipv6Address, &version, &system.SystemKey, &system.Organization.LogtoID,
		&customDataJSON, &system.Notes, &system.CreatedAt, &system.UpdatedAt, &createdByJSON, &registeredAt, &deletedAt,
		&organizationName, &organizationType, &organizationDBID,
	)

	if err == sql.ErrNoRows {
		return fmt.Errorf("system not found")
	}
	if err != nil {
		return fmt.Errorf("failed to query system: %w", err)
	}

	// Check if system is actually deleted
	if !deletedAt.Valid {
		return fmt.Errorf("system is not deleted")
	}

	// Convert nullable fields
	if systemType.Valid {
		system.Type = &systemType.String
	}
	system.FQDN = fqdn.String
	system.IPv4Address = ipv4Address.String
	system.IPv6Address = ipv6Address.String
	system.Version = version.String
	system.Organization.ID = organizationDBID.String
	system.Organization.Name = organizationName.String
	system.Organization.Type = organizationType.String

	// Convert registered_at
	if registeredAt.Valid {
		system.RegisteredAt = &registeredAt.Time
	}

	// Parse created_by JSON
	if err := json.Unmarshal(createdByJSON, &system.CreatedBy); err != nil {
		return fmt.Errorf("failed to parse created_by: %w", err)
	}

	// Parse custom_data JSON
	if customDataJSON != nil {
		if err := json.Unmarshal(customDataJSON, &system.CustomData); err != nil {
			return fmt.Errorf("failed to parse custom_data: %w", err)
		}
	}

	// Validate delete permissions based on created_by
	// Use same permission check as delete - if user could delete it, they can restore it
	if canDelete, reason := s.CanDeleteSystemByCreator(userOrgRole, userOrgID, &system.CreatedBy); !canDelete {
		return fmt.Errorf("access denied: %s", reason)
	}

	// Restore system in database (set deleted_at to NULL and status to 'unknown')
	restoreQuery := `UPDATE systems SET deleted_at = NULL, updated_at = NOW(), status = 'unknown' WHERE id = $1 AND deleted_at IS NOT NULL`

	result, err := database.DB.Exec(restoreQuery, systemID)
	if err != nil {
		return fmt.Errorf("failed to restore system: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("system not found or already restored")
	}

	logger.Info().
		Str("system_id", systemID).
		Str("restored_by", userID).
		Msg("System restored successfully")

	return nil
}

// DestroySystem permanently deletes a system from the database
func (s *LocalSystemsService) DestroySystem(systemID, userID, userOrgID, userOrgRole string) error {
	// Query system including deleted (same pattern as RestoreSystem)
	query := `
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version,
		       s.system_key, s.organization_id, s.custom_data, s.notes, s.created_at, s.updated_at, s.created_by, s.registered_at, s.deleted_at,
		       COALESCE(d.name, r.name, c.name, 'Owner') as organization_name,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END as organization_type,
		       COALESCE(d.id::text, r.id::text, c.id::text, '') as organization_db_id
		FROM systems s
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id::text) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id::text) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id::text) AND c.deleted_at IS NULL
		WHERE s.id = $1
	`

	system := &models.System{}
	var customDataJSON []byte
	var createdByJSON []byte
	var fqdn, ipv4Address, ipv6Address, version sql.NullString
	var registeredAt, deletedAt sql.NullTime
	var organizationName, organizationType, organizationDBID sql.NullString
	var systemType sql.NullString

	err := database.DB.QueryRow(query, systemID).Scan(
		&system.ID, &system.Name, &systemType, &system.Status, &fqdn,
		&ipv4Address, &ipv6Address, &version, &system.SystemKey, &system.Organization.LogtoID,
		&customDataJSON, &system.Notes, &system.CreatedAt, &system.UpdatedAt, &createdByJSON, &registeredAt, &deletedAt,
		&organizationName, &organizationType, &organizationDBID,
	)

	if err == sql.ErrNoRows {
		return fmt.Errorf("system not found")
	}
	if err != nil {
		return fmt.Errorf("failed to query system: %w", err)
	}

	// Parse created_by JSON for permission check
	if err := json.Unmarshal(createdByJSON, &system.CreatedBy); err != nil {
		return fmt.Errorf("failed to parse created_by: %w", err)
	}

	// Validate permissions based on created_by
	if canDelete, reason := s.CanDeleteSystemByCreator(userOrgRole, userOrgID, &system.CreatedBy); !canDelete {
		return fmt.Errorf("access denied: %s", reason)
	}

	// Hard-delete system from DB
	systemRepo := entities.NewLocalSystemRepository()
	err = systemRepo.HardDelete(systemID)
	if err != nil {
		return fmt.Errorf("failed to hard-delete system: %w", err)
	}

	cache.InvalidateSystemAuth(context.Background(), system.SystemKey)

	logger.Info().
		Str("system_id", systemID).
		Str("destroyed_by", userID).
		Msg("System permanently destroyed")

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

	// Generate new token (format: my_<public>.<secret>)
	fullToken, publicPart, secretPart, err := s.generateSystemSecretToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system secret token: %w", err)
	}

	now := time.Now()

	// Hash the secret part using SHA256 (fast, negligible memory)
	hashedSecretSHA256, err := helpers.HashSystemSecretSHA256(secretPart)
	if err != nil {
		return nil, fmt.Errorf("failed to hash system secret: %w", err)
	}

	// Update secret with new SHA256 hash
	query := `
		UPDATE systems
		SET system_secret_public = $2, system_secret_sha256 = $3, updated_at = $4
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err = database.DB.Exec(query, systemID, publicPart, hashedSecretSHA256, now)
	if err != nil {
		return nil, fmt.Errorf("failed to update system credentials: %w", err)
	}

	// Update the existing system object in memory instead of re-fetching
	system.UpdatedAt = now
	system.SystemSecret = fullToken

	cache.InvalidateSystemAuth(context.Background(), system.SystemKey)

	logger.Info().
		Str("system_id", systemID).
		Str("regenerated_by", userID).
		Msg("System secret regenerated successfully")

	return system, nil
}

// RegisterSystem registers a system using its system_secret token and returns the system_key
func (s *LocalSystemsService) RegisterSystem(systemSecret string) (*models.RegisterSystemResponse, error) {
	// Split token format: my_<public>.<secret>
	parts := strings.Split(systemSecret, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid system secret format")
	}

	// Extract public part (remove "my_" prefix)
	publicPart := strings.TrimPrefix(parts[0], "my_")
	if publicPart == parts[0] {
		return nil, fmt.Errorf("invalid system secret format: missing 'my_' prefix")
	}
	secretPart := parts[1]

	// Fast lookup using system_secret_public
	query := `
		SELECT id, system_key, system_secret_sha256, deleted_at, registered_at
		FROM systems
		WHERE system_secret_public = $1
	`

	var systemID, systemKey string
	var hashedSecretSHA256 sql.NullString
	var deletedAt, registeredAt sql.NullTime

	err := database.DB.QueryRow(query, publicPart).Scan(&systemID, &systemKey, &hashedSecretSHA256, &deletedAt, &registeredAt)
	if err == sql.ErrNoRows {
		logger.Warn().Str("public_part", publicPart).Msg("System not found with provided public part")
		return nil, fmt.Errorf("invalid system secret")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query system: %w", err)
	}

	// Check if system is deleted
	if deletedAt.Valid {
		logger.Warn().Str("system_id", systemID).Msg("Attempted to register deleted system")
		return nil, fmt.Errorf("system has been deleted")
	}

	// Check if system is already registered
	if registeredAt.Valid {
		logger.Warn().Str("system_id", systemID).Msg("Attempted to register already registered system")
		return nil, fmt.Errorf("system is already registered")
	}

	// Verify the secret part using SHA256
	valid := false
	if hashedSecretSHA256.Valid {
		valid, err = helpers.VerifySystemSecretSHA256(secretPart, hashedSecretSHA256.String)
		if err != nil {
			return nil, fmt.Errorf("failed to verify system secret: %w", err)
		}
	}
	if !valid {
		logger.Warn().Str("system_id", systemID).Msg("Invalid secret part provided during registration")
		return nil, fmt.Errorf("invalid system secret")
	}

	// Update registered_at timestamp
	now := time.Now()
	updateQuery := `
		UPDATE systems
		SET registered_at = $1
		WHERE id = $2
	`

	_, err = database.DB.Exec(updateQuery, now, systemID)
	if err != nil {
		return nil, fmt.Errorf("failed to update registration timestamp: %w", err)
	}

	logger.Info().
		Str("system_id", systemID).
		Str("system_key", systemKey).
		Msg("System registered successfully")

	return &models.RegisterSystemResponse{
		SystemKey:    systemKey,
		RegisteredAt: now,
	}, nil
}

// GetTotals returns total counts and status for systems based on hierarchical RBAC
func (s *LocalSystemsService) GetTotals(userOrgRole, userOrgID string, timeoutMinutes int) (*models.SystemTotals, error) {
	// Normalize role to lowercase for case-insensitive comparison
	normalizedRole := strings.ToLower(userOrgRole)

	// First validate that the user can access systems (basic RBAC check)
	switch normalizedRole {
	case "owner", "distributor", "reseller", "customer":
		// These roles can access system totals
	default:
		return nil, fmt.Errorf("insufficient permissions to access system totals")
	}

	// Owner can access all systems - pass nil to skip RBAC filtering
	var allowedOrgIDs []string
	if normalizedRole != "owner" {
		userService := NewUserService()
		var err error
		allowedOrgIDs, err = userService.GetHierarchicalOrganizationIDs(normalizedRole, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
		}
	}

	// Get totals with the specified timeout based on created_by organizations
	return s.GetTotalsByCreatedByOrganizations(allowedOrgIDs, timeoutMinutes)
}

// Systems can now be created by any authenticated user
// Access control is based on the created_by field which is set to the creator's organization

// CanUpdateSystemByCreator validates if a user can update a system based on created_by organization
func (s *LocalSystemsService) CanUpdateSystemByCreator(userOrgRole, userOrgID string, creator *models.SystemCreator) (bool, string) {
	// Normalize organization role to lowercase for case-insensitive comparison
	normalizedOrgRole := strings.ToLower(userOrgRole)

	switch normalizedOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can update systems created by organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedOrgRole, userOrgID, creator.OrganizationID) {
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
	// Normalize organization role to lowercase for case-insensitive comparison
	normalizedOrgRole := strings.ToLower(userOrgRole)

	switch normalizedOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can delete systems created by organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedOrgRole, userOrgID, creator.OrganizationID) {
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
	// Normalize role to lowercase for case-insensitive comparison
	normalizedRole := strings.ToLower(userOrgRole)

	switch normalizedRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can access systems created by organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedRole, userOrgID, system.CreatedBy.OrganizationID) {
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

// applyUnifiedStatus overrides the DB status with the unified status.
// Priority: suspended > deleted > DB status (unknown/active/inactive)
func (s *LocalSystemsService) applyUnifiedStatus(system *models.System) {
	if system.SuspendedAt != nil {
		system.Status = "suspended"
		return
	}
	if system.DeletedAt != nil {
		system.Status = "deleted"
		return
	}
	// DB status is already unknown/active/inactive - keep as is
}

// generateSystemKey generates a unique UUID-based system key with prefix
// Format: NETH-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX
func (s *LocalSystemsService) generateSystemKey() (string, error) {
	// Generate a new UUID
	id := uuid.New()

	// Convert UUID to uppercase hex string without dashes
	hexStr := strings.ToUpper(strings.ReplaceAll(id.String(), "-", ""))

	// Format as: NETH-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX-XXXX
	// Group into 4-character segments for readability
	var segments []string
	for i := 0; i < len(hexStr); i += 4 {
		end := i + 4
		if end > len(hexStr) {
			end = len(hexStr)
		}
		segments = append(segments, hexStr[i:end])
	}

	return "NETH-" + strings.Join(segments, "-"), nil
}

// generateSecretPublicPart generates the public part of the token (20 random characters)
func (s *LocalSystemsService) generateSecretPublicPart() (string, error) {
	bytes := make([]byte, 15) // 15 bytes = 20 base64 chars (raw URL encoding)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use RawURLEncoding (no padding, URL-safe)
	return strings.ToLower(hex.EncodeToString(bytes)[:20]), nil
}

// generateSecretPrivatePart generates the secret part of the token (40 random characters)
func (s *LocalSystemsService) generateSecretPrivatePart() (string, error) {
	bytes := make([]byte, 30) // 30 bytes = 40 base64 chars
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	// Use hex encoding for simplicity
	return strings.ToLower(hex.EncodeToString(bytes)[:40]), nil
}

// generateSystemSecretToken generates a complete token in format: my_<public>.<secret>
// Returns: fullToken, publicPart, secretPart, error
func (s *LocalSystemsService) generateSystemSecretToken() (string, string, string, error) {
	publicPart, err := s.generateSecretPublicPart()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate public part: %w", err)
	}

	secretPart, err := s.generateSecretPrivatePart()
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate secret part: %w", err)
	}

	fullToken := fmt.Sprintf("my_%s.%s", publicPart, secretPart)
	return fullToken, publicPart, secretPart, nil
}

// GetTotalsByCreatedByOrganizations returns total counts and status for systems created by specified organizations
// nil allowedOrgIDs = owner (no RBAC filter), empty = no access
func (s *LocalSystemsService) GetTotalsByCreatedByOrganizations(allowedOrgIDs []string, timeoutMinutes int) (*models.SystemTotals, error) {
	// nil = owner (no filter), empty = no access
	if allowedOrgIDs != nil && len(allowedOrgIDs) == 0 {
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

	args := []interface{}{cutoff}
	orgClause := ""
	if allowedOrgIDs != nil {
		args = append(args, pq.Array(allowedOrgIDs))
		orgClause = " AND s.created_by ->> 'organization_id' = ANY($2::text[])"
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat > $1 THEN 1 ELSE 0 END), 0) as active,
			COALESCE(SUM(CASE WHEN h.last_heartbeat IS NOT NULL AND h.last_heartbeat <= $1 THEN 1 ELSE 0 END), 0) as inactive,
			COALESCE(SUM(CASE WHEN h.last_heartbeat IS NULL THEN 1 ELSE 0 END), 0) as unknown
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.deleted_at IS NULL%s
	`, orgClause)

	var total, active, inactive, unknown int
	err := database.DB.QueryRow(query, args...).Scan(&total, &active, &inactive, &unknown)
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

// CanCreateSystemForOrganization validates if a user can create systems for a specific organization
func (s *LocalSystemsService) CanCreateSystemForOrganization(userOrgRole, userOrgID, targetOrgID string) (bool, string) {
	// Normalize organization role to lowercase for case-insensitive comparison
	normalizedOrgRole := strings.ToLower(userOrgRole)

	switch normalizedOrgRole {
	case "owner":
		// Owner can assign to any organization in the hierarchy. We still
		// gate the assignment on existence so a typo or stale logto_id can't
		// strand a system under a phantom organization (which would silently
		// route its backups to an unreachable S3 prefix on reassignment).
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedOrgRole, userOrgID, targetOrgID) {
			return true, ""
		}
		return false, "target organization does not exist"
	case "distributor":
		// Distributor can create systems for organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedOrgRole, userOrgID, targetOrgID) {
			return true, ""
		}
		return false, "distributors can only create systems for organizations they manage"
	case "reseller":
		// Reseller can create systems for their own organization and customers they manage
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedOrgRole, userOrgID, targetOrgID) {
			return true, ""
		}
		return false, "resellers can only create systems for their own organization and customers they manage"
	case "customer":
		// Customers can create systems for their own organization only
		if targetOrgID == userOrgID {
			return true, ""
		}
		return false, "customers can only create systems for their own organization"
	default:
		return false, "insufficient permissions to create systems"
	}
}

// getOrganizationInfo fetches organization info (name, type and IDs) from distributors, resellers, or customers tables
func (s *LocalSystemsService) getOrganizationInfo(logtoOrgID string) models.Organization {
	query := `
		SELECT
			COALESCE(d.name, r.name, c.name, 'Owner') as name,
			CASE
				WHEN d.logto_id IS NOT NULL THEN 'distributor'
				WHEN r.logto_id IS NOT NULL THEN 'reseller'
				WHEN c.logto_id IS NOT NULL THEN 'customer'
				ELSE 'owner'
			END as type,
			COALESCE(d.id::text, r.id::text, c.id::text, '') as db_id
		FROM (SELECT $1 as logto_id) o
		LEFT JOIN distributors d ON (o.logto_id = d.logto_id OR o.logto_id = d.id::text) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (o.logto_id = r.logto_id OR o.logto_id = r.id::text) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (o.logto_id = c.logto_id OR o.logto_id = c.id::text) AND c.deleted_at IS NULL
	`

	var org models.Organization
	org.LogtoID = logtoOrgID

	err := database.DB.QueryRow(query, logtoOrgID).Scan(&org.Name, &org.Type, &org.ID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("organization_id", logtoOrgID).
			Msg("Failed to fetch organization info")
		return models.Organization{
			ID:      "",
			LogtoID: logtoOrgID,
			Name:    "Owner",
			Type:    "owner",
		}
	}

	return org
}

// GetSystemsTrend calculates trend data for systems over a specified period
func (s *LocalSystemsService) GetSystemsTrend(period int, userOrgRole, userOrgID string) (*models.TrendResponse, error) {
	// Get hierarchical organization IDs
	userService := NewUserService()
	allowedOrgIDs, err := userService.GetHierarchicalOrganizationIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchical organization IDs: %w", err)
	}

	if len(allowedOrgIDs) == 0 {
		return nil, fmt.Errorf("no accessible organizations")
	}

	// Determine period label and grouping
	var periodLabel string
	var grouping string

	switch period {
	case 7:
		periodLabel = "Last 7 days"
		grouping = "daily"
	case 30:
		periodLabel = "Last 30 days"
		grouping = "daily"
	case 180:
		periodLabel = "Last 6 months"
		grouping = "weekly"
	case 365:
		periodLabel = "Last year"
		grouping = "monthly"
	default:
		return nil, fmt.Errorf("invalid period: %d (supported: 7, 30, 180, 365)", period)
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs)+1)
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orgID
	}
	args[len(allowedOrgIDs)] = period

	placeholdersStr := strings.Join(placeholders, ",")

	// Query for trend data based on grouping
	var query string
	switch grouping {
	case "daily":
		query = fmt.Sprintf(`
			WITH date_series AS (
				SELECT generate_series(
					CURRENT_DATE - INTERVAL '%d days',
					CURRENT_DATE,
					INTERVAL '1 day'
				)::date AS date
			)
			SELECT
				ds.date::text AS period_date,
				COALESCE(
					(SELECT COUNT(*)
					 FROM systems
					 WHERE deleted_at IS NULL
					   AND created_by ->> 'organization_id' IN (%s)
					   AND created_at <= ds.date + INTERVAL '23 hours 59 minutes 59 seconds'),
					0
				) AS count
			FROM date_series ds
			ORDER BY ds.date
		`, period-1, placeholdersStr)
	case "weekly":
		query = fmt.Sprintf(`
			WITH week_series AS (
				SELECT generate_series(
					DATE_TRUNC('week', CURRENT_DATE - INTERVAL '%d days'),
					DATE_TRUNC('week', CURRENT_DATE),
					INTERVAL '1 week'
				)::date AS week_start
			)
			SELECT
				ws.week_start::text AS period_date,
				COALESCE(
					(SELECT COUNT(*)
					 FROM systems
					 WHERE deleted_at IS NULL
					   AND created_by ->> 'organization_id' IN (%s)
					   AND created_at <= ws.week_start + INTERVAL '6 days 23 hours 59 minutes 59 seconds'),
					0
				) AS count
			FROM week_series ws
			ORDER BY ws.week_start
		`, period, placeholdersStr)
	default: // monthly
		query = fmt.Sprintf(`
			WITH month_series AS (
				SELECT generate_series(
					DATE_TRUNC('month', CURRENT_DATE - INTERVAL '%d days'),
					DATE_TRUNC('month', CURRENT_DATE),
					INTERVAL '1 month'
				)::date AS month_start
			)
			SELECT
				ms.month_start::text AS period_date,
				COALESCE(
					(SELECT COUNT(*)
					 FROM systems
					 WHERE deleted_at IS NULL
					   AND created_by ->> 'organization_id' IN (%s)
					   AND created_at <= ms.month_start + INTERVAL '1 month' - INTERVAL '1 second'),
					0
				) AS count
			FROM month_series ms
			ORDER BY ms.month_start
		`, period, placeholdersStr)
	}

	// Execute query
	rows, err := database.DB.Query(query, args[:len(args)-1]...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trend data: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// Parse results
	var trendData []models.TrendDataPoint
	for rows.Next() {
		var dp models.TrendDataPoint
		if err := rows.Scan(&dp.Date, &dp.Count); err != nil {
			return nil, fmt.Errorf("failed to scan trend data: %w", err)
		}
		trendData = append(trendData, dp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating trend data: %w", err)
	}

	// Calculate current and previous totals
	currentTotal := 0
	previousTotal := 0
	if len(trendData) > 0 {
		currentTotal = trendData[len(trendData)-1].Count
		if len(trendData) > 1 {
			previousTotal = trendData[0].Count
		}
	}

	// Calculate delta and trend
	delta := currentTotal - previousTotal
	deltaPercentage := 0.0
	if previousTotal > 0 {
		deltaPercentage = (float64(delta) / float64(previousTotal)) * 100
	}

	trend := "stable"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	return &models.TrendResponse{
		Period:          period,
		PeriodLabel:     periodLabel,
		CurrentTotal:    currentTotal,
		PreviousTotal:   previousTotal,
		Delta:           delta,
		DeltaPercentage: deltaPercentage,
		Trend:           trend,
		DataPoints:      trendData,
	}, nil
}

// CanSuspendSystem validates if a user can suspend/reactivate a system based on hierarchical permissions
func (s *LocalSystemsService) CanSuspendSystem(userOrgRole, userOrgID, systemOrgID string) (bool, string) {
	normalizedRole := strings.ToLower(userOrgRole)

	// Check the target organization type is manageable by the user's role
	if database.DB != nil {
		userService := NewUserService()
		targetOrgType := userService.GetOrganizationType(systemOrgID)
		if !userService.CanAccessOrgType(normalizedRole, targetOrgType) {
			return false, "cannot suspend systems in higher-level organizations"
		}
	}

	switch normalizedRole {
	case "owner":
		return true, ""
	case "distributor":
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedRole, userOrgID, systemOrgID) {
			return true, ""
		}
		return false, "distributors can only suspend systems in organizations they manage"
	case "reseller":
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(normalizedRole, userOrgID, systemOrgID) {
			return true, ""
		}
		return false, "resellers can only suspend systems in their own organization or customers they manage"
	case "customer":
		if systemOrgID == userOrgID {
			return true, ""
		}
		return false, "customers can only suspend systems in their own organization"
	default:
		return false, "insufficient permissions to suspend systems"
	}
}

// SuspendSystem suspends a single system
func (s *LocalSystemsService) SuspendSystem(systemID, userOrgRole, userOrgID string) error {
	systemRepo := entities.NewLocalSystemRepository()
	system, err := systemRepo.GetByID(systemID)
	if err != nil {
		return err
	}

	if canSuspend, reason := s.CanSuspendSystem(userOrgRole, userOrgID, system.Organization.LogtoID); !canSuspend {
		return fmt.Errorf("access denied: %s", reason)
	}

	return systemRepo.SuspendSystem(systemID)
}

// ReactivateSystem reactivates a suspended system
func (s *LocalSystemsService) ReactivateSystem(systemID, userOrgRole, userOrgID string) error {
	systemRepo := entities.NewLocalSystemRepository()
	system, err := systemRepo.GetByID(systemID)
	if err != nil {
		return err
	}

	if canSuspend, reason := s.CanSuspendSystem(userOrgRole, userOrgID, system.Organization.LogtoID); !canSuspend {
		return fmt.Errorf("access denied: %s", reason)
	}

	// If the system was cascade-suspended by an org, the user must have authority over that org to reactivate
	if system.SuspendedByOrgID != nil && *system.SuspendedByOrgID != "" {
		if canReactivate, reason := s.CanSuspendSystem(userOrgRole, userOrgID, *system.SuspendedByOrgID); !canReactivate {
			return fmt.Errorf("access denied: system was suspended by organization cascade, %s", reason)
		}
	}

	return systemRepo.ReactivateSystem(systemID)
}
