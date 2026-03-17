/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package ssh

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	glssh "github.com/gliderlabs/ssh"
	"github.com/redis/go-redis/v9"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
	"github.com/nethesis/my/services/ssh-gateway/models"
)

const (
	nonceKeyPrefix  = "ssh_auth:nonce:"
	resultKeyPrefix = "ssh_auth:result:"
	keyAuthPrefix   = "ssh_key_auth:"
	nonceBytes      = 32
)

// AuthHandler manages the keyboard-interactive SSH authentication flow
type AuthHandler struct {
	redis *redis.Client
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(redisClient *redis.Client) *AuthHandler {
	return &AuthHandler{redis: redisClient}
}

// PublicKeyAuth checks if the client's SSH public key has a cached auth result.
// After a successful browser OAuth, the key fingerprint is cached in Redis.
// Subsequent connections with the same key skip the browser prompt.
func (h *AuthHandler) PublicKeyAuth(ctx glssh.Context, key glssh.PublicKey) bool {
	log := logger.ComponentLogger("auth")

	systemKey := ctx.User()
	if systemKey == "" {
		return false
	}

	fingerprint := gossh.FingerprintSHA256(key)
	cacheKey := keyAuthPrefix + systemKey + ":" + fingerprint

	// Store fingerprint in context so KeyboardInteractive can cache it after success
	ctx.SetValue("ssh_pubkey_fingerprint", fingerprint)

	log.Debug().
		Str("system_key", systemKey).
		Str("fingerprint", fingerprint).
		Msg("checking cached public key")

	rctx := context.Background()
	val, err := h.redis.Get(rctx, cacheKey).Result()
	if err != nil {
		log.Debug().Str("system_key", systemKey).Str("fingerprint", fingerprint).Msg("no cached key found")
		return false
	}

	var result models.AuthResult
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return false
	}

	// Store auth result and auth method in session context
	ctx.SetValue("auth_result", &result)
	ctx.SetValue("ssh_auth_method", "cached_key")

	log.Info().
		Str("system_key", systemKey).
		Str("user_id", result.UserID).
		Str("session_id", result.SessionID).
		Str("fingerprint", fingerprint).
		Msg("SSH auth via cached public key")

	return true
}

// KeyboardInteractive handles the keyboard-interactive SSH auth challenge.
// It generates a nonce, displays the browser auth URL, and polls Redis for the result.
// After success, caches the client's SSH public key (if available) for future connections.
func (h *AuthHandler) KeyboardInteractive(ctx glssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
	log := logger.ComponentLogger("auth")

	systemKey := ctx.User()
	if systemKey == "" {
		log.Warn().Msg("empty SSH username")
		return false
	}

	// Generate nonce
	nonce, err := generateNonce()
	if err != nil {
		log.Error().Err(err).Msg("failed to generate nonce")
		return false
	}

	// Store nonce in Redis
	nonceData := models.NonceData{
		SystemKey: systemKey,
		CreatedAt: time.Now(),
	}
	data, err := json.Marshal(nonceData)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal nonce data")
		return false
	}

	rctx := context.Background()
	if err := h.redis.Set(rctx, nonceKeyPrefix+nonce, data, configuration.Config.AuthNonceTTL).Err(); err != nil {
		log.Error().Err(err).Msg("failed to store nonce in Redis")
		return false
	}

	// Display auth URL to the operator
	authURL := fmt.Sprintf("%s/ssh-auth?code=%s", configuration.Config.AppURL, nonce)
	banner := fmt.Sprintf(
		"\nMy Nethesis - SSH Gateway\n"+
			"-------------------------\n\n"+
			"Open this URL to authenticate:\n\n"+
			"  %s\n\n"+
			"Waiting for login...\n",
		authURL,
	)
	_, err = challenger("", banner, nil, nil)
	if err != nil {
		log.Debug().Err(err).Msg("keyboard-interactive challenge send failed")
	}

	// Poll Redis for auth result
	resultKey := resultKeyPrefix + nonce
	deadline := time.Now().Add(configuration.Config.AuthPollTimeout)
	ticker := time.NewTicker(configuration.Config.AuthPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			h.redis.Del(rctx, nonceKeyPrefix+nonce) //nolint:errcheck
			return false
		case <-ticker.C:
			if time.Now().After(deadline) {
				log.Info().Str("system_key", systemKey).Msg("SSH auth timed out")
				h.redis.Del(rctx, nonceKeyPrefix+nonce) //nolint:errcheck
				return false
			}

			val, err := h.redis.GetDel(rctx, resultKey).Result()
			if err == redis.Nil {
				continue
			}
			if err != nil {
				log.Error().Err(err).Msg("failed to check auth result in Redis")
				continue
			}

			var result models.AuthResult
			if err := json.Unmarshal([]byte(val), &result); err != nil {
				log.Error().Err(err).Msg("failed to unmarshal auth result")
				return false
			}

			h.redis.Del(rctx, nonceKeyPrefix+nonce) //nolint:errcheck

			// Show success message
			connectTarget := systemKey
			if result.SystemType != "" {
				connectTarget = fmt.Sprintf("[%s] %s", result.SystemType, systemKey)
			}
			successMsg := fmt.Sprintf(
				"Authenticated as:\n"+
					"  - %s\n"+
					"  - %s\n"+
					"  - %s\n"+
					"Connecting to %s...\n",
				result.Username, result.UserEmail, result.OrganizationName, connectTarget,
			)
			_, _ = challenger("", successMsg, nil, nil)

			// Store auth result in session context
			ctx.SetValue("auth_result", &result)

			// Cache the client's SSH public key for future connections
			h.cachePublicKey(ctx, systemKey, &result)

			log.Info().
				Str("system_key", systemKey).
				Str("user_id", result.UserID).
				Str("session_id", result.SessionID).
				Msg("SSH auth successful via browser")

			return true
		}
	}
}

// cachePublicKey stores the client's SSH public key fingerprint in Redis
// so future connections with the same key skip browser auth.
func (h *AuthHandler) cachePublicKey(ctx glssh.Context, systemKey string, result *models.AuthResult) {
	log := logger.ComponentLogger("auth")

	fp := ctx.Value("ssh_pubkey_fingerprint")
	if fp == nil {
		log.Debug().Str("system_key", systemKey).Msg("no public key fingerprint in context, cannot cache for future auth")
		return
	}

	fingerprint, ok := fp.(string)
	if !ok {
		return
	}

	cacheKey := keyAuthPrefix + systemKey + ":" + fingerprint

	data, err := json.Marshal(result)
	if err != nil {
		return
	}

	rctx := context.Background()
	h.redis.Set(rctx, cacheKey, data, configuration.Config.KeyAuthTTL) //nolint:errcheck

	logger.ComponentLogger("auth").Info().
		Str("system_key", systemKey).
		Str("fingerprint", fingerprint).
		Msg("SSH public key cached for future auth")
}

// generateNonce creates a cryptographically random hex nonce
func generateNonce() (string, error) {
	b := make([]byte, nonceBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
