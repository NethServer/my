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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/logto"
)

// LocalOrganizationService handles local-first CRUD operations with Logto sync
type LocalOrganizationService struct {
	distributorRepo *entities.LocalDistributorRepository
	resellerRepo    *entities.LocalResellerRepository
	customerRepo    *entities.LocalCustomerRepository
	userRepo        *entities.LocalUserRepository
	systemRepo      *entities.LocalSystemRepository
	logtoClient     *logto.LogtoManagementClient
}

// NewLocalOrganizationService creates a new local organization service
func NewOrganizationService() *LocalOrganizationService {
	return &LocalOrganizationService{
		distributorRepo: entities.NewLocalDistributorRepository(),
		resellerRepo:    entities.NewLocalResellerRepository(),
		customerRepo:    entities.NewLocalCustomerRepository(),
		userRepo:        entities.NewLocalUserRepository(),
		systemRepo:      entities.NewLocalSystemRepository(),
		logtoClient:     logto.NewManagementClient(),
	}
}

// CreateDistributor creates a distributor locally and syncs to Logto
func (s *LocalOrganizationService) CreateDistributor(req *models.CreateLocalDistributorRequest, createdByUserID, createdByOrgID string) (*models.LocalDistributor, error) {
	// Validate required fields
	var validationErrors []response.ValidationError

	if strings.TrimSpace(req.Name) == "" {
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "name",
			Message: "cannot_be_empty",
			Value:   req.Name,
		})
	}

	// Validate VAT in custom_data
	if req.CustomData == nil || req.CustomData["vat"] == nil {
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "custom_data.vat",
			Message: "required",
			Value:   "",
		})
	} else if vatStr, ok := req.CustomData["vat"].(string); !ok || strings.TrimSpace(vatStr) == "" {
		vatValue := ""
		if req.CustomData["vat"] != nil {
			vatValue = fmt.Sprintf("%v", req.CustomData["vat"])
		}
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "custom_data.vat",
			Message: "cannot_be_empty",
			Value:   vatValue,
		})
	}

	if len(validationErrors) > 0 {
		validationErr := &ValidationError{
			StatusCode: 400,
			ErrorData: response.ErrorData{
				Type:   "validation_error",
				Errors: validationErrors,
			},
		}
		return nil, validationErr
	}
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Prepare CustomData with user fields and system fields
	// Start with user-provided custom data (allows custom fields)
	customData := make(map[string]interface{})
	if req.CustomData != nil {
		for k, v := range req.CustomData {
			customData[k] = v
		}
	}

	// Security: Prevent creation of organizations with reserved type "owner"
	if customDataType, ok := customData["type"]; ok {
		if typeStr, ok := customDataType.(string); ok && strings.ToLower(typeStr) == "owner" {
			return nil, &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Errors: []response.ValidationError{
						{
							Key:     "custom_data.type",
							Message: "organization type 'owner' is reserved and cannot be used",
							Value:   typeStr,
						},
					},
				},
			}
		}
	}

	// System fields - these override any user-provided values and are always maintained
	customData["type"] = "distributor"
	customData["createdBy"] = createdByOrgID
	customData["createdAt"] = time.Now().Format(time.RFC3339)

	// Update the request with the properly managed customData
	req.CustomData = customData

	// 2. Create in local DB with CustomData
	distributor, err := s.distributorRepo.Create(req)
	if err != nil {
		// Check for VAT constraint violation (from entities/database)
		if strings.Contains(err.Error(), "already exists") {
			vatValue := ""
			if req.CustomData != nil && req.CustomData["vat"] != nil {
				vatValue = fmt.Sprintf("%v", req.CustomData["vat"])
			}
			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.vat",
						Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
						Value:   vatValue,
					}},
				},
			}
			return nil, validationErr
		}
		return nil, fmt.Errorf("failed to create distributor locally: %w", err)
	}

	// 3. Sync to Logto using the same CustomData
	logtoOrg, err := s.logtoClient.CreateOrganization(models.CreateOrganizationRequest{
		Name:        distributor.Name,
		Description: distributor.Description,
		CustomData:  distributor.CustomData,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("distributor_id", distributor.ID).
			Str("distributor_name", distributor.Name).
			Msg("Failed to sync distributor to Logto")
		return nil, fmt.Errorf("failed to sync distributor to Logto: %w", err)
	}

	// 4. Configure JIT roles for the new organization
	err = s.configureOrganizationJitRoles(logtoOrg.ID, "distributor")
	if err != nil {
		logger.Warn().
			Err(err).
			Str("distributor_id", distributor.ID).
			Str("logto_org_id", logtoOrg.ID).
			Msg("Failed to configure JIT roles for distributor organization")
	}

	// 5. Mark as synced
	err = s.markDistributorSynced(distributor.ID, logtoOrg.ID)
	if err != nil {
		// Log but don't fail - distributor is created in both places
		logger.Warn().
			Err(err).
			Str("distributor_id", distributor.ID).
			Msg("Failed to mark distributor as synced")
	}
	distributor.LogtoID = &logtoOrg.ID

	// 6. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("distributor_id", distributor.ID).
		Str("distributor_name", distributor.Name).
		Str("logto_org_id", logtoOrg.ID).
		Str("created_by", createdByUserID).
		Msg("Distributor created successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()
	return distributor, nil
}

// CreateReseller creates a reseller locally and syncs to Logto
func (s *LocalOrganizationService) CreateReseller(req *models.CreateLocalResellerRequest, createdByUserID, createdByOrgID string) (*models.LocalReseller, error) {
	// Validate required fields
	var validationErrors []response.ValidationError

	if strings.TrimSpace(req.Name) == "" {
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "name",
			Message: "cannot_be_empty",
			Value:   req.Name,
		})
	}

	// Validate VAT in custom_data
	if req.CustomData == nil || req.CustomData["vat"] == nil {
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "custom_data.vat",
			Message: "required",
			Value:   "",
		})
	} else if vatStr, ok := req.CustomData["vat"].(string); !ok || strings.TrimSpace(vatStr) == "" {
		vatValue := ""
		if req.CustomData["vat"] != nil {
			vatValue = fmt.Sprintf("%v", req.CustomData["vat"])
		}
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "custom_data.vat",
			Message: "cannot_be_empty",
			Value:   vatValue,
		})
	}

	if len(validationErrors) > 0 {
		validationErr := &ValidationError{
			StatusCode: 400,
			ErrorData: response.ErrorData{
				Type:   "validation_error",
				Errors: validationErrors,
			},
		}
		return nil, validationErr
	}
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// No validation needed - hierarchy is tracked in CustomData

	// 1. Prepare CustomData with user fields and system fields
	// Start with user-provided custom data (allows custom fields)
	customData := make(map[string]interface{})
	if req.CustomData != nil {
		for k, v := range req.CustomData {
			customData[k] = v
		}
	}

	// Security: Prevent creation of organizations with reserved type "owner"
	if customDataType, ok := customData["type"]; ok {
		if typeStr, ok := customDataType.(string); ok && strings.ToLower(typeStr) == "owner" {
			return nil, &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Errors: []response.ValidationError{
						{
							Key:     "custom_data.type",
							Message: "organization type 'owner' is reserved and cannot be used",
							Value:   typeStr,
						},
					},
				},
			}
		}
	}

	// System fields - these override any user-provided values and are always maintained
	customData["type"] = "reseller"
	customData["createdBy"] = createdByOrgID
	customData["createdAt"] = time.Now().Format(time.RFC3339)

	// Update the request with the properly managed customData
	req.CustomData = customData

	// 2. Create in local DB with CustomData
	reseller, err := s.resellerRepo.Create(req)
	if err != nil {
		// Check for VAT constraint violation (from entities/database)
		if strings.Contains(err.Error(), "already exists") {
			vatValue := ""
			if req.CustomData != nil && req.CustomData["vat"] != nil {
				vatValue = fmt.Sprintf("%v", req.CustomData["vat"])
			}
			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.vat",
						Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
						Value:   vatValue,
					}},
				},
			}
			return nil, validationErr
		}
		return nil, fmt.Errorf("failed to create reseller locally: %w", err)
	}

	// 3. Sync to Logto using the same CustomData
	logtoOrg, err := s.logtoClient.CreateOrganization(models.CreateOrganizationRequest{
		Name:        reseller.Name,
		Description: reseller.Description,
		CustomData:  reseller.CustomData,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("reseller_id", reseller.ID).
			Str("reseller_name", reseller.Name).
			Msg("Failed to sync reseller to Logto")
		return nil, fmt.Errorf("failed to sync reseller to Logto: %w", err)
	}

	// 3. Configure JIT roles for the new organization
	err = s.configureOrganizationJitRoles(logtoOrg.ID, "reseller")
	if err != nil {
		logger.Warn().
			Err(err).
			Str("reseller_id", reseller.ID).
			Str("logto_org_id", logtoOrg.ID).
			Msg("Failed to configure JIT roles for reseller organization")
	}

	// 4. Mark as synced
	err = s.markResellerSynced(reseller.ID, logtoOrg.ID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("reseller_id", reseller.ID).
			Msg("Failed to mark reseller as synced")
	}
	reseller.LogtoID = &logtoOrg.ID

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("reseller_id", reseller.ID).
		Str("reseller_name", reseller.Name).
		Str("logto_org_id", logtoOrg.ID).
		Str("created_by", createdByUserID).
		Msg("Reseller created successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()
	return reseller, nil
}

// CreateCustomer creates a customer locally and syncs to Logto
func (s *LocalOrganizationService) CreateCustomer(req *models.CreateLocalCustomerRequest, createdByUserID, createdByOrgID string) (*models.LocalCustomer, error) {
	// Validate required fields
	var validationErrors []response.ValidationError

	if strings.TrimSpace(req.Name) == "" {
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "name",
			Message: "cannot_be_empty",
			Value:   req.Name,
		})
	}

	// Validate VAT in custom_data
	if req.CustomData == nil || req.CustomData["vat"] == nil {
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "custom_data.vat",
			Message: "required",
			Value:   "",
		})
	} else if vatStr, ok := req.CustomData["vat"].(string); !ok || strings.TrimSpace(vatStr) == "" {
		vatValue := ""
		if req.CustomData["vat"] != nil {
			vatValue = fmt.Sprintf("%v", req.CustomData["vat"])
		}
		validationErrors = append(validationErrors, response.ValidationError{
			Key:     "custom_data.vat",
			Message: "cannot_be_empty",
			Value:   vatValue,
		})
	}

	if len(validationErrors) > 0 {
		validationErr := &ValidationError{
			StatusCode: 400,
			ErrorData: response.ErrorData{
				Type:   "validation_error",
				Errors: validationErrors,
			},
		}
		return nil, validationErr
	}
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// No hierarchy validation needed - handled via CustomData
	creatorType := "customer"

	// 1. Prepare CustomData with user fields and system fields
	// Start with user-provided custom data (allows custom fields)
	customData := make(map[string]interface{})
	if req.CustomData != nil {
		for k, v := range req.CustomData {
			customData[k] = v
		}
	}

	// Security: Prevent creation of organizations with reserved type "owner"
	if customDataType, ok := customData["type"]; ok {
		if typeStr, ok := customDataType.(string); ok && strings.ToLower(typeStr) == "owner" {
			return nil, &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Errors: []response.ValidationError{
						{
							Key:     "custom_data.type",
							Message: "organization type 'owner' is reserved and cannot be used",
							Value:   typeStr,
						},
					},
				},
			}
		}
	}

	// System fields - these override any user-provided values and are always maintained
	customData["type"] = "customer"
	customData["createdBy"] = createdByOrgID
	customData["createdAt"] = time.Now().Format(time.RFC3339)

	// Update the request with the properly managed customData
	req.CustomData = customData

	// 2. Create in local DB with CustomData
	customer, err := s.customerRepo.Create(req)
	if err != nil {
		// Check for VAT constraint violation (from entities/database)
		if strings.Contains(err.Error(), "already exists") {
			vatValue := ""
			if req.CustomData != nil && req.CustomData["vat"] != nil {
				vatValue = fmt.Sprintf("%v", req.CustomData["vat"])
			}
			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.vat",
						Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
						Value:   vatValue,
					}},
				},
			}
			return nil, validationErr
		}
		return nil, fmt.Errorf("failed to create customer locally: %w", err)
	}

	// 3. Sync to Logto using the same CustomData
	logtoOrg, err := s.logtoClient.CreateOrganization(models.CreateOrganizationRequest{
		Name:        customer.Name,
		Description: customer.Description,
		CustomData:  customer.CustomData,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("customer_id", customer.ID).
			Str("customer_name", customer.Name).
			Msg("Failed to sync customer to Logto")
		return nil, fmt.Errorf("failed to sync customer to Logto: %w", err)
	}

	// 3. Configure JIT roles for the new organization
	err = s.configureOrganizationJitRoles(logtoOrg.ID, "customer")
	if err != nil {
		logger.Warn().
			Err(err).
			Str("customer_id", customer.ID).
			Str("logto_org_id", logtoOrg.ID).
			Msg("Failed to configure JIT roles for customer organization")
	}

	// 4. Mark as synced
	err = s.markCustomerSynced(customer.ID, logtoOrg.ID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("customer_id", customer.ID).
			Msg("Failed to mark customer as synced")
	}
	customer.LogtoID = &logtoOrg.ID

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("customer_id", customer.ID).
		Str("customer_name", customer.Name).
		Str("creator_type", creatorType).
		Str("logto_org_id", logtoOrg.ID).
		Str("created_by", createdByUserID).
		Msg("Customer created successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()
	return customer, nil
}

// Helper methods to mark entities as synced
func (s *LocalOrganizationService) markDistributorSynced(id, logtoID string) error {
	query := `UPDATE distributors SET logto_id = $1, logto_synced_at = $2, logto_sync_error = NULL WHERE id = $3`
	_, err := database.DB.Exec(query, logtoID, time.Now(), id)
	return err
}

func (s *LocalOrganizationService) markResellerSynced(id, logtoID string) error {
	query := `UPDATE resellers SET logto_id = $1, logto_synced_at = $2, logto_sync_error = NULL WHERE id = $3`
	_, err := database.DB.Exec(query, logtoID, time.Now(), id)
	return err
}

func (s *LocalOrganizationService) markCustomerSynced(id, logtoID string) error {
	query := `UPDATE customers SET logto_id = $1, logto_synced_at = $2, logto_sync_error = NULL WHERE id = $3`
	_, err := database.DB.Exec(query, logtoID, time.Now(), id)
	return err
}

// RBAC validation methods
func (s *LocalOrganizationService) CanCreateDistributor(userOrgRole, userOrgID string) (bool, string) {
	// Only Owner can create distributors
	if userOrgRole != "owner" {
		return false, "only owners can create distributors"
	}
	return true, ""
}

func (s *LocalOrganizationService) CanCreateReseller(userOrgRole, userOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		return true, ""
	default:
		return false, "insufficient permissions to create resellers"
	}
}

