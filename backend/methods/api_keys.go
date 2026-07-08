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
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	"github.com/nethesis/my/backend/services/logto"
)

// CreateAPIKey issues a personal API key for the current user.
// POST /me/api-keys
func CreateAPIKey(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	// Step-up: re-verify the user's password before minting a long-lived
	// credential. Issuing a key is more sensitive than a normal request, so a
	// valid session alone is not enough.
	if user.LogtoID == nil || *user.LogtoID == "" {
		c.JSON(http.StatusForbidden, response.Forbidden(local.ErrAPIKeyNoLocalUser.Error(), nil))
		return
	}
	if err := logto.NewManagementClient().VerifyUserPassword(*user.LogtoID, req.Password); err != nil {
		logger.LogBusinessOperation(c, "api-keys", "create", "api_key", "", false, err)
		// Mirror the change-password flow: a wrong password is a field validation
		// error (400), not a 401. A 401 would trip the client interceptor into
		// logging the user out.
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "validation failed",
			"data": gin.H{
				"type": "validation_error",
				"errors": []gin.H{
					{"key": "password", "message": "incorrect_password", "value": ""},
				},
			},
		})
		return
	}

	key, token, err := local.NewAPIKeysService().CreateAPIKey(user, req.Name, req.Mode, req.ExpiresInDays)
	if err != nil {
		switch {
		case errors.Is(err, local.ErrAPIKeyNoLocalUser):
			c.JSON(http.StatusForbidden, response.Forbidden(err.Error(), nil))
		case errors.Is(err, local.ErrAPIKeyLimitReached):
			logger.LogBusinessOperation(c, "api-keys", "create", "api_key", "", false, err)
			c.JSON(http.StatusConflict, response.Error(http.StatusConflict, "validation failed", response.ErrorData{
				Type: "validation_error",
				Errors: []response.ValidationError{
					{Key: "limit", Message: "max_keys_reached", Value: strconv.Itoa(local.APIKeyMaxPerUser)},
				},
			}))
		case strings.Contains(err.Error(), "invalid mode"):
			c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		default:
			logger.LogBusinessOperation(c, "api-keys", "create", "api_key", "", false, err)
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create api key", nil))
		}
		return
	}

	logger.LogBusinessOperationDetails(c, "api-keys", "create", "api_key", key.ID, true, nil, map[string]interface{}{
		"name":       key.Name,
		"mode":       key.Mode,
		"expires_at": key.ExpiresAt,
	})
	local.NewAPIKeysService().RecordAPIKeyEvent(models.APIKeyAuditRecord{
		APIKeyID:       key.ID,
		UserID:         local.NewAPIKeysService().APIKeyAnchor(user),
		OrganizationID: user.OrganizationID,
		Event:          models.APIKeyEventCreated,
		KeyName:        key.Name,
		KeyMode:        key.Mode,
		IP:             c.ClientIP(),
		Method:         c.Request.Method,
		Path:           c.Request.URL.Path,
	})
	c.JSON(http.StatusCreated, response.Created("api key created successfully", models.CreateAPIKeyResponse{
		APIKey: *key,
		Token:  token,
	}))
}

// ListAPIKeys returns the current user's API keys (without secrets).
// GET /me/api-keys
func ListAPIKeys(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	keys, err := local.NewAPIKeysService().ListAPIKeys(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to list api keys", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("api keys retrieved successfully", gin.H{"api_keys": keys}))
}

// RevokeAPIKey revokes one of the current user's API keys.
// DELETE /me/api-keys/:id
func RevokeAPIKey(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	keyID := c.Param("id")
	err := local.NewAPIKeysService().RevokeAPIKey(user, keyID)
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, response.NotFound("api key not found", nil))
		return
	}
	if err != nil {
		logger.LogBusinessOperation(c, "api-keys", "revoke", "api_key", keyID, false, err)
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to revoke api key", nil))
		return
	}

	logger.LogBusinessOperation(c, "api-keys", "revoke", "api_key", keyID, true, nil)
	local.NewAPIKeysService().RecordAPIKeyEvent(models.APIKeyAuditRecord{
		APIKeyID:       keyID,
		UserID:         local.NewAPIKeysService().APIKeyAnchor(user),
		OrganizationID: user.OrganizationID,
		Event:          models.APIKeyEventRevoked,
		IP:             c.ClientIP(),
		Method:         c.Request.Method,
		Path:           c.Request.URL.Path,
	})
	c.JSON(http.StatusOK, response.OK("api key revoked successfully", nil))
}

// ListAPIKeyAudit returns the current user's API key audit trail (lifecycle
// events and security failures), paginated. Optional filters: event, api_key_id.
// GET /me/api-keys/audit
func ListAPIKeyAudit(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	page, pageSize := helpers.GetPaginationFromQuery(c)
	entries, total, err := local.NewAPIKeysService().ListAPIKeyAudit(
		local.NewAPIKeysService().APIKeyAnchor(user), c.Query("event"), c.Query("api_key_id"), page, pageSize,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve api key audit", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("api key audit retrieved successfully", gin.H{
		"audit":      entries,
		"pagination": helpers.BuildPaginationInfoWithSorting(page, pageSize, total, "created_at", "desc"),
	}))
}
