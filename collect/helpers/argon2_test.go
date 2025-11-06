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
	"encoding/base64"
	"fmt"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestVerifySystemSecret(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		hash          string
		expectValid   bool
		expectError   bool
		errorContains string
	}{
		{
			name:          "empty hash",
			secret:        "any-secret",
			hash:          "",
			expectValid:   false,
			expectError:   true,
			errorContains: "invalid hash format",
		},
		{
			name:          "malformed hash - too few parts",
			secret:        "any-secret",
			hash:          "$argon2id$v=19$m=65536",
			expectValid:   false,
			expectError:   true,
			errorContains: "invalid hash format",
		},
		{
			name:          "malformed hash - wrong algorithm",
			secret:        "any-secret",
			hash:          "$sha256$v=19$m=65536,t=3,p=2$salt$hash",
			expectValid:   false,
			expectError:   true,
			errorContains: "unsupported hash algorithm",
		},
		{
			name:          "malformed hash - invalid base64 salt",
			secret:        "any-secret",
			hash:          "$argon2id$v=19$m=65536,t=3,p=2$invalid!!!$hash",
			expectValid:   false,
			expectError:   true,
			errorContains: "failed to decode salt",
		},
		{
			name:          "malformed hash - invalid base64 hash",
			secret:        "any-secret",
			hash:          "$argon2id$v=19$m=65536,t=3,p=2$dGVzdA$invalid!!!",
			expectValid:   false,
			expectError:   true,
			errorContains: "failed to decode hash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := VerifySystemSecret(tt.secret, tt.hash)

			if tt.expectError {
				if err == nil {
					t.Errorf("VerifySystemSecret() expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("VerifySystemSecret() error = %v, should contain %q", err, tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("VerifySystemSecret() unexpected error = %v", err)
				}
			}

			if valid != tt.expectValid {
				t.Errorf("VerifySystemSecret() valid = %v, want %v", valid, tt.expectValid)
			}
		})
	}
}

func TestVerifySystemSecret_EmptySecret(t *testing.T) {
	hash := "$argon2id$v=19$m=65536,t=3,p=2$dGVzdA$dGVzdA"
	valid, err := VerifySystemSecret("", hash)
	if err != nil {
		t.Errorf("VerifySystemSecret() unexpected error = %v", err)
	}
	if valid {
		t.Error("VerifySystemSecret() should return false for empty secret")
	}
}

func TestVerifySystemSecret_WithDynamicHash(t *testing.T) {
	// Generate a hash dynamically for testing
	secret := "test-secret-123"

	// Argon2 parameters (same as in production)
	memory := uint32(64 * 1024)
	iterations := uint32(3)
	parallelism := uint8(2)
	saltLength := 16
	keyLength := uint32(32)

	// Generate salt
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	// Generate hash
	hash := argon2.IDKey([]byte(secret), salt, iterations, memory, parallelism, keyLength)

	// Encode in PHC format
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		memory,
		iterations,
		parallelism,
		b64Salt,
		b64Hash,
	)

	// Test with correct secret
	valid, err := VerifySystemSecret(secret, encodedHash)
	if err != nil {
		t.Fatalf("VerifySystemSecret() unexpected error = %v", err)
	}
	if !valid {
		t.Error("VerifySystemSecret() should return true for correct secret")
	}

	// Test with wrong secret
	valid, err = VerifySystemSecret("wrong-secret", encodedHash)
	if err != nil {
		t.Fatalf("VerifySystemSecret() unexpected error = %v", err)
	}
	if valid {
		t.Error("VerifySystemSecret() should return false for wrong secret")
	}

	// Test case sensitivity
	valid, err = VerifySystemSecret("Test-Secret-123", encodedHash)
	if err != nil {
		t.Fatalf("VerifySystemSecret() unexpected error = %v", err)
	}
	if valid {
		t.Error("VerifySystemSecret() should return false for case-different secret")
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && indexOf(s, substr) >= 0))
}

// indexOf returns the index of substr in s, or -1 if not found
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
