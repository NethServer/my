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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	"github.com/nethesis/my/backend/services/logto"
)

// =============================================================================
// CONSENT MANAGEMENT ENDPOINTS
// =============================================================================

// EnableImpersonationConsent allows a user to enable consent for being impersonated
// POST /api/impersonate/consent
func EnableImpersonationConsent(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse request body
	var req models.EnableConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.RequestLogger(c, "impersonate").Warn().
			Err(err).
			Str("operation", "enable_consent_parse_request").
			Str("user_id", user.ID).
			Msg("Invalid enable consent request JSON")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// Enable consent
	consent, err := impersonationService.EnableConsent(user.ID, req.DurationHours)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "enable_consent").
			Str("user_id", user.ID).
			Int("duration_hours", req.DurationHours).
			Msg("Failed to enable impersonation consent")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to enable consent: "+err.Error(),
			nil,
		))
		return
	}

	// Log consent enabling
	logger.RequestLogger(c, "impersonate").Info().
		Str("operation", "consent_enabled").
		Str("user_id", user.ID).
		Str("username", user.Username).
		Str("consent_id", consent.ID).
		Int("duration_hours", req.DurationHours).
		Time("expires_at", consent.ExpiresAt).
		Msg("User enabled impersonation consent")

	c.JSON(http.StatusOK, response.OK(
		"impersonation consent enabled successfully",
		gin.H{
			"consent": consent,
		},
	))
}

// DisableImpersonationConsent allows a user to disable consent for being impersonated
// DELETE /api/impersonate/consent
func DisableImpersonationConsent(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// Disable consent
	err := impersonationService.DisableConsent(user.ID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "disable_consent").
			Str("user_id", user.ID).
			Msg("Failed to disable impersonation consent")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to disable consent: "+err.Error(),
			nil,
		))
		return
	}

	// Log consent disabling
	logger.RequestLogger(c, "impersonate").Info().
		Str("operation", "consent_disabled").
		Str("user_id", user.ID).
		Str("username", user.Username).
		Msg("User disabled impersonation consent")

	c.JSON(http.StatusOK, response.OK(
		"impersonation consent disabled successfully",
		nil,
	))
}

// GetImpersonationConsentStatus gets the current consent status for the user
// GET /api/impersonate/consent
func GetImpersonationConsentStatus(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// Get consent status
	consent, err := impersonationService.GetConsentStatus(user.ID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_consent_status").
			Str("user_id", user.ID).
			Msg("Failed to get impersonation consent status")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to get consent status: "+err.Error(),
			nil,
		))
		return
	}

	if consent == nil {
		c.JSON(http.StatusOK, response.OK(
			"no active impersonation consent",
			gin.H{
				"consent": nil,
			},
		))
		return
	}

	c.JSON(http.StatusOK, response.OK(
		"consent status retrieved successfully",
		gin.H{
			"consent": consent,
		},
	))
}

// =============================================================================
// IMPERSONATION ENDPOINTS (Updated with Consent Validation)
// =============================================================================

