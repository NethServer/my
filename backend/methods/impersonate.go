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
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/jwt"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/local"
	"github.com/nethesis/my/backend/services/logto"
)

// =============================================================================
// IMPERSONATION ENDPOINTS
// =============================================================================

// ImpersonateUser lets a user holding impersonate:users act as another user,
// for configuration.Config.ImpersonationSessionDuration. The target's
// consent is not required: every session is audited instead.
// POST /api/impersonate
func ImpersonateUser(c *gin.Context) {
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
		// Get session details to check if it's expired
		activeSession, _ := sessionManager.GetActiveSession(user.ID)

		// If session is expired, clean it up automatically
		if activeSession != nil && time.Now().After(activeSession.ExpiresAt) {
			logger.RequestLogger(c, "impersonate").Info().
				Str("operation", "cleanup_expired_session").
				Str("user_id", user.ID).
				Str("session_id", activeSession.SessionID).
				Time("expired_at", activeSession.ExpiresAt).
				Msg("Cleaning up expired impersonation session")

			err := sessionManager.ClearSession(user.ID)
			if err != nil {
				logger.RequestLogger(c, "impersonate").Warn().
					Err(err).
					Str("operation", "cleanup_expired_session_failed").
					Str("user_id", user.ID).
					Msg("Failed to clean up expired session - continuing")
			}
			// Continue with the impersonation request since session was expired
		} else {
			// Session is still active - prevent duplicate
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

	// Get target user information for impersonation (req.UserID is a Logto ID)
	targetUser, err := logto.GetUserForImpersonation(req.UserID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "impersonate_get_target_user").
			Str("user_id", user.ID).
			Str("target_user_logto_id", req.UserID).
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
			Str("target_user_logto_id", req.UserID).
			Msg("User attempted to impersonate themselves")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"cannot impersonate yourself",
			nil,
		))
		return
	}

	// Privilege ceiling: the break-glass account (user role Owner) is never a
	// target. Acting as it would hand a Staff user destroy:systems|users and
	// the Owner-organization membership gates that exist to contain Staff.
	if models.HasOwnerUserRole(targetUser.UserRoles) {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "impersonate_owner_role_target_refused").
			Str("user_id", user.ID).
			Str("target_user_id", targetUser.ID).
			Msg("Attempt to impersonate an Owner-role account refused")
		c.JSON(http.StatusForbidden, response.Forbidden(
			"an Owner-role account cannot be impersonated",
			nil,
		))
		return
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	sessionDuration := configuration.Config.ImpersonationSessionDuration
	expiresAt := time.Now().Add(sessionDuration)

	// Generate session ID for audit tracking
	sessionID := uuid.New().String()

	impersonationToken, err := jwt.GenerateImpersonationTokenWithDuration(*targetUser, *user, sessionID, sessionDuration)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "impersonate_generate_token").
			Str("user_id", user.ID).
			Str("target_user_id", targetUser.ID).
			Str("session_id", sessionID).
			Msg("Failed to generate impersonation token")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate impersonation token: "+err.Error(),
			nil,
		))
		return
	}

	// Create active session in Redis to prevent duplicate impersonation
	// Store logto_id as ImpersonatedUserID for consistency with GetUserForImpersonation
	err = sessionManager.CreateSession(user.ID, sessionID, *targetUser.LogtoID, sessionDuration)
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
		ImpersonatorUserID:   helpers.GetEffectiveUserID(user),
		ImpersonatedUserID:   targetUser.ID,
		ActionType:           "session_start",
		ImpersonatorUsername: user.Username,
		ImpersonatedUsername: targetUser.Username,
		ImpersonatorName:     user.Name,
		ImpersonatedName:     targetUser.Name,
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
		Str("operation", "impersonation_started").
		Str("impersonator_user_id", user.ID).
		Str("impersonator_username", user.Username).
		Str("impersonated_user_id", targetUser.ID).
		Str("impersonated_username", targetUser.Username).
		Str("session_id", sessionID).
		Str("impersonator_org", user.OrganizationID).
		Str("impersonated_org", targetUser.OrganizationID).
		Int64("duration_seconds", int64(sessionDuration.Seconds())).
		Time("expires_at", expiresAt).
		Msg("User impersonation started successfully")

	c.JSON(http.StatusOK, response.OK(
		"impersonation started successfully",
		gin.H{
			"is_impersonating":  true,
			"token":             impersonationToken,
			"expires_in":        int64(sessionDuration.Seconds()),
			"expires_at":        expiresAt,
			"session_id":        sessionID,
			"impersonated_user": targetUser,
			"impersonator":      user,
		},
	))
}

