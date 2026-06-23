/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
)

// APIKeyPrefix marks personal API key tokens. It distinguishes them from custom
// JWTs in the auth middleware and lets secret scanners recognise leaked keys.
const APIKeyPrefix = "myk_"

// GenerateAPIKey builds an opaque token in the format myk_<public>.<secret>.
// Only the public part is stored in clear; the secret part is verified against
// a salted SHA-256 hash (see HashSystemSecretSHA256).
func GenerateAPIKey() (full, public, secret string, err error) {
	public, err = randomHex(15) // 30 hex chars
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate public part: %w", err)
	}
	secret, err = randomHex(30) // 60 hex chars
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate secret part: %w", err)
	}
	full = APIKeyPrefix + public + "." + secret
	return full, public, secret, nil
}

// ParseAPIKey splits a token myk_<public>.<secret> into its public and secret
// parts. It returns an error if the token is not a well-formed API key.
func ParseAPIKey(token string) (public, secret string, err error) {
	if !strings.HasPrefix(token, APIKeyPrefix) {
		return "", "", fmt.Errorf("invalid api key format: missing %q prefix", APIKeyPrefix)
	}
	body := strings.TrimPrefix(token, APIKeyPrefix)
	parts := strings.SplitN(body, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid api key format")
	}
	return parts[0], parts[1], nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
