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
	"strings"
	"testing"
)

func TestHashSystemSecret(t *testing.T) {
	tests := []struct {
		name   string
		secret string
	}{
		{
			name:   "simple secret",
			secret: "mysecret123",
		},
		{
			name:   "long secret",
			secret: "this-is-a-very-long-secret-with-many-characters-0123456789",
		},
		{
			name:   "special characters",
			secret: "secret!@#$%^&*(){}[]|\\:;\"'<>,.?/~`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashSystemSecret(tt.secret)
			if err != nil {
				t.Fatalf("HashSystemSecret() error = %v", err)
			}

			// Verify hash format
			if !strings.HasPrefix(hash, "$argon2id$") {
				t.Errorf("hash does not start with $argon2id$, got: %s", hash)
			}

			// Verify hash has correct number of parts
			parts := strings.Split(hash, "$")
			if len(parts) != 6 {
				t.Errorf("hash has incorrect format, expected 6 parts, got %d", len(parts))
			}
		})
	}
}

func TestHashSystemSecretUniqueness(t *testing.T) {
	secret := "test-secret"

	// Generate two hashes of the same secret
	hash1, err := HashSystemSecret(secret)
	if err != nil {
		t.Fatalf("HashSystemSecret() error = %v", err)
	}

	hash2, err := HashSystemSecret(secret)
	if err != nil {
		t.Fatalf("HashSystemSecret() error = %v", err)
	}

	// Hashes should be different due to different salts
	if hash1 == hash2 {
		t.Error("two hashes of the same secret should be different due to unique salts")
	}
}

func TestVerifySystemSecret(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		wrongSecret   string
		expectSuccess bool
	}{
		{
			name:          "correct secret",
			secret:        "correct-secret",
			wrongSecret:   "",
			expectSuccess: true,
		},
		{
			name:          "wrong secret",
			secret:        "correct-secret",
			wrongSecret:   "wrong-secret",
			expectSuccess: false,
		},
		{
			name:          "case sensitive",
			secret:        "Secret123",
			wrongSecret:   "secret123",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate hash
			hash, err := HashSystemSecret(tt.secret)
			if err != nil {
				t.Fatalf("HashSystemSecret() error = %v", err)
			}

			// Verify with correct secret
			if tt.expectSuccess {
				valid, err := VerifySystemSecret(tt.secret, hash)
				if err != nil {
					t.Fatalf("VerifySystemSecret() error = %v", err)
				}
				if !valid {
					t.Error("VerifySystemSecret() should return true for correct secret")
				}
			} else {
				// Verify with wrong secret
				valid, err := VerifySystemSecret(tt.wrongSecret, hash)
				if err != nil {
					t.Fatalf("VerifySystemSecret() error = %v", err)
				}
				if valid {
					t.Error("VerifySystemSecret() should return false for wrong secret")
				}
			}
		})
	}
}

func TestVerifySystemSecretInvalidHash(t *testing.T) {
	tests := []struct {
		name        string
		hash        string
		expectError bool
	}{
		{
			name:        "empty hash",
			hash:        "",
			expectError: true,
		},
		{
			name:        "malformed hash - too few parts",
			hash:        "$argon2id$v=19$m=65536",
			expectError: true,
		},
		{
			name:        "malformed hash - wrong algorithm",
			hash:        "$sha256$v=19$m=65536,t=3,p=2$salt$hash",
			expectError: true,
		},
		{
			name:        "malformed hash - invalid base64 salt",
			hash:        "$argon2id$v=19$m=65536,t=3,p=2$invalid!!!$hash",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := VerifySystemSecret("any-secret", tt.hash)
			if tt.expectError {
				if err == nil {
					t.Error("VerifySystemSecret() should return error for invalid hash")
				}
				if valid {
					t.Error("VerifySystemSecret() should return false for invalid hash")
				}
			}
		})
	}
}

func TestHashSystemSecretLength(t *testing.T) {
	secret := "test-secret"

	hash, err := HashSystemSecret(secret)
	if err != nil {
		t.Fatalf("HashSystemSecret() error = %v", err)
	}

	// Argon2id hash should be well under the 512 character database limit
	if len(hash) > 512 {
		t.Errorf("hash length %d exceeds database limit of 512 characters", len(hash))
	}

	// Typical Argon2id hash should be around 100-150 characters
	if len(hash) < 80 || len(hash) > 200 {
		t.Logf("Warning: hash length %d is outside expected range (80-200)", len(hash))
	}
}