func (s *LocalOrganizationService) CanCreateCustomer(userOrgRole, userOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		return true, ""
	case "reseller":
		return true, ""
	default:
		return false, "insufficient permissions to create customers"
	}
}

// ============================================
// READ OPERATIONS (GetByID and List)
// ============================================

// GetDistributor retrieves a distributor by ID
func (s *LocalOrganizationService) GetDistributor(id string) (*models.LocalDistributor, error) {
	return s.distributorRepo.GetByID(id)
}

// GetReseller retrieves a reseller by ID
func (s *LocalOrganizationService) GetReseller(id string) (*models.LocalReseller, error) {
	return s.resellerRepo.GetByID(id)
}

// GetCustomer retrieves a customer by ID
func (s *LocalOrganizationService) GetCustomer(id string) (*models.LocalCustomer, error) {
	return s.customerRepo.GetByID(id)
}

// ListDistributors returns paginated distributors based on RBAC
func (s *LocalOrganizationService) ListDistributors(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalDistributor, int, error) {
	return s.distributorRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection, statuses)
}

// ListResellers returns paginated resellers based on RBAC
func (s *LocalOrganizationService) ListResellers(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalReseller, int, error) {
	return s.resellerRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection, statuses)
}

// ListCustomers returns paginated customers based on RBAC
func (s *LocalOrganizationService) ListCustomers(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses []string) ([]*models.LocalCustomer, int, error) {
	return s.customerRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection, statuses)
}

// ============================================
// UPDATE OPERATIONS
// ============================================

