/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/response"
)

// GetAlertFilters handles GET /api/filters/alerts - returns the static alert
// catalog used to populate the "alert name" dropdown in the alerts views.
//
// Systems and organizations dropdowns are populated by the dedicated
// /api/systems and /api/organizations endpoints respectively; severities are
// a fixed enum (critical|warning|info) the UI hardcodes. None of those needed
// a scope-aware aggregation here, which was costing seconds per call.
func GetAlertFilters(c *gin.Context) {
	c.JSON(http.StatusOK, response.OK("alert filters retrieved successfully", gin.H{
		"alerts": AlertCatalog(),
	}))
}
