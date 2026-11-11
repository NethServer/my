/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// GetResellerTransferPreview handles GET /api/resellers/:id/transfer — what
// moving the reseller under ?organization_id= would do, without doing it.
func GetResellerTransferPreview(c *gin.Context) {
	resellerID := c.Param("id")
	targetOrgID := c.Query("organization_id")
	if resellerID == "" || targetOrgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID and organization_id required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	plan, err := local.NewOrganizationService().PlanResellerTransfer(resellerID, targetOrgID, transferActor(c, user))
	if err != nil {
		respondResellerTransferError(c, resellerID, err, "preview")
		return
	}

	c.JSON(http.StatusOK, response.OK("reseller transfer preview retrieved successfully", plan))
}

// TransferReseller handles PATCH /api/resellers/:id/transfer — moves the
// reseller, with its customers, systems and users, under another distributor.
func TransferReseller(c *gin.Context) {
	resellerID := c.Param("id")
	if resellerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("reseller ID required", nil))
		return
	}

	var request models.TransferResellerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	result, err := local.NewOrganizationService().TransferReseller(resellerID, request.OrganizationID, transferActor(c, user))
	if err != nil {
		respondResellerTransferError(c, resellerID, err, "transfer")
		return
	}

	logger.LogBusinessOperation(c, "resellers", "transfer", "reseller", resellerID, true, nil)
	c.JSON(http.StatusOK, response.OK("reseller transferred successfully", result))
}

// respondResellerTransferError maps the transfer guardrails onto HTTP, with
// the same stable codes as the customer transfer.
func respondResellerTransferError(c *gin.Context, resellerID string, err error, operation string) {
	if validationErr := getValidationError(err); validationErr != nil {
		c.JSON(validationErr.StatusCode, response.Error(validationErr.StatusCode, "validation failed", validationErr.ErrorData))
		return
	}

	conflicts := map[error]string{
		local.ErrTransferResellerSuspended:      "reseller_is_suspended",
		local.ErrTransferResellerNotSynced:      "reseller_not_synced",
		local.ErrTransferResellerSameManager:    "same_organization",
		local.ErrTransferResellerTargetInactive: "target_is_suspended",
		local.ErrTransferResellerConcurrent:     "transfer_in_progress",
		cache.ErrResellerTransferLocked:         "transfer_in_progress",
	}
	for sentinel, code := range conflicts {
		if errors.Is(err, sentinel) {
			if operation == "transfer" {
				logger.LogBusinessOperation(c, "resellers", "transfer", "reseller", resellerID, false, err)
			}
			c.JSON(http.StatusConflict, response.Error(http.StatusConflict, "validation failed", response.ErrorData{
				Type:   "validation_error",
				Errors: []response.ValidationError{{Key: "transfer", Message: code}},
			}))
			return
		}
	}

	switch {
	case errors.Is(err, local.ErrTransferResellerAccessDenied):
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to transfer reseller", nil))
	case strings.Contains(err.Error(), "not found"):
		c.JSON(http.StatusNotFound, response.NotFound("reseller not found", nil))
	default:
		logger.Error().Err(err).Str("reseller_id", resellerID).Str("operation", operation).Msg("Reseller transfer failed")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to transfer reseller", nil))
	}
}
