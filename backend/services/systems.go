/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package services

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// SystemsService handles business logic for systems management
type SystemsService struct {
	collectURL  string
	logtoClient *LogtoManagementClient
}

// NewSystemsService creates a new systems service
func NewSystemsService() *SystemsService {
	return &SystemsService{
		collectURL:  getCollectURL(),
		logtoClient: NewLogtoManagementClient(),
	}
}

func getCollectURL() string {
	return "http://localhost:8081"
}

// CreateSystem creates a new system with automatic secret generation
func (s *SystemsService) CreateSystem(request *models.CreateSystemRequest, creatorInfo *models.SystemCreator) (*models.System, error) {
	// Validate system type is allowed
	if err := s.validateSystemType(request.Type); err != nil {
		return nil, err
	}

	// Validate customer_id exists in Logto as Customer organization
	if err := s.validateCustomerID(request.CustomerID); err != nil {
		return nil, fmt.Errorf("invalid customer_id: %w", err)
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

// generateSystemSecret generates a cryptographically secure random secret
func (s *SystemsService) generateSystemSecret() (string, error) {
	// Generate 32 random bytes (will be 64 hex characters)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// validateCustomerID validates that the customer_id exists in Logto as a Customer organization
func (s *SystemsService) validateCustomerID(customerID string) error {
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
func (s *SystemsService) validateSystemType(systemType string) error {
	for _, allowedType := range configuration.Config.SystemTypes {
		if systemType == allowedType {
			return nil
		}
	}

	return fmt.Errorf("invalid system type '%s', allowed types: %v", systemType, configuration.Config.SystemTypes)
}

// GetSystemsByOrganization retrieves systems filtered by organization access
func (s *SystemsService) GetSystemsByOrganization(userID string, userOrgRole, userRole string) ([]*models.System, error) {
	query := `
		SELECT id, name, type, status, fqdn, ipv4_address, ipv6_address, version, last_seen, custom_data, customer_id, created_at, updated_at, created_by
		FROM systems
		ORDER BY created_at DESC
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query systems: %w", err)
	}
	defer rows.Close()

	var systems []*models.System
	for rows.Next() {
		system := &models.System{}
		var customDataJSON []byte
		var createdByJSON []byte
		var fqdn, ipv4Address, ipv6Address, version sql.NullString

		err := rows.Scan(
			&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
			&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON, &system.CustomerID,
			&system.CreatedAt, &system.UpdatedAt, &createdByJSON,
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

// GetSystemByID retrieves a specific system with access validation
func (s *SystemsService) GetSystemByID(systemID, userID string, userOrgRole, userRole string) (*models.System, error) {
	query := `
		SELECT id, name, type, status, fqdn, ipv4_address, ipv6_address, version, last_seen, custom_data, customer_id, created_at, updated_at, created_by
		FROM systems
		WHERE id = $1
	`

	system := &models.System{}
	var customDataJSON []byte
	var createdByJSON []byte
	var fqdn, ipv4Address, ipv6Address, version sql.NullString

	err := database.DB.QueryRow(query, systemID).Scan(
		&system.ID, &system.Name, &system.Type, &system.Status, &fqdn,
		&ipv4Address, &ipv6Address, &version, &system.LastSeen, &customDataJSON, &system.CustomerID,
		&system.CreatedAt, &system.UpdatedAt, &createdByJSON,
	)

	if err == nil {
		// Convert NullString to string
		system.FQDN = fqdn.String
		system.IPv4Address = ipv4Address.String
		system.IPv6Address = ipv6Address.String
		system.Version = version.String
	}

	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("system not found")
		}
		return nil, fmt.Errorf("failed to query system: %w", err)
	}

	// Parse custom_data JSON
	if len(customDataJSON) > 0 {
		if err := json.Unmarshal(customDataJSON, &system.CustomData); err != nil {
			logger.Warn().Err(err).Str("system_id", systemID).Msg("Failed to parse system custom_data")
			system.CustomData = make(map[string]string)
		}
	} else {
		system.CustomData = make(map[string]string)
	}

	// Parse created_by JSON
	if len(createdByJSON) > 0 {
		if err := json.Unmarshal(createdByJSON, &system.CreatedBy); err != nil {
			logger.Warn().Err(err).Str("system_id", systemID).Msg("Failed to parse created_by")
		}
	}

	logger.Debug().
		Str("system_id", systemID).
		Str("user_id", userID).
		Msg("Retrieved system by ID")

	return system, nil
}

// UpdateSystem updates an existing system with access validation
func (s *SystemsService) UpdateSystem(systemID string, request *models.UpdateSystemRequest, userID string, userOrgRole, userRole string) (*models.System, error) {
	// Validate access to the system
	system, err := s.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		return nil, err
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
func (s *SystemsService) DeleteSystem(systemID, userID string, userOrgRole, userRole string) error {
	// Validate access to the system
	_, err := s.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		return err
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
func (s *SystemsService) RegenerateSystemSecret(systemID, userID string, userOrgRole, userRole string) (*models.System, error) {
	// Validate access to the system
	_, err := s.GetSystemByID(systemID, userID, userOrgRole, userRole)
	if err != nil {
		return nil, err
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
	system, err := s.GetSystemByID(systemID, userID, userOrgRole, userRole)
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