// UpdateDistributor updates a distributor locally and syncs to Logto
func (s *LocalOrganizationService) UpdateDistributor(id string, req *models.UpdateLocalDistributorRequest, updatedByUserID, updatedByOrgID string) (*models.LocalDistributor, error) {
	// 1. Get current distributor before update to preserve system fields
	currentDistributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get current distributor for update: %w", err)
	}

	// Check if distributor is synced to Logto
	if currentDistributor.LogtoID == nil {
		return nil, fmt.Errorf("distributor not synced to Logto yet - missing logto_id")
	}

	// 2. Prepare CustomData with proper field management
	// Start with existing custom data to preserve user-defined fields
	finalCustomData := make(map[string]interface{})
	if currentDistributor.CustomData != nil {
		for k, v := range currentDistributor.CustomData {
			finalCustomData[k] = v
		}
	}

	// Merge user-provided custom data (allows users to update their custom fields)
	if req.CustomData != nil {
		for k, v := range *req.CustomData {
			finalCustomData[k] = v
		}
	}

	// Security: Prevent updating organizations to reserved type "owner"
	if customDataType, ok := finalCustomData["type"]; ok {
		if typeStr, ok := customDataType.(string); ok && strings.ToLower(typeStr) == "owner" {
			return nil, &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Errors: []response.ValidationError{
						{
							Key:     "custom_data.type",
							Message: "organization type 'owner' is reserved and cannot be used",
							Value:   typeStr,
						},
					},
				},
			}
		}
	}

	// System fields - these override any user-provided values and are always maintained
	// CRITICAL: Preserve original type and createdBy - never change them
	finalCustomData["type"] = "distributor"
	if existingCreatedBy, exists := finalCustomData["createdBy"]; exists {
		finalCustomData["createdBy"] = existingCreatedBy
	} else {
		// Fallback if somehow missing (should not happen)
		finalCustomData["createdBy"] = updatedByOrgID
	}

	// Add update tracking (these are additional fields, not replacements)
	finalCustomData["updatedBy"] = updatedByOrgID
	finalCustomData["updatedAt"] = time.Now().Format(time.RFC3339)

	// 3. Validate changes in Logto FIRST (before consuming local resources)
	updateReq := models.UpdateOrganizationRequest{}
	if req.Name != nil {
		updateReq.Name = req.Name
	}
	if req.Description != nil {
		updateReq.Description = req.Description
	}
	// Use the properly managed customData
	updateReq.CustomData = finalCustomData

	// Try the update in Logto first for validation
	_, err = s.logtoClient.UpdateOrganization(*currentDistributor.LogtoID, updateReq)
	if err != nil {
		logger.Error().
			Err(err).
			Str("distributor_id", id).
			Str("distributor_name", currentDistributor.Name).
			Msg("Failed to validate distributor update in Logto")

		// No rollback needed since we haven't changed anything locally yet
		return nil, fmt.Errorf("failed to validate distributor update in Logto: %w", err)
	}

	// 4. Begin transaction for local operations (after Logto validation passes)
	tx, err := database.DB.Begin()
	if err != nil {
		// Revert the Logto changes since we can't proceed with local update
		originalReq := models.UpdateOrganizationRequest{
			Name:        &currentDistributor.Name,
			Description: &currentDistributor.Description,
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentDistributor.CustomData != nil {
			for k, v := range currentDistributor.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateOrganization(*currentDistributor.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("distributor_id", id).
				Msg("Failed to revert Logto changes after local transaction failure")
		}

		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update the request with properly managed customData
	req.CustomData = &finalCustomData

	// 5. Update in local DB (after Logto validation passes)
	distributor, err := s.distributorRepo.Update(id, req)
	if err != nil {

		// Check for VAT constraint violation (from entities/database)
		if strings.Contains(err.Error(), "already exists") {
			vatValue := ""
			if req.CustomData != nil && (*req.CustomData)["vat"] != nil {
				vatValue = fmt.Sprintf("%v", (*req.CustomData)["vat"])
			}

			// Revert Logto changes before returning validation error
			originalReq := models.UpdateOrganizationRequest{
				Name:        &currentDistributor.Name,
				Description: &currentDistributor.Description,
			}
			// Restore original custom data
			originalCustomData := make(map[string]interface{})
			if currentDistributor.CustomData != nil {
				for k, v := range currentDistributor.CustomData {
					originalCustomData[k] = v
				}
			}
			originalReq.CustomData = originalCustomData

			if _, revertErr := s.logtoClient.UpdateOrganization(*currentDistributor.LogtoID, originalReq); revertErr != nil {
				logger.Warn().
					Err(revertErr).
					Str("distributor_id", id).
					Msg("Failed to revert Logto changes after VAT constraint violation")
			}

			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.vat",
						Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
						Value:   vatValue,
					}},
				},
			}
			return nil, validationErr
		}

		// Revert the Logto changes since local update failed
		originalReq := models.UpdateOrganizationRequest{
			Name:        &currentDistributor.Name,
			Description: &currentDistributor.Description,
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentDistributor.CustomData != nil {
			for k, v := range currentDistributor.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateOrganization(*currentDistributor.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("distributor_id", id).
				Msg("Failed to revert Logto changes after local update failure")
		}

		// Transaction will be rolled back by defer
		return nil, fmt.Errorf("failed to update distributor locally: %w", err)
	}

	// 6. Mark as synced
	err = s.markDistributorSynced(id, *distributor.LogtoID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("distributor_id", id).
			Msg("Failed to mark distributor as synced after update")
	}

	// 7. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Str("updated_by", updatedByUserID).
		Msg("Distributor updated successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()
	return distributor, nil
}

// UpdateReseller updates a reseller locally and syncs to Logto
func (s *LocalOrganizationService) UpdateReseller(id string, req *models.UpdateLocalResellerRequest, updatedByUserID, updatedByOrgID string) (*models.LocalReseller, error) {
	// 1. Get current reseller before update to preserve system fields
	currentReseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get current reseller for update: %w", err)
	}

	// Check if reseller is synced to Logto
	if currentReseller.LogtoID == nil {
		return nil, fmt.Errorf("reseller not synced to Logto yet - missing logto_id")
	}

	// 2. Prepare CustomData with proper field management
	// Start with existing custom data to preserve user-defined fields
	finalCustomData := make(map[string]interface{})
	if currentReseller.CustomData != nil {
		for k, v := range currentReseller.CustomData {
			finalCustomData[k] = v
		}
	}

	// Merge user-provided custom data (allows users to update their custom fields)
	if req.CustomData != nil {
		for k, v := range *req.CustomData {
			finalCustomData[k] = v
		}
	}

	// Security: Prevent updating organizations to reserved type "owner"
	if customDataType, ok := finalCustomData["type"]; ok {
		if typeStr, ok := customDataType.(string); ok && strings.ToLower(typeStr) == "owner" {
			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.type",
						Message: "organization type 'owner' is reserved and cannot be used",
						Value:   typeStr,
					}},
				},
			}
			return nil, validationErr
		}
	}

	// System fields - these override any user-provided values and are always maintained
	// CRITICAL: Preserve original type and createdBy - never change them
	finalCustomData["type"] = "reseller"
	if existingCreatedBy, exists := finalCustomData["createdBy"]; exists {
		finalCustomData["createdBy"] = existingCreatedBy
	} else {
		// Fallback if somehow missing (should not happen)
		finalCustomData["createdBy"] = updatedByOrgID
	}

	// Add update tracking (these are additional fields, not replacements)
	finalCustomData["updatedBy"] = updatedByOrgID
	finalCustomData["updatedAt"] = time.Now().Format(time.RFC3339)

	// 3. Validate changes in Logto FIRST (before consuming local resources)
	updateReq := models.UpdateOrganizationRequest{}
	if req.Name != nil {
		updateReq.Name = req.Name
	}
	if req.Description != nil {
		updateReq.Description = req.Description
	}
	// Use the properly managed customData
	updateReq.CustomData = finalCustomData

	// Try the update in Logto first for validation
	_, err = s.logtoClient.UpdateOrganization(*currentReseller.LogtoID, updateReq)
	if err != nil {
		logger.Error().
			Err(err).
			Str("reseller_id", id).
			Str("reseller_name", currentReseller.Name).
			Msg("Failed to validate reseller update in Logto")

		// No rollback needed since we haven't changed anything locally yet
		return nil, fmt.Errorf("failed to validate reseller update in Logto: %w", err)
	}

	// 4. Begin transaction for local operations (after Logto validation passes)
	tx, err := database.DB.Begin()
	if err != nil {
		// Revert the Logto changes since we can't proceed with local update
		originalReq := models.UpdateOrganizationRequest{
			Name:        &currentReseller.Name,
			Description: &currentReseller.Description,
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentReseller.CustomData != nil {
			for k, v := range currentReseller.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateOrganization(*currentReseller.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("reseller_id", id).
				Msg("Failed to revert Logto changes after local transaction failure")
		}

		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update the request with properly managed customData
	req.CustomData = &finalCustomData

	// 5. Update in local DB (after Logto validation passes)
	reseller, err := s.resellerRepo.Update(id, req)
	if err != nil {

		// Check for VAT constraint violation (from entities/database)
		if strings.Contains(err.Error(), "already exists") {
			vatValue := ""
			if req.CustomData != nil && (*req.CustomData)["vat"] != nil {
				vatValue = fmt.Sprintf("%v", (*req.CustomData)["vat"])
			}

			// Revert Logto changes before returning validation error
			originalReq := models.UpdateOrganizationRequest{
				Name:        &currentReseller.Name,
				Description: &currentReseller.Description,
			}
			// Restore original custom data
			originalCustomData := make(map[string]interface{})
			if currentReseller.CustomData != nil {
				for k, v := range currentReseller.CustomData {
					originalCustomData[k] = v
				}
			}
			originalReq.CustomData = originalCustomData

			if _, revertErr := s.logtoClient.UpdateOrganization(*currentReseller.LogtoID, originalReq); revertErr != nil {
				logger.Warn().
					Err(revertErr).
					Str("reseller_id", id).
					Msg("Failed to revert Logto changes after VAT constraint violation")
			}

			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.vat",
						Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
						Value:   vatValue,
					}},
				},
			}
			return nil, validationErr
		}

		// Revert the Logto changes since local update failed
		originalReq := models.UpdateOrganizationRequest{
			Name:        &currentReseller.Name,
			Description: &currentReseller.Description,
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentReseller.CustomData != nil {
			for k, v := range currentReseller.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateOrganization(*currentReseller.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("reseller_id", id).
				Msg("Failed to revert Logto changes after local update failure")
		}

		// Transaction will be rolled back by defer
		return nil, fmt.Errorf("failed to update reseller locally: %w", err)
	}

	// 6. Mark as synced
	err = s.markResellerSynced(id, *reseller.LogtoID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("reseller_id", id).
			Msg("Failed to mark reseller as synced after update")
	}

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Str("updated_by", updatedByUserID).
		Msg("Reseller updated successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()
	return reseller, nil
}

// UpdateCustomer updates a customer locally and syncs to Logto
func (s *LocalOrganizationService) UpdateCustomer(id string, req *models.UpdateLocalCustomerRequest, updatedByUserID, updatedByOrgID string) (*models.LocalCustomer, error) {
	// 1. Get current customer before update to preserve system fields
	currentCustomer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get current customer for update: %w", err)
	}

	// Check if customer is synced to Logto
	if currentCustomer.LogtoID == nil {
		return nil, fmt.Errorf("customer not synced to Logto yet - missing logto_id")
	}

	// 2. Prepare CustomData with proper field management
	// Start with existing custom data to preserve user-defined fields
	finalCustomData := make(map[string]interface{})
	if currentCustomer.CustomData != nil {
		for k, v := range currentCustomer.CustomData {
			finalCustomData[k] = v
		}
	}

	// Merge user-provided custom data (allows users to update their custom fields)
	if req.CustomData != nil {
		for k, v := range *req.CustomData {
			finalCustomData[k] = v
		}
	}

	// Security: Prevent updating organizations to reserved type "owner"
	if customDataType, ok := finalCustomData["type"]; ok {
		if typeStr, ok := customDataType.(string); ok && strings.ToLower(typeStr) == "owner" {
			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.type",
						Message: "organization type 'owner' is reserved and cannot be used",
						Value:   typeStr,
					}},
				},
			}
			return nil, validationErr
		}
	}

	// System fields - these override any user-provided values and are always maintained
	// CRITICAL: Preserve original type and createdBy - never change them
	finalCustomData["type"] = "customer"
	if existingCreatedBy, exists := finalCustomData["createdBy"]; exists {
		finalCustomData["createdBy"] = existingCreatedBy
	} else {
		// Fallback if somehow missing (should not happen)
		finalCustomData["createdBy"] = updatedByOrgID
	}

	// Add update tracking (these are additional fields, not replacements)
	finalCustomData["updatedBy"] = updatedByOrgID
	finalCustomData["updatedAt"] = time.Now().Format(time.RFC3339)

	// 3. Validate changes in Logto FIRST (before consuming local resources)
	updateReq := models.UpdateOrganizationRequest{}
	if req.Name != nil {
		updateReq.Name = req.Name
	}
	if req.Description != nil {
		updateReq.Description = req.Description
	}
	// Use the properly managed customData
	updateReq.CustomData = finalCustomData

	// Try the update in Logto first for validation
	_, err = s.logtoClient.UpdateOrganization(*currentCustomer.LogtoID, updateReq)
	if err != nil {
		logger.Error().
			Err(err).
			Str("customer_id", id).
			Str("customer_name", currentCustomer.Name).
			Msg("Failed to validate customer update in Logto")

		// No rollback needed since we haven't changed anything locally yet
		return nil, fmt.Errorf("failed to validate customer update in Logto: %w", err)
	}

	// 4. Begin transaction for local operations (after Logto validation passes)
	tx, err := database.DB.Begin()
	if err != nil {
		// Revert the Logto changes since we can't proceed with local update
		originalReq := models.UpdateOrganizationRequest{
			Name:        &currentCustomer.Name,
			Description: &currentCustomer.Description,
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentCustomer.CustomData != nil {
			for k, v := range currentCustomer.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateOrganization(*currentCustomer.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("customer_id", id).
				Msg("Failed to revert Logto changes after local transaction failure")
		}

		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Update the request with properly managed customData
	req.CustomData = &finalCustomData

	// 5. Update in local DB (after Logto validation passes)
	customer, err := s.customerRepo.Update(id, req)
	if err != nil {

		// Check for VAT constraint violation (from entities/database)
		if strings.Contains(err.Error(), "already exists") {
			vatValue := ""
			if req.CustomData != nil && (*req.CustomData)["vat"] != nil {
				vatValue = fmt.Sprintf("%v", (*req.CustomData)["vat"])
			}

			// Revert Logto changes before returning validation error
			originalReq := models.UpdateOrganizationRequest{
				Name:        &currentCustomer.Name,
				Description: &currentCustomer.Description,
			}
			// Restore original custom data
			originalCustomData := make(map[string]interface{})
			if currentCustomer.CustomData != nil {
				for k, v := range currentCustomer.CustomData {
					originalCustomData[k] = v
				}
			}
			originalReq.CustomData = originalCustomData

			if _, revertErr := s.logtoClient.UpdateOrganization(*currentCustomer.LogtoID, originalReq); revertErr != nil {
				logger.Warn().
					Err(revertErr).
					Str("customer_id", id).
					Msg("Failed to revert Logto changes after VAT constraint violation")
			}

			validationErr := &ValidationError{
				StatusCode: 400,
				ErrorData: response.ErrorData{
					Type: "validation_error",
					Errors: []response.ValidationError{{
						Key:     "custom_data.vat",
						Message: strings.ToLower(strings.ReplaceAll(err.Error(), "VAT", "vat")),
						Value:   vatValue,
					}},
				},
			}
			return nil, validationErr
		}

		// Revert the Logto changes since local update failed
		originalReq := models.UpdateOrganizationRequest{
			Name:        &currentCustomer.Name,
			Description: &currentCustomer.Description,
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentCustomer.CustomData != nil {
			for k, v := range currentCustomer.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateOrganization(*currentCustomer.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("customer_id", id).
				Msg("Failed to revert Logto changes after local update failure")
		}

		// Transaction will be rolled back by defer
		return nil, fmt.Errorf("failed to update customer locally: %w", err)
	}

	// 6. Mark as synced
	err = s.markCustomerSynced(id, *customer.LogtoID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("customer_id", id).
			Msg("Failed to mark customer as synced after update")
	}

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Str("updated_by", updatedByUserID).
		Msg("Customer updated successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()
	return customer, nil
}

// ============================================
// DELETE OPERATIONS (Soft Delete)
// ============================================

// DeleteDistributor soft-deletes a distributor locally and syncs to Logto
// Returns the count of cascade-deleted systems
func (s *LocalOrganizationService) DeleteDistributor(id, deletedByUserID, deletedByOrgID string) (int, int, error) {
	// Get distributor before deletion for logging and logto_id
	distributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get distributor: %w", err)
	}

	var deletedSystemsCount int
	var deletedUsersCount int

	if distributor.LogtoID != nil && *distributor.LogtoID != "" {
		distLogtoID := *distributor.LogtoID

		// Find child resellers
		childResellerLogtoIDs, err := s.resellerRepo.GetLogtoIDsByCreatedBy(distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to get child reseller logto_ids for cascade deletion")
		}

		// Find child customers (created by distributor OR by child resellers)
		createdByOrgIDs := append([]string{distLogtoID}, childResellerLogtoIDs...)
		childCustomerLogtoIDs, err := s.customerRepo.GetLogtoIDsByCreatedByMultiple(createdByOrgIDs)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to get child customer logto_ids for cascade deletion")
		}

		// All org IDs in the hierarchy
		allOrgIDs := append([]string{distLogtoID}, childResellerLogtoIDs...)
		allOrgIDs = append(allOrgIDs, childCustomerLogtoIDs...)

		// 1. Cascade soft-delete users across the entire hierarchy
		deletedUsersCount, err = s.userRepo.SoftDeleteByMultipleOrgIDs(allOrgIDs, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Int("org_ids_count", len(allOrgIDs)).Msg("Failed to cascade soft-delete users for distributor hierarchy")
		} else if deletedUsersCount > 0 {
			logger.Info().Int("deleted_users", deletedUsersCount).Str("distributor_id", id).Str("distributor_name", distributor.Name).Msg("Cascade soft-deleted users for distributor hierarchy")
		}

		// 2. Cascade soft-delete systems across the entire hierarchy
		deletedSystemsCount, err = s.systemRepo.SoftDeleteSystemsByMultipleOrgIDs(allOrgIDs, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Int("org_ids_count", len(allOrgIDs)).Msg("Failed to cascade soft-delete systems for distributor hierarchy")
		} else if deletedSystemsCount > 0 {
			logger.Info().Int("deleted_systems", deletedSystemsCount).Str("distributor_id", id).Str("distributor_name", distributor.Name).Msg("Cascade soft-deleted systems for distributor hierarchy")
		}
	}

	// 3. Soft delete the distributor locally
	err = s.distributorRepo.Delete(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to delete distributor locally: %w", err)
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Str("deleted_by", deletedByUserID).
		Int("deleted_systems", deletedSystemsCount).
		Int("deleted_users", deletedUsersCount).
		Msg("Distributor soft-deleted successfully")

	refreshUnifiedOrganizationsAsync()
	return deletedSystemsCount, deletedUsersCount, nil
}

// DeleteReseller soft-deletes a reseller locally (no Logto deletion)
// Returns the count of cascade-deleted systems and users
func (s *LocalOrganizationService) DeleteReseller(id, deletedByUserID, deletedByOrgID string) (int, int, error) {
	// Get reseller before deletion for logging and logto_id
	reseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get reseller: %w", err)
	}

	var deletedSystemsCount int
	var deletedUsersCount int

	if reseller.LogtoID != nil && *reseller.LogtoID != "" {
		resLogtoID := *reseller.LogtoID

		// Find child customers
		childCustomerLogtoIDs, err := s.customerRepo.GetLogtoIDsByCreatedByMultiple([]string{resLogtoID})
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to get child customer logto_ids for cascade deletion")
		}

		// All org IDs: reseller + child customers
		allOrgIDs := append([]string{resLogtoID}, childCustomerLogtoIDs...)

		// 1. Cascade soft-delete users across the hierarchy
		deletedUsersCount, err = s.userRepo.SoftDeleteByMultipleOrgIDs(allOrgIDs, resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Int("org_ids_count", len(allOrgIDs)).Msg("Failed to cascade soft-delete users for reseller hierarchy")
		} else if deletedUsersCount > 0 {
			logger.Info().Int("deleted_users", deletedUsersCount).Str("reseller_id", id).Str("reseller_name", reseller.Name).Msg("Cascade soft-deleted users for reseller hierarchy")
		}

		// 2. Cascade soft-delete systems across the hierarchy
		deletedSystemsCount, err = s.systemRepo.SoftDeleteSystemsByMultipleOrgIDs(allOrgIDs, resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Int("org_ids_count", len(allOrgIDs)).Msg("Failed to cascade soft-delete systems for reseller hierarchy")
		} else if deletedSystemsCount > 0 {
			logger.Info().Int("deleted_systems", deletedSystemsCount).Str("reseller_id", id).Str("reseller_name", reseller.Name).Msg("Cascade soft-deleted systems for reseller hierarchy")
		}
	}

	// 3. Soft delete the reseller locally
	err = s.resellerRepo.Delete(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to delete reseller locally: %w", err)
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Str("deleted_by", deletedByUserID).
		Int("deleted_systems", deletedSystemsCount).
		Int("deleted_users", deletedUsersCount).
		Msg("Reseller soft-deleted successfully")

	refreshUnifiedOrganizationsAsync()
	return deletedSystemsCount, deletedUsersCount, nil
}

// DeleteCustomer soft-deletes a customer locally (no Logto deletion)
// Returns the count of cascade-deleted systems and users
func (s *LocalOrganizationService) DeleteCustomer(id, deletedByUserID, deletedByOrgID string) (int, int, error) {
	// Get customer before deletion for logging and logto_id
	customer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get customer: %w", err)
	}

	var deletedSystemsCount int
	var deletedUsersCount int

	if customer.LogtoID != nil && *customer.LogtoID != "" {
		custLogtoID := *customer.LogtoID

		// 1. Cascade soft-delete users for this customer
		deletedUsersCount, err = s.userRepo.SoftDeleteByMultipleOrgIDs([]string{custLogtoID}, custLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("customer_id", id).Msg("Failed to cascade soft-delete users for customer")
		} else if deletedUsersCount > 0 {
			logger.Info().Int("deleted_users", deletedUsersCount).Str("customer_id", id).Str("customer_name", customer.Name).Msg("Cascade soft-deleted users for customer")
		}

		// 2. Cascade soft-delete systems for this customer
		deletedSystemsCount, err = s.systemRepo.SoftDeleteSystemsByMultipleOrgIDs([]string{custLogtoID}, custLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("customer_id", id).Msg("Failed to cascade soft-delete systems for customer")
		} else if deletedSystemsCount > 0 {
			logger.Info().Int("deleted_systems", deletedSystemsCount).Str("customer_id", id).Str("customer_name", customer.Name).Msg("Cascade soft-deleted systems for customer")
		}
	}

	// 3. Soft delete the customer locally
	err = s.customerRepo.Delete(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to delete customer locally: %w", err)
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Str("deleted_by", deletedByUserID).
		Int("deleted_systems", deletedSystemsCount).
		Int("deleted_users", deletedUsersCount).
		Msg("Customer soft-deleted successfully")

	refreshUnifiedOrganizationsAsync()
	return deletedSystemsCount, deletedUsersCount, nil
}

// ============================================
// RESTORE OPERATIONS
// ============================================

// RestoreDistributor restores a soft-deleted distributor and cascade-restores users and systems
func (s *LocalOrganizationService) RestoreDistributor(id, restoredByUserID, restoredByOrgID string) (int, int, error) {
	// Get distributor including deleted
	distributor, err := s.distributorRepo.GetByIDIncludeDeleted(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get distributor: %w", err)
	}

	if distributor.DeletedAt == nil {
		return 0, 0, fmt.Errorf("distributor is not deleted")
	}

	// Restore the distributor
	if distributor.LogtoID != nil {
		err = s.distributorRepo.Restore(*distributor.LogtoID)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to restore distributor: %w", err)
		}
	}

	var restoredSystemsCount int
	var restoredUsersCount int

	if distributor.LogtoID != nil && *distributor.LogtoID != "" {
		distLogtoID := *distributor.LogtoID

		// Cascade restore users soft-deleted by this distributor
		restoredUsersCount, err = s.userRepo.RestoreByDeletedByOrgID(distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade restore users for distributor")
		}

		// Cascade restore systems soft-deleted by this distributor
		restoredSystemsCount, err = s.systemRepo.RestoreSystemsByDeletedByOrgID(distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade restore systems for distributor")
		}
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Str("restored_by", restoredByUserID).
		Int("restored_systems", restoredSystemsCount).
		Int("restored_users", restoredUsersCount).
		Msg("Distributor restored successfully")

	refreshUnifiedOrganizationsAsync()
	return restoredSystemsCount, restoredUsersCount, nil
}

// RestoreReseller restores a soft-deleted reseller and cascade-restores users and systems
func (s *LocalOrganizationService) RestoreReseller(id, restoredByUserID, restoredByOrgID string) (int, int, error) {
	// Get reseller including deleted
	reseller, err := s.resellerRepo.GetByIDIncludeDeleted(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get reseller: %w", err)
	}

	if reseller.DeletedAt == nil {
		return 0, 0, fmt.Errorf("reseller is not deleted")
	}

	// Restore the reseller
	if reseller.LogtoID != nil {
		err = s.resellerRepo.Restore(*reseller.LogtoID)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to restore reseller: %w", err)
		}
	}

	var restoredSystemsCount int
	var restoredUsersCount int

	if reseller.LogtoID != nil && *reseller.LogtoID != "" {
		resLogtoID := *reseller.LogtoID

		// Cascade restore users soft-deleted by this reseller
		restoredUsersCount, err = s.userRepo.RestoreByDeletedByOrgID(resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade restore users for reseller")
		}

		// Cascade restore systems soft-deleted by this reseller
		restoredSystemsCount, err = s.systemRepo.RestoreSystemsByDeletedByOrgID(resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade restore systems for reseller")
		}
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Str("restored_by", restoredByUserID).
		Int("restored_systems", restoredSystemsCount).
		Int("restored_users", restoredUsersCount).
		Msg("Reseller restored successfully")

	refreshUnifiedOrganizationsAsync()
	return restoredSystemsCount, restoredUsersCount, nil
}

// RestoreCustomer restores a soft-deleted customer and cascade-restores users and systems
func (s *LocalOrganizationService) RestoreCustomer(id, restoredByUserID, restoredByOrgID string) (int, int, error) {
	// Get customer including deleted
	customer, err := s.customerRepo.GetByIDIncludeDeleted(id)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get customer: %w", err)
	}

	if customer.DeletedAt == nil {
		return 0, 0, fmt.Errorf("customer is not deleted")
	}

	// Restore the customer
	if customer.LogtoID != nil {
		err = s.customerRepo.Restore(*customer.LogtoID)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to restore customer: %w", err)
		}
	}

	var restoredSystemsCount int
	var restoredUsersCount int

	if customer.LogtoID != nil && *customer.LogtoID != "" {
		custLogtoID := *customer.LogtoID

		// Cascade restore users soft-deleted by this customer
		restoredUsersCount, err = s.userRepo.RestoreByDeletedByOrgID(custLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("customer_id", id).Msg("Failed to cascade restore users for customer")
		}

		// Cascade restore systems soft-deleted by this customer
		restoredSystemsCount, err = s.systemRepo.RestoreSystemsByDeletedByOrgID(custLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("customer_id", id).Msg("Failed to cascade restore systems for customer")
		}
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Str("restored_by", restoredByUserID).
		Int("restored_systems", restoredSystemsCount).
		Int("restored_users", restoredUsersCount).
		Msg("Customer restored successfully")

	refreshUnifiedOrganizationsAsync()
	return restoredSystemsCount, restoredUsersCount, nil
}

// ============================================
// DESTROY OPERATIONS (Permanent Delete)
// ============================================

// DestroyDistributor permanently deletes a distributor and its entire hierarchy from DB and Logto
func (s *LocalOrganizationService) DestroyDistributor(id, destroyedByUserID, destroyedByOrgID string) error {
	// Get distributor including deleted
	distributor, err := s.distributorRepo.GetByIDIncludeDeleted(id)
	if err != nil {
		return fmt.Errorf("failed to get distributor: %w", err)
	}

	if distributor.LogtoID == nil || *distributor.LogtoID == "" {
		return fmt.Errorf("distributor has no logto_id")
	}

	distLogtoID := *distributor.LogtoID

	// Find child resellers
	childResellerLogtoIDs, err := s.resellerRepo.GetLogtoIDsByCreatedBy(distLogtoID)
	if err != nil {
		logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to get child reseller logto_ids for destroy")
	}

	// Find child customers (created by distributor OR by child resellers)
	createdByOrgIDs := append([]string{distLogtoID}, childResellerLogtoIDs...)
	childCustomerLogtoIDs, err := s.customerRepo.GetLogtoIDsByCreatedByMultiple(createdByOrgIDs)
	if err != nil {
		logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to get child customer logto_ids for destroy")
	}

	// All org IDs in the hierarchy
	allOrgIDs := append([]string{distLogtoID}, childResellerLogtoIDs...)
	allOrgIDs = append(allOrgIDs, childCustomerLogtoIDs...)

	// 1. Hard-delete users from DB and Logto for each org
	for _, orgID := range allOrgIDs {
		userLogtoIDs, err := s.userRepo.HardDeleteByOrgID(orgID)
		if err != nil {
			logger.Warn().Err(err).Str("org_id", orgID).Msg("Failed to hard-delete users from DB")
		}
		for _, userLogtoID := range userLogtoIDs {
			if err := s.logtoClient.DeleteUser(userLogtoID); err != nil {
				logger.Warn().Err(err).Str("user_logto_id", userLogtoID).Msg("Failed to delete user from Logto")
			}
		}
	}

	// 2. Hard-delete systems from DB for entire hierarchy
	_, err = s.systemRepo.HardDeleteByMultipleOrgIDs(allOrgIDs)
	if err != nil {
		logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to hard-delete systems from DB")
	}

	// 3. Hard-delete child orgs bottom-up: customers first, then resellers
	for _, custLogtoID := range childCustomerLogtoIDs {
		if err := s.customerRepo.HardDelete(custLogtoID); err != nil {
			logger.Warn().Err(err).Str("customer_logto_id", custLogtoID).Msg("Failed to hard-delete customer from DB")
		}
		if err := s.logtoClient.DeleteOrganization(custLogtoID); err != nil {
			logger.Warn().Err(err).Str("customer_logto_id", custLogtoID).Msg("Failed to delete customer from Logto")
		}
	}
	for _, resLogtoID := range childResellerLogtoIDs {
		if err := s.resellerRepo.HardDelete(resLogtoID); err != nil {
			logger.Warn().Err(err).Str("reseller_logto_id", resLogtoID).Msg("Failed to hard-delete reseller from DB")
		}
		if err := s.logtoClient.DeleteOrganization(resLogtoID); err != nil {
			logger.Warn().Err(err).Str("reseller_logto_id", resLogtoID).Msg("Failed to delete reseller from Logto")
		}
	}

	// 4. Hard-delete the distributor itself
	if err := s.distributorRepo.HardDelete(distLogtoID); err != nil {
		return fmt.Errorf("failed to hard-delete distributor from DB: %w", err)
	}
	if err := s.logtoClient.DeleteOrganization(distLogtoID); err != nil {
		logger.Warn().Err(err).Str("distributor_logto_id", distLogtoID).Msg("Failed to delete distributor from Logto")
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Str("destroyed_by", destroyedByUserID).
		Int("child_resellers", len(childResellerLogtoIDs)).
		Int("child_customers", len(childCustomerLogtoIDs)).
		Msg("Distributor permanently destroyed with entire hierarchy")

	refreshUnifiedOrganizationsAsync()
	return nil
}

// DestroyReseller permanently deletes a reseller and its child hierarchy from DB and Logto
func (s *LocalOrganizationService) DestroyReseller(id, destroyedByUserID, destroyedByOrgID string) error {
	// Get reseller including deleted
	reseller, err := s.resellerRepo.GetByIDIncludeDeleted(id)
	if err != nil {
		return fmt.Errorf("failed to get reseller: %w", err)
	}

	if reseller.LogtoID == nil || *reseller.LogtoID == "" {
		return fmt.Errorf("reseller has no logto_id")
	}

	resLogtoID := *reseller.LogtoID

	// Find child customers
	childCustomerLogtoIDs, err := s.customerRepo.GetLogtoIDsByCreatedByMultiple([]string{resLogtoID})
	if err != nil {
		logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to get child customer logto_ids for destroy")
	}

	// All org IDs: reseller + child customers
	allOrgIDs := append([]string{resLogtoID}, childCustomerLogtoIDs...)

	// 1. Hard-delete users from DB and Logto for each org
	for _, orgID := range allOrgIDs {
		userLogtoIDs, err := s.userRepo.HardDeleteByOrgID(orgID)
		if err != nil {
			logger.Warn().Err(err).Str("org_id", orgID).Msg("Failed to hard-delete users from DB")
		}
		for _, userLogtoID := range userLogtoIDs {
			if err := s.logtoClient.DeleteUser(userLogtoID); err != nil {
				logger.Warn().Err(err).Str("user_logto_id", userLogtoID).Msg("Failed to delete user from Logto")
			}
		}
	}

	// 2. Hard-delete systems from DB for entire hierarchy
	_, err = s.systemRepo.HardDeleteByMultipleOrgIDs(allOrgIDs)
	if err != nil {
		logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to hard-delete systems from DB")
	}

	// 3. Hard-delete child customers bottom-up
	for _, custLogtoID := range childCustomerLogtoIDs {
		if err := s.customerRepo.HardDelete(custLogtoID); err != nil {
			logger.Warn().Err(err).Str("customer_logto_id", custLogtoID).Msg("Failed to hard-delete customer from DB")
		}
		if err := s.logtoClient.DeleteOrganization(custLogtoID); err != nil {
			logger.Warn().Err(err).Str("customer_logto_id", custLogtoID).Msg("Failed to delete customer from Logto")
		}
	}

	// 4. Hard-delete the reseller itself
	if err := s.resellerRepo.HardDelete(resLogtoID); err != nil {
		return fmt.Errorf("failed to hard-delete reseller from DB: %w", err)
	}
	if err := s.logtoClient.DeleteOrganization(resLogtoID); err != nil {
		logger.Warn().Err(err).Str("reseller_logto_id", resLogtoID).Msg("Failed to delete reseller from Logto")
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Str("destroyed_by", destroyedByUserID).
		Int("child_customers", len(childCustomerLogtoIDs)).
		Msg("Reseller permanently destroyed with child hierarchy")

	refreshUnifiedOrganizationsAsync()
	return nil
}

// DestroyCustomer permanently deletes a customer from DB and Logto
func (s *LocalOrganizationService) DestroyCustomer(id, destroyedByUserID, destroyedByOrgID string) error {
	// Get customer including deleted
	customer, err := s.customerRepo.GetByIDIncludeDeleted(id)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	if customer.LogtoID == nil || *customer.LogtoID == "" {
		return fmt.Errorf("customer has no logto_id")
	}

	custLogtoID := *customer.LogtoID

	// 1. Hard-delete users from DB and Logto
	userLogtoIDs, err := s.userRepo.HardDeleteByOrgID(custLogtoID)
	if err != nil {
		logger.Warn().Err(err).Str("customer_id", id).Msg("Failed to hard-delete users from DB")
	}
	for _, userLogtoID := range userLogtoIDs {
		if err := s.logtoClient.DeleteUser(userLogtoID); err != nil {
			logger.Warn().Err(err).Str("user_logto_id", userLogtoID).Msg("Failed to delete user from Logto")
		}
	}

	// 2. Hard-delete systems from DB
	_, err = s.systemRepo.HardDeleteByOrgID(custLogtoID)
	if err != nil {
		logger.Warn().Err(err).Str("customer_id", id).Msg("Failed to hard-delete systems from DB")
	}

	// 3. Hard-delete the customer itself
	if err := s.customerRepo.HardDelete(custLogtoID); err != nil {
		return fmt.Errorf("failed to hard-delete customer from DB: %w", err)
	}
	if err := s.logtoClient.DeleteOrganization(custLogtoID); err != nil {
		logger.Warn().Err(err).Str("customer_logto_id", custLogtoID).Msg("Failed to delete customer from Logto")
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Str("destroyed_by", destroyedByUserID).
		Msg("Customer permanently destroyed")

	refreshUnifiedOrganizationsAsync()
	return nil
}

// ============================================
// AGGREGATED ORGANIZATION OPERATIONS
// ============================================

// GetAllOrganizationsPaginated returns all organizations (distributors + resellers + customers) with pagination
// This replaces the Logto API call with local database for better performance
func (s *LocalOrganizationService) GetAllOrganizationsPaginated(userOrgRole, userOrgID string, page, pageSize int, filters models.OrganizationFilters) (*models.PaginatedOrganizations, error) {
	var allOrganizations []models.LogtoOrganization

	// For aggregated results, we need to get more data to apply filters and pagination correctly
	// We'll get a reasonable amount and then paginate after filtering
	fetchSize := pageSize * 10 // Get more data to account for filtering

	// Add own organization first (if user is distributor/reseller/customer)
	if userOrgRole != "owner" && userOrgID != "" {
		ownOrg, err := s.getUserOwnOrganization(userOrgRole, userOrgID)
		if err == nil && ownOrg != nil {
			allOrganizations = append(allOrganizations, *ownOrg)
		}
	}

	// Fetch distributors
	distributors, _, err := s.distributorRepo.List(userOrgRole, userOrgID, 1, fetchSize, "", "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get distributors: %w", err)
	}
	for _, d := range distributors {
		logtoID := ""
		if d.LogtoID != nil {
			logtoID = *d.LogtoID
		}

		allOrganizations = append(allOrganizations, models.LogtoOrganization{
			ID:          logtoID,
			Name:        d.Name,
			Description: d.Description,
			CustomData: map[string]interface{}{
				"type":        "distributor",
				"database_id": d.ID,
			},
		})
	}

	// Fetch resellers
	resellers, _, err := s.resellerRepo.List(userOrgRole, userOrgID, 1, fetchSize, "", "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get resellers: %w", err)
	}
	for _, r := range resellers {
		logtoID := ""
		if r.LogtoID != nil {
			logtoID = *r.LogtoID
		}

		customData := map[string]interface{}{
			"type":        "reseller",
			"database_id": r.ID,
		}
		// Preserve other custom data fields
		if r.CustomData != nil {
			for k, v := range r.CustomData {
				if k != "type" && k != "database_id" {
					customData[k] = v
				}
			}
		}
		allOrganizations = append(allOrganizations, models.LogtoOrganization{
			ID:          logtoID,
			Name:        r.Name,
			Description: r.Description,
			CustomData:  customData,
		})
	}

	// Fetch customers
	customers, _, err := s.customerRepo.List(userOrgRole, userOrgID, 1, fetchSize, "", "", "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}
	for _, c := range customers {
		logtoID := ""
		if c.LogtoID != nil {
			logtoID = *c.LogtoID
		}

		customData := map[string]interface{}{
			"type":        "customer",
			"database_id": c.ID,
		}
		// Preserve other custom data fields
		if c.CustomData != nil {
			for k, v := range c.CustomData {
				if k != "type" && k != "database_id" {
					customData[k] = v
				}
			}
		}

		allOrganizations = append(allOrganizations, models.LogtoOrganization{
			ID:          logtoID,
			Name:        c.Name,
			Description: c.Description,
			CustomData:  customData,
		})
	}

	// Apply client-side filters
	filteredOrgs := s.applyLocalFilters(allOrganizations, filters)

	// Apply pagination
	totalCount := len(filteredOrgs)
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize

	if startIndex >= totalCount {
		filteredOrgs = []models.LogtoOrganization{}
	} else if endIndex > totalCount {
		filteredOrgs = filteredOrgs[startIndex:]
	} else {
		filteredOrgs = filteredOrgs[startIndex:endIndex]
	}

	totalPages := (totalCount + pageSize - 1) / pageSize

	paginationInfo := models.PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	if paginationInfo.HasNext {
		nextPage := page + 1
		paginationInfo.NextPage = &nextPage
	}

	if paginationInfo.HasPrev {
		prevPage := page - 1
		paginationInfo.PrevPage = &prevPage
	}

	return &models.PaginatedOrganizations{
		Data:       filteredOrgs,
		Pagination: paginationInfo,
	}, nil
}

// applyLocalFilters applies filters to local organization data
func (s *LocalOrganizationService) applyLocalFilters(orgs []models.LogtoOrganization, filters models.OrganizationFilters) []models.LogtoOrganization {
	if filters.Search == "" && filters.Name == "" && filters.Description == "" && filters.Type == "" && filters.CreatedBy == "" {
		return orgs
	}

	var filtered []models.LogtoOrganization
	for _, org := range orgs {
		// Search filter (matches name or description)
		if filters.Search != "" {
			searchLower := strings.ToLower(filters.Search)
			if !strings.Contains(strings.ToLower(org.Name), searchLower) &&
				!strings.Contains(strings.ToLower(org.Description), searchLower) {
				continue
			}
		}

		// Name filter (exact match)
		if filters.Name != "" && org.Name != filters.Name {
			continue
		}

		// Description filter (exact match)
		if filters.Description != "" && org.Description != filters.Description {
			continue
		}

		// Type filter
		if filters.Type != "" {
			if org.CustomData == nil {
				continue
			}
			if orgType, ok := org.CustomData["type"].(string); !ok || orgType != filters.Type {
				continue
			}
		}

		// CreatedBy filter
		if filters.CreatedBy != "" {
			if org.CustomData == nil {
				continue
			}
			matched := false
			if distributorId, ok := org.CustomData["distributorId"].(string); ok && distributorId == filters.CreatedBy {
				matched = true
			}
			if resellerId, ok := org.CustomData["resellerId"].(string); ok && resellerId == filters.CreatedBy {
				matched = true
			}
			if !matched {
				continue
			}
		}

		filtered = append(filtered, org)
	}

	return filtered
}

// configureOrganizationJitRoles configures JIT roles for an organization based on its type
func (s *LocalOrganizationService) configureOrganizationJitRoles(logtoOrgID, orgType string) error {
	// Map organization type to organization role name
	var roleName string
	switch orgType {
	case "distributor":
		roleName = "Distributor"
	case "reseller":
		roleName = "Reseller"
	case "customer":
		roleName = "Customer"
	default:
		return fmt.Errorf("unknown organization type: %s", orgType)
	}

	// Get the organization role ID by name
	orgRole, err := s.logtoClient.GetOrganizationRoleByName(roleName)
	if err != nil {
		return fmt.Errorf("failed to get organization role '%s': %w", roleName, err)
	}

	// Configure JIT roles for the organization
	err = s.logtoClient.SetOrganizationJitRoles(logtoOrgID, []string{orgRole.ID})
	if err != nil {
		return fmt.Errorf("failed to set JIT roles for organization: %w", err)
	}

	logger.Info().
		Str("logto_org_id", logtoOrgID).
		Str("org_type", orgType).
		Str("role_name", roleName).
		Str("role_id", orgRole.ID).
		Msg("Successfully configured JIT roles for organization")

	return nil
}

// getUserOwnOrganization returns the user's own organization based on their role and org ID
func (s *LocalOrganizationService) getUserOwnOrganization(userOrgRole, userOrgID string) (*models.LogtoOrganization, error) {
	switch userOrgRole {
	case "distributor":
		// Search for distributor by logto_id (since userOrgID is the logto_id from JWT)
		var distributor *models.LocalDistributor
		query := `SELECT id, logto_id, name, description, custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
		          FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL LIMIT 1`
		row := database.DB.QueryRow(query, userOrgID)

		distributor = &models.LocalDistributor{}
		var customDataJSON []byte
		err := row.Scan(&distributor.ID, &distributor.LogtoID, &distributor.Name, &distributor.Description,
			&customDataJSON, &distributor.CreatedAt, &distributor.UpdatedAt,
			&distributor.LogtoSyncedAt, &distributor.LogtoSyncError, &distributor.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to get distributor organization: %w", err)
		}

		return &models.LogtoOrganization{
			ID:          userOrgID,
			Name:        distributor.Name,
			Description: distributor.Description,
			CustomData: map[string]interface{}{
				"type":        "distributor",
				"database_id": distributor.ID,
			},
		}, nil

	case "reseller":
		// Search for reseller by logto_id (since userOrgID is the logto_id from JWT)
		var reseller *models.LocalReseller
		query := `SELECT id, logto_id, name, description, custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
		          FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL LIMIT 1`
		row := database.DB.QueryRow(query, userOrgID)

		reseller = &models.LocalReseller{}
		var customDataJSON []byte
		err := row.Scan(&reseller.ID, &reseller.LogtoID, &reseller.Name, &reseller.Description,
			&customDataJSON, &reseller.CreatedAt, &reseller.UpdatedAt,
			&reseller.LogtoSyncedAt, &reseller.LogtoSyncError, &reseller.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to get reseller organization: %w", err)
		}

		// Parse custom data
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &reseller.CustomData); err != nil {
				reseller.CustomData = make(map[string]interface{})
			}
		}

		customData := map[string]interface{}{
			"type":        "reseller",
			"database_id": reseller.ID,
		}
		// Preserve other custom data fields
		if reseller.CustomData != nil {
			for k, v := range reseller.CustomData {
				if k != "type" && k != "database_id" {
					customData[k] = v
				}
			}
		}

		return &models.LogtoOrganization{
			ID:          userOrgID,
			Name:        reseller.Name,
			Description: reseller.Description,
			CustomData:  customData,
		}, nil

	case "customer":
		// Search for customer by logto_id (since userOrgID is the logto_id from JWT)
		var customer *models.LocalCustomer
		query := `SELECT id, logto_id, name, description, custom_data, created_at, updated_at, logto_synced_at, logto_sync_error, deleted_at
		          FROM customers WHERE logto_id = $1 AND deleted_at IS NULL LIMIT 1`
		row := database.DB.QueryRow(query, userOrgID)

		customer = &models.LocalCustomer{}
		var customDataJSON []byte
		err := row.Scan(&customer.ID, &customer.LogtoID, &customer.Name, &customer.Description,
			&customDataJSON, &customer.CreatedAt, &customer.UpdatedAt,
			&customer.LogtoSyncedAt, &customer.LogtoSyncError, &customer.DeletedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to get customer organization: %w", err)
		}

		// Parse custom data
		if len(customDataJSON) > 0 {
			if err := json.Unmarshal(customDataJSON, &customer.CustomData); err != nil {
				customer.CustomData = make(map[string]interface{})
			}
		}

		customData := map[string]interface{}{
			"type":        "customer",
			"database_id": customer.ID,
		}
		// Preserve other custom data fields
		if customer.CustomData != nil {
			for k, v := range customer.CustomData {
				if k != "type" && k != "database_id" {
					customData[k] = v
				}
			}
		}

		return &models.LogtoOrganization{
			ID:          userOrgID,
			Name:        customer.Name,
			Description: customer.Description,
			CustomData:  customData,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported user organization role: %s", userOrgRole)
	}
}

// SuspendDistributor suspends a distributor and all its users and systems
func (s *LocalOrganizationService) SuspendDistributor(id, suspendedByUserID, suspendedByOrgID string) (*models.LocalDistributor, int, int, int, int, error) {
	// Get distributor to verify it exists and get logto_id
	distributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("failed to get distributor: %w", err)
	}

	// Suspend the distributor locally
	err = s.distributorRepo.Suspend(id)
	if err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("failed to suspend distributor: %w", err)
	}

	suspendedResellersCount := 0
	suspendedCustomersCount := 0
	suspendedUsersCount := 0
	suspendedSystemsCount := 0

	if distributor.LogtoID != nil && *distributor.LogtoID != "" {
		distLogtoID := *distributor.LogtoID

		// 1. Suspend child resellers (created by this distributor)
		childResellerLogtoIDs, resellersCount, err := s.resellerRepo.SuspendByCreatedBy(distLogtoID, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade suspend resellers")
		}
		suspendedResellersCount = resellersCount

		// 2. Suspend child customers (created by distributor OR by child resellers)
		createdByOrgIDs := []string{distLogtoID}
		createdByOrgIDs = append(createdByOrgIDs, childResellerLogtoIDs...)
		childCustomerLogtoIDs, customersCount, err := s.customerRepo.SuspendByCreatedByMultiple(createdByOrgIDs, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade suspend customers")
		}
		suspendedCustomersCount = customersCount

		// 3. Collect ALL org IDs involved (distributor + resellers + customers) for user/system suspension
		allOrgIDs := []string{distLogtoID}
		allOrgIDs = append(allOrgIDs, childResellerLogtoIDs...)
		allOrgIDs = append(allOrgIDs, childCustomerLogtoIDs...)

		// 4. Cascade suspend users across all orgs
		users, usersCount, err := s.userRepo.SuspendUsersByMultipleOrgIDs(allOrgIDs, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade suspend users")
		}
		suspendedUsersCount = usersCount

		// Sync user suspensions to Logto
		for _, user := range users {
			if user.LogtoID != nil && *user.LogtoID != "" {
				if syncErr := s.logtoClient.SuspendUser(*user.LogtoID); syncErr != nil {
					logger.Warn().Err(syncErr).Str("user_id", user.ID).Msg("Failed to suspend user in Logto")
				}
			}
		}

		// 5. Cascade suspend systems across all orgs
		systemsCount, err := s.systemRepo.SuspendSystemsByMultipleOrgIDs(allOrgIDs, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade suspend systems")
		}
		suspendedSystemsCount = systemsCount
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Int("suspended_resellers_count", suspendedResellersCount).
		Int("suspended_customers_count", suspendedCustomersCount).
		Int("suspended_users_count", suspendedUsersCount).
		Int("suspended_systems_count", suspendedSystemsCount).
		Str("suspended_by_user_id", suspendedByUserID).
		Str("suspended_by_org_id", suspendedByOrgID).
		Msg("Distributor suspended successfully with full hierarchy cascade")

	// Return updated distributor
	updatedDistributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return nil, suspendedResellersCount, suspendedCustomersCount, suspendedUsersCount, suspendedSystemsCount, fmt.Errorf("failed to get updated distributor: %w", err)
	}
	return updatedDistributor, suspendedResellersCount, suspendedCustomersCount, suspendedUsersCount, suspendedSystemsCount, nil
}

// ReactivateDistributor reactivates a distributor and all entities cascade-suspended by it
func (s *LocalOrganizationService) ReactivateDistributor(id, reactivatedByUserID, reactivatedByOrgID string) (*models.LocalDistributor, int, int, int, int, error) {
	// Get distributor to verify it exists and get logto_id
	distributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("failed to get distributor: %w", err)
	}

	// Reactivate the distributor locally
	err = s.distributorRepo.Reactivate(id)
	if err != nil {
		return nil, 0, 0, 0, 0, fmt.Errorf("failed to reactivate distributor: %w", err)
	}

	reactivatedResellersCount := 0
	reactivatedCustomersCount := 0
	reactivatedUsersCount := 0
	reactivatedSystemsCount := 0

	if distributor.LogtoID != nil && *distributor.LogtoID != "" {
		distLogtoID := *distributor.LogtoID

		// Reactivate all entities with suspended_by_org_id = distributor.logto_id
		resellersCount, err := s.resellerRepo.ReactivateBySuspendedByOrgID(distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade reactivate resellers")
		}
		reactivatedResellersCount = resellersCount

		customersCount, err := s.customerRepo.ReactivateBySuspendedByOrgID(distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade reactivate customers")
		}
		reactivatedCustomersCount = customersCount

		reactivatedUsersCount, err = s.cascadeReactivateUsers(distLogtoID, "distributor", distributor.Name)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade reactivate users")
		}

		reactivatedSystemsCount, err = s.cascadeReactivateSystems(distLogtoID, "distributor", distributor.Name)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade reactivate systems")
		}
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Int("reactivated_resellers_count", reactivatedResellersCount).
		Int("reactivated_customers_count", reactivatedCustomersCount).
		Int("reactivated_users_count", reactivatedUsersCount).
		Int("reactivated_systems_count", reactivatedSystemsCount).
		Str("reactivated_by_user_id", reactivatedByUserID).
		Str("reactivated_by_org_id", reactivatedByOrgID).
		Msg("Distributor reactivated successfully with full hierarchy cascade")

	// Return updated distributor
	updatedDistributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return nil, reactivatedResellersCount, reactivatedCustomersCount, reactivatedUsersCount, reactivatedSystemsCount, fmt.Errorf("failed to get updated distributor: %w", err)
	}
	return updatedDistributor, reactivatedResellersCount, reactivatedCustomersCount, reactivatedUsersCount, reactivatedSystemsCount, nil
}

