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
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/csvimport"
	"github.com/nethesis/my/backend/services/local"
)

// GetUsersImportTemplate handles GET /api/users/import/template
func GetUsersImportTemplate(c *gin.Context) {
	sendTemplateCSV(c, "users_import_template.csv",
		csvimport.UserCSVHeaders, csvimport.UserTemplateExamples)
}

// ValidateUsersImport handles POST /api/users/import/validate
func ValidateUsersImport(c *gin.Context) {
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
	userOrgRole := strings.ToLower(user.OrgRole)

	// Parse CSV
	headers, rows, err := csvimport.ParseCSV(data)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		return
	}

	// Validate headers
	if err := csvimport.ValidateHeaders(headers, csvimport.UserCSVHeaders); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		return
	}

	// Check role cache availability
	roleCache := cache.GetRoleNames()
	if !roleCache.IsLoaded() {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("role cache not available", nil))
		return
	}

	// Validate each row
	result := models.ImportValidationResult{
		TotalRows: len(rows),
		Rows:      make([]models.ImportRow, 0, len(rows)),
	}

	// Pre-load allowed organization IDs for this user's hierarchy
	userRepo := entities.NewLocalUserRepository()
	allowedOrgIDs, err := userRepo.GetHierarchicalOrganizationIDs(userOrgRole, user.OrganizationID)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load hierarchical organization IDs")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to load organization hierarchy", nil))
		return
	}

	seenEmails := make(map[string]int)

	for i, row := range rows {
		rowNum := i + 2
		rowMap := csvimport.RowToMap(headers, row)

		// Field-level validation
		errs := csvimport.ValidateUserRow(rowMap)

		// Duplicate email within CSV
		if dupErr := csvimport.CheckDuplicateInSet("email", rowMap["email"], seenEmails, rowNum); dupErr != nil {
			errs = append(errs, *dupErr)
		}

		// Resolve organization name to ID (scoped to user's hierarchy)
		var orgLogtoID string
		if rowMap["organization"] != "" && !hasFieldError(errs, "organization") {
			orgID, _, resolveErr := csvimport.ResolveOrganizationByName(rowMap["organization"], allowedOrgIDs)
			if resolveErr != nil {
				if resolveErr.Error() == "ambiguous_name" {
					errs = append(errs, models.ImportFieldError{
						Field:   "organization",
						Message: "ambiguous_name",
						Value:   rowMap["organization"],
					})
				} else {
					logger.Error().Err(resolveErr).Str("organization", rowMap["organization"]).Msg("Failed to resolve organization")
					errs = append(errs, models.ImportFieldError{
						Field:   "organization",
						Message: "lookup_failed",
						Value:   rowMap["organization"],
					})
				}
			} else if orgID == "" {
				errs = append(errs, models.ImportFieldError{
					Field:   "organization",
					Message: "not_found",
					Value:   rowMap["organization"],
				})
			} else {
				orgLogtoID = orgID
			}
		}

		// Resolve role names to IDs
		var roleIDs []string
		if rowMap["roles"] != "" && !hasFieldError(errs, "roles") {
			ids, invalidNames := csvimport.ResolveRolesByNames(rowMap["roles"])
			if len(invalidNames) > 0 {
				errs = append(errs, models.ImportFieldError{
					Field:   "roles",
					Message: "unknown_roles: " + strings.Join(invalidNames, ", "),
					Value:   rowMap["roles"],
				})
			} else if len(ids) == 0 {
				errs = append(errs, models.ImportFieldError{
					Field:   "roles",
					Message: "at_least_one_role_required",
					Value:   rowMap["roles"],
				})
			} else {
				roleIDs = ids

				// Validate role assignment permissions
				for _, roleID := range roleIDs {
					accessControl, exists := roleCache.GetAccessControl(roleID)
					if exists && accessControl.HasAccessControl {
						if !HasOrgRolePermission(user.OrgRole, accessControl.RequiredOrgRole) {
							roleName := rowMap["roles"]
							errs = append(errs, models.ImportFieldError{
								Field:   "roles",
								Message: "insufficient_privileges_to_assign_role",
								Value:   roleName,
							})
							break
						}
					}
				}
			}
		}

		importRow := models.ImportRow{
			RowNumber: rowNum,
			Data:      csvimport.UserRowToData(rowMap, orgLogtoID, roleIDs),
		}

		// Database duplicate check
		if rowMap["email"] != "" && !hasFieldError(errs, "email") {
			exists, dbErr := csvimport.CheckUserExistsByEmail(rowMap["email"])
			if dbErr != nil {
				logger.Error().Err(dbErr).Str("email", rowMap["email"]).Msg("Failed to check user duplicate")
			} else if exists {
				importRow.Status = models.ImportRowDuplicate
				importRow.Errors = append(errs, models.ImportFieldError{
					Field:   "email",
					Message: "already_exists",
					Value:   rowMap["email"],
				})
				result.DuplicateRows++
				result.Rows = append(result.Rows, importRow)
				continue
			}
		}

		if len(errs) > 0 {
			importRow.Status = models.ImportRowInvalid
			importRow.Errors = errs
			result.ErrorRows++
		} else {
			importRow.Status = models.ImportRowValid
			result.ValidRows++
		}

		result.Rows = append(result.Rows, importRow)
	}

	// Save session to Redis
	session := &models.ImportSessionData{
		EntityType: "users",
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

	logger.RequestLogger(c, "users").Info().
		Str("operation", "import_validate").
		Int("total_rows", result.TotalRows).
		Int("valid_rows", result.ValidRows).
		Int("error_rows", result.ErrorRows).
		Int("duplicate_rows", result.DuplicateRows).
		Str("import_id", importID).
		Msg("CSV import validated")

	c.JSON(http.StatusOK, response.OK("users import validated", result))
}