// ImpersonateUserWithConsent allows owner users to impersonate another user (only if consent is active)
// POST /api/impersonate
func ImpersonateUserWithConsent(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check if user is already impersonating someone (via impersonation token)
	isImpersonated, exists := c.Get("is_impersonated")
	if exists && isImpersonated.(bool) {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "impersonate_chaining_prevented").
			Str("user_id", user.ID).
			Msg("User attempted to impersonate while already impersonating")

		c.JSON(http.StatusForbidden, response.Forbidden(
			"cannot impersonate while already impersonating another user. exit current impersonation first.",
			nil,
		))
		return
	}

	// Check if this Owner already has an active impersonation session (via Redis)
	sessionManager := cache.NewImpersonationSessionManager()
	hasActiveSession, err := sessionManager.HasActiveSession(user.ID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "check_active_session_failed").
			Str("user_id", user.ID).
			Msg("Failed to check for active impersonation session - allowing request (fail open)")
		// Continue with fail-open approach for infrastructure issues
	} else if hasActiveSession {
		// Get session details for better error message
		activeSession, _ := sessionManager.GetActiveSession(user.ID)

		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "impersonate_duplicate_session_prevented").
			Str("user_id", user.ID).
			Str("existing_session_id", func() string {
				if activeSession != nil {
					return activeSession.SessionID
				}
				return "unknown"
			}()).
			Msg("User attempted to start impersonation while having an active session")

		c.JSON(http.StatusConflict, response.Conflict(
			"you already have an active impersonation session. exit the current session before starting a new one.",
			gin.H{
				"active_session": activeSession,
			},
		))
		return
	}

	// Only owner organization users can impersonate
	if user.OrgRole != "Owner" {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "impersonate_unauthorized").
			Str("user_id", user.ID).
			Str("org_role", user.OrgRole).
			Msg("Non-owner user attempted impersonation")

		c.JSON(http.StatusForbidden, response.Forbidden(
			"only owner users can impersonate other users",
			nil,
		))
		return
	}

	// Parse request body
	var req ImpersonateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.RequestLogger(c, "impersonate").Warn().
			Err(err).
			Str("operation", "impersonate_parse_request").
			Str("user_id", user.ID).
			Msg("Invalid impersonate request JSON")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// Check if target user has active consent
	canBeImpersonated, err := impersonationService.CanBeImpersonated(req.UserID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "check_consent").
			Str("user_id", user.ID).
			Str("target_user_id", req.UserID).
			Msg("Failed to check impersonation consent")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to check impersonation consent: "+err.Error(),
			nil,
		))
		return
	}

	if !canBeImpersonated {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "impersonate_no_consent").
			Str("user_id", user.ID).
			Str("target_user_id", req.UserID).
			Msg("Attempted to impersonate user without active consent")

		c.JSON(http.StatusForbidden, response.Forbidden(
			"target user has not provided consent for impersonation or consent has expired",
			nil,
		))
		return
	}

	// Get target user information for impersonation
	targetUser, err := logto.GetUserForImpersonation(req.UserID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "impersonate_get_target_user").
			Str("user_id", user.ID).
			Str("target_user_id", req.UserID).
			Msg("Failed to get target user for impersonation")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"target user not found or inaccessible: "+err.Error(),
			nil,
		))
		return
	}

	// Prevent self-impersonation
	if targetUser.ID == user.ID || (user.LogtoID != nil && targetUser.LogtoID != nil && *user.LogtoID == *targetUser.LogtoID) {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "impersonate_self_attempt").
			Str("user_id", user.ID).
			Str("target_user_id", req.UserID).
			Msg("User attempted to impersonate themselves")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"cannot impersonate yourself",
			nil,
		))
		return
	}

	// Get consent details to determine token duration
	consent, err := impersonationService.GetConsentStatus(req.UserID)
	if err != nil || consent == nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_consent_details").
			Str("user_id", user.ID).
			Str("target_user_id", req.UserID).
			Msg("Failed to get consent details for impersonation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to get consent details: "+err.Error(),
			nil,
		))
		return
	}

	// Generate session ID for audit tracking
	sessionID := uuid.New().String()

	// Generate impersonation token with custom duration from consent
	impersonationDuration := time.Duration(consent.MaxDurationMinutes) * time.Minute
	impersonationToken, err := jwt.GenerateImpersonationTokenWithDuration(*targetUser, *user, sessionID, impersonationDuration)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "impersonate_generate_token").
			Str("user_id", user.ID).
			Str("target_user_id", req.UserID).
			Str("session_id", sessionID).
			Msg("Failed to generate impersonation token")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate impersonation token: "+err.Error(),
			nil,
		))
		return
	}

	// Create active session in Redis to prevent duplicate impersonation
	err = sessionManager.CreateSession(user.ID, sessionID, targetUser.ID, impersonationDuration)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Warn().
			Err(err).
			Str("operation", "create_active_session_failed").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Failed to create active session - continuing (fail open)")
		// Don't fail the request for Redis issues, but log the problem
	}

	// Log session start in audit
	auditEntry := &models.ImpersonationAuditEntry{
		SessionID:            sessionID,
		ImpersonatorUserID:   user.ID,
		ImpersonatedUserID:   targetUser.ID,
		ActionType:           "session_start",
		ImpersonatorUsername: user.Username,
		ImpersonatedUsername: targetUser.Username,
	}

	err = impersonationService.LogImpersonationAction(auditEntry)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Warn().
			Err(err).
			Str("operation", "log_session_start").
			Str("session_id", sessionID).
			Msg("Failed to log impersonation session start")
		// Don't fail the request for logging errors
	}

	// Log successful impersonation start
	logger.RequestLogger(c, "impersonate").Info().
		Str("operation", "impersonation_started_with_consent").
		Str("impersonator_user_id", user.ID).
		Str("impersonator_username", user.Username).
		Str("impersonated_user_id", targetUser.ID).
		Str("impersonated_username", targetUser.Username).
		Str("session_id", sessionID).
		Str("impersonator_org", user.OrganizationID).
		Str("impersonated_org", targetUser.OrganizationID).
		Int("max_duration_minutes", consent.MaxDurationMinutes).
		Msg("User impersonation started successfully with consent")

	c.JSON(http.StatusOK, response.OK(
		"impersonation started successfully",
		gin.H{
			"token":             impersonationToken,
			"expires_in":        consent.MaxDurationMinutes * 60, // Convert to seconds
			"expires_at":        consent.ExpiresAt,               // ISO timestamp when consent expires
			"session_id":        sessionID,
			"impersonated_user": targetUser,
			"impersonator":      user,
			"is_impersonated":   true,
			"max_duration":      consent.MaxDurationMinutes,
		},
	))
}

