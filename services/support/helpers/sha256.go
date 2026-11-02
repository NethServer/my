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
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

// VerifySystemSecretSHA256 verifies a system secret against a salted SHA256 hash.
// Expected format: hex_salt:hex_hash
func VerifySystemSecretSHA256(secret, encodedHash string) (bool, error) {
	parts := strings.SplitN(encodedHash, ":", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid sha256 hash format")
	}

	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	actualHash := sha256.Sum256(append(salt, []byte(secret)...))
	if subtle.ConstantTimeCompare(actualHash[:], expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}