// SuspendReseller suspends a reseller and all its child customers, users and systems
func (s *LocalOrganizationService) SuspendReseller(id, suspendedByUserID, suspendedByOrgID string) (*models.LocalReseller, int, int, int, error) {
	// Get reseller to verify it exists and get logto_id
	reseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to get reseller: %w", err)
	}

	// Suspend the reseller locally
	err = s.resellerRepo.Suspend(id)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to suspend reseller: %w", err)
	}

	suspendedCustomersCount := 0
	suspendedUsersCount := 0
	suspendedSystemsCount := 0

	if reseller.LogtoID != nil && *reseller.LogtoID != "" {
		resLogtoID := *reseller.LogtoID

		// 1. Suspend child customers (created by this reseller)
		childCustomerLogtoIDs, customersCount, err := s.customerRepo.SuspendByCreatedByMultiple([]string{resLogtoID}, resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade suspend customers")
		}
		suspendedCustomersCount = customersCount

		// 2. Collect ALL org IDs (reseller + customers) for user/system suspension
		allOrgIDs := []string{resLogtoID}
		allOrgIDs = append(allOrgIDs, childCustomerLogtoIDs...)

		// 3. Cascade suspend users across all orgs
		users, usersCount, err := s.userRepo.SuspendUsersByMultipleOrgIDs(allOrgIDs, resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade suspend users")
		}
		suspendedUsersCount = usersCount

		// Sync user suspensions to Logto
		for _, user := range users {
			if user.LogtoID != nil && *user.LogtoID != "" {
				if syncErr := s.logtoClient.SuspendUser(*user.LogtoID); syncErr != nil {
					logger.Warn().Err(syncErr).Str("user_id", user.ID).Msg("Failed to suspend user in Logto")
				}
			}
		}

		// 4. Cascade suspend systems across all orgs
		systemsCount, err := s.systemRepo.SuspendSystemsByMultipleOrgIDs(allOrgIDs, resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade suspend systems")
		}
		suspendedSystemsCount = systemsCount
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Int("suspended_customers_count", suspendedCustomersCount).
		Int("suspended_users_count", suspendedUsersCount).
		Int("suspended_systems_count", suspendedSystemsCount).
		Str("suspended_by_user_id", suspendedByUserID).
		Str("suspended_by_org_id", suspendedByOrgID).
		Msg("Reseller suspended successfully with hierarchy cascade")

	// Return updated reseller
	updatedReseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return nil, suspendedCustomersCount, suspendedUsersCount, suspendedSystemsCount, fmt.Errorf("failed to get updated reseller: %w", err)
	}
	return updatedReseller, suspendedCustomersCount, suspendedUsersCount, suspendedSystemsCount, nil
}

