/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nethesis/my/backend/logger"
)

const (
	// BlacklistKeyPrefix is the Redis key prefix for blacklisted tokens
	BlacklistKeyPrefix = "jwt_blacklist:"
	// UserBlacklistKeyPrefix is the Redis key prefix for user-level blacklists
	UserBlacklistKeyPrefix = "jwt_user_blacklist:"
)

// TokenBlacklist handles JWT token blacklisting operations
type TokenBlacklist struct {
	redisClient *RedisClient
}

// BlacklistData represents the data stored for a blacklisted token
type BlacklistData struct {
	Reason        string `json:"reason"`
	BlacklistedAt int64  `json:"blacklisted_at"`
	TTLSeconds    int64  `json:"ttl_seconds"`
}

// NewTokenBlacklist creates a new TokenBlacklist instance
func NewTokenBlacklist() *TokenBlacklist {
	return &TokenBlacklist{
		redisClient: GetRedisClient(),
	}
}

// hashToken creates a SHA256 hash of the token for storage
// This prevents storing the actual token content in Redis
func (tb *TokenBlacklist) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// BlacklistToken adds a token to the blacklist
func (tb *TokenBlacklist) BlacklistToken(tokenString string, reason string) error {
	if tb.redisClient == nil {
		return fmt.Errorf("redis client unavailable")
	}

	// Parse token to get expiration time
	token, err := jwt.Parse(tokenString, nil)
	if err != nil {
		// Even if parsing fails, we still want to blacklist the token
		logger.ComponentLogger("blacklist").Warn().
			Err(err).
			Str("operation", "parse_token_for_blacklist").
			Msg("Failed to parse token for blacklisting, using default TTL")
	}

	// Calculate TTL based on token expiration
	var ttl time.Duration
	if token != nil && token.Claims != nil {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if exp, ok := claims["exp"].(float64); ok {
				expTime := time.Unix(int64(exp), 0)
				ttl = time.Until(expTime)
				if ttl <= 0 {
					// Token already expired, no need to blacklist
					logger.ComponentLogger("blacklist").Debug().
						Str("operation", "skip_expired_token").
						Time("exp_time", expTime).
						Msg("Skipping blacklist of already expired token")
					return nil
				}
			}
		}
	}

	// Default TTL if we couldn't parse expiration (24 hours)
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	// Hash the token for storage
	tokenHash := tb.hashToken(tokenString)
	key := BlacklistKeyPrefix + tokenHash

	// Store blacklist entry with metadata
	blacklistData := BlacklistData{
		Reason:        reason,
		BlacklistedAt: time.Now().Unix(),
		TTLSeconds:    int64(ttl.Seconds()),
	}

	err = tb.redisClient.Set(key, blacklistData, ttl)
	if err != nil {
		logger.ComponentLogger("blacklist").Error().
			Err(err).
			Str("operation", "blacklist_token").
			Str("key", key).
			Str("reason", reason).
			Msg("Failed to blacklist token")
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	logger.ComponentLogger("blacklist").Info().
		Str("operation", "token_blacklisted").
		Str("reason", reason).
		Dur("ttl", ttl).
		Msg("Token successfully blacklisted")

	return nil
}

// IsTokenBlacklisted checks if a token is blacklisted
func (tb *TokenBlacklist) IsTokenBlacklisted(tokenString string) (bool, string, error) {
	if tb.redisClient == nil {
		logger.ComponentLogger("blacklist").Warn().
			Str("operation", "blacklist_check_failed").
			Msg("Redis client unavailable for blacklist check")
		// Fail open - allow token if Redis is unavailable
		return false, "", fmt.Errorf("redis client unavailable")
	}

	tokenHash := tb.hashToken(tokenString)
	key := BlacklistKeyPrefix + tokenHash

	// Try to get blacklist data
	var blacklistData BlacklistData
	err := tb.redisClient.Get(key, &blacklistData)
	if err != nil {
		// Token not blacklisted (key doesn't exist)
		return false, "", nil
	}

	logger.ComponentLogger("blacklist").Info().
		Str("operation", "token_blacklisted_found").
		Str("reason", blacklistData.Reason).
		Msg("Token found in blacklist")

	return true, blacklistData.Reason, nil
}

// BlacklistAllUserTokens blacklists all tokens for a specific user
func (tb *TokenBlacklist) BlacklistAllUserTokens(userID string, reason string) error {
	if tb.redisClient == nil {
		return fmt.Errorf("redis client unavailable")
	}

	userKey := UserBlacklistKeyPrefix + userID

	// Set user-level blacklist with 7 days TTL (longer than max token lifetime)
	blacklistData := BlacklistData{
		Reason:        reason,
		BlacklistedAt: time.Now().Unix(),
		TTLSeconds:    int64((7 * 24 * time.Hour).Seconds()),
	}

	err := tb.redisClient.Set(userKey, blacklistData, 7*24*time.Hour)
	if err != nil {
		logger.ComponentLogger("blacklist").Error().
			Err(err).
			Str("operation", "blacklist_user_tokens").
			Str("user_id", userID).
			Str("reason", reason).
			Msg("Failed to blacklist user tokens")
		return fmt.Errorf("failed to blacklist user tokens: %w", err)
	}

	logger.ComponentLogger("blacklist").Info().
		Str("operation", "user_tokens_blacklisted").
		Str("user_id", userID).
		Str("reason", reason).
		Msg("All user tokens successfully blacklisted")

	return nil
}

// IsUserBlacklisted checks if all tokens for a user are blacklisted
func (tb *TokenBlacklist) IsUserBlacklisted(userID string) (bool, string, error) {
	if tb.redisClient == nil {
		// Fail open
		return false, "", fmt.Errorf("redis client unavailable")
	}

	userKey := UserBlacklistKeyPrefix + userID

	// Try to get user blacklist data
	var blacklistData BlacklistData
	err := tb.redisClient.Get(userKey, &blacklistData)
	if err != nil {
		// User not blacklisted (key doesn't exist)
		return false, "", nil
	}

	logger.ComponentLogger("blacklist").Info().
		Str("operation", "user_blacklisted_found").
		Str("user_id", userID).
		Str("reason", blacklistData.Reason).
		Msg("User found in blacklist")

	return true, blacklistData.Reason, nil
}

// RemoveUserFromBlacklist removes a user from the blacklist
func (tb *TokenBlacklist) RemoveUserFromBlacklist(userID string) error {
	if tb.redisClient == nil {
		return fmt.Errorf("redis client unavailable")
	}

	userKey := UserBlacklistKeyPrefix + userID

	err := tb.redisClient.Delete(userKey)
	if err != nil {
		logger.ComponentLogger("blacklist").Error().
			Err(err).
			Str("operation", "remove_user_blacklist").
			Str("user_id", userID).
			Msg("Failed to remove user from blacklist")
		return fmt.Errorf("failed to remove user from blacklist: %w", err)
	}

	logger.ComponentLogger("blacklist").Info().
		Str("operation", "user_blacklist_removed").
		Str("user_id", userID).
		Msg("User successfully removed from blacklist")

	return nil
}

// GetGlobalBlacklist returns the global token blacklist instance
var globalBlacklist *TokenBlacklist

func GetTokenBlacklist() *TokenBlacklist {
	if globalBlacklist == nil {
		globalBlacklist = NewTokenBlacklist()
	}
	return globalBlacklist
}
