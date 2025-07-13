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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)


// SystemsService handles business logic for systems management
type SystemsService struct {
	collectURL string
}

// NewSystemsService creates a new systems service
func NewSystemsService() *SystemsService {
	return &SystemsService{
		collectURL: getCollectURL(),
	}
}

// getCollectURL returns the collect service URL from configuration
func getCollectURL() string {
	// This should come from configuration
	// For now, we'll use localhost
	return "http://localhost:8081"
}

// CreateSystem creates a new system with automatic secret generation
func (s *SystemsService) CreateSystem(request *models.CreateSystemRequest, createdBy string) (*models.CreateSystemResponse, error) {
	// Generate unique system ID
	systemID := uuid.New().String()
	
	// Generate secure system secret (64 characters)
	secret, err := s.generateSystemSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate system secret: %w", err)
	}
	
	now := time.Now()
	
	// Convert metadata to JSON for storage
	metadataJSON, err := json.Marshal(request.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	// Insert system into database
	query := `
		INSERT INTO systems (id, name, type, status, ip_address, version, last_seen, metadata, organization_id, created_at, updated_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err = database.DB.Exec(query, systemID, request.Name, request.Type, "offline", request.IPAddress,
		request.Version, now, metadataJSON, request.OrganizationID, now, now, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create system: %w", err)
	}
	
	// Hash the secret for storage
	hash := sha256.Sum256([]byte(secret))
	hashedSecret := hex.EncodeToString(hash[:])
	
	// Insert system credentials
	credQuery := `
		INSERT INTO system_credentials (system_id, secret_hash, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	
	_, err = database.DB.Exec(credQuery, systemID, hashedSecret, true, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to store system credentials: %w", err)
	}
	
	// Create system object
	system := &models.System{
		ID:             systemID,
		Name:           request.Name,
		Type:           request.Type,
		Status:         "offline",
		IPAddress:      request.IPAddress,
		Version:        request.Version,
		LastSeen:       now,
		Metadata:       request.Metadata,
		OrganizationID: request.OrganizationID,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      createdBy,
	}
	
	// Create system secret object
	systemSecret := &models.SystemSecret{
		SystemID:   systemID,
		Secret:     secret,
		SecretHint: secret[len(secret)-4:],
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  createdBy,
	}
	
	logger.Info().
		Str("system_id", systemID).
		Str("system_name", system.Name).
		Str("organization_id", system.OrganizationID).
		Str("created_by", createdBy).
		Msg("System created successfully")
	
	return &models.CreateSystemResponse{
		System:       system,
		SystemSecret: systemSecret,
	}, nil
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


// ValidateOrganizationAccess validates that the user can create systems in the specified organization
func (s *SystemsService) ValidateOrganizationAccess(userID, organizationID string, userOrgRole, userRole string) error {
	// Implement RBAC validation logic
	
	// Owner can create systems in any organization
	if userOrgRole == "Owner" {
		return nil
	}
	
	// Distributors can create systems in their own organization and customer organizations
	if userOrgRole == "Distributor" {
		// TODO: Check if organizationID is user's org or a customer org
		return nil
	}
	
	// Resellers can create systems in their own organization and customer organizations
	if userOrgRole == "Reseller" {
		// TODO: Check if organizationID is user's org or a customer org
		return nil
	}
	
	// Customers can only create systems in their own organization
	if userOrgRole == "Customer" {
		// TODO: Check if organizationID matches user's organization
		return nil
	}
	
	// Support and Admin user roles can manage systems regardless of organization
	if userRole == "Support" || userRole == "Admin" {
		return nil
	}
	
	return fmt.Errorf("insufficient permissions to create systems in organization %s", organizationID)
}

// GetSystemsByOrganization retrieves systems filtered by organization access
func (s *SystemsService) GetSystemsByOrganization(userID string, userOrgRole, userRole string) ([]*models.System, error) {
	query := `
		SELECT id, name, type, status, ip_address, version, last_seen, metadata, organization_id, created_at, updated_at, created_by
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
		var metadataJSON []byte
		
		err := rows.Scan(
			&system.ID, &system.Name, &system.Type, &system.Status, &system.IPAddress,
			&system.Version, &system.LastSeen, &metadataJSON, &system.OrganizationID,
			&system.CreatedAt, &system.UpdatedAt, &system.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan system: %w", err)
		}
		
		// Parse metadata JSON
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &system.Metadata); err != nil {
				logger.Warn().Err(err).Str("system_id", system.ID).Msg("Failed to parse system metadata")
				system.Metadata = make(map[string]string)
			}
		} else {
			system.Metadata = make(map[string]string)
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
		SELECT id, name, type, status, ip_address, version, last_seen, metadata, organization_id, created_at, updated_at, created_by
		FROM systems
		WHERE id = $1
	`
	
	system := &models.System{}
	var metadataJSON []byte
	
	err := database.DB.QueryRow(query, systemID).Scan(
		&system.ID, &system.Name, &system.Type, &system.Status, &system.IPAddress,
		&system.Version, &system.LastSeen, &metadataJSON, &system.OrganizationID,
		&system.CreatedAt, &system.UpdatedAt, &system.CreatedBy,
	)
	
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("system not found")
		}
		return nil, fmt.Errorf("failed to query system: %w", err)
	}
	
	// Parse metadata JSON
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &system.Metadata); err != nil {
			logger.Warn().Err(err).Str("system_id", systemID).Msg("Failed to parse system metadata")
			system.Metadata = make(map[string]string)
		}
	} else {
		system.Metadata = make(map[string]string)
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
		system.Type = request.Type
	}
	if request.Status != "" {
		system.Status = request.Status
	}
	if request.IPAddress != "" {
		system.IPAddress = request.IPAddress
	}
	if request.Version != "" {
		system.Version = request.Version
	}
	if request.Metadata != nil {
		system.Metadata = request.Metadata
	}
	if request.OrganizationID != "" {
		// Validate access to target organization
		if err := s.ValidateOrganizationAccess(userID, request.OrganizationID, userOrgRole, userRole); err != nil {
			return nil, fmt.Errorf("cannot move system to organization: %w", err)
		}
		system.OrganizationID = request.OrganizationID
	}
	
	system.UpdatedAt = now
	
	// Convert metadata to JSON for storage
	metadataJSON, err := json.Marshal(system.Metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}
	
	// Update system in database
	query := `
		UPDATE systems
		SET name = $2, type = $3, status = $4, ip_address = $5, version = $6, metadata = $7, organization_id = $8, updated_at = $9
		WHERE id = $1
	`
	
	_, err = database.DB.Exec(query, systemID, system.Name, system.Type, system.Status,
		system.IPAddress, system.Version, metadataJSON, system.OrganizationID, now)
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
func (s *SystemsService) RegenerateSystemSecret(systemID, userID string, userOrgRole, userRole string) (*models.SystemSecret, error) {
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
	
	// Update system credentials
	query := `
		UPDATE system_credentials
		SET secret_hash = $2, updated_at = $3, last_used = NULL
		WHERE system_id = $1
	`
	
	_, err = database.DB.Exec(query, systemID, hashedSecret, now)
	if err != nil {
		return nil, fmt.Errorf("failed to update system credentials: %w", err)
	}
	
	systemSecret := &models.SystemSecret{
		SystemID:   systemID,
		Secret:     secret,
		SecretHint: secret[len(secret)-4:],
		CreatedAt:  now,
		UpdatedAt:  now,
		CreatedBy:  userID,
	}
	
	logger.Info().
		Str("system_id", systemID).
		Str("regenerated_by", userID).
		Msg("System secret regenerated successfully")
	
	return systemSecret, nil
}