// ReactivateReseller reactivates a reseller and all entities cascade-suspended by it
func (s *LocalOrganizationService) ReactivateReseller(id, reactivatedByUserID, reactivatedByOrgID string) (*models.LocalReseller, int, int, int, error) {
	// Get reseller to verify it exists and get logto_id
	reseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to get reseller: %w", err)
	}

	// Reactivate the reseller locally
	err = s.resellerRepo.Reactivate(id)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to reactivate reseller: %w", err)
	}

	reactivatedCustomersCount := 0
	reactivatedUsersCount := 0
	reactivatedSystemsCount := 0

	if reseller.LogtoID != nil && *reseller.LogtoID != "" {
		resLogtoID := *reseller.LogtoID

		// Reactivate all entities with suspended_by_org_id = reseller.logto_id
		customersCount, err := s.customerRepo.ReactivateBySuspendedByOrgID(resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade reactivate customers")
		}
		reactivatedCustomersCount = customersCount

		reactivatedUsersCount, err = s.cascadeReactivateUsers(resLogtoID, "reseller", reseller.Name)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade reactivate users")
		}

		reactivatedSystemsCount, err = s.cascadeReactivateSystems(resLogtoID, "reseller", reseller.Name)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade reactivate systems")
		}
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Int("reactivated_customers_count", reactivatedCustomersCount).
		Int("reactivated_users_count", reactivatedUsersCount).
		Int("reactivated_systems_count", reactivatedSystemsCount).
		Str("reactivated_by_user_id", reactivatedByUserID).
		Str("reactivated_by_org_id", reactivatedByOrgID).
		Msg("Reseller reactivated successfully with hierarchy cascade")

	// Return updated reseller
	updatedReseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return nil, reactivatedCustomersCount, reactivatedUsersCount, reactivatedSystemsCount, fmt.Errorf("failed to get updated reseller: %w", err)
	}
	return updatedReseller, reactivatedCustomersCount, reactivatedUsersCount, reactivatedSystemsCount, nil
}

