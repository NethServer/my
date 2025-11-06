/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2 parameters - these provide strong security while maintaining reasonable performance
const (
	// Memory cost in KB (64 MB = 64 * 1024 KB)
	argon2Memory = 64 * 1024
	// Number of iterations
	argon2Iterations = 3
	// Degree of parallelism
	argon2Parallelism = 2
	// Salt length in bytes
	argon2SaltLength = 16
	// Key (hash) length in bytes
	argon2KeyLength = 32
)

// HashSystemSecret generates an Argon2id hash of the system secret
// The hash is stored in the PHC string format: $argon2id$v=19$m=65536,t=3,p=2$salt$hash
func HashSystemSecret(secret string) (string, error) {
	// Generate random salt
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Generate the hash using Argon2id
	hash := argon2.IDKey(
		[]byte(secret),
		salt,
		argon2Iterations,
		argon2Memory,
		argon2Parallelism,
		argon2KeyLength,
	)

	// Encode salt and hash in base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return PHC string format
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

// VerifySystemSecret verifies a system secret against an Argon2id hash
// Returns true if the secret matches the hash, false otherwise
func VerifySystemSecret(secret, encodedHash string) (bool, error) {
	// Parse the encoded hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, fmt.Errorf("invalid hash format")
	}

	// Verify it's an Argon2id hash
	if parts[1] != "argon2id" {
		return false, fmt.Errorf("unsupported hash algorithm: %s", parts[1])
	}

	// Parse parameters
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("failed to parse version: %w", err)
	}

	var memory, iterations uint32
	var parallelism uint8
	var parallelismTmp int
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelismTmp); err != nil {
		return false, fmt.Errorf("failed to parse parameters: %w", err)
	}
	parallelism = uint8(parallelismTmp)

	// Decode salt and hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Generate hash with the same parameters
	actualHash := argon2.IDKey(
		[]byte(secret),
		salt,
		iterations,
		memory,
		parallelism,
		uint32(len(expectedHash)),
	)

	// Compare hashes using constant-time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}