// ConfirmUsersImport handles POST /api/users/import/confirm
func ConfirmUsersImport(c *gin.Context) {
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

	// Retrieve session
	session, err := csvimport.GetImportSession(req.ImportID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("import session not found or expired. please re-validate the CSV file", nil))
		return
	}

	if session.UserID != user.ID {
		c.JSON(http.StatusForbidden, response.Forbidden("this import session belongs to another user", nil))
		return
	}
	if session.EntityType != "users" {
		c.JSON(http.StatusBadRequest, response.BadRequest("import session entity type mismatch", nil))
		return
	}

	// Build skip set
	skipSet := make(map[int]bool, len(req.SkipRows))
	for _, row := range req.SkipRows {
		skipSet[row] = true
	}

	// Create users
	userService := local.NewUserService()
	result := models.ImportConfirmResult{
		Results: make([]models.ImportResultRow, 0),
	}

	for _, row := range session.Rows {
		if row.Status != models.ImportRowValid {
			result.Skipped++
			result.Results = append(result.Results, models.ImportResultRow{
				RowNumber: row.RowNumber,
				Status:    models.ImportResultSkipped,
			})
			continue
		}

		if skipSet[row.RowNumber] {
			result.Skipped++
			result.Results = append(result.Results, models.ImportResultRow{
				RowNumber: row.RowNumber,
				Status:    models.ImportResultSkipped,
			})
			continue
		}

		createReq := csvimport.UserDataToCreateRequest(row.Data)

		account, createErr := userService.CreateUser(createReq, user.ID, user.OrganizationID)
		if createErr != nil {
			result.Failed++
			result.Results = append(result.Results, models.ImportResultRow{
				RowNumber: row.RowNumber,
				Status:    models.ImportResultFailed,
				Error:     createErr.Error(),
			})
			logger.Error().
				Err(createErr).
				Int("row_number", row.RowNumber).
				Str("email", createReq.Email).
				Msg("Failed to create user from import")
		} else {
			result.Created++
			result.Results = append(result.Results, models.ImportResultRow{
				RowNumber: row.RowNumber,
				Status:    models.ImportResultCreated,
				ID:        account.ID,
			})

			logger.LogBusinessOperation(c, "users", "create", "user", account.ID, true, nil)
		}
	}

	// Clean up session
	csvimport.DeleteImportSession(req.ImportID)

	// Invalidate caches
	cache.GetRBACCache().InvalidateAll()

	logger.RequestLogger(c, "users").Info().
		Str("operation", "import_confirm").
		Int("created", result.Created).
		Int("skipped", result.Skipped).
		Int("failed", result.Failed).
		Str("import_id", req.ImportID).
		Msg("Users CSV import confirmed")

	c.JSON(http.StatusOK, response.OK("users imported successfully", result))
}