// ExtendImpersonation restarts the current impersonation session for a full
// configuration.Config.ImpersonationSessionDuration: same session, new token.
// Called with the impersonation token itself; API keys never reach it
// (RejectAPIKey on the group). The impersonator is re-read from its current
// state, so a user who lost impersonate:users cannot keep a session alive; the
// new token still names the impersonator as the session was started, since
// its id keys the session in Redis and the audit trail.
// POST /api/impersonate/extend
func ExtendImpersonation(c *gin.Context) {
	isImpersonated, exists := c.Get("is_impersonated")
	if !exists || !isImpersonated.(bool) {
		c.JSON(http.StatusBadRequest, response.BadRequest("not currently impersonating a user", nil))
		return
	}

	sessionIDStr, _ := c.Get("session_id")
	sessionID, _ := sessionIDStr.(string)
	impersonator, _ := c.Get("impersonated_by")
	impersonatorUser, ok := impersonator.(*models.User)
	impersonatedUser, hasTarget := helpers.GetUserFromContext(c)
	if !hasTarget {
		return // GetUserFromContext already answered
	}
	if !ok || impersonatorUser == nil || sessionID == "" ||
		impersonatorUser.LogtoID == nil || impersonatedUser.LogtoID == nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid impersonation state", nil))
		return
	}

	freshImpersonator, err := local.ResolveUserByLogtoID(*impersonatorUser.LogtoID)
	if err != nil {
		if errors.Is(err, local.ErrUserInactive) {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("account suspended or deleted", nil))
			return
		}
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "extend_impersonation_resolve_impersonator").
			Str("impersonator_user_id", impersonatorUser.ID).
			Msg("Failed to resolve impersonator for extending impersonation")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to extend impersonation", nil))
		return
	}
	if !HasPermission(freshImpersonator.UserPermissions, freshImpersonator.OrgPermissions, "impersonate:users") {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "extend_impersonation_permission_lost").
			Str("impersonator_user_id", impersonatorUser.ID).
			Str("session_id", sessionID).
			Msg("Impersonator no longer holds impersonate:users")
		c.JSON(http.StatusForbidden, response.Forbidden("insufficient permissions to impersonate users", nil))
		return
	}

	// The target is re-read too: it may have been suspended, deleted or given
	// the Owner user role since the session started, and the new token must
	// carry its current permissions.
	freshTarget, err := logto.GetUserForImpersonation(*impersonatedUser.LogtoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("target user not found or inaccessible", nil))
		return
	}
	if models.HasOwnerUserRole(freshTarget.UserRoles) {
		c.JSON(http.StatusForbidden, response.Forbidden("an Owner-role account cannot be impersonated", nil))
		return
	}

	sessionDuration := configuration.Config.ImpersonationSessionDuration
	expiresAt := time.Now().Add(sessionDuration)

	sessionManager := cache.NewImpersonationSessionManager()
	if err := sessionManager.ExtendSession(impersonatorUser.ID, sessionID, sessionDuration); err != nil {
		if errors.Is(err, cache.ErrImpersonationSessionNotFound) {
			c.JSON(http.StatusBadRequest, response.BadRequest("impersonation session expired or ended", nil))
			return
		}
		logger.RequestLogger(c, "impersonate").Warn().
			Err(err).
			Str("operation", "extend_active_session_failed").
			Str("session_id", sessionID).
			Msg("Failed to extend active session - continuing (fail open)")
	}

	impersonationToken, err := jwt.GenerateImpersonationTokenWithDuration(*freshTarget, *impersonatorUser, sessionID, sessionDuration)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "extend_impersonation_generate_token").
			Str("session_id", sessionID).
			Msg("Failed to generate impersonation token")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to extend impersonation", nil))
		return
	}

	// The old token would otherwise stay valid until its own expiry
	if presented := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "); presented != "" {
		if blErr := cache.GetTokenBlacklist().BlacklistToken(presented, "impersonation extended"); blErr != nil {
			logger.RequestLogger(c, "impersonate").Warn().
				Err(blErr).
				Str("operation", "extend_impersonation_blacklist_failed").
				Str("session_id", sessionID).
				Msg("Failed to revoke the previous impersonation token")
		}
	}

	auditEntry := &models.ImpersonationAuditEntry{
		SessionID:            sessionID,
		ImpersonatorUserID:   helpers.GetEffectiveUserID(impersonatorUser),
		ImpersonatedUserID:   freshTarget.ID,
		ActionType:           "session_extended",
		ImpersonatorUsername: impersonatorUser.Username,
		ImpersonatedUsername: freshTarget.Username,
		ImpersonatorName:     impersonatorUser.Name,
		ImpersonatedName:     freshTarget.Name,
	}
	if err := local.NewImpersonationService().LogImpersonationAction(auditEntry); err != nil {
		logger.RequestLogger(c, "impersonate").Warn().
			Err(err).
			Str("operation", "log_session_extended").
			Str("session_id", sessionID).
			Msg("Failed to log impersonation session extension")
	}

	logger.RequestLogger(c, "impersonate").Info().
		Str("operation", "impersonation_extended").
		Str("impersonator_user_id", impersonatorUser.ID).
		Str("impersonated_user_id", freshTarget.ID).
		Str("session_id", sessionID).
		Time("expires_at", expiresAt).
		Msg("Impersonation session extended")

	c.JSON(http.StatusOK, response.OK(
		"impersonation extended successfully",
		gin.H{
			"is_impersonating":  true,
			"token":             impersonationToken,
			"expires_in":        int64(sessionDuration.Seconds()),
			"expires_at":        expiresAt,
			"session_id":        sessionID,
			"impersonated_user": freshTarget,
			"impersonator":      impersonatorUser,
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

	// The impersonation token dies here: it would otherwise stay valid for the
	// rest of the session (up to a week), leaving a usable target identity in
	// the caller's hands. The presented token is the one to burn.
	if presented := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "); presented != "" {
		if blErr := cache.GetTokenBlacklist().BlacklistToken(presented, "impersonation exited"); blErr != nil {
			logger.RequestLogger(c, "impersonate").Warn().
				Err(blErr).
				Str("operation", "exit_impersonation_blacklist_failed").
				Str("session_id", sessionIDStr).
				Msg("Failed to revoke impersonation token on exit")
		}
	}

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
			ImpersonatorUserID:   helpers.GetEffectiveUserID(impersonatorUser),
			ImpersonatedUserID:   impersonatedUser.ID,
			ActionType:           "session_end",
			ImpersonatorUsername: impersonatorUser.Username,
			ImpersonatedUsername: impersonatedUser.Username,
			ImpersonatorName:     impersonatorUser.Name,
			ImpersonatedName:     impersonatedUser.Name,
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

	// The impersonator's own session is re-minted from its CURRENT state, not
	// from the claims frozen in the impersonation token: roles may have
	// changed and the account may have been suspended or deleted meanwhile.
	if impersonatorUser.LogtoID == nil || *impersonatorUser.LogtoID == "" {
		c.JSON(http.StatusInternalServerError, response.InternalServerError("invalid impersonation state", nil))
		return
	}
	freshImpersonator, err := local.ResolveUserByLogtoID(*impersonatorUser.LogtoID)
	if err != nil {
		if errors.Is(err, local.ErrUserInactive) {
			logger.RequestLogger(c, "impersonate").Warn().
				Str("operation", "exit_impersonation_inactive_impersonator").
				Str("impersonator_user_id", impersonatorUser.ID).
				Msg("Impersonator account is no longer active; no session re-issued")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("account suspended or deleted", nil))
			return
		}
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "exit_impersonation_resolve_impersonator").
			Str("impersonator_user_id", impersonatorUser.ID).
			Msg("Failed to resolve impersonator for exiting impersonation")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to restore session", nil))
		return
	}
	impersonatorUser = freshImpersonator

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
	expDuration, err := time.ParseDuration(configuration.Config.JWTExpiration)
	if err != nil {
		expDuration = 30 * time.Minute // mirrors the configuration default
	}
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
			"user":          impersonatorUser,
		},
	))
}

