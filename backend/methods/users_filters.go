/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// GetUserFilters handles GET /api/filters/users - returns the roles available
// to assign users in the caller's scope.
//
// Organizations dropdown is populated by /api/organizations, which supports
// search + pagination and scales to tenants with thousands of rows.
func GetUserFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	roles, err := FetchFilteredRoles(user)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed in user filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve user filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("user filters retrieved successfully", gin.H{
		"roles": helpers.EnsureSlice(roles),
	}))
}
