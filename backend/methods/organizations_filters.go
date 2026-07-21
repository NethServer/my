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
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// creatorFilterOption is one selectable creator in the filters dropdown. It
// carries the creator's org (name + id) alongside the user so the UI can
// disambiguate homonyms — two different people with the same name but in
// different organizations — mirroring the "Created by" table column.
type creatorFilterOption struct {
	UserID           string `json:"user_id"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
}

// toCreatorOptions maps the creator snapshots returned by the repository to the
// filter dropdown shape (one entry per user, already deduplicated and sorted by
// name). The org is attached only to label homonyms in the UI.
func toCreatorOptions(creators []models.OrgCreator) []creatorFilterOption {
	out := make([]creatorFilterOption, 0, len(creators))
	for _, cr := range creators {
		out = append(out, creatorFilterOption{
			UserID:           cr.UserID,
			Name:             cr.Name,
			Email:            cr.Email,
			OrganizationID:   cr.OrganizationID,
			OrganizationName: cr.OrganizationName,
		})
	}
	return out
}

// GetDistributorFilters handles GET /api/filters/distributors - returns the
// distinct creators of the distributors visible to the caller, for the
// created_by filter on GET /api/distributors.
func GetDistributorFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	service := local.NewOrganizationService()
	userOrgRole := strings.ToLower(user.OrgRole)
	creators, err := service.ListDistributorCreators(userOrgRole, user.OrganizationID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve distributor filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve distributor filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("distributor filters retrieved successfully", gin.H{
		"created_by": toCreatorOptions(creators),
	}))
}

// GetResellerFilters handles GET /api/filters/resellers - returns the distinct
// creators of the resellers visible to the caller.
func GetResellerFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	service := local.NewOrganizationService()
	userOrgRole := strings.ToLower(user.OrgRole)
	creators, err := service.ListResellerCreators(userOrgRole, user.OrganizationID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve reseller filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve reseller filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("reseller filters retrieved successfully", gin.H{
		"created_by": toCreatorOptions(creators),
	}))
}

// GetCustomerFilters handles GET /api/filters/customers - returns the distinct
// creators of the customers visible to the caller.
func GetCustomerFilters(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	service := local.NewOrganizationService()
	userOrgRole := strings.ToLower(user.OrgRole)
	creators, err := service.ListCustomerCreators(userOrgRole, user.OrganizationID)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve customer filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve customer filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("customer filters retrieved successfully", gin.H{
		"created_by": toCreatorOptions(creators),
	}))
}