// SuspendCustomer suspends a customer and all its users and systems
func (s *LocalOrganizationService) SuspendCustomer(id, suspendedByUserID, suspendedByOrgID string) (*models.LocalCustomer, int, int, error) {
	// Get customer to verify it exists and get logto_id
	customer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get customer: %w", err)
	}

	// Suspend the customer locally
	err = s.customerRepo.Suspend(id)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to suspend customer: %w", err)
	}

	// Cascade suspend users and systems if customer has logto_id
	suspendedUsersCount := 0
	suspendedSystemsCount := 0
	if customer.LogtoID != nil && *customer.LogtoID != "" {
		suspendedUsersCount, err = s.cascadeSuspendUsers(*customer.LogtoID, "customer", customer.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("customer_id", id).
				Str("logto_id", *customer.LogtoID).
				Msg("Failed to cascade suspend users for customer")
		}
		suspendedSystemsCount, err = s.cascadeSuspendSystems(*customer.LogtoID, "customer", customer.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("customer_id", id).
				Str("logto_id", *customer.LogtoID).
				Msg("Failed to cascade suspend systems for customer")
		}
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Int("suspended_users_count", suspendedUsersCount).
		Int("suspended_systems_count", suspendedSystemsCount).
		Str("suspended_by_user_id", suspendedByUserID).
		Str("suspended_by_org_id", suspendedByOrgID).
		Msg("Customer suspended successfully")

	// Return updated customer
	updatedCustomer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return nil, suspendedUsersCount, suspendedSystemsCount, fmt.Errorf("failed to get updated customer: %w", err)
	}
	return updatedCustomer, suspendedUsersCount, suspendedSystemsCount, nil
}

