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
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/repositories"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services"
)

// GetSystemsTotals returns the total count of systems and their liveness status based on heartbeat data
// Respects RBAC hierarchy - users only see totals for systems they can access
func GetSystemsTotals(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Get query parameter for timeout (default 15 minutes)
	timeoutStr := c.DefaultQuery("timeout", "15")
	timeoutMinutes, err := strconv.Atoi(timeoutStr)
	if err != nil || timeoutMinutes <= 0 {
		logger.Warn().
			Str("component", "totals").
			Str("operation", "parse_timeout").
			Str("timeout", timeoutStr).
			Err(err).
			Msg("invalid timeout format, using default")
		timeoutMinutes = 15
	}

	// Use local repository for fast totals
	repo := repositories.NewLocalSystemRepository()
	totals, err := repo.GetTotals(strings.ToLower(userOrgRole), userOrgID, timeoutMinutes)
	if err != nil {
		logger.Error().
			Str("component", "totals").
			Str("operation", "get_systems").
			Err(err).
			Msg("failed to retrieve systems totals")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve systems totals", nil))
		return
	}

	// Convert to response format
	result := map[string]interface{}{
		"total":           totals.Total,
		"alive":           totals.Alive,
		"dead":            totals.Dead,
		"zombie":          totals.Zombie,
		"timeout_minutes": totals.TimeoutMinutes,
	}

	logger.Info().
		Str("component", "totals").
		Str("operation", "systems_totals").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("total", totals.Total).
		Int("alive", totals.Alive).
		Int("dead", totals.Dead).
		Int("zombie", totals.Zombie).
		Int("timeout_minutes", totals.TimeoutMinutes).
		Msg("systems totals retrieved")

	c.JSON(http.StatusOK, response.OK("systems totals retrieved", result))
}

// GetDistributorsTotals returns the total count of distributors accessible to the user
func GetDistributorsTotals(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Use local repository for fast totals
	repo := repositories.NewLocalDistributorRepository()
	count, err := repo.GetTotals(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		logger.Error().
			Str("component", "totals").
			Str("operation", "get_distributors").
			Err(err).
			Msg("failed to retrieve distributors totals")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve distributors totals", nil))
		return
	}

	result := map[string]interface{}{
		"total": count,
	}

	logger.Info().
		Str("component", "totals").
		Str("operation", "distributors_totals").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("total", count).
		Msg("distributors totals retrieved")

	c.JSON(http.StatusOK, response.OK("distributors totals retrieved", result))
}

// GetResellersTotals returns the total count of resellers accessible to the user
func GetResellersTotals(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	// Use local repository for fast totals
	repo := repositories.NewLocalResellerRepository()
	count, err := repo.GetTotals(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		logger.Error().
			Str("component", "totals").
			Str("operation", "get_resellers").
			Err(err).
			Msg("failed to retrieve resellers totals")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve resellers totals", nil))
		return
	}

	result := map[string]interface{}{
		"total": count,
	}

	logger.Info().
		Str("component", "totals").
		Str("operation", "resellers_totals").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("total", count).
		Msg("resellers totals retrieved")

	c.JSON(http.StatusOK, response.OK("resellers totals retrieved", result))
}

// GetCustomersTotals returns the total count of customers accessible to the user
func GetCustomersTotals(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	repo := repositories.NewLocalCustomerRepository()
	count, err := repo.GetTotals(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		logger.Error().Str("component", "totals").Str("operation", "get_customers").Err(err).Msg("failed to retrieve customers totals")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve customers totals", nil))
		return
	}

	result := map[string]interface{}{"total": count}
	logger.Info().Str("component", "totals").Str("operation", "customers_totals").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("total", count).Msg("customers totals retrieved")
	c.JSON(http.StatusOK, response.OK("customers totals retrieved", result))
}

// GetAccountsTotals returns the total count of user accounts accessible to the user
func GetAccountsTotals(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("User context required", nil))
		return
	}

	service := services.NewLocalUserService()
	count, err := service.GetTotals(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		logger.Error().Str("component", "totals").Str("operation", "get_accounts").Err(err).Msg("failed to retrieve accounts totals")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve accounts totals", nil))
		return
	}

	result := map[string]interface{}{"total": count}
	logger.Info().Str("component", "totals").Str("operation", "accounts_totals").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("total", count).Msg("accounts totals retrieved")
	c.JSON(http.StatusOK, response.OK("accounts totals retrieved", result))
}
