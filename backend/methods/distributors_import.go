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
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
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

		// Duplicate-in-CSV check on VAT — sanity-level, applied to all entity
		// types. Name uniqueness isn't enforced by the DB (two distributors
		// can share a name as long as their VAT differs), so we don't flag
		// duplicate names in the CSV either.
		if dupErr := csvimport.CheckDuplicateInSet("vat_number", rowMap["vat_number"], seenVATs, rowNum); dupErr != nil {
			errs = append(errs, *dupErr)
		}

		// Database VAT check.
		//   - distributors/resellers: DB triggers enforce global VAT uniqueness.
		//       active org same VAT  → WARNING (override=true turns it into UPDATE)
		//       soft-deleted same VAT → ERROR (admin must restore or destroy first)
		//   - customers: no DB uniqueness, so we de-duplicate at import time,
		//       scoped to the importing org (custom_data.createdBy) and skipping
		//       placeholder VATs. An existing active customer with the same VAT
		//       under this org → WARNING (skipped on confirm unless override).
		var warns []models.ImportFieldError
		if rowMap["vat_number"] != "" && !hasFieldError(errs, "vat_number") {
			if entityType == "customers" {
				exists, dbErr := csvimport.CustomerVATExistsForOwner(rowMap["vat_number"], user.OrganizationID)
				if dbErr != nil {
					logger.Error().Err(dbErr).Str("vat_number", rowMap["vat_number"]).Msg("Failed to check customer duplicate")
				} else if exists {
					warns = append(warns, models.ImportFieldError{
						Field:   "vat_number",
						Message: "already_exists",
						Values:  []string{rowMap["vat_number"]},
					})
				}
			} else {
				state, dbErr := csvimport.CheckOrganizationExistenceStateByVAT(rowMap["vat_number"], entityType)
				if dbErr != nil {
					logger.Error().Err(dbErr).Str("vat_number", rowMap["vat_number"]).Msg("Failed to check organization duplicate")
				} else {
					switch state {
					case csvimport.OrgExistsActive:
						warns = append(warns, models.ImportFieldError{
							Field:   "vat_number",
							Message: "already_exists",
							Values:  []string{rowMap["vat_number"]},
						})
					case csvimport.OrgSoftDeleted:
						errs = append(errs, models.ImportFieldError{
							Field:   "vat_number",
							Message: "archived",
							Values:  []string{rowMap["vat_number"]},
						})
					}
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
		Results: make([]models.ImportResultRow, 0, len(session.Rows)),
	}

	// Each valid/override row is created (or updated) against Logto — a few
	// hundred sequential round-trips easily blow past a minute. Process rows
	// concurrently with a bounded pool. The cap is kept at the DB connection
	// pool size because each create holds one connection through its Logto
	// calls (see CreateCustomer); going higher just queues on the pool and
	// starves other requests.
	const importConfirmConcurrency = 6

	results := make([]models.ImportResultRow, len(session.Rows))
	sem := make(chan struct{}, importConfirmConcurrency)
	var wg sync.WaitGroup

	for i, row := range session.Rows {
		switch row.Status {
		case models.ImportRowInvalid:
			results[i] = models.ImportResultRow{
				RowNumber: row.RowNumber,
				Status:    models.ImportResultSkipped,
				Reason:    models.ImportSkipError,
			}

		case models.ImportRowWarning:
			if !req.Override {
				results[i] = models.ImportResultRow{
					RowNumber: row.RowNumber,
					Status:    models.ImportResultSkipped,
					Reason:    models.ImportSkipWarningNotOverride,
				}
				continue
			}
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, row models.ImportRow) {
				defer wg.Done()
				defer func() { <-sem }()
				results[i] = updateOrganizationFromImportRow(service, user, entityType, row)
			}(i, row)

		case models.ImportRowValid:
			wg.Add(1)
			sem <- struct{}{}
			go func(i int, row models.ImportRow) {
				defer wg.Done()
				defer func() { <-sem }()
				results[i] = createOrganizationFromImportRow(service, user, entityType, row)
			}(i, row)
		}
	}
	wg.Wait()

	// Tally single-threaded so the per-row audit log can safely touch the
	// (non-concurrency-safe) gin.Context.
	for _, r := range results {
		switch r.Status {
		case models.ImportResultCreated:
			result.Created++
			logger.LogBusinessOperation(c, entityType, "create", entityType, r.ID, true, nil)
		case models.ImportResultUpdated:
			result.Updated++
			logger.LogBusinessOperation(c, entityType, "update", entityType, r.ID, true, nil)
		case models.ImportResultFailed:
			result.Failed++
		case models.ImportResultSkipped:
			result.Skipped++
		}
		result.Results = append(result.Results, r)
	}

	// Clean up the import session
	csvimport.DeleteImportSession(req.ImportID)

	// Invalidate caches
	cache.GetRBACCache().InvalidateAll()

	// Refresh planner statistics for the imported table. A bulk import can add
	// hundreds of rows at once and shift the custom_data->>'createdBy'
	// distribution the org list/stats count queries rely on; stale statistics
	// make those queries fall back to sequential scans until autoanalyze
	// catches up. entityType is an internal constant (distributors/resellers/
	// customers), quoted defensively. Best-effort: never fail the import on it.
	if result.Created+result.Updated > 0 {
		if _, err := database.DB.Exec("ANALYZE " + pq.QuoteIdentifier(entityType)); err != nil {
			logger.RequestLogger(c, entityType).Warn().Err(err).Msg("post-import ANALYZE failed")
		}
	}

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

// createOrganizationFromImportRow creates a distributor/reseller/customer from a
// validated row and returns the per-row result. Safe for concurrent use: it
// touches neither the shared accumulator nor the gin.Context (audit logging is
// done single-threaded by the caller).
func createOrganizationFromImportRow(service *local.LocalOrganizationService, user *models.User, entityType string, row models.ImportRow) models.ImportResultRow {
	createReq := csvimport.OrganizationDataToCreateRequest(row.Data)

	var createdID string
	var createErr error

	switch entityType {
	case "distributors":
		org, err := service.CreateDistributor(createReq, models.NewOrgCreatorFromUser(*user), user.OrganizationID)
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
		org, err := service.CreateReseller(resellerReq, models.NewOrgCreatorFromUser(*user), user.OrganizationID)
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
		org, err := service.CreateCustomer(customerReq, models.NewOrgCreatorFromUser(*user), user.OrganizationID)
		if err != nil {
			createErr = err
		} else {
			createdID = org.ID
		}
	}

	if createErr != nil {
		logger.Error().
			Err(createErr).
			Int("row_number", row.RowNumber).
			Str("entity_type", entityType).
			Msg("Failed to create organization from import")
		return models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     formatImportError(createErr),
		}
	}
	return models.ImportResultRow{
		RowNumber: row.RowNumber,
		Status:    models.ImportResultCreated,
		ID:        createdID,
	}
}

