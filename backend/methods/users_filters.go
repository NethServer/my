/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// GetUserFilters handles GET /api/filters/users - returns the roles available
// to assign users in the caller's scope and the distinct creators of the users
// visible to the caller, for the created_by filter on GET /api/users.
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

	repo := entities.NewLocalUserRepository()
	creators, err := repo.ListCreators(strings.ToLower(user.OrgRole), user.OrganizationID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve user created-by filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve user filters", nil))
		return
	}

	// Same option shape as the organization filters (one entry per user, org
	// attached to label homonyms in the UI). Already sorted by name.
	createdBy := make([]creatorFilterOption, 0, len(creators))
	for _, cr := range creators {
		createdBy = append(createdBy, creatorFilterOption{
			UserID:           cr.UserID,
			Name:             cr.Name,
			Email:            cr.Email,
			OrganizationID:   cr.OrganizationID,
			OrganizationName: cr.OrganizationName,
		})
	}

	c.JSON(http.StatusOK, response.OK("user filters retrieved successfully", gin.H{
		"roles":      helpers.EnsureSlice(roles),
		"created_by": createdBy,
	}))
}
