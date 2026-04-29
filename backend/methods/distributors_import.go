/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/csvimport"
	"github.com/nethesis/my/backend/services/local"
)

// GetDistributorsImportTemplate handles GET /api/distributors/import/template
func GetDistributorsImportTemplate(c *gin.Context) {
	sendTemplateCSV(c, "distributors_import_template.csv",
		csvimport.OrganizationCSVHeaders, csvimport.OrganizationTemplateExamples)
}

// ValidateDistributorsImport handles POST /api/distributors/import/validate
func ValidateDistributorsImport(c *gin.Context) {
	validateOrganizationImport(c, "distributors")
}

// ConfirmDistributorsImport handles POST /api/distributors/import/confirm
func ConfirmDistributorsImport(c *gin.Context) {
	confirmOrganizationImport(c, "distributors")
}

// validateOrganizationImport is the shared validation logic for distributor/reseller/customer imports
func validateOrganizationImport(c *gin.Context, entityType string) {
	// Read file
	data := readCSVFromRequest(c)
	if data == nil {
		return
	}

	// Get user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check create permission
	service := local.NewOrganizationService()
	userOrgRole := strings.ToLower(user.OrgRole)
	var canCreate bool
	var reason string
	switch entityType {
	case "distributors":
		canCreate, reason = service.CanCreateDistributor(userOrgRole, user.OrganizationID)
	case "resellers":
		canCreate, reason = service.CanCreateReseller(userOrgRole, user.OrganizationID)
	case "customers":
		canCreate, reason = service.CanCreateCustomer(userOrgRole, user.OrganizationID)
	}
	if !canCreate {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied: "+reason, nil))
		return
	}

	// Parse CSV
	headers, rows, err := csvimport.ParseCSV(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		return
	}

	// Validate headers
	if err := csvimport.ValidateHeaders(headers, csvimport.OrganizationCSVHeaders); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		return
	}

	// Validate each row
	result := models.ImportValidationResult{
		TotalRows: len(rows),
		Rows:      make([]models.ImportRow, 0, len(rows)),
	}

	seenNames := make(map[string]int)
	seenVATs := make(map[string]int)

	for i, row := range rows {
		rowNum := i + 2 // 1-indexed, skip header row
		rowMap := csvimport.RowToMap(headers, row)

		importRow := models.ImportRow{
			RowNumber: rowNum,
			Data:      csvimport.OrganizationRowToData(rowMap),
		}

		// Field-level validation
		errs := csvimport.ValidateOrganizationRow(rowMap)

		// Duplicate checks within CSV
		if dupErr := csvimport.CheckDuplicateInSet("name", rowMap["name"], seenNames, rowNum); dupErr != nil {
			errs = append(errs, *dupErr)
		}
		if dupErr := csvimport.CheckDuplicateInSet("vat", rowMap["vat"], seenVATs, rowNum); dupErr != nil {
			errs = append(errs, *dupErr)
		}

		// Database name check — three outcomes:
		//   - active org with same name  → WARNING (override=true turns it into UPDATE)
		//   - soft-deleted org same name → ERROR (admin must restore or destroy first)
		//   - no org                      → no flag, row stays valid
		var warns []models.ImportFieldError
		if rowMap["name"] != "" && !hasFieldError(errs, "name") {
			state, dbErr := csvimport.CheckOrganizationExistenceState(rowMap["name"], entityType)
			if dbErr != nil {
				logger.Error().Err(dbErr).Str("name", rowMap["name"]).Msg("Failed to check organization duplicate")
			} else {
				switch state {
				case csvimport.OrgExistsActive:
					warns = append(warns, models.ImportFieldError{
						Field:   "name",
						Message: "already_exists",
						Values:  []string{rowMap["name"]},
					})
				case csvimport.OrgSoftDeleted:
					errs = append(errs, models.ImportFieldError{
						Field:   "name",
						Message: "archived",
						Values:  []string{rowMap["name"]},
					})
				}
			}
		}

		switch {
		case len(errs) > 0:
			importRow.Status = models.ImportRowInvalid
			importRow.Errors = errs
			importRow.Warnings = warns
			result.ErrorRows++
		case len(warns) > 0:
			importRow.Status = models.ImportRowWarning
			importRow.Warnings = warns
			result.WarningRows++
		default:
			importRow.Status = models.ImportRowValid
			result.ValidRows++
		}

		result.Rows = append(result.Rows, importRow)
	}

	// Save session to Redis for confirmation
	session := &models.ImportSessionData{
		EntityType: entityType,
		Rows:       result.Rows,
		UserID:     user.ID,
		UserOrgID:  user.OrganizationID,
		OrgRole:    userOrgRole,
	}

	importID, err := csvimport.SaveImportSession(session)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to save import session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save import session", nil))
		return
	}

	result.ImportID = importID

	logger.RequestLogger(c, entityType).Info().
		Str("operation", "import_validate").
		Int("total_rows", result.TotalRows).
		Int("valid_rows", result.ValidRows).
		Int("error_rows", result.ErrorRows).
		Int("warning_rows", result.WarningRows).
		Str("import_id", importID).
		Msg("CSV import validated")

	c.JSON(http.StatusOK, response.OK(entityType+" import validated", result))
}

