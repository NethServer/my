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

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
)

// GetSystemsTotals returns the total count of systems and their liveness status based on heartbeat data
// Respects RBAC hierarchy - users only see totals for systems they can access
func GetSystemsTotals(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
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

	// Use service which handles timeout parameter properly
	systemsService := local.NewSystemsService()
	totals, err := systemsService.GetTotals(strings.ToLower(userOrgRole), userOrgID, timeoutMinutes)
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
		"active":          totals.Active,
		"inactive":        totals.Inactive,
		"unknown":         totals.Unknown,
		"timeout_minutes": totals.TimeoutMinutes,
	}

	logger.Info().
		Str("component", "totals").
		Str("operation", "systems_totals").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("total", totals.Total).
		Int("active", totals.Active).
		Int("inactive", totals.Inactive).
		Int("unknown", totals.Unknown).
		Int("timeout_minutes", totals.TimeoutMinutes).
		Msg("systems totals retrieved")

	c.JSON(http.StatusOK, response.OK("systems totals retrieved", result))
}

// GetDistributorsTotals returns the total count of distributors accessible to the user
func GetDistributorsTotals(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Use local repository for fast totals
	repo := entities.NewLocalDistributorRepository()
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
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Use local repository for fast totals
	repo := entities.NewLocalResellerRepository()
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
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	repo := entities.NewLocalCustomerRepository()
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

// GetUsersTotals returns the total count of user accounts accessible to the user
func GetUsersTotals(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	service := local.NewUserService()
	count, err := service.GetTotals(strings.ToLower(userOrgRole), userOrgID)
	if err != nil {
		logger.Error().Str("component", "totals").Str("operation", "get_users").Err(err).Msg("failed to retrieve users totals")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve users totals", nil))
		return
	}

	result := map[string]interface{}{"total": count}
	logger.Info().Str("component", "totals").Str("operation", "users_totals").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("total", count).Msg("users totals retrieved")
	c.JSON(http.StatusOK, response.OK("users totals retrieved", result))
}

// GetSystemsTrend returns trend data for systems over a specified period
func GetSystemsTrend(c *gin.Context) {
	// Get current user context for hierarchical filtering
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get period parameter (default: 7 days)
	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	// Get trend data
	systemsService := local.NewSystemsService()
	trend, err := systemsService.GetSystemsTrend(period, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Str("component", "trend").
			Str("operation", "get_systems_trend").
			Err(err).
			Int("period", period).
			Msg("failed to retrieve systems trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve systems trend", nil))
		return
	}

	logger.Info().
		Str("component", "trend").
		Str("operation", "systems_trend").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("period", period).
		Int("current_total", trend.CurrentTotal).
		Int("delta", trend.Delta).
		Str("trend", trend.Trend).
		Msg("systems trend retrieved")

	c.JSON(http.StatusOK, response.OK("systems trend retrieved successfully", trend))
}

// GetDistributorsTrend returns trend data for distributors over a specified period
func GetDistributorsTrend(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get period parameter (default: 7 days)
	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	// Get trend data from repository
	repo := entities.NewLocalDistributorRepository()
	dataPoints, currentTotal, previousTotal, err := repo.GetTrend(strings.ToLower(userOrgRole), userOrgID, period)
	if err != nil {
		logger.Error().Str("component", "trend").Str("operation", "get_distributors_trend").Err(err).Int("period", period).Msg("failed to retrieve distributors trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve distributors trend", nil))
		return
	}

	// Build response
	delta := currentTotal - previousTotal
	deltaPercentage := 0.0
	if previousTotal > 0 {
		deltaPercentage = (float64(delta) / float64(previousTotal)) * 100
	}

	trend := "stable"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	periodLabel := map[int]string{7: "7 days", 30: "30 days", 180: "180 days", 365: "365 days"}[period]

	response := map[string]interface{}{
		"period":           period,
		"period_label":     periodLabel,
		"current_total":    currentTotal,
		"previous_total":   previousTotal,
		"delta":            delta,
		"delta_percentage": deltaPercentage,
		"trend":            trend,
		"data_points":      dataPoints,
	}

	logger.Info().Str("component", "trend").Str("operation", "distributors_trend").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("period", period).Int("current_total", currentTotal).Int("delta", delta).Str("trend", trend).Msg("distributors trend retrieved")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "distributors trend retrieved successfully", "data": response})
}

