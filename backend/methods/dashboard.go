/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/middleware"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/dashboard"
)

// Injected so the handler tests never reach the model SDK.
var (
	generateDashboard       = dashboard.Generate
	dashboardCatalogForUser = dashboard.CatalogResponse
)

// GetDashboardCatalog returns the widgets the caller is allowed to see, so the
// customization wizard offers only topics that would actually render something.
func GetDashboardCatalog(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user not found in context", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("dashboard catalog retrieved successfully", dashboardCatalogForUser(user)))
}

// GenerateDashboard picks and orders the widgets for the caller's dashboard.
//
// It answers 200 even when the model is unreachable or incoherent: the service
// falls back to a deterministic selection over the same catalog and says so in
// generated_by, so the frontend always has a dashboard to render.
func GenerateDashboard(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user not found in context", nil))
		return
	}

	var req models.DashboardGenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			middleware.PayloadTooLarge(c, "request body too large")
			return
		}
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	c.JSON(http.StatusOK, response.OK("dashboard generated successfully", generateDashboard(c.Request.Context(), user, req)))
}
