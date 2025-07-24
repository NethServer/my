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

	// Validate customer_id exists in Logto as Customer organization
	if err := s.validateCustomerID(request.CustomerID); err != nil {
		return nil, fmt.Errorf("invalid customer_id: %w", err)
	}

	// Validate hierarchical access - creator must be able to create systems in the customer organization
	if canCreate, reason := s.CanCreateSystem(userOrgRole, userOrgID, request); !canCreate {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

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
		INSERT INTO systems (id, name, type, status, custom_data, customer_id, secret_hash, secret_hint, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = database.DB.Exec(query, systemID, request.Name, request.Type, "offline",
		customDataJSON, request.CustomerID, hashedSecret, secret[len(secret)-4:], now, now, createdByJSON)
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
		CustomerID: request.CustomerID,
		Secret:     secret,
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  *creatorInfo,
	}

	logger.Info().
		Str("system_id", systemID).
		Str("system_name", system.Name).
		Str("customer_id", system.CustomerID).
		Str("created_by_user", creatorInfo.UserID).
		Str("created_by_org", creatorInfo.OrganizationID).
		Msg("System created successfully")

	return system, nil
}

// GetSystemsByOrganization retrieves systems filtered by organization access
func (s *LocalSystemsService) GetSystemsByOrganization(userID string, userOrgRole, userRole string) ([]*models.System, error) {
	query := `
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version, s.last_seen,
		       s.custom_data, s.customer_id, s.created_at, s.updated_at, s.created_by, h.last_heartbeat
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
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
			&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON, &system.CustomerID,
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

// GetSystemsByOrganizationPaginated retrieves systems filtered by organization with pagination and RBAC
func (s *LocalSystemsService) GetSystemsByOrganizationPaginated(userID, userOrgID, userOrgRole string, page, pageSize int) ([]*models.System, int, error) {
	// Calculate offset
	offset := (page - 1) * pageSize

	// Get all customer organization IDs the user can access hierarchically
	// Systems can only be associated with customers, so we only need customer IDs
	systemRepo := entities.NewLocalSystemRepository()
	allowedOrgIDs, err := systemRepo.GetHierarchicalCustomerIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get hierarchical customer IDs: %w", err)
	}

	if len(allowedOrgIDs) == 0 {
		return []*models.System{}, 0, nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs))
	for i, orgID := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = orgID
	}
	placeholdersStr := strings.Join(placeholders, ",")

	// Get total count first
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM systems s
		WHERE s.customer_id IN (%s)`, placeholdersStr)

	var totalCount int
	err = database.DB.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get systems count: %w", err)
	}

	if totalCount == 0 {
		return []*models.System{}, 0, nil
	}

	// Get paginated systems with hierarchical filtering
	listArgs := make([]interface{}, len(args)+2)
	copy(listArgs, args)
	listArgs[len(args)] = pageSize
	listArgs[len(args)+1] = offset

	query := fmt.Sprintf(`
		SELECT s.id, s.name, s.type, s.status, s.fqdn, s.ipv4_address, s.ipv6_address, s.version, s.last_seen,
		       s.custom_data, s.customer_id, s.created_at, s.updated_at, s.created_by, h.last_heartbeat
		FROM systems s
		LEFT JOIN system_heartbeats h ON s.id = h.system_id
		WHERE s.customer_id IN (%s)
		ORDER BY s.created_at DESC
		LIMIT $%d OFFSET $%d`, placeholdersStr, len(args)+1, len(args)+2)

	rows, err := database.DB.Query(query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query systems: %w", err)
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
			&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON, &system.CustomerID,
			&system.CreatedAt, &system.UpdatedAt, &createdByJSON, &lastHeartbeat,
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
		return nil, 0, fmt.Errorf("error iterating systems: %w", err)
	}

	logger.Debug().
		Str("user_id", userID).
		Int("page", page).
		Int("page_size", pageSize).
		Int("count", len(systems)).
		Int("total_count", totalCount).
		Msg("Retrieved paginated systems for user")

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

	// Validate update permissions
	if canUpdate, reason := s.CanUpdateSystem(userOrgRole, userOrgID, system.CustomerID); !canUpdate {
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
	if request.CustomerID != "" {
		// Validate customer_id exists in Logto as Customer organization
		if err := s.validateCustomerID(request.CustomerID); err != nil {
			return nil, fmt.Errorf("invalid customer_id: %w", err)
		}
		system.CustomerID = request.CustomerID
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
		SET name = $2, type = $3, custom_data = $4, customer_id = $5, updated_at = $6
		WHERE id = $1
	`

	_, err = database.DB.Exec(query, systemID, system.Name, system.Type,
		customDataJSON, system.CustomerID, now)
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

	// Validate delete permissions
	if canDelete, reason := s.CanDeleteSystem(userOrgRole, userOrgID, system.CustomerID); !canDelete {
		return fmt.Errorf("access denied: %s", reason)
	}

	// Delete system from database (CASCADE will handle system_credentials)
	query := `DELETE FROM systems WHERE id = $1`

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

	// Validate update permissions (regenerating secret is an update operation)
	if canUpdate, reason := s.CanUpdateSystem(userOrgRole, userOrgID, system.CustomerID); !canUpdate {
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
		WHERE id = $1
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

	systemRepo := entities.NewLocalSystemRepository()

	// Get hierarchical customer IDs first
	allowedOrgIDs, err := systemRepo.GetHierarchicalCustomerIDs(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hierarchical customer IDs: %w", err)
	}

	// Get totals with the specified timeout
	return systemRepo.GetTotalsByCustomerIDs(allowedOrgIDs, timeoutMinutes)
}

// CanCreateSystem validates if a user can create a system based on hierarchical permissions
func (s *LocalSystemsService) CanCreateSystem(userOrgRole, userOrgID string, req *models.CreateSystemRequest) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can create systems in customer organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, req.CustomerID) {
			return true, ""
		}
		return false, "distributors can only create systems in organizations they manage"
	case "reseller":
		// Reseller can create systems in customer organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, req.CustomerID) {
			return true, ""
		}
		return false, "resellers can only create systems in their own organization or customers they manage"
	case "customer":
		// Customer can only create systems in their own organization
		if req.CustomerID == userOrgID {
			return true, ""
		}
		return false, "customers can only create systems in their own organization"
	default:
		return false, "insufficient permissions to create systems"
	}
}

// CanUpdateSystem validates if a user can update a system based on hierarchical permissions
func (s *LocalSystemsService) CanUpdateSystem(userOrgRole, userOrgID, targetSystemCustomerID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can update systems in organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetSystemCustomerID) {
			return true, ""
		}
		return false, "distributors can only update systems in organizations they manage"
	case "reseller":
		// Reseller can update systems in organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetSystemCustomerID) {
			return true, ""
		}
		return false, "resellers can only update systems in their own organization or customers they manage"
	case "customer":
		// Customer can only update systems in their own organization
		if targetSystemCustomerID == userOrgID {
			return true, ""
		}
		return false, "customers can only update systems in their own organization"
	default:
		return false, "insufficient permissions to update systems"
	}
}

// CanDeleteSystem validates if a user can delete a system based on hierarchical permissions
func (s *LocalSystemsService) CanDeleteSystem(userOrgRole, userOrgID, targetSystemCustomerID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can delete systems in organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetSystemCustomerID) {
			return true, ""
		}
		return false, "distributors can only delete systems in organizations they manage"
	case "reseller":
		// Reseller can delete systems in organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetSystemCustomerID) {
			return true, ""
		}
		return false, "resellers can only delete systems in their own organization or customers they manage"
	case "customer":
		// Customer can only delete systems in their own organization
		if targetSystemCustomerID == userOrgID {
			return true, ""
		}
		return false, "customers can only delete systems in their own organization"
	default:
		return false, "insufficient permissions to delete systems"
	}
}

// CanAccessSystem validates if a user can access a specific system based on hierarchical permissions
func (s *LocalSystemsService) CanAccessSystem(system *models.System, userOrgRole, userOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can access systems in organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, system.CustomerID) {
			return true, ""
		}
		return false, "distributors can only access systems in organizations they manage"
	case "reseller":
		// Reseller can access systems in organizations they manage hierarchically
		userService := NewUserService()
		if userService.IsOrganizationInHierarchy(userOrgRole, userOrgID, system.CustomerID) {
			return true, ""
		}
		return false, "resellers can only access systems in their own organization or customers they manage"
	case "customer":
		// Customer can only access systems in their own organization
		if system.CustomerID == userOrgID {
			return true, ""
		}
		return false, "customers can only access systems in their own organization"
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

// validateCustomerID validates that the customer_id exists in Logto as a Customer organization
func (s *LocalSystemsService) validateCustomerID(customerID string) error {
	// Get organization from Logto
	org, err := s.logtoClient.GetOrganizationByID(customerID)
	if err != nil {
		return fmt.Errorf("customer organization not found: %w", err)
	}

	// Check if organization has "Customer" type in custom data
	if org.CustomData == nil {
		return fmt.Errorf("organization %s has no custom data - cannot verify type", customerID)
	}

	orgType, ok := org.CustomData["type"].(string)
	if !ok {
		return fmt.Errorf("organization %s has no type in custom data", customerID)
	}

	if orgType != "customer" {
		return fmt.Errorf("organization %s is not a customer (type: %s)", customerID, orgType)
	}

	return nil
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