// GetImpersonationStatus checks if user is currently impersonating and returns session info
// GET /api/impersonate/status
func GetImpersonationStatus(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Check if this is an impersonation token first
	isImpersonated, exists := c.Get("is_impersonated")
	if exists && isImpersonated.(bool) {
		// This is an impersonation token - return session details from token
		sessionID, _ := c.Get("session_id")
		impersonator, _ := c.Get("impersonated_by")

		sessionIDStr := ""
		if sessionID != nil {
			sessionIDStr = sessionID.(string)
		}

		var impersonatorUser *models.User
		if impersonator != nil {
			if imp, ok := impersonator.(*models.User); ok {
				impersonatorUser = imp
			}
		}

		c.JSON(http.StatusOK, response.OK(
			"currently impersonating",
			gin.H{
				"is_impersonating":  true,
				"session_id":        sessionIDStr,
				"impersonated_user": user,
				"impersonator":      impersonatorUser,
			},
		))
		return
	}

	// This is a regular token - check if user has an active impersonation session
	sessionManager := cache.NewImpersonationSessionManager()
	activeSession, err := sessionManager.GetActiveSession(user.ID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_impersonation_status").
			Str("user_id", user.ID).
			Msg("Failed to check for active impersonation session")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to check impersonation status: "+err.Error(),
			nil,
		))
		return
	}

	if activeSession == nil {
		// No active impersonation session
		c.JSON(http.StatusOK, response.OK(
			"not currently impersonating",
			gin.H{
				"is_impersonating": false,
			},
		))
		return
	}

	// User has an active impersonation session - get impersonated user details
	impersonatedUser, err := logto.GetUserForImpersonation(activeSession.ImpersonatedUserID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_impersonated_user_details").
			Str("user_id", user.ID).
			Str("impersonated_user_id", activeSession.ImpersonatedUserID).
			Str("session_id", activeSession.SessionID).
			Msg("Failed to get impersonated user details")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to get impersonated user details: "+err.Error(),
			nil,
		))
		return
	}

	// Calculate remaining duration for the impersonation token
	remainingDuration := time.Until(activeSession.ExpiresAt)
	if remainingDuration <= 0 {
		// Session has expired - clean it up
		logger.RequestLogger(c, "impersonate").Info().
			Str("operation", "expired_session_cleanup").
			Str("user_id", user.ID).
			Str("session_id", activeSession.SessionID).
			Time("expired_at", activeSession.ExpiresAt).
			Msg("Impersonation session expired during status check - cleaning up")

		err := sessionManager.ClearSession(user.ID)
		if err != nil {
			logger.RequestLogger(c, "impersonate").Warn().
				Err(err).
				Str("user_id", user.ID).
				Msg("Failed to clear expired session")
		}

		c.JSON(http.StatusOK, response.OK(
			"not currently impersonating",
			gin.H{
				"is_impersonating": false,
			},
		))
		return
	}

	// Generate a new impersonation token for the active session
	impersonationToken, err := jwt.GenerateImpersonationTokenWithDuration(*impersonatedUser, *user, activeSession.SessionID, remainingDuration)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "generate_impersonation_token").
			Str("user_id", user.ID).
			Str("session_id", activeSession.SessionID).
			Msg("Failed to generate impersonation token for active session")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to generate impersonation token: "+err.Error(),
			nil,
		))
		return
	}

	// Calculate expires_in in seconds
	expiresIn := int64(remainingDuration.Seconds())

	logger.RequestLogger(c, "impersonate").Info().
		Str("operation", "impersonation_status_with_token").
		Str("user_id", user.ID).
		Str("impersonated_user_id", impersonatedUser.ID).
		Str("session_id", activeSession.SessionID).
		Int64("expires_in_seconds", expiresIn).
		Msg("Returning active impersonation status with token")

	c.JSON(http.StatusOK, response.OK(
		"currently impersonating",
		gin.H{
			"is_impersonating":  true,
			"session_id":        activeSession.SessionID,
			"impersonated_user": impersonatedUser,
			"impersonator":      user,
			"expires_at":        activeSession.ExpiresAt,
			"created_at":        activeSession.CreatedAt,
			"token":             impersonationToken,
			"expires_in":        expiresIn,
		},
	))
}