// confirmOrganizationImport is the shared confirmation logic for distributor/reseller/customer imports
func confirmOrganizationImport(c *gin.Context, entityType string) {
	var req models.ImportConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body", nil))
		return
	}

	if req.ImportID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("import_id is required", nil))
		return
	}

	// Get user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Retrieve session from Redis
	session, err := csvimport.GetImportSession(req.ImportID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("import session not found or expired. please re-validate the CSV file", nil))
		return
	}

	// Validate session belongs to this user and entity type
	if session.UserID != user.ID {
		c.JSON(http.StatusForbidden, response.Forbidden("this import session belongs to another user", nil))
		return
	}
	if session.EntityType != entityType {
		c.JSON(http.StatusBadRequest, response.BadRequest("import session entity type mismatch", nil))
		return
	}

	service := local.NewOrganizationService()
	result := models.ImportConfirmResult{
		Results: make([]models.ImportResultRow, 0),
	}

	for _, row := range session.Rows {
		switch row.Status {
		case models.ImportRowInvalid:
			result.Skipped++
			result.Results = append(result.Results, models.ImportResultRow{
				RowNumber: row.RowNumber,
				Status:    models.ImportResultSkipped,
				Reason:    models.ImportSkipError,
			})

		case models.ImportRowWarning:
			if !req.Override {
				result.Skipped++
				result.Results = append(result.Results, models.ImportResultRow{
					RowNumber: row.RowNumber,
					Status:    models.ImportResultSkipped,
					Reason:    models.ImportSkipWarningNotOverride,
				})
				continue
			}
			updateOrganizationFromImportRow(c, service, user, entityType, row, &result)

		case models.ImportRowValid:
			createOrganizationFromImportRow(c, service, user, entityType, row, &result)
		}
	}

	// Clean up the import session
	csvimport.DeleteImportSession(req.ImportID)

	// Invalidate caches
	cache.GetRBACCache().InvalidateAll()

	logger.RequestLogger(c, entityType).Info().
		Str("operation", "import_confirm").
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("failed", result.Failed).
		Bool("override", req.Override).
		Str("import_id", req.ImportID).
		Msg("CSV import confirmed")

	c.JSON(http.StatusOK, response.OK(entityType+" imported successfully", result))
}

