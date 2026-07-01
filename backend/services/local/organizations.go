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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/cache"
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

// orgCreatorUserID safely extracts the creator's user logto_id for logging.
func orgCreatorUserID(creator *models.OrgCreator) string {
	if creator == nil {
		return ""
	}
	return creator.UserID
}

// CreateDistributor creates a distributor locally and syncs to Logto
func (s *LocalOrganizationService) CreateDistributor(req *models.CreateLocalDistributorRequest, creator *models.OrgCreator, createdByOrgID string) (*models.LocalDistributor, error) {
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
	if creator != nil {
		customData["createdByUser"] = creator
	}

	// Update the request with the properly managed customData
	req.CustomData = customData

	// 2. Create in local DB with CustomData inside the transaction
	distributor, err := s.distributorRepo.CreateWithTx(tx, req)
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

	// 5. Mark as synced (inside the transaction so logto_id commits atomically)
	err = s.markDistributorSynced(tx, distributor.ID, logtoOrg.ID)
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
		Str("created_by", orgCreatorUserID(creator)).
		Msg("Distributor created successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()

	distributor.CreatedBy = creator
	delete(distributor.CustomData, "createdByUser")
	return distributor, nil
}

// CreateReseller creates a reseller locally and syncs to Logto
func (s *LocalOrganizationService) CreateReseller(req *models.CreateLocalResellerRequest, creator *models.OrgCreator, createdByOrgID string) (*models.LocalReseller, error) {
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
	if creator != nil {
		customData["createdByUser"] = creator
	}

	// Update the request with the properly managed customData
	req.CustomData = customData

	// 2. Create in local DB with CustomData inside the transaction
	reseller, err := s.resellerRepo.CreateWithTx(tx, req)
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

	// 4. Mark as synced (inside the transaction so logto_id commits atomically)
	err = s.markResellerSynced(tx, reseller.ID, logtoOrg.ID)
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
		Str("created_by", orgCreatorUserID(creator)).
		Msg("Reseller created successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()

	reseller.CreatedBy = creator
	delete(reseller.CustomData, "createdByUser")
	return reseller, nil
}

// CreateCustomer creates a customer locally and syncs to Logto
func (s *LocalOrganizationService) CreateCustomer(req *models.CreateLocalCustomerRequest, creator *models.OrgCreator, createdByOrgID string) (*models.LocalCustomer, error) {
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
	if creator != nil {
		customData["createdByUser"] = creator
	}

	// Update the request with the properly managed customData
	req.CustomData = customData

	// 2. Create in local DB with CustomData inside the transaction
	customer, err := s.customerRepo.CreateWithTx(tx, req)
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

	// 4. Mark as synced (inside the transaction so logto_id commits atomically)
	err = s.markCustomerSynced(tx, customer.ID, logtoOrg.ID)
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
		Str("created_by", orgCreatorUserID(creator)).
		Msg("Customer created successfully with Logto sync")

	refreshUnifiedOrganizationsAsync()

	customer.CreatedBy = creator
	delete(customer.CustomData, "createdByUser")
	return customer, nil
}

// Helper methods to mark entities as synced. The executor lets the create flow run
// the UPDATE inside its transaction so logto_id commits atomically with the row.
func (s *LocalOrganizationService) markDistributorSynced(exec sqlExecer, id, logtoID string) error {
	query := `UPDATE distributors SET logto_id = $1, logto_synced_at = $2, logto_sync_error = NULL WHERE id = $3`
	_, err := exec.Exec(query, logtoID, time.Now(), id)
	return err
}

func (s *LocalOrganizationService) markResellerSynced(exec sqlExecer, id, logtoID string) error {
	query := `UPDATE resellers SET logto_id = $1, logto_synced_at = $2, logto_sync_error = NULL WHERE id = $3`
	_, err := exec.Exec(query, logtoID, time.Now(), id)
	return err
}

func (s *LocalOrganizationService) markCustomerSynced(exec sqlExecer, id, logtoID string) error {
	query := `UPDATE customers SET logto_id = $1, logto_synced_at = $2, logto_sync_error = NULL WHERE id = $3`
	_, err := exec.Exec(query, logtoID, time.Now(), id)
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

// ResolveCreatedByOrg determines the owning organization (custom_data.createdBy)
// for a newly created reseller or customer.
//
// Visibility in this system keys ENTIRELY on custom_data.createdBy: a reseller
// sees customers whose createdBy is the reseller, a distributor sees resellers
// (and their customers) whose createdBy chains up to it. So an entity created by
// an upper tier on behalf of a lower one (e.g. the distributor-token migration
// import creating customers that belong to a reseller) must be stamped with the
// lower tier's org id, otherwise it stays invisible to that tier.
//
// By default the entity is owned by the caller's own org (current behaviour).
// An owner or distributor may instead pass created_by_organization_id to
// attribute it to an ancestor org, subject to:
//   - only owner/distributor may override;
//   - the target must sit within the caller's manageable hierarchy;
//   - the target must be a valid parent type for childType — a "reseller" must
//     be attributed to a distributor, a "customer" or "system" to a reseller or
//     distributor.
//
// It returns the effective createdBy org id, the attributed org's display name
// (empty when the entity stays owned by the caller's own org), whether the request
// is allowed, and a human-readable reason when it is not.
func (s *LocalOrganizationService) ResolveCreatedByOrg(userOrgRole, userOrgID, targetOrgID, childType string) (string, string, bool, string) {
	// Default: own org (unchanged behaviour).
	if targetOrgID == "" || targetOrgID == userOrgID {
		return userOrgID, "", true, ""
	}

	role := strings.ToLower(userOrgRole)
	if role != "owner" && role != "distributor" {
		return "", "", false, "only owner or distributor can set created_by_organization_id"
	}

	// The target must sit within the caller's manageable hierarchy.
	if !NewUserService().IsOrganizationInHierarchy(role, userOrgID, targetOrgID) {
		return "", "", false, "created_by_organization_id is not within your hierarchy"
	}

	// The target must be a valid parent type for the entity being created; capture
	// its name so the caller can stamp the creator snapshot's org.
	var orgName string
	switch strings.ToLower(childType) {
	case "reseller":
		distributor, err := s.distributorRepo.GetByID(targetOrgID)
		if err != nil {
			return "", "", false, "a reseller can only be attributed to a distributor"
		}
		orgName = distributor.Name
	case "customer", "system":
		if reseller, err := s.resellerRepo.GetByID(targetOrgID); err == nil {
			orgName = reseller.Name
		} else if distributor, err := s.distributorRepo.GetByID(targetOrgID); err == nil {
			orgName = distributor.Name
		} else {
			return "", "", false, "created_by_organization_id must be a reseller or distributor"
		}
	default:
		return "", "", false, "unsupported entity type for created_by_organization_id"
	}

	return targetOrgID, orgName, true, ""
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
func (s *LocalOrganizationService) ListDistributors(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses, createdBy []string) ([]*models.LocalDistributor, int, error) {
	return s.distributorRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection, statuses, createdBy)
}

// ListResellers returns paginated resellers based on RBAC
func (s *LocalOrganizationService) ListResellers(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses, createdBy []string) ([]*models.LocalReseller, int, error) {
	return s.resellerRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection, statuses, createdBy)
}

// ListCustomers returns paginated customers based on RBAC
func (s *LocalOrganizationService) ListCustomers(userOrgRole, userOrgID string, page, pageSize int, search, sortBy, sortDirection string, statuses, createdBy []string) ([]*models.LocalCustomer, int, error) {
	return s.customerRepo.List(userOrgRole, userOrgID, page, pageSize, search, sortBy, sortDirection, statuses, createdBy)
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
	err = s.markDistributorSynced(database.DB, id, *distributor.LogtoID)
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
	err = s.markResellerSynced(database.DB, id, *reseller.LogtoID)
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
	err = s.markCustomerSynced(database.DB, id, *customer.LogtoID)
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

// GetAllOrganizationsPaginated returns all organizations (distributors + resellers
// + customers) with true SQL-level pagination across the caller's hierarchy.
//
// RBAC: Owner sees every active organization. Non-owner sees own org + descendants
// (the allowed set is computed by GetAllowedOrganizationIDs, cached). Filters,
// search, sort and LIMIT/OFFSET are pushed to the database in a single UNION ALL
// + COUNT pair, so total_count is accurate even past the first page.
func (s *LocalOrganizationService) GetAllOrganizationsPaginated(userOrgRole, userOrgID string, page, pageSize int, sortBy, sortDirection string, filters models.OrganizationFilters) (*models.PaginatedOrganizations, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	// Resolve RBAC scope. Owner = no scope filter; others get the cached
	// hierarchy (own org + descendants) from the applications service.
	var scopeClause string
	var scopeArgs []interface{}
	nextIdx := 1

	if strings.ToLower(userOrgRole) != "owner" {
		appsService := NewApplicationsService()
		allowedIDs, err := appsService.GetAllowedOrganizationIDs(userOrgRole, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve allowed organizations: %w", err)
		}
		if len(allowedIDs) == 0 {
			return emptyPaginatedOrganizations(page, pageSize), nil
		}
		placeholders := make([]string, len(allowedIDs))
		for i, id := range allowedIDs {
			placeholders[i] = fmt.Sprintf("$%d", nextIdx)
			scopeArgs = append(scopeArgs, id)
			nextIdx++
		}
		scopeClause = fmt.Sprintf(" AND logto_id IN (%s)", strings.Join(placeholders, ","))
	}

	// Build filter clauses applied on top of the UNION.
	var filterClauses []string
	var filterArgs []interface{}

	if filters.Type != "" {
		filterClauses = append(filterClauses, fmt.Sprintf("type = $%d", nextIdx))
		filterArgs = append(filterArgs, filters.Type)
		nextIdx++
	}
	if filters.Name != "" {
		filterClauses = append(filterClauses, fmt.Sprintf("name = $%d", nextIdx))
		filterArgs = append(filterArgs, filters.Name)
		nextIdx++
	}
	if filters.Description != "" {
		filterClauses = append(filterClauses, fmt.Sprintf("description = $%d", nextIdx))
		filterArgs = append(filterArgs, filters.Description)
		nextIdx++
	}
	if filters.Search != "" {
		filterClauses = append(filterClauses, fmt.Sprintf("(LOWER(name) LIKE LOWER('%%' || $%d || '%%') OR LOWER(description) LIKE LOWER('%%' || $%d || '%%'))", nextIdx, nextIdx))
		filterArgs = append(filterArgs, filters.Search)
		nextIdx++
	}
	if filters.CreatedBy != "" {
		// Matches the old in-memory behavior: createdBy hits either distributorId
		// or resellerId in custom_data.
		filterClauses = append(filterClauses, fmt.Sprintf("(custom_data->>'distributorId' = $%d OR custom_data->>'resellerId' = $%d)", nextIdx, nextIdx))
		filterArgs = append(filterArgs, filters.CreatedBy)
		nextIdx++
	}

	filterClause := ""
	if len(filterClauses) > 0 {
		filterClause = " WHERE " + strings.Join(filterClauses, " AND ")
	}

	// UNION the three org tables, applying the RBAC scope per branch so the
	// planner can use the logto_id index instead of materialising every row.
	unionSQL := fmt.Sprintf(`
		SELECT id, logto_id, name, description, custom_data, 'distributor' AS type
		FROM distributors WHERE deleted_at IS NULL AND logto_id IS NOT NULL%s
		UNION ALL
		SELECT id, logto_id, name, description, custom_data, 'reseller' AS type
		FROM resellers WHERE deleted_at IS NULL AND logto_id IS NOT NULL%s
		UNION ALL
		SELECT id, logto_id, name, description, custom_data, 'customer' AS type
		FROM customers WHERE deleted_at IS NULL AND logto_id IS NOT NULL%s
	`, scopeClause, scopeClause, scopeClause)

	// Placeholders for scope are inserted into all three UNION branches via the
	// same %s, so they share positional args ($1..$N). Bind scope once + filters.
	queryArgs := make([]interface{}, 0, len(scopeArgs)+len(filterArgs)+2)
	queryArgs = append(queryArgs, scopeArgs...)
	queryArgs = append(queryArgs, filterArgs...)

	// True total — counts rows after scope + filters, before paging.
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM (%s) AS all_orgs%s`, unionSQL, filterClause)
	var totalCount int
	if err := database.DB.QueryRow(countQuery, queryArgs...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count organizations: %w", err)
	}

	// Whitelist sortable columns; string columns sort case-insensitively to
	// match the other list endpoints (and the alphabetical default the UI wants).
	orderClause := "ORDER BY LOWER(name) ASC"
	if sortBy != "" {
		validSortFields := map[string]string{
			"name":        "LOWER(name)",
			"description": "LOWER(description)",
			"type":        "type",
		}
		if dbField, valid := validSortFields[sortBy]; valid {
			direction := "ASC"
			if strings.ToUpper(sortDirection) == "DESC" {
				direction = "DESC"
			}
			orderClause = fmt.Sprintf("ORDER BY %s %s", dbField, direction)
		}
	}

	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(`SELECT id, logto_id, name, description, custom_data, type FROM (%s) AS all_orgs%s %s LIMIT $%d OFFSET $%d`, unionSQL, filterClause, orderClause, nextIdx, nextIdx+1)
	dataArgs := append(queryArgs, pageSize, offset)

	rows, err := database.DB.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	orgs := make([]models.LogtoOrganization, 0, pageSize)
	for rows.Next() {
		var (
			dbID, logtoID, name, orgType string
			description                  string
			customDataJSON               []byte
		)
		if err := rows.Scan(&dbID, &logtoID, &name, &description, &customDataJSON, &orgType); err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}

		customData := map[string]interface{}{
			"type":        orgType,
			"database_id": dbID,
		}
		if len(customDataJSON) > 0 {
			var existing map[string]interface{}
			if err := json.Unmarshal(customDataJSON, &existing); err == nil {
				for k, v := range existing {
					if k == "type" || k == "database_id" {
						continue
					}
					customData[k] = v
				}
			}
		}

		orgs = append(orgs, models.LogtoOrganization{
			ID:          logtoID,
			Name:        name,
			Description: description,
			CustomData:  customData,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate organizations: %w", err)
	}

	totalPages := 0
	if totalCount > 0 {
		totalPages = (totalCount + pageSize - 1) / pageSize
	}
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
	if sortBy != "" {
		paginationInfo.SortBy = &sortBy
		paginationInfo.SortDirection = &sortDirection
	}

	return &models.PaginatedOrganizations{
		Data:       orgs,
		Pagination: paginationInfo,
	}, nil
}

// emptyPaginatedOrganizations returns a zero-result paginated response.
func emptyPaginatedOrganizations(page, pageSize int) *models.PaginatedOrganizations {
	return &models.PaginatedOrganizations{
		Data: []models.LogtoOrganization{},
		Pagination: models.PaginationInfo{
			Page:       page,
			PageSize:   pageSize,
			TotalCount: 0,
			TotalPages: 0,
			HasNext:    false,
			HasPrev:    page > 1,
		},
	}
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
		systemKeys, err := s.systemRepo.SuspendSystemsByMultipleOrgIDs(allOrgIDs, distLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("distributor_id", id).Msg("Failed to cascade suspend systems")
		}
		invalidateSystemAuthKeys(systemKeys)
		suspendedSystemsCount = len(systemKeys)
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
		systemKeys, err := s.systemRepo.SuspendSystemsByMultipleOrgIDs(allOrgIDs, resLogtoID)
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", id).Msg("Failed to cascade suspend systems")
		}
		invalidateSystemAuthKeys(systemKeys)
		suspendedSystemsCount = len(systemKeys)
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
	systemKeys, err := s.systemRepo.SuspendSystemsByOrgID(orgLogtoID)
	if err != nil {
		return 0, fmt.Errorf("failed to suspend systems locally: %w", err)
	}
	invalidateSystemAuthKeys(systemKeys)

	logger.Info().
		Int("suspended_systems", len(systemKeys)).
		Str("organization_logto_id", orgLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed cascade system suspension for organization")

	return len(systemKeys), nil
}

// cascadeReactivateSystems reactivates all cascade-suspended systems of an organization
func (s *LocalOrganizationService) cascadeReactivateSystems(orgLogtoID, orgType, orgName string) (int, error) {
	systemKeys, err := s.systemRepo.ReactivateSystemsByOrgID(orgLogtoID)
	if err != nil {
		return 0, fmt.Errorf("failed to reactivate systems locally: %w", err)
	}
	invalidateSystemAuthKeys(systemKeys)

	logger.Info().
		Int("reactivated_systems", len(systemKeys)).
		Str("organization_logto_id", orgLogtoID).
		Str("org_type", orgType).
		Str("org_name", orgName).
		Msg("Completed cascade system reactivation for organization")

	return len(systemKeys), nil
}

// invalidateSystemAuthKeys purges collect's cached credentials for each system so a
// suspension or reactivation takes effect immediately instead of waiting for the
// auth-cache TTL. Best effort: InvalidateSystemAuth never blocks (entries also
// expire on their own within SystemAuthCacheTTL).
func invalidateSystemAuthKeys(systemKeys []string) {
	for _, key := range systemKeys {
		cache.InvalidateSystemAuth(context.Background(), key)
	}
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
