/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package helpers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/response"
)

// HandleAccessError handles common entity access errors (not found, access denied, generic).
// Returns true if the error was handled and the caller should return.
func HandleAccessError(c *gin.Context, err error, entityType, entityID string) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()

	if errMsg == entityType+" not found" {
		c.JSON(http.StatusNotFound, response.NotFound(entityType+" not found", nil))
		return true
	}

	if strings.Contains(errMsg, "access denied") {
		c.JSON(http.StatusForbidden, response.Forbidden("access denied to "+entityType, map[string]interface{}{
			entityType + "_id": entityID,
		}))
		return true
	}

	c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to process "+entityType+" request", map[string]interface{}{
		"error": errMsg,
	}))
	return true
}
