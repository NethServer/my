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

// GetCustomerTransferPreview handles GET /api/customers/:id/transfer — what
// moving the customer under ?organization_id= would do, without doing it.
func GetCustomerTransferPreview(c *gin.Context) {
	customerID := c.Param("id")
	targetOrgID := c.Query("organization_id")
	if customerID == "" || targetOrgID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID and organization_id required", nil))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	plan, err := local.NewOrganizationService().PlanCustomerTransfer(customerID, targetOrgID, transferActor(c, user))
	if err != nil {
		respondTransferError(c, customerID, err, "preview")
		return
	}

	c.JSON(http.StatusOK, response.OK("customer transfer preview retrieved successfully", plan))
}

// TransferCustomer handles PATCH /api/customers/:id/transfer — moves the
// customer, with its systems and users, under another managing organization.
func TransferCustomer(c *gin.Context) {
	customerID := c.Param("id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("customer ID required", nil))
		return
	}

	var request models.TransferCustomerRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	result, err := local.NewOrganizationService().TransferCustomer(customerID, request.OrganizationID, transferActor(c, user))
	if err != nil {
		respondTransferError(c, customerID, err, "transfer")
		return
	}

	logger.LogBusinessOperation(c, "customers", "transfer", "customer", customerID, true, nil)
	c.JSON(http.StatusOK, response.OK("customer transferred successfully", result))
}

func transferActor(c *gin.Context, user *models.User) local.CustomerTransferActor {
	return local.CustomerTransferActor{
		UserID:    user.ID,
		OrgID:     user.OrganizationID,
		OrgRole:   user.OrgRole,
		ActorIP:   c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
}

// respondTransferError maps the transfer guardrails onto HTTP: the conflicts
// carry a stable code in errors[].message, like the promotion, so the client
// builds its i18n key from it.
func respondTransferError(c *gin.Context, customerID string, err error, operation string) {
	if validationErr := getValidationError(err); validationErr != nil {
		c.JSON(validationErr.StatusCode, response.Error(validationErr.StatusCode, "validation failed", validationErr.ErrorData))
		return
	}

	conflicts := map[error]string{
		local.ErrTransferCustomerSuspended:      "customer_is_suspended",
		local.ErrTransferCustomerNotSynced:      "customer_not_synced",
		local.ErrTransferCustomerSameManager:    "same_organization",
		local.ErrTransferCustomerTargetInactive: "target_is_suspended",
		local.ErrTransferCustomerConcurrent:     "transfer_in_progress",
		cache.ErrCustomerTransferLocked:         "transfer_in_progress",
	}
	for sentinel, code := range conflicts {
		if errors.Is(err, sentinel) {
			if operation == "transfer" {
				logger.LogBusinessOperation(c, "customers", "transfer", "customer", customerID, false, err)
			}
			c.JSON(http.StatusConflict, response.Error(http.StatusConflict, "validation failed", response.ErrorData{
				Type:   "validation_error",
				Errors: []response.ValidationError{{Key: "transfer", Message: code}},
			}))
			return
		}
	}

	switch {
	case errors.Is(err, local.ErrTransferCustomerAccessDenied):
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to transfer customer", nil))
	case strings.Contains(err.Error(), "not found"):
		c.JSON(http.StatusNotFound, response.NotFound("customer not found", nil))
	default:
		logger.Error().Err(err).Str("customer_id", customerID).Str("operation", operation).Msg("Customer transfer failed")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to transfer customer", nil))
	}
}
