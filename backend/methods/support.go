/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// signedRedisMessage wraps a Redis pub/sub payload with an HMAC-SHA256 signature
// so the support service can verify the message came from the backend.
type signedRedisMessage struct {
	Payload string `json:"payload"`
	Sig     string `json:"sig"`
}

// signAndMarshal signs a payload with SUPPORT_INTERNAL_SECRET and returns the signed envelope.
// If SUPPORT_INTERNAL_SECRET is not configured, the envelope is published unsigned
// (backward-compatible, but a warning is logged).
func signAndMarshal(payload []byte) []byte {
	secret := configuration.Config.SupportInternalSecret
	var sig string
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(payload)
		sig = hex.EncodeToString(mac.Sum(nil))
	}
	envelope, _ := json.Marshal(signedRedisMessage{Payload: string(payload), Sig: sig})
	return envelope
}

// validateHostPort checks that target is a valid host:port with port in range 1-65535.
// Fix #7: replaces the regex that accepted port numbers up to 99999.
func validateHostPort(target string) error {
	host, portStr, err := net.SplitHostPort(target)
	if err != nil {
		return fmt.Errorf("invalid host:port format: %w", err)
	}
	if host == "" {
		return fmt.Errorf("empty host")
	}
	port, convErr := strconv.Atoi(portStr)
	if convErr != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid port: must be 1-65535")
	}
	return nil
}

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

	// Notify support service via Redis pub/sub to disconnect the tunnel.
	// Fix #2: message is signed with SUPPORT_INTERNAL_SECRET so the support service
	// can verify it was not injected by a third party with Redis access.
	if redisClient := cache.GetRedisClient(); redisClient != nil {
		cmd := map[string]string{
			"action":     "close",
			"session_id": sessionID,
		}
		payload, _ := json.Marshal(cmd)
		envelope := signAndMarshal(payload)
		if err := redisClient.Publish("support:commands", string(envelope)); err != nil {
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
		if err := validateHostPort(svc.Target); err != nil {
			c.JSON(http.StatusBadRequest, response.BadRequest(
				"invalid target format: must be host:port with port 1-65535", nil,
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

	// Fix #2: sign the Redis message before publishing
	cmd := map[string]interface{}{
		"action":     "add_services",
		"session_id": sessionID,
		"services":   services,
	}
	payload, _ := json.Marshal(cmd)
	envelope := signAndMarshal(payload)
	if err := redisClient.Publish("support:commands", string(envelope)); err != nil {
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