// GetResellersTrend returns trend data for resellers over a specified period
func GetResellersTrend(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get period parameter (default: 7 days)
	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	// Get trend data from repository
	repo := entities.NewLocalResellerRepository()
	dataPoints, currentTotal, previousTotal, err := repo.GetTrend(strings.ToLower(userOrgRole), userOrgID, period)
	if err != nil {
		logger.Error().Str("component", "trend").Str("operation", "get_resellers_trend").Err(err).Int("period", period).Msg("failed to retrieve resellers trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve resellers trend", nil))
		return
	}

	// Build response
	delta := currentTotal - previousTotal
	deltaPercentage := 0.0
	if previousTotal > 0 {
		deltaPercentage = (float64(delta) / float64(previousTotal)) * 100
	}

	trend := "stable"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	periodLabel := map[int]string{7: "7 days", 30: "30 days", 180: "180 days", 365: "365 days"}[period]

	response := map[string]interface{}{
		"period":           period,
		"period_label":     periodLabel,
		"current_total":    currentTotal,
		"previous_total":   previousTotal,
		"delta":            delta,
		"delta_percentage": deltaPercentage,
		"trend":            trend,
		"data_points":      dataPoints,
	}

	logger.Info().Str("component", "trend").Str("operation", "resellers_trend").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("period", period).Int("current_total", currentTotal).Int("delta", delta).Str("trend", trend).Msg("resellers trend retrieved")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "resellers trend retrieved successfully", "data": response})
}

// GetCustomersTrend returns trend data for customers over a specified period
func GetCustomersTrend(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get period parameter (default: 7 days)
	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	// Get trend data from repository
	repo := entities.NewLocalCustomerRepository()
	dataPoints, currentTotal, previousTotal, err := repo.GetTrend(strings.ToLower(userOrgRole), userOrgID, period)
	if err != nil {
		logger.Error().Str("component", "trend").Str("operation", "get_customers_trend").Err(err).Int("period", period).Msg("failed to retrieve customers trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve customers trend", nil))
		return
	}

	// Build response
	delta := currentTotal - previousTotal
	deltaPercentage := 0.0
	if previousTotal > 0 {
		deltaPercentage = (float64(delta) / float64(previousTotal)) * 100
	}

	trend := "stable"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	periodLabel := map[int]string{7: "7 days", 30: "30 days", 180: "180 days", 365: "365 days"}[period]

	response := map[string]interface{}{
		"period":           period,
		"period_label":     periodLabel,
		"current_total":    currentTotal,
		"previous_total":   previousTotal,
		"delta":            delta,
		"delta_percentage": deltaPercentage,
		"trend":            trend,
		"data_points":      dataPoints,
	}

	logger.Info().Str("component", "trend").Str("operation", "customers_trend").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("period", period).Int("current_total", currentTotal).Int("delta", delta).Str("trend", trend).Msg("customers trend retrieved")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "customers trend retrieved successfully", "data": response})
}

// GetUsersTrend returns trend data for users over a specified period
func GetUsersTrend(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get period parameter (default: 7 days)
	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	// Get trend data
	userService := local.NewUserService()
	trend, err := userService.GetUsersTrend(period, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().
			Str("component", "trend").
			Str("operation", "get_users_trend").
			Err(err).
			Int("period", period).
			Msg("failed to retrieve users trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve users trend", nil))
		return
	}

	logger.Info().
		Str("component", "trend").
		Str("operation", "users_trend").
		Str("user_org_id", userOrgID).
		Str("user_org_role", userOrgRole).
		Int("period", period).
		Int("current_total", trend.CurrentTotal).
		Int("delta", trend.Delta).
		Str("trend", trend.Trend).
		Msg("users trend retrieved")

	c.JSON(http.StatusOK, response.OK("users trend retrieved successfully", trend))
}

// GetApplicationsTrend returns trend data for applications over a specified period
func GetApplicationsTrend(c *gin.Context) {
	userID, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("user context required", nil))
		return
	}

	// Get period parameter (default: 7 days)
	periodStr := c.DefaultQuery("period", "7")
	period, err := strconv.Atoi(periodStr)
	if err != nil || (period != 7 && period != 30 && period != 180 && period != 365) {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid period parameter (supported: 7, 30, 180, 365)", nil))
		return
	}

	// Get trend data from service
	appsService := local.NewApplicationsService()
	dataPoints, currentTotal, previousTotal, err := appsService.GetApplicationsTrend(strings.ToLower(userOrgRole), userOrgID, period)
	if err != nil {
		logger.Error().Str("component", "trend").Str("operation", "get_applications_trend").Err(err).Int("period", period).Msg("failed to retrieve applications trend")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve applications trend", nil))
		return
	}

	// Build response
	delta := currentTotal - previousTotal
	deltaPercentage := 0.0
	if previousTotal > 0 {
		deltaPercentage = (float64(delta) / float64(previousTotal)) * 100
	}

	trend := "stable"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	periodLabel := map[int]string{7: "7 days", 30: "30 days", 180: "180 days", 365: "365 days"}[period]

	trendResponse := map[string]interface{}{
		"period":           period,
		"period_label":     periodLabel,
		"current_total":    currentTotal,
		"previous_total":   previousTotal,
		"delta":            delta,
		"delta_percentage": deltaPercentage,
		"trend":            trend,
		"data_points":      dataPoints,
	}

	logger.Info().Str("component", "trend").Str("operation", "applications_trend").Str("user_org_id", userOrgID).Str("user_org_role", userOrgRole).Int("period", period).Int("current_total", currentTotal).Int("delta", delta).Str("trend", trend).Msg("applications trend retrieved")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "applications trend retrieved successfully", "data": trendResponse})
}
