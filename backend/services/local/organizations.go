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
	logtoClient     *logto.LogtoManagementClient
}

// NewLocalOrganizationService creates a new local organization service
func NewOrganizationService() *LocalOrganizationService {
	return &LocalOrganizationService{
		distributorRepo: entities.NewLocalDistributorRepository(),
		resellerRepo:    entities.NewLocalResellerRepository(),
		customerRepo:    entities.NewLocalCustomerRepository(),
		userRepo:        entities.NewLocalUserRepository(),
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
		if strings.Contains(err.Error(), "VAT already exists") {
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
		if strings.Contains(err.Error(), "VAT already exists") {
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
		if strings.Contains(err.Error(), "VAT already exists") {
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
func (s *LocalOrganizationService) ListDistributors(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalDistributor, int, error) {
	return s.distributorRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection)
}

// ListResellers returns paginated resellers based on RBAC
func (s *LocalOrganizationService) ListResellers(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalReseller, int, error) {
	return s.resellerRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection)
}

// ListCustomers returns paginated customers based on RBAC
func (s *LocalOrganizationService) ListCustomers(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string) ([]*models.LocalCustomer, int, error) {
	return s.customerRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection)
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
		if strings.Contains(err.Error(), "VAT already exists") {
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
		if strings.Contains(err.Error(), "VAT already exists") {
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
		if strings.Contains(err.Error(), "VAT already exists") {
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

	return customer, nil
}

// ============================================
// DELETE OPERATIONS (Soft Delete)
// ============================================

// DeleteDistributor soft-deletes a distributor locally and syncs to Logto
func (s *LocalOrganizationService) DeleteDistributor(id, deletedByUserID, deletedByOrgID string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get distributor before deletion for logging and logto_id
	distributor, err := s.distributorRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get distributor: %w", err)
	}

	// 1. Delete all users associated with this organization (both locally and from Logto)
	if distributor.LogtoID != nil {
		err = s.deleteOrganizationUsers(*distributor.LogtoID, "distributor", distributor.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("distributor_id", id).
				Str("distributor_name", distributor.Name).
				Msg("Failed to delete some users associated with distributor organization")
			// Continue with organization deletion even if some users failed to delete
		}
	}

	// 2. Soft delete in local DB
	err = s.distributorRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete distributor locally: %w", err)
	}

	// 3. Delete from Logto using logto_id
	if distributor.LogtoID != nil {
		err = s.logtoClient.DeleteOrganization(*distributor.LogtoID)
	} else {
		logger.Warn().Str("distributor_id", id).Msg("Distributor has no logto_id, skipping Logto deletion")
		err = nil // Don't fail if not synced to Logto
	}
	if err != nil {
		logger.Error().
			Err(err).
			Str("distributor_id", id).
			Str("distributor_name", distributor.Name).
			Msg("Failed to sync distributor deletion to Logto")
		return fmt.Errorf("failed to sync distributor deletion to Logto: %w", err)
	}

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("distributor_id", id).
		Str("distributor_name", distributor.Name).
		Str("deleted_by", deletedByUserID).
		Msg("Distributor deleted successfully with Logto sync")

	return nil
}

// DeleteReseller soft-deletes a reseller locally and syncs to Logto
func (s *LocalOrganizationService) DeleteReseller(id, deletedByUserID, deletedByOrgID string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get reseller before deletion for logging and logto_id
	reseller, err := s.resellerRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get reseller: %w", err)
	}

	// 1. Delete all users associated with this organization (both locally and from Logto)
	if reseller.LogtoID != nil {
		err = s.deleteOrganizationUsers(*reseller.LogtoID, "reseller", reseller.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("reseller_id", id).
				Str("reseller_name", reseller.Name).
				Msg("Failed to delete some users associated with reseller organization")
			// Continue with organization deletion even if some users failed to delete
		}
	}

	// 2. Soft delete in local DB
	err = s.resellerRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete reseller locally: %w", err)
	}

	// 3. Delete from Logto using logto_id
	if reseller.LogtoID != nil {
		err = s.logtoClient.DeleteOrganization(*reseller.LogtoID)
	} else {
		logger.Warn().Str("reseller_id", id).Msg("Reseller has no logto_id, skipping Logto deletion")
		err = nil // Don't fail if not synced to Logto
	}
	if err != nil {
		logger.Error().
			Err(err).
			Str("reseller_id", id).
			Str("reseller_name", reseller.Name).
			Msg("Failed to sync reseller deletion to Logto")
		return fmt.Errorf("failed to sync reseller deletion to Logto: %w", err)
	}

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("reseller_id", id).
		Str("reseller_name", reseller.Name).
		Str("deleted_by", deletedByUserID).
		Msg("Reseller deleted successfully with Logto sync")

	return nil
}

// DeleteCustomer soft-deletes a customer locally and syncs to Logto
func (s *LocalOrganizationService) DeleteCustomer(id, deletedByUserID, deletedByOrgID string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get customer before deletion for logging and logto_id
	customer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get customer: %w", err)
	}

	// 1. Delete all users associated with this organization (both locally and from Logto)
	if customer.LogtoID != nil {
		err = s.deleteOrganizationUsers(*customer.LogtoID, "customer", customer.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("customer_id", id).
				Str("customer_name", customer.Name).
				Msg("Failed to delete some users associated with customer organization")
			// Continue with organization deletion even if some users failed to delete
		}
	}

	// 2. Soft delete in local DB
	err = s.customerRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete customer locally: %w", err)
	}

	// 3. Delete from Logto using logto_id
	if customer.LogtoID != nil {
		err = s.logtoClient.DeleteOrganization(*customer.LogtoID)
	} else {
		logger.Warn().Str("customer_id", id).Msg("Customer has no logto_id, skipping Logto deletion")
		err = nil // Don't fail if not synced to Logto
	}
	if err != nil {
		logger.Error().
			Err(err).
			Str("customer_id", id).
			Str("customer_name", customer.Name).
			Msg("Failed to sync customer deletion to Logto")
		return fmt.Errorf("failed to sync customer deletion to Logto: %w", err)
	}

	// 4. Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("customer_id", id).
		Str("customer_name", customer.Name).
		Str("deleted_by", deletedByUserID).
		Msg("Customer deleted successfully with Logto sync")

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
	distributors, _, err := s.distributorRepo.List(userOrgRole, userOrgID, 1, fetchSize, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get distributors: %w", err)
	}
	for _, d := range distributors {
		// Use logto_id if available, otherwise fallback to local ID
		orgID := d.ID
		if d.LogtoID != nil {
			orgID = *d.LogtoID
		}

		allOrganizations = append(allOrganizations, models.LogtoOrganization{
			ID:          orgID,
			Name:        d.Name,
			Description: d.Description,
			CustomData: map[string]interface{}{
				"type": "distributor",
			},
		})
	}

	// Fetch resellers
	resellers, _, err := s.resellerRepo.List(userOrgRole, userOrgID, 1, fetchSize, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get resellers: %w", err)
	}
	for _, r := range resellers {
		// Use logto_id if available, otherwise fallback to local ID
		orgID := r.ID
		if r.LogtoID != nil {
			orgID = *r.LogtoID
		}

		customData := r.CustomData
		if customData == nil {
			customData = map[string]interface{}{"type": "reseller"}
		}
		allOrganizations = append(allOrganizations, models.LogtoOrganization{
			ID:          orgID,
			Name:        r.Name,
			Description: r.Description,
			CustomData:  customData,
		})
	}

	// Fetch customers
	customers, _, err := s.customerRepo.List(userOrgRole, userOrgID, 1, fetchSize, "", "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get customers: %w", err)
	}
	for _, c := range customers {
		// Use logto_id if available, otherwise fallback to local ID
		orgID := c.ID
		if c.LogtoID != nil {
			orgID = *c.LogtoID
		}

		customData := c.CustomData
		if customData == nil {
			customData = map[string]interface{}{"type": "customer"}
		}

		allOrganizations = append(allOrganizations, models.LogtoOrganization{
			ID:          orgID,
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
				"type": "distributor",
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

		customData := reseller.CustomData
		if customData == nil {
			customData = map[string]interface{}{"type": "reseller"}
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

		customData := customer.CustomData
		if customData == nil {
			customData = map[string]interface{}{"type": "customer"}
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

// deleteOrganizationUsers deletes all users associated with an organization
// This function is called when deleting an organization to ensure user consistency
func (s *LocalOrganizationService) deleteOrganizationUsers(organizationLogtoID, orgType, orgName string) error {
	// 1. Find all users associated with this organization
	query := `
		SELECT id, logto_id, username, email, name
		FROM users
		WHERE organization_id = $1 AND deleted_at IS NULL
	`

	rows, err := database.DB.Query(query, organizationLogtoID)
	if err != nil {
		return fmt.Errorf("failed to query users for organization %s: %w", organizationLogtoID, err)
	}
	defer func() { _ = rows.Close() }()

	type userToDelete struct {
		ID       string
		LogtoID  *string
		Username string
		Email    string
		Name     string
	}

	var usersToDelete []userToDelete
	for rows.Next() {
		var user userToDelete
		err := rows.Scan(&user.ID, &user.LogtoID, &user.Username, &user.Email, &user.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("organization_logto_id", organizationLogtoID).
				Msg("Failed to scan user for deletion")
			continue
		}
		usersToDelete = append(usersToDelete, user)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating users for organization %s: %w", organizationLogtoID, err)
	}

	// 2. Delete each user both locally and from Logto
	deletedCount := 0
	failedCount := 0

	for _, user := range usersToDelete {
		// Delete from local database first (soft delete)
		err := s.userRepo.Delete(user.ID)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("user_id", user.ID).
				Str("username", user.Username).
				Str("organization_logto_id", organizationLogtoID).
				Str("org_type", orgType).
				Msg("Failed to delete user locally")
			failedCount++
			continue
		}

		// Delete from Logto if user has been synced
		if user.LogtoID != nil && *user.LogtoID != "" {
			err := s.logtoClient.DeleteUser(*user.LogtoID)
			if err != nil {
				logger.Warn().
					Err(err).
					Str("user_id", user.ID).
					Str("logto_user_id", *user.LogtoID).
					Str("username", user.Username).
					Str("organization_logto_id", organizationLogtoID).
					Str("org_type", orgType).
					Msg("Failed to delete user from Logto (user deleted locally)")
				// User is deleted locally but failed in Logto - this is acceptable
				// The inconsistency will be logged but won't block the organization deletion
			}
		}

		deletedCount++
		logger.Info().
			Str("user_id", user.ID).
			Str("username", user.Username).
			Str("email", user.Email).
			Str("organization_logto_id", organizationLogtoID).
			Str("org_type", orgType).
			Str("org_name", orgName).
			Msg("User deleted successfully due to organization deletion")
	}

	logger.Info().
		Int("total_users", len(usersToDelete)).
		Int("deleted_successfully", deletedCount).
		Int("failed_deletions", failedCount).
		Str("organization_logto_id", organizationLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed user deletion for organization")

	// Return error only if all deletions failed, otherwise log warnings and continue
	if failedCount > 0 && deletedCount == 0 {
		return fmt.Errorf("failed to delete any users for organization %s (%s)", orgName, orgType)
	}

	return nil
}
