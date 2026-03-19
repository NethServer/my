/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"encoding/json"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// validHostPort matches a host:port string (IPv4, IPv6, or hostname with port).
var validHostPort = regexp.MustCompile(`^(?:[a-zA-Z0-9._\-\[\]]+):\d{1,5}$`)

// GetSupportSessions handles GET /api/support-sessions
// Returns support sessions grouped by system with server-side pagination.
func GetSupportSessions(c *gin.Context) {
	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	page, pageSize, sortBy, sortDirection := helpers.GetPaginationAndSortingFromQuery(c)

	status := c.Query("status")
	systemID := c.Query("system_id")

	repo := entities.NewSupportRepository()
	groups, totalCount, err := repo.GetSystemSessions(
		userOrgRole, userOrgID, page, pageSize, status, systemID, sortBy, sortDirection,
	)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve support sessions")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve support sessions", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("support sessions retrieved successfully", gin.H{
		"support_sessions": helpers.EnsureSlice(groups),
		"pagination":       helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, sortBy, sortDirection),
	}))
}

// GetSupportSession handles GET /api/support-sessions/:id
func GetSupportSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)

	repo := entities.NewSupportRepository()
	session, err := repo.GetSessionByID(sessionID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support session", nil))
		return
	}
	if session == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("support session retrieved successfully", session))
}

// ExtendSupportSession handles PATCH /api/support-sessions/:id/extend
func ExtendSupportSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	var request models.ExtendSessionRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	repo := entities.NewSupportRepository()
	if err := repo.ExtendSession(sessionID, request.Hours); err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to extend support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to extend support session", nil))
		return
	}

	logger.LogBusinessOperation(c, "support", "extend", "session", sessionID, true, nil)

	c.JSON(http.StatusOK, response.OK("support session extended successfully", gin.H{
		"session_id":        sessionID,
		"extended_by_hours": request.Hours,
	}))
}

// CloseSupportSession handles DELETE /api/support-sessions/:id
func CloseSupportSession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	repo := entities.NewSupportRepository()
	if err := repo.CloseSession(sessionID); err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to close support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to close support session", nil))
		return
	}

	// Notify support service via Redis pub/sub to disconnect the tunnel
	if redisClient := cache.GetRedisClient(); redisClient != nil {
		cmd := map[string]string{
			"action":     "close",
			"session_id": sessionID,
		}
		payload, _ := json.Marshal(cmd)
		if err := redisClient.Publish("support:commands", string(payload)); err != nil {
			logger.Warn().Err(err).Str("session_id", sessionID).Msg("failed to publish close command to support service")
		}
	}

	logger.LogBusinessOperation(c, "support", "close", "session", sessionID, true, nil)

	c.JSON(http.StatusOK, response.OK("support session closed successfully", gin.H{
		"session_id": sessionID,
	}))
}

// GetSupportSessionLogs handles GET /api/support-sessions/:id/logs
func GetSupportSessionLogs(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	page, pageSize, _, _ := helpers.GetPaginationAndSortingFromQuery(c)

	repo := entities.NewSupportRepository()
	logs, totalCount, err := repo.GetAccessLogs(sessionID, page, pageSize)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get access logs")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get access logs", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("access logs retrieved successfully", gin.H{
		"access_logs": helpers.EnsureSlice(logs),
		"pagination":  helpers.BuildPaginationInfoWithSorting(page, pageSize, totalCount, "connected_at", "desc"),
	}))
}

// GetSupportSessionDiagnostics handles GET /api/support-sessions/:id/diagnostics
func GetSupportSessionDiagnostics(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)

	repo := entities.NewSupportRepository()
	data, at, err := repo.GetDiagnostics(sessionID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get diagnostics")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get diagnostics", nil))
		return
	}

	c.JSON(http.StatusOK, response.OK("diagnostics retrieved successfully", gin.H{
		"session_id":     sessionID,
		"diagnostics":    data,
		"diagnostics_at": at,
	}))
}

// AddSupportSessionServices handles POST /api/support-sessions/:id/services
// It sends an add_services command to the tunnel-client via Redis pub/sub,
// dynamically injecting static services into the running tunnel without reconnection.
func AddSupportSessionServices(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("session id required", nil))
		return
	}

	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)

	// RBAC: verify session belongs to the caller's scope
	repo := entities.NewSupportRepository()
	sess, err := repo.GetSessionByID(sessionID, userOrgRole, userOrgID)
	if err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to get support session")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to get support session", nil))
		return
	}
	if sess == nil {
		c.JSON(http.StatusNotFound, response.NotFound("support session not found", nil))
		return
	}
	if sess.Status != "active" {
		c.JSON(http.StatusConflict, response.Conflict("session is not active", nil))
		return
	}

	var request models.AddSessionServicesRequest
	if err := c.ShouldBindBodyWith(&request, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	// Validate each service entry
	services := make(map[string]interface{})
	for _, svc := range request.Services {
		if !validServiceName.MatchString(svc.Name) {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				"invalid service name: use alphanumeric characters, dots, hyphens only", nil,
			))
			return
		}
		if !validHostPort.MatchString(svc.Target) {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				"invalid target format: must be host:port", nil,
			))
			return
		}
		services[svc.Name] = map[string]interface{}{
			"target": svc.Target,
			"label":  svc.Label,
			"tls":    svc.TLS,
		}
	}

	// Publish add_services command via Redis pub/sub
	redisClient := cache.GetRedisClient()
	if redisClient == nil {
		c.JSON(http.StatusServiceUnavailable, response.InternalServerError("redis not available", nil))
		return
	}

	cmd := map[string]interface{}{
		"action":     "add_services",
		"session_id": sessionID,
		"services":   services,
	}
	payload, _ := json.Marshal(cmd)
	if err := redisClient.Publish("support:commands", string(payload)); err != nil {
		logger.Error().Err(err).Str("session_id", sessionID).Msg("failed to publish add_services command")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to send command to support service", nil))
		return
	}

	logger.LogBusinessOperation(c, "support", "add_services", "session", sessionID, true, nil)

	c.JSON(http.StatusOK, response.OK("services added successfully", gin.H{
		"session_id": sessionID,
		"count":      len(request.Services),
	}))
}
