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
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
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

	key, token, err := local.NewAPIKeysService().CreateAPIKey(user.ID, user.OrganizationID, req.Name, req.Mode, req.ExpiresInDays)
	if err != nil {
		switch {
		case errors.Is(err, local.ErrAPIKeyNoLocalUser):
			c.JSON(http.StatusForbidden, response.Forbidden(err.Error(), nil))
		case strings.Contains(err.Error(), "maximum number"):
			c.JSON(http.StatusConflict, response.Conflict(err.Error(), nil))
		case strings.Contains(err.Error(), "invalid mode"):
			c.JSON(http.StatusBadRequest, response.BadRequest(err.Error(), nil))
		default:
			logger.LogBusinessOperation(c, "api-keys", "create", "api_key", "", false, err)
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to create api key", nil))
		}
		return
	}

	logger.LogBusinessOperation(c, "api-keys", "create", "api_key", key.ID, true, nil)
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

	keys, err := local.NewAPIKeysService().ListAPIKeys(user.ID)
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
	err := local.NewAPIKeysService().RevokeAPIKey(user.ID, keyID)
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
	c.JSON(http.StatusOK, response.OK("api key revoked successfully", nil))
}
