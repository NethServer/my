/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"github.com/gin-gonic/gin"
)

// getAuthenticatedSystemID extracts the system_id from the gin context set by BasicAuthMiddleware.
// Returns the system ID as a string and true on success, or empty string and false on failure.
func getAuthenticatedSystemID(c *gin.Context) (string, bool) {
	systemID, exists := c.Get("system_id")
	if !exists {
		return "", false
	}
	systemIDStr, ok := systemID.(string)
	if !ok {
		return "", false
	}
	return systemIDStr, true
}