// ReactivateCustomer reactivates a customer and all its cascade-suspended users and systems
func (s *LocalOrganizationService) ReactivateCustomer(id, reactivatedByUserID, reactivatedByOrgID string) (*models.LocalCustomer, int, int, error) {
	// Get customer to verify it exists and get logto_id
	customer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to get customer: %w", err)
	}

	// Reactivate the customer locally
	err = s.customerRepo.Reactivate(id)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to reactivate customer: %w", err)
	}

	// Cascade reactivate users and systems if customer has logto_id
	reactivatedUsersCount := 0
	reactivatedSystemsCount := 0
	if customer.LogtoID != nil && *customer.LogtoID != "" {
		reactivatedUsersCount, err = s.cascadeReactivateUsers(*customer.LogtoID, "customer", customer.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("customer_id", id).
				Str("logto_id", *customer.LogtoID).
				Msg("Failed to cascade reactivate users for customer")
		}
		reactivatedSystemsCount, err = s.cascadeReactivateSystems(*customer.LogtoID, "customer", customer.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("customer_id", id).
				Str("logto_id", *customer.LogtoID).
				Msg("Failed to cascade reactivate systems for customer")
		}
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Int("reactivated_users_count", reactivatedUsersCount).
		Int("reactivated_systems_count", reactivatedSystemsCount).
		Str("reactivated_by_user_id", reactivatedByUserID).
		Str("reactivated_by_org_id", reactivatedByOrgID).
		Msg("Customer reactivated successfully")

	// Return updated customer
	updatedCustomer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return nil, reactivatedUsersCount, reactivatedSystemsCount, fmt.Errorf("failed to get updated customer: %w", err)
	}
	return updatedCustomer, reactivatedUsersCount, reactivatedSystemsCount, nil
}