// updateOrganizationFromImportRow looks up the existing org by VAT (matching
// the validate-time uniqueness check) and overwrites name, description and
// custom_data with the CSV-provided values. RBAC is already enforced at
// validate time (CanCreateDistributor/Reseller/Customer for the caller); the
// underlying Update*  service performs additional checks.
//
// Reached for any entity whose row was flagged as a duplicate WARNING and then
// confirmed with override=true. For distributors/resellers the match is global
// by VAT; for customers it is scoped to the importing org (custom_data.createdBy),
// mirroring the validate-time dedup.
func updateOrganizationFromImportRow(service *local.LocalOrganizationService, user *models.User, entityType string, row models.ImportRow) models.ImportResultRow {
	createReq := csvimport.OrganizationDataToCreateRequest(row.Data)

	vat, _ := createReq.CustomData["vat"].(string)
	var existingID string
	var err error
	if entityType == "customers" {
		existingID, err = csvimport.GetCustomerIDByVATForOwner(vat, user.OrganizationID)
	} else {
		existingID, err = csvimport.GetOrganizationIDByVAT(vat, entityType)
	}
	if err != nil || existingID == "" {
		errMsg := "organization not found in database"
		if err != nil {
			errMsg = err.Error()
		}
		return models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     errMsg,
		}
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
		logger.Error().
			Err(updateErr).
			Int("row_number", row.RowNumber).
			Str("entity_type", entityType).
			Msg("Failed to update organization from import")
		return models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     formatImportError(updateErr),
		}
	}
	return models.ImportResultRow{
		RowNumber: row.RowNumber,
		Status:    models.ImportResultUpdated,
		ID:        updatedID,
	}
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