// ExitImpersonationWithAudit allows user to exit impersonation mode and logs the session end
// DELETE /api/impersonate
func ExitImpersonationWithAudit(c *gin.Context) {
	// Check if this is an impersonation token
	isImpersonated, exists := c.Get("is_impersonated")
	logger.RequestLogger(c, "impersonate").Debug().
		Bool("is_impersonated_exists", exists).
		Interface("is_impersonated_value", isImpersonated).
		Msg("ExitImpersonation - checking impersonation state")

	if !exists || !isImpersonated.(bool) {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "exit_impersonation_not_impersonating").
			Msg("Exit impersonation called without active impersonation")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"not currently impersonating a user",
			nil,
		))
		return
	}

	// Get session ID from context (if available)
	sessionID, _ := c.Get("session_id")
	sessionIDStr := ""
	if sessionID != nil {
		sessionIDStr = sessionID.(string)
	}

	// Get impersonator information
	impersonator, exists := c.Get("impersonated_by")
	if !exists || impersonator == nil {
		logger.RequestLogger(c, "impersonate").Error().
			Str("operation", "exit_impersonation_missing_impersonator").
			Bool("exists", exists).
			Interface("impersonator", impersonator).
			Msg("Impersonation context missing impersonator information")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"invalid impersonation state",
			nil,
		))
		return
	}

	impersonatorUser, ok := impersonator.(*models.User)
	if !ok || impersonatorUser == nil {
		logger.RequestLogger(c, "impersonate").Error().
			Str("operation", "exit_impersonation_invalid_impersonator_type").
			Interface("impersonator_type", impersonator).
			Msg("Invalid impersonator type in context")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"invalid impersonation state",
			nil,
		))
		return
	}

	impersonatedUser, _ := helpers.GetUserFromContext(c)

	// Clear active session in Redis
	sessionManager := cache.NewImpersonationSessionManager()
	if sessionIDStr != "" {
		err := sessionManager.ClearSessionByID(impersonatorUser.ID, sessionIDStr)
		if err != nil {
			logger.RequestLogger(c, "impersonate").Warn().
				Err(err).
				Str("operation", "clear_active_session_failed").
				Str("impersonator_user_id", impersonatorUser.ID).
				Str("session_id", sessionIDStr).
				Msg("Failed to clear active session - continuing (fail open)")
			// Don't fail the request for Redis issues, but log the problem
		}
	} else {
		// If no session ID, try to clear any active session for this impersonator
		err := sessionManager.ClearSession(impersonatorUser.ID)
		if err != nil {
			logger.RequestLogger(c, "impersonate").Warn().
				Err(err).
				Str("operation", "clear_active_session_fallback_failed").
				Str("impersonator_user_id", impersonatorUser.ID).
				Msg("Failed to clear active session (fallback) - continuing (fail open)")
		}
	}

	// Log session end in audit (if session ID is available)
	if sessionIDStr != "" {
		impersonationService := local.NewImpersonationService()
		auditEntry := &models.ImpersonationAuditEntry{
			SessionID:            sessionIDStr,
			ImpersonatorUserID:   impersonatorUser.ID,
			ImpersonatedUserID:   impersonatedUser.ID,
			ActionType:           "session_end",
			ImpersonatorUsername: impersonatorUser.Username,
			ImpersonatedUsername: impersonatedUser.Username,
		}

		err := impersonationService.LogImpersonationAction(auditEntry)
		if err != nil {
			logger.RequestLogger(c, "impersonate").Warn().
				Err(err).
				Str("operation", "log_session_end").
				Str("session_id", sessionIDStr).
				Msg("Failed to log impersonation session end")
			// Don't fail the request for logging errors
		}
	}

	// Generate new regular token for the original user
	newToken, err := jwt.GenerateCustomToken(*impersonatorUser)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "exit_impersonation_generate_token").
			Str("impersonator_user_id", impersonatorUser.ID).
			Msg("Failed to generate token for exiting impersonation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate exit token: "+err.Error(),
			nil,
		))
		return
	}

	// Generate refresh token for the original user
	refreshToken, err := jwt.GenerateRefreshToken(*impersonatorUser.LogtoID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "exit_impersonation_generate_refresh_token").
			Str("impersonator_user_id", impersonatorUser.ID).
			Msg("Failed to generate refresh token for exiting impersonation")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate refresh token: "+err.Error(),
			nil,
		))
		return
	}

	// Calculate expiration in seconds for regular token
	expDuration := 24 * time.Hour // Regular token duration
	expiresIn := int64(expDuration.Seconds())

	// Log successful impersonation exit
	logger.RequestLogger(c, "impersonate").Info().
		Str("operation", "impersonation_exited_with_audit").
		Str("impersonator_user_id", impersonatorUser.ID).
		Str("impersonator_username", impersonatorUser.Username).
		Str("impersonated_user_id", impersonatedUser.ID).
		Str("impersonated_username", impersonatedUser.Username).
		Str("session_id", sessionIDStr).
		Msg("User successfully exited impersonation with audit")

	c.JSON(http.StatusOK, response.OK(
		"impersonation ended successfully",
		gin.H{
			"token":         newToken,
			"refresh_token": refreshToken,
			"expires_in":    expiresIn,
		},
	))
}

// =============================================================================
// AUDIT ENDPOINTS
// =============================================================================

// GetImpersonationAudit retrieves audit history for the current user (being impersonated)
// GET /api/impersonate/audit
func GetImpersonationAudit(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 1000 {
		limit = 50
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// Get audit history
	entries, total, err := impersonationService.GetUserAuditHistory(user.ID, limit, offset)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_audit_history").
			Str("user_id", user.ID).
			Msg("Failed to get impersonation audit history")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to get audit history: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, response.OK(
		"audit history retrieved successfully",
		gin.H{
			"entries":  entries,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
			"has_more": offset+len(entries) < total,
		},
	))
}