// cascadeSuspendUsers suspends all active users of an organization and syncs to Logto
func (s *LocalOrganizationService) cascadeSuspendUsers(orgLogtoID, orgType, orgName string) (int, error) {
	// Suspend users in local database
	users, count, err := s.userRepo.SuspendUsersByOrgID(orgLogtoID)
	if err != nil {
		return 0, fmt.Errorf("failed to suspend users locally: %w", err)
	}

	if count == 0 {
		return 0, nil
	}

	// Sync suspensions to Logto
	failedCount := 0
	for _, user := range users {
		if user.LogtoID != nil && *user.LogtoID != "" {
			err := s.logtoClient.SuspendUser(*user.LogtoID)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("user_id", user.ID).
					Str("logto_user_id", *user.LogtoID).
					Str("username", user.Username).
					Str("organization_logto_id", orgLogtoID).
					Str("org_type", orgType).
					Msg("Failed to suspend user in Logto (user suspended locally)")
				failedCount++
			}
		}
	}

	logger.Info().
		Int("total_users", len(users)).
		Int("suspended_locally", count).
		Int("failed_logto_sync", failedCount).
		Str("organization_logto_id", orgLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed cascade user suspension for organization")

	return count, nil
}

// cascadeReactivateUsers reactivates all cascade-suspended users of an organization and syncs to Logto
func (s *LocalOrganizationService) cascadeReactivateUsers(orgLogtoID, orgType, orgName string) (int, error) {
	// Reactivate users in local database
	users, count, err := s.userRepo.ReactivateUsersByOrgID(orgLogtoID)
	if err != nil {
		return 0, fmt.Errorf("failed to reactivate users locally: %w", err)
	}

	if count == 0 {
		return 0, nil
	}

	// Sync reactivations to Logto
	failedCount := 0
	for _, user := range users {
		if user.LogtoID != nil && *user.LogtoID != "" {
			err := s.logtoClient.ReactivateUser(*user.LogtoID)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("user_id", user.ID).
					Str("logto_user_id", *user.LogtoID).
					Str("username", user.Username).
					Str("organization_logto_id", orgLogtoID).
					Str("org_type", orgType).
					Msg("Failed to reactivate user in Logto (user reactivated locally)")
				failedCount++
			}
		}
	}

	logger.Info().
		Int("total_users", len(users)).
		Int("reactivated_locally", count).
		Int("failed_logto_sync", failedCount).
		Str("organization_logto_id", orgLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed cascade user reactivation for organization")

	return count, nil
}

// cascadeSuspendSystems suspends all active systems of an organization
func (s *LocalOrganizationService) cascadeSuspendSystems(orgLogtoID, orgType, orgName string) (int, error) {
	count, err := s.systemRepo.SuspendSystemsByOrgID(orgLogtoID)
	if err != nil {
		return 0, fmt.Errorf("failed to suspend systems locally: %w", err)
	}

	logger.Info().
		Int("suspended_systems", count).
		Str("organization_logto_id", orgLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed cascade system suspension for organization")

	return count, nil
}

// cascadeReactivateSystems reactivates all cascade-suspended systems of an organization
func (s *LocalOrganizationService) cascadeReactivateSystems(orgLogtoID, orgType, orgName string) (int, error) {
	count, err := s.systemRepo.ReactivateSystemsByOrgID(orgLogtoID)
	if err != nil {
		return 0, fmt.Errorf("failed to reactivate systems locally: %w", err)
	}

	logger.Info().
		Int("reactivated_systems", count).
		Str("organization_logto_id", orgLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed cascade system reactivation for organization")

	return count, nil
}

// refreshUnifiedOrganizationsAsync refreshes the unified_organizations materialized
// view in a background goroutine. Called after organization CRUD operations.
func refreshUnifiedOrganizationsAsync() {
	go func() {
		if err := database.RefreshUnifiedOrganizations(); err != nil {
			logger.Warn().Err(err).Msg("Failed to refresh unified_organizations materialized view")
		}
	}()
}
