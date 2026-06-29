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
	"sort"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// orgFiltersScanLimit bounds how many organizations are scanned to build the
// created_by filter options. Organization counts are in the low thousands, so a
// single generous page captures every distinct creator the user can see.
const orgFiltersScanLimit = 100000

// creatorFilterOption is one selectable creator in the filters dropdown,
// mirroring the systems created_by filter shape (user_id, name, email).
type creatorFilterOption struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// distinctCreators collects the unique creators from a page of organizations,
// deduplicated by user_id and sorted by name. Organizations without a creator
// snapshot are skipped.
func distinctCreators[T any](items []T, get func(T) *models.OrgCreator) []creatorFilterOption {
	seen := make(map[string]bool)
	out := make([]creatorFilterOption, 0)
	for _, it := range items {
		cb := get(it)
		if cb == nil || cb.UserID == "" || seen[cb.UserID] {
			continue
		}
		seen[cb.UserID] = true
		out = append(out, creatorFilterOption{UserID: cb.UserID, Name: cb.Name, Email: cb.Email})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
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
	distributors, _, err := service.ListDistributors(userOrgRole, user.OrganizationID, 1, orgFiltersScanLimit, "", "name", "asc", nil, nil)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve distributor filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve distributor filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("distributor filters retrieved successfully", gin.H{
		"created_by": distinctCreators(distributors, func(d *models.LocalDistributor) *models.OrgCreator { return d.CreatedBy }),
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
	resellers, _, err := service.ListResellers(userOrgRole, user.OrganizationID, 1, orgFiltersScanLimit, "", "name", "asc", nil, nil)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve reseller filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve reseller filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("reseller filters retrieved successfully", gin.H{
		"created_by": distinctCreators(resellers, func(r *models.LocalReseller) *models.OrgCreator { return r.CreatedBy }),
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
	customers, _, err := service.ListCustomers(userOrgRole, user.OrganizationID, 1, orgFiltersScanLimit, "", "name", "asc", nil, nil)
	if err != nil {
		logger.Error().Err(err).Str("user_id", user.ID).Msg("Failed to retrieve customer filters")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve customer filters", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("customer filters retrieved successfully", gin.H{
		"created_by": distinctCreators(customers, func(cu *models.LocalCustomer) *models.OrgCreator { return cu.CreatedBy }),
	}))
}