// =============================================================================
// AUDIT ENDPOINTS
// =============================================================================

// GetImpersonationSessions retrieves all impersonation sessions for current user
// GET /api/impersonate/sessions
func GetImpersonationSessions(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Parse pagination parameters
	page, pageSize := helpers.GetPaginationFromQuery(c)

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// Get sessions
	sessions, total, err := impersonationService.GetUserSessions(user.ID, page, pageSize)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_sessions").
			Str("user_id", user.ID).
			Msg("Failed to get impersonation sessions")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to get sessions: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, response.OK(
		"sessions retrieved successfully",
		gin.H{
			"sessions":   sessions,
			"pagination": helpers.BuildPaginationInfo(page, pageSize, total),
		},
	))
}

// GetImpersonationSession retrieves details for a specific impersonation session
// GET /api/impersonate/sessions/:session_id
func GetImpersonationSession(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get session_id parameter
	sessionID := c.Param("session_id")
	if sessionID == "" {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "get_session_details").
			Str("user_id", user.ID).
			Msg("Missing session_id parameter")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"session_id parameter is required",
			nil,
		))
		return
	}

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// First, check if the session exists in the database at all
	sessionExists, err := impersonationService.SessionExists(sessionID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "check_session_exists").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Failed to check if session exists")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to check session existence: "+err.Error(),
			nil,
		))
		return
	}

	// If session doesn't exist, return 404
	if !sessionExists {
		logger.RequestLogger(c, "impersonate").Debug().
			Str("operation", "session_not_found").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Session not found in database")

		c.JSON(http.StatusNotFound, response.NotFound(
			"session not found",
			nil,
		))
		return
	}

	// Session exists - now verify that it belongs to the current user
	sessions, _, err := impersonationService.GetUserSessions(user.ID, 1, 1000)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "verify_session_ownership").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Failed to verify session ownership")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to verify session ownership: "+err.Error(),
			nil,
		))
		return
	}

	// Find the specific session
	var session *models.ImpersonationSession
	for _, s := range sessions {
		if s.SessionID == sessionID {
			session = &s
			break
		}
	}

	// Session exists but doesn't belong to user - return 403
	if session == nil {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "session_access_denied").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("User attempted to access session that doesn't belong to them")

		c.JSON(http.StatusForbidden, response.Forbidden(
			"access denied: this session belongs to another user",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, response.OK(
		"session details retrieved successfully",
		gin.H{
			"session": session,
		},
	))
}

