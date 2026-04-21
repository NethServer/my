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

// getAuthenticatedSystemKey extracts the system_key from the gin context
// set by BasicAuthMiddleware. The key is the stable user-facing identifier
// (NETH-XXXX-…) used both for BasicAuth and for the S3 object prefix, so
// operators reading a raw bucket listing can tie each object back to a
// recognisable system without an extra DB lookup.
func getAuthenticatedSystemKey(c *gin.Context) (string, bool) {
	systemKey, exists := c.Get("system_key")
	if !exists {
		return "", false
	}
	systemKeyStr, ok := systemKey.(string)
	if !ok {
		return "", false
	}
	return systemKeyStr, true
}
