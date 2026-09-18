/*
 * Copyright (C) 2026 Nethesis S.r.l.
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
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/logto"
)

// Self-service email change, in two steps.
//
// The address on a my account is the identity every SSO consumer trusts: the
// shop and the training site both match their local account on it, and Logto
// stamps every address as verified. Letting a user type any address and have it
// written straight to Logto therefore let a user become, at the next SSO login,
// whoever owned that address on those sites. So a new address is parked, a
// one-time code is mailed to it through Logto, and only the code applies it.
//
// Step 1 is POST /me/change-info with a different email: name and phone are
// applied at once, the email is parked and the answer is 202 with
// email_verification_required. Step 2 is POST /me/change-info/verify-email
// with the code. The verify step never takes the address from the request: the
// address that was mailed is the address that gets written.

// emailShape is the same acceptance test Logto applies to the address it will
// send the code to. Anything else is refused here, before a code is requested.
var emailShape = regexp.MustCompile(`^\S+@\S+\.\S+$`)

// VerifyEmailChangeRequest is the body of POST /me/change-info/verify-email.
type VerifyEmailChangeRequest struct {
	Code string `json:"code" binding:"required"`
}

// normalizeEmail is how two addresses are compared and stored: trimmed and
// lower-cased, the way Logto stores them.
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// changeInfoValidationError is the 400 shape the profile form already knows
// how to render field by field.
func changeInfoValidationError(c *gin.Context, key, message, value string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    400,
		"message": "validation failed",
		"data": gin.H{
			"type": "validation_error",
			"errors": []gin.H{{
				"key":     key,
				"message": message,
				"value":   value,
			}},
		},
	})
}

// startEmailChange parks the address and mails the code. It answers the
// request itself on failure and returns false; on success the caller finishes
// the 202.
func startEmailChange(c *gin.Context, user *models.User, newEmail string) bool {
	log := logger.RequestLogger(c, "auth").With().
		Str("operation", "change_info_email").
		Str("user_id", user.ID).
		Str("logto_id", *user.LogtoID).
		Logger()

	pending, err := cache.GetPendingEmailChange(*user.LogtoID)
	if err != nil {
		log.Error().Err(err).Msg("Pending email change store unavailable")
		c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("email verification temporarily unavailable", nil))
		return false
	}
	if pending != nil && time.Since(pending.RequestedAt) < cache.PendingEmailChangeResendCooldown {
		log.Warn().Msg("Verification code requested again inside the cooldown")
		c.JSON(http.StatusTooManyRequests, response.TooManyRequests(
			"a verification code was sent recently, wait before requesting another",
			gin.H{"retry_after_seconds": int((cache.PendingEmailChangeResendCooldown - time.Since(pending.RequestedAt)).Seconds()) + 1},
		))
		return false
	}

	if err := logto.NewManagementClient().SendEmailVerificationCode(newEmail); err != nil {
		log.Error().Err(err).Msg("Failed to send verification code through Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to send verification code", nil))
		return false
	}

	if err := cache.SetPendingEmailChange(*user.LogtoID, newEmail); err != nil {
		log.Error().Err(err).Msg("Verification code sent but the pending change could not be stored")
		c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("email verification temporarily unavailable", nil))
		return false
	}

	log.Info().Msg("Verification code sent for email change")
	return true
}

// VerifyEmailChange applies a parked email change once the code proves the
// mailbox is the user's.
// POST /api/me/change-info/verify-email
func VerifyEmailChange(c *gin.Context) {
	user, ok := helpers.GetUserFromContext(c)
	if !ok {
		return
	}

	if user.LogtoID == nil || *user.LogtoID == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("profile update not available for this user", nil))
		return
	}

	log := logger.RequestLogger(c, "auth").With().
		Str("operation", "verify_email_change").
		Str("user_id", user.ID).
		Str("logto_id", *user.LogtoID).
		Logger()

	var req VerifyEmailChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" || len(code) > 16 {
		changeInfoValidationError(c, "code", "invalid verification code", "")
		return
	}

	pending, err := cache.GetPendingEmailChange(*user.LogtoID)
	if err != nil {
		log.Error().Err(err).Msg("Pending email change store unavailable")
		c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("email verification temporarily unavailable", nil))
		return
	}
	if pending == nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("no email change waiting for verification", gin.H{"type": "no_pending_email_change"}))
		return
	}

	logtoClient := logto.NewManagementClient()
	if err := logtoClient.VerifyEmailCode(pending.Email, code); err != nil {
		if errors.Is(err, logto.ErrVerificationCodeInvalid) {
			exhausted, recErr := cache.RecordPendingEmailChangeAttempt(*user.LogtoID, pending)
			if recErr != nil {
				log.Error().Err(recErr).Msg("Could not record the failed attempt")
				c.JSON(http.StatusServiceUnavailable, response.ServiceUnavailable("email verification temporarily unavailable", nil))
				return
			}
			log.Warn().Int("attempts", pending.Attempts).Bool("exhausted", exhausted).Msg("Wrong or expired verification code")
			if exhausted {
				c.JSON(http.StatusBadRequest, response.BadRequest(
					"too many wrong codes, request a new one",
					gin.H{"type": "no_pending_email_change"},
				))
				return
			}
			changeInfoValidationError(c, "code", "invalid or expired verification code", "")
			return
		}
		log.Error().Err(err).Msg("Failed to verify the code through Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to verify code", nil))
		return
	}

	newEmail := pending.Email
	if _, err := logtoClient.UpdateUser(*user.LogtoID, models.UpdateUserRequest{PrimaryEmail: &newEmail}); err != nil {
		// Only now, with the mailbox proven, is "already taken" an answer the
		// caller may have: before the code it would have been an oracle on
		// other people's addresses.
		if strings.Contains(err.Error(), "email_already_in_use") {
			_ = cache.DeletePendingEmailChange(*user.LogtoID)
			log.Warn().Msg("Verified address already belongs to another account")
			c.JSON(http.StatusConflict, response.Conflict("email already in use by another account", gin.H{"type": "already_exists", "key": "email"}))
			return
		}
		log.Error().Err(err).Msg("Failed to update the email in Logto")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to update profile", nil))
		return
	}

	// Same mirror as ChangeInfo: Logto is the source of truth and is already
	// updated, the local copy only feeds the lists.
	if err := entities.NewLocalUserRepository().UpdateProfileInfo(*user.LogtoID, nil, &newEmail, nil); err != nil {
		log.Warn().Err(err).Msg("Email updated in Logto but failed to sync to local database")
	}

	if err := cache.DeletePendingEmailChange(*user.LogtoID); err != nil {
		log.Warn().Err(err).Msg("Email applied but the pending record could not be removed; it expires on its own")
	}
	if rc := cache.GetRedisClient(); rc != nil {
		_ = rc.Delete("user_profile:" + *user.LogtoID)
	}

	logger.LogAccountOperation(c, "change_email", user.ID, user.OrganizationID, user.ID, user.OrganizationID, true, nil)
	log.Info().Msg("Email changed after verification")

	c.JSON(http.StatusOK, response.OK("profile updated successfully", gin.H{
		"updated_fields": []string{"email"},
	}))
}