// createOrganizationFromImportRow creates a distributor/reseller/customer from a validated row.
func createOrganizationFromImportRow(c *gin.Context, service *local.LocalOrganizationService, user *models.User, entityType string, row models.ImportRow, result *models.ImportConfirmResult) {
	createReq := csvimport.OrganizationDataToCreateRequest(row.Data)

	var createdID string
	var createErr error

	switch entityType {
	case "distributors":
		org, err := service.CreateDistributor(createReq, user.ID, user.OrganizationID)
		if err != nil {
			createErr = err
		} else {
			createdID = org.ID
		}
	case "resellers":
		resellerReq := &models.CreateLocalResellerRequest{
			Name:        createReq.Name,
			Description: createReq.Description,
			CustomData:  createReq.CustomData,
		}
		org, err := service.CreateReseller(resellerReq, user.ID, user.OrganizationID)
		if err != nil {
			createErr = err
		} else {
			createdID = org.ID
		}
	case "customers":
		customerReq := &models.CreateLocalCustomerRequest{
			Name:        createReq.Name,
			Description: createReq.Description,
			CustomData:  createReq.CustomData,
		}
		org, err := service.CreateCustomer(customerReq, user.ID, user.OrganizationID)
		if err != nil {
			createErr = err
		} else {
			createdID = org.ID
		}
	}

	if createErr != nil {
		result.Failed++
		result.Results = append(result.Results, models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     formatImportError(createErr),
		})
		logger.Error().
			Err(createErr).
			Int("row_number", row.RowNumber).
			Str("entity_type", entityType).
			Msg("Failed to create organization from import")
		return
	}
	result.Created++
	result.Results = append(result.Results, models.ImportResultRow{
		RowNumber: row.RowNumber,
		Status:    models.ImportResultCreated,
		ID:        createdID,
	})
	logger.LogBusinessOperation(c, entityType, "create", entityType, createdID, true, nil)
}

// updateOrganizationFromImportRow looks up the existing org by name (within the entity type)
// and overwrites name, description and custom_data with the CSV-provided values. RBAC is
// already enforced at validate time (CanCreateDistributor/Reseller/Customer for the caller),
// and the underlying Update*  service performs additional checks.
func updateOrganizationFromImportRow(c *gin.Context, service *local.LocalOrganizationService, user *models.User, entityType string, row models.ImportRow, result *models.ImportConfirmResult) {
	createReq := csvimport.OrganizationDataToCreateRequest(row.Data)

	existingID, err := csvimport.GetOrganizationIDByName(createReq.Name, entityType)
	if err != nil || existingID == "" {
		result.Failed++
		errMsg := "organization not found in database"
		if err != nil {
			errMsg = err.Error()
		}
		result.Results = append(result.Results, models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     errMsg,
		})
		return
	}

	customDataPtr := &createReq.CustomData
	descPtr := &createReq.Description

	var updatedID string
	var updateErr error

	switch entityType {
	case "distributors":
		updateReq := &models.UpdateLocalDistributorRequest{
			Name:        &createReq.Name,
			Description: descPtr,
			CustomData:  customDataPtr,
		}
		org, err := service.UpdateDistributor(existingID, updateReq, user.ID, user.OrganizationID)
		if err != nil {
			updateErr = err
		} else {
			updatedID = org.ID
		}
	case "resellers":
		updateReq := &models.UpdateLocalResellerRequest{
			Name:        &createReq.Name,
			Description: descPtr,
			CustomData:  customDataPtr,
		}
		org, err := service.UpdateReseller(existingID, updateReq, user.ID, user.OrganizationID)
		if err != nil {
			updateErr = err
		} else {
			updatedID = org.ID
		}
	case "customers":
		updateReq := &models.UpdateLocalCustomerRequest{
			Name:        &createReq.Name,
			Description: descPtr,
			CustomData:  customDataPtr,
		}
		org, err := service.UpdateCustomer(existingID, updateReq, user.ID, user.OrganizationID)
		if err != nil {
			updateErr = err
		} else {
			updatedID = org.ID
		}
	}

	if updateErr != nil {
		result.Failed++
		result.Results = append(result.Results, models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     formatImportError(updateErr),
		})
		logger.Error().
			Err(updateErr).
			Int("row_number", row.RowNumber).
			Str("entity_type", entityType).
			Msg("Failed to update organization from import")
		return
	}
	result.Updated++
	result.Results = append(result.Results, models.ImportResultRow{
		RowNumber: row.RowNumber,
		Status:    models.ImportResultUpdated,
		ID:        updatedID,
	})
	logger.LogBusinessOperation(c, entityType, "update", entityType, updatedID, true, nil)
}

// hasFieldError checks if there's already an error for the given field
func hasFieldError(errs []models.ImportFieldError, field string) bool {
	for _, e := range errs {
		if e.Field == field {
			return true
		}
	}
	return false
}