// GetSessionAudit retrieves audit history for a specific impersonation session
// GET /api/impersonate/sessions/:session_id/audit
func GetSessionAudit(c *gin.Context) {
	// Get current user context
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	// Get session_id parameter
	sessionID := c.Param("session_id")
	if sessionID == "" {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "get_session_audit").
			Str("user_id", user.ID).
			Msg("Missing session_id parameter")

		c.JSON(http.StatusBadRequest, response.BadRequest(
			"session_id parameter is required",
			nil,
		))
		return
	}

	// Parse pagination parameters
	page, pageSize := helpers.GetPaginationFromQuery(c)

	// Create impersonation service
	impersonationService := local.NewImpersonationService()

	// First, check if the session exists in the database at all
	sessionExists, err := impersonationService.SessionExists(sessionID)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "check_session_exists").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Failed to check if session exists")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to check session existence: "+err.Error(),
			nil,
		))
		return
	}

	// If session doesn't exist, return 200 with empty list
	if !sessionExists {
		logger.RequestLogger(c, "impersonate").Debug().
			Str("operation", "session_not_found").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Session not found in database")

		c.JSON(http.StatusOK, response.OK(
			"session audit retrieved successfully",
			gin.H{
				"session_id": sessionID,
				"entries":    []models.ImpersonationAuditEntry{},
				"pagination": helpers.BuildPaginationInfo(page, pageSize, 0),
			},
		))
		return
	}

	// Session exists - now verify that it belongs to the current user
	sessions, _, err := impersonationService.GetUserSessions(user.ID, 1, 1000) // Get all sessions (page 1, large page size)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "verify_session_ownership").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Failed to verify session ownership")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to verify session ownership: "+err.Error(),
			nil,
		))
		return
	}

	// Check if the session belongs to this user
	var sessionOwned bool
	for _, session := range sessions {
		if session.SessionID == sessionID {
			sessionOwned = true
			break
		}
	}

	// Session exists but doesn't belong to user - return 403
	if !sessionOwned {
		logger.RequestLogger(c, "impersonate").Warn().
			Str("operation", "session_access_denied").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("User attempted to access session that doesn't belong to them")

		c.JSON(http.StatusForbidden, response.Forbidden(
			"access denied: this session belongs to another user",
			nil,
		))
		return
	}

	// Get session audit history with pagination
	entries, total, err := impersonationService.GetSessionAuditHistory(sessionID, page, pageSize)
	if err != nil {
		logger.RequestLogger(c, "impersonate").Error().
			Err(err).
			Str("operation", "get_session_audit").
			Str("user_id", user.ID).
			Str("session_id", sessionID).
			Msg("Failed to get session audit history")

		c.JSON(http.StatusInternalServerError, response.InternalServerError(
			"failed to get session audit: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, response.OK(
		"session audit retrieved successfully",
		gin.H{
			"session_id": sessionID,
			"entries":    entries,
			"pagination": helpers.BuildPaginationInfo(page, pageSize, total),
		},
	))
}
