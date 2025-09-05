/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/logger"
)

const (
	// StateBlacklistTTL is how long to keep used state tokens in blacklist (24 hours)
	StateBlacklistTTL = 24 * time.Hour
	// StateBlacklistPrefix is the Redis key prefix for used state tokens
	StateBlacklistPrefix = "used_state_token:"
	// StateTokenExpirationTime is the maximum age for a state token (1 hour)
	StateTokenExpirationTime = 1 * time.Hour
	// StateTokenPrefix is the expected prefix for state tokens
	StateTokenPrefix = "state_"
)

// StateTokenData represents the structure of a time-based state token
type StateTokenData struct {
	Timestamp int64  `json:"timestamp"`
	Random    string `json:"random"`
}

// ValidateAndBlacklistStateToken validates a time-based state token and adds it to blacklist (one-shot use)
func ValidateAndBlacklistStateToken(state string) error {
	// Basic length validation
	if len(state) < 10 {
		return fmt.Errorf("state token too short")
	}

	// Validate state token format and extract timestamp
	tokenData, err := extractStateTokenData(state)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("component", "state_validation").
			Str("state", state).
			Msg("Invalid state token format")
		return fmt.Errorf("invalid state token format")
	}

	// Check if token is expired (older than 1 hour)
	tokenTime := time.Unix(0, tokenData.Timestamp*int64(time.Millisecond))
	if time.Since(tokenTime) > StateTokenExpirationTime {
		logger.Warn().
			Str("component", "state_validation").
			Str("state", state).
			Time("token_time", tokenTime).
			Dur("age", time.Since(tokenTime)).
			Msg("State token expired (older than 1 hour)")
		return fmt.Errorf("state token expired")
	}

	blacklistKey := StateBlacklistPrefix + state

	// Check if token is already in blacklist (already used)
	exists, err := cache.GetRedisClient().Exists(blacklistKey)
	if err != nil {
		logger.Error().
			Err(err).
			Str("component", "state_validation").
			Str("state", state).
			Msg("Failed to check state blacklist in Redis")
		return fmt.Errorf("failed to validate state token")
	}

	if exists {
		logger.Warn().
			Str("component", "state_validation").
			Str("state", state).
			Msg("State token already used (found in blacklist)")
		return fmt.Errorf("state token already used")
	}

	// Add token to blacklist with TTL
	err = cache.GetRedisClient().Set(blacklistKey, "used", StateBlacklistTTL)
	if err != nil {
		logger.Error().
			Err(err).
			Str("component", "state_validation").
			Str("state", state).
			Msg("Failed to add state token to blacklist")
		return fmt.Errorf("failed to consume state token")
	}

	logger.Info().
		Str("component", "state_validation").
		Str("state", state).
		Time("token_time", tokenTime).
		Dur("token_age", time.Since(tokenTime)).
		Dur("blacklist_ttl", StateBlacklistTTL).
		Msg("State token validated and blacklisted successfully")

	return nil
}

// extractStateTokenData extracts and validates the timestamp and random data from a time-based state token
func extractStateTokenData(state string) (*StateTokenData, error) {
	// Validate prefix
	if !strings.HasPrefix(state, StateTokenPrefix) {
		return nil, fmt.Errorf("state token missing required prefix")
	}

	// Extract base64 encoded part
	encodedData := strings.TrimPrefix(state, StateTokenPrefix)
	if len(encodedData) == 0 {
		return nil, fmt.Errorf("state token missing data section")
	}

	// Decode base64
	jsonData, err := base64.StdEncoding.DecodeString(encodedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state token: %w", err)
	}

	// Parse JSON
	var tokenData StateTokenData
	if err := json.Unmarshal(jsonData, &tokenData); err != nil {
		return nil, fmt.Errorf("failed to parse state token JSON: %w", err)
	}

	// Validate required fields
	if tokenData.Timestamp == 0 {
		return nil, fmt.Errorf("state token missing timestamp")
	}
	if len(tokenData.Random) == 0 {
		return nil, fmt.Errorf("state token missing random data")
	}

	return &tokenData, nil
}
