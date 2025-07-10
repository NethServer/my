/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package initcmd

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSecurePassword(t *testing.T) {
	t.Run("password length", func(t *testing.T) {
		password := GenerateSecurePassword()
		assert.Equal(t, 16, len(password), "Password should be 16 characters long")
	})

	t.Run("password character sets", func(t *testing.T) {
		password := GenerateSecurePassword()

		hasLower := false
		hasUpper := false
		hasDigit := false
		hasSymbol := false

		lowerCase := "abcdefghijklmnopqrstuvwxyz"
		upperCase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits := "0123456789"
		symbols := "!@#$%^&*"

		for _, char := range password {
			charStr := string(char)
			if strings.Contains(lowerCase, charStr) {
				hasLower = true
			}
			if strings.Contains(upperCase, charStr) {
				hasUpper = true
			}
			if strings.Contains(digits, charStr) {
				hasDigit = true
			}
			if strings.Contains(symbols, charStr) {
				hasSymbol = true
			}
		}

		assert.True(t, hasLower, "Password should contain lowercase letters")
		assert.True(t, hasUpper, "Password should contain uppercase letters")
		assert.True(t, hasDigit, "Password should contain digits")
		assert.True(t, hasSymbol, "Password should contain symbols")
	})

	t.Run("password uniqueness", func(t *testing.T) {
		passwords := make(map[string]bool)

		// Generate multiple passwords and check they're unique
		for i := 0; i < 100; i++ {
			password := GenerateSecurePassword()
			assert.False(t, passwords[password], "Passwords should be unique")
			passwords[password] = true
		}
	})
}

func TestGenerateJWTSecret(t *testing.T) {
	t.Run("secret length", func(t *testing.T) {
		secret := GenerateJWTSecret()

		// Base64 encoded 32 bytes should be longer than 32 characters
		assert.Greater(t, len(secret), 32, "JWT secret should be longer than 32 characters")
	})

	t.Run("secret is base64", func(t *testing.T) {
		secret := GenerateJWTSecret()

		// Should be valid base64
		decoded, err := base64.URLEncoding.DecodeString(secret)
		if err != nil {
			// If it's not valid base64, it uses the default
			assert.Equal(t, "your-super-secret-jwt-key-please-change-in-production", secret)
		} else {
			// If it's valid base64, it should decode to 32 bytes
			assert.Equal(t, 32, len(decoded), "Decoded secret should be 32 bytes")
		}
	})

	t.Run("secret uniqueness", func(t *testing.T) {
		secrets := make(map[string]bool)

		// Generate multiple secrets and check they're unique
		for i := 0; i < 10; i++ {
			secret := GenerateJWTSecret()
			// Allow the default secret to appear multiple times
			if secret != "your-super-secret-jwt-key-please-change-in-production" {
				assert.False(t, secrets[secret], "JWT secrets should be unique")
				secrets[secret] = true
			}
		}
	})
}

func TestRandomInt(t *testing.T) {
	t.Run("range validation", func(t *testing.T) {
		max := 10
		results := make(map[int]bool)

		// Generate multiple random numbers
		for i := 0; i < 100; i++ {
			n := RandomInt(max)
			assert.GreaterOrEqual(t, n, 0, "Random int should be >= 0")
			assert.Less(t, n, max, "Random int should be < max")
			results[n] = true
		}

		// Should have some variation (at least 3 different values out of 10 possible)
		assert.GreaterOrEqual(t, len(results), 3, "Should generate varied random numbers")
	})

	t.Run("edge cases", func(t *testing.T) {
		// Test with max = 1
		for i := 0; i < 10; i++ {
			n := RandomInt(1)
			assert.Equal(t, 0, n, "randomInt(1) should always return 0")
		}
	})
}
