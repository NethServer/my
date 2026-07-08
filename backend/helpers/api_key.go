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

// APIKeyOwnerPrefix marks owner-account API keys. These carry owner-wide
// privileges, so they get their own prefix: a leaked token is recognisable as
// high-privilege at a glance (logs, secret scanners), and the middleware can
// route it before any DB lookup. Owner keys are anchored to the Logto ID
// instead of a local users row (the Owner organization has no local users).
const APIKeyOwnerPrefix = "myo_"

// GenerateAPIKey builds an opaque token in the format myk_<public>.<secret>.
// Only the public part is stored in clear; the secret part is verified against
// a salted SHA-256 hash (see HashSystemSecretSHA256).
func GenerateAPIKey() (full, public, secret string, err error) {
	return newAPIKey(APIKeyPrefix)
}

// GenerateOwnerAPIKey builds an owner token in the format myo_<public>.<secret>.
func GenerateOwnerAPIKey() (full, public, secret string, err error) {
	return newAPIKey(APIKeyOwnerPrefix)
}

func newAPIKey(prefix string) (full, public, secret string, err error) {
	public, err = randomHex(15) // 30 hex chars
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate public part: %w", err)
	}
	secret, err = randomHex(30) // 60 hex chars
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate secret part: %w", err)
	}
	full = prefix + public + "." + secret
	return full, public, secret, nil
}

// ParseAPIKey splits a token myk_<public>.<secret> or myo_<public>.<secret>
// into its public and secret parts; owner reports which prefix it carried. It
// returns an error if the token is not a well-formed API key.
func ParseAPIKey(token string) (public, secret string, owner bool, err error) {
	var body string
	switch {
	case strings.HasPrefix(token, APIKeyPrefix):
		body = strings.TrimPrefix(token, APIKeyPrefix)
	case strings.HasPrefix(token, APIKeyOwnerPrefix):
		body = strings.TrimPrefix(token, APIKeyOwnerPrefix)
		owner = true
	default:
		return "", "", false, fmt.Errorf("invalid api key format: missing %q or %q prefix", APIKeyPrefix, APIKeyOwnerPrefix)
	}
	parts := strings.SplitN(body, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false, fmt.Errorf("invalid api key format")
	}
	return parts[0], parts[1], owner, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
