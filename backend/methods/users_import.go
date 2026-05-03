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
	"errors"
	"fmt"
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
		var ambiguousCandidates []models.ImportOrgCandidate
		if rowMap["company_name"] != "" && !hasFieldError(errs, "company_name") {
			orgID, _, resolveErr := csvimport.ResolveOrganizationByName(rowMap["company_name"], allowedOrgIDs)
			if resolveErr != nil {
				var ambErr *csvimport.AmbiguousOrgError
				if errors.As(resolveErr, &ambErr) {
					candidates := make([]models.ImportOrgCandidate, len(ambErr.Candidates))
					for j, c := range ambErr.Candidates {
						candidates[j] = models.ImportOrgCandidate{
							LogtoID: c.LogtoID,
							Name:    c.Name,
							Type:    c.OrgType,
						}
					}
					ambiguousCandidates = candidates
				} else {
					logger.Error().Err(resolveErr).Str("organization", rowMap["company_name"]).Msg("Failed to resolve organization")
					errs = append(errs, models.ImportFieldError{
						Field:   "company_name",
						Message: "lookup_failed",
						Values:  []string{rowMap["company_name"]},
					})
				}
			} else if orgID == "" {
				errs = append(errs, models.ImportFieldError{
					Field:   "company_name",
					Message: "not_found",
					Values:  []string{rowMap["company_name"]},
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
				// `unknown_roles`: each invalid role name becomes its own value, so
				// the frontend can render the list with proper i18n separators.
				errs = append(errs, models.ImportFieldError{
					Field:   "roles",
					Message: "unknown",
					Values:  invalidNames,
				})
			} else if len(ids) == 0 {
				errs = append(errs, models.ImportFieldError{
					Field:   "roles",
					Message: "at_least_one_required",
				})
			} else {
				roleIDs = ids

				// Validate role assignment permissions
				for _, roleID := range roleIDs {
					accessControl, exists := roleCache.GetAccessControl(roleID)
					if exists && accessControl.HasAccessControl {
						if !HasOrgRolePermission(user.OrgRole, accessControl.RequiredOrgRole) {
							errs = append(errs, models.ImportFieldError{
								Field:   "roles",
								Message: "insufficient_privileges",
								Values:  []string{rowMap["roles"]},
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

		// Database email check — three outcomes:
		//   - active user with same email  → WARNING (override=true turns it into UPDATE)
		//   - soft-deleted user same email → ERROR (admin must restore or destroy first)
		//   - no user                       → no flag, row stays valid
		var warns []models.ImportFieldError
		if rowMap["email"] != "" && !hasFieldError(errs, "email") {
			state, dbErr := csvimport.CheckUserExistenceState(rowMap["email"])
			if dbErr != nil {
				logger.Error().Err(dbErr).Str("email", rowMap["email"]).Msg("Failed to check user duplicate")
			} else {
				switch state {
				case csvimport.UserExistsActive:
					warns = append(warns, models.ImportFieldError{
						Field:   "email",
						Message: "already_exists",
						Values:  []string{rowMap["email"]},
					})
				case csvimport.UserSoftDeleted:
					errs = append(errs, models.ImportFieldError{
						Field:   "email",
						Message: "archived",
						Values:  []string{rowMap["email"]},
					})
				}
			}
		}

		// Phone uniqueness check — Logto enforces phone uniqueness, so a row
		// whose phone is already used by a *different* user would fail at confirm
		// time. Catching it here surfaces the conflict at validate.
		// The check is excluded for self-referential warning rows (same email
		// just keeping their existing phone) so override=true still works.
		if rowMap["phone"] != "" && !hasFieldError(errs, "phone") {
			collidingEmail, dbErr := csvimport.GetUserEmailByPhone(rowMap["phone"], rowMap["email"])
			if dbErr != nil {
				logger.Error().Err(dbErr).Str("phone", rowMap["phone"]).Msg("Failed to check phone collision")
			} else if collidingEmail != "" {
				errs = append(errs, models.ImportFieldError{
					Field:   "phone",
					Message: "already_used",
					Values:  []string{rowMap["phone"], collidingEmail},
				})
			}
		}

		switch {
		case len(errs) > 0:
			// Errors are blocking — they win over warnings and ambiguity.
			importRow.Status = models.ImportRowInvalid
			importRow.Errors = errs
			importRow.Warnings = warns
			result.ErrorRows++
		case ambiguousCandidates != nil:
			importRow.Status = models.ImportRowAmbiguous
			importRow.Errors = []models.ImportFieldError{{
				Field:      "company_name",
				Message:    "ambiguous",
				Values:     []string{rowMap["company_name"]},
				Candidates: ambiguousCandidates,
			}}
			importRow.Warnings = warns
			result.AmbiguousRows++
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
		Int("warning_rows", result.WarningRows).
		Int("ambiguous_rows", result.AmbiguousRows).
		Str("import_id", importID).
		Msg("CSV import validated")

	c.JSON(http.StatusOK, response.OK("users import validated", result))
}

// ConfirmUsersImport handles POST /api/users/import/confirm.
//
// Behavior matrix:
//
//	valid     → CREATE
//	error     → skipped (reason=error)
//	ambiguous → CREATE with the chosen org if a resolution is provided, otherwise skipped
//	warning   → UPDATE the existing user (looked up by email) when override=true,
//	            otherwise skipped
//
// `skip_rows` no longer exists in the request: the backend computes skips automatically.
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

	userService := local.NewUserService()
	userOrgRole := strings.ToLower(user.OrgRole)
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

		case models.ImportRowAmbiguous:
			rowKey := fmt.Sprintf("%d", row.RowNumber)
			resolution, hasResolution := req.Resolutions[rowKey]
			if !hasResolution || resolution.OrganizationID == "" {
				result.Skipped++
				result.Results = append(result.Results, models.ImportResultRow{
					RowNumber: row.RowNumber,
					Status:    models.ImportResultSkipped,
					Reason:    models.ImportSkipAmbiguousUnresolve,
				})
				continue
			}

			// Validate the chosen org ID is among the candidates
			validChoice := false
			for _, e := range row.Errors {
				for _, cand := range e.Candidates {
					if cand.LogtoID == resolution.OrganizationID {
						validChoice = true
						break
					}
				}
			}
			if !validChoice {
				result.Failed++
				result.Results = append(result.Results, models.ImportResultRow{
					RowNumber: row.RowNumber,
					Status:    models.ImportResultFailed,
					Error:     "resolved organization_id is not among the candidates",
				})
				continue
			}

			row.Data["organization_id"] = resolution.OrganizationID
			createUserFromImportRow(c, userService, user, row, &result)

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
			updateUserFromImportRow(c, userService, user, userOrgRole, row, &result)

		case models.ImportRowValid:
			createUserFromImportRow(c, userService, user, row, &result)
		}
	}

	// Clean up session
	csvimport.DeleteImportSession(req.ImportID)

	// Invalidate caches
	cache.GetRBACCache().InvalidateAll()

	logger.RequestLogger(c, "users").Info().
		Str("operation", "import_confirm").
		Int("created", result.Created).
		Int("updated", result.Updated).
		Int("skipped", result.Skipped).
		Int("failed", result.Failed).
		Bool("override", req.Override).
		Str("import_id", req.ImportID).
		Msg("Users CSV import confirmed")

	c.JSON(http.StatusOK, response.OK("users imported successfully", result))
}

// createUserFromImportRow creates a new user from an import row and appends the result.
func createUserFromImportRow(c *gin.Context, userService *local.LocalUserService, user *models.User, row models.ImportRow, result *models.ImportConfirmResult) {
	createReq := csvimport.UserDataToCreateRequest(row.Data)
	account, createErr := userService.CreateUser(createReq, user.ID, user.OrganizationID)
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
			Str("email", createReq.Email).
			Msg("Failed to create user from import")
		return
	}
	result.Created++
	result.Results = append(result.Results, models.ImportResultRow{
		RowNumber: row.RowNumber,
		Status:    models.ImportResultCreated,
		ID:        account.ID,
	})
	logger.LogBusinessOperation(c, "users", "create", "user", account.ID, true, nil)
}

// updateUserFromImportRow looks up the existing user by email and overwrites all CSV-provided
// fields (name, phone, organization, roles). RBAC is enforced explicitly: the caller must be
// allowed to update the existing user (target user's current org must be in caller's hierarchy).
// Moving a user across orgs is allowed because the destination org has already been resolved
// against the caller's hierarchy at validate time.
func updateUserFromImportRow(c *gin.Context, userService *local.LocalUserService, user *models.User, userOrgRole string, row models.ImportRow, result *models.ImportConfirmResult) {
	createReq := csvimport.UserDataToCreateRequest(row.Data)

	existingID, err := csvimport.GetUserIDByEmail(createReq.Email)
	if err != nil || existingID == "" {
		// Race: row was a warning at validate time but the user is gone now.
		result.Failed++
		errMsg := "user not found in database"
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

	// RBAC check on the existing target user — fails fast with a per-row error if the caller
	// has no permission to update users in the target user's current org.
	existing, err := userService.GetUser(existingID, userOrgRole, user.OrganizationID)
	if err != nil {
		result.Failed++
		result.Results = append(result.Results, models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     formatImportError(err),
		})
		return
	}
	_ = existing // existence + RBAC verified

	// Email is the lookup key — we deliberately do not allow changing it via import.
	updateReq := &models.UpdateLocalUserRequest{
		Name:           &createReq.Name,
		Phone:          createReq.Phone,
		UserRoleIDs:    &createReq.UserRoleIDs,
		OrganizationID: createReq.OrganizationID,
	}

	updated, err := userService.UpdateUser(existingID, updateReq, user.ID, user.OrganizationID)
	if err != nil {
		result.Failed++
		result.Results = append(result.Results, models.ImportResultRow{
			RowNumber: row.RowNumber,
			Status:    models.ImportResultFailed,
			Error:     formatImportError(err),
		})
		logger.Error().
			Err(err).
			Int("row_number", row.RowNumber).
			Str("email", createReq.Email).
			Msg("Failed to update user from import")
		return
	}
	result.Updated++
	result.Results = append(result.Results, models.ImportResultRow{
		RowNumber: row.RowNumber,
		Status:    models.ImportResultUpdated,
		ID:        updated.ID,
	})
	logger.LogBusinessOperation(c, "users", "update", "user", updated.ID, true, nil)
}
