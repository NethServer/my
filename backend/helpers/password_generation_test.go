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
	"unicode"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/testutils"
)

func TestGeneratePassword(t *testing.T) {
	testutils.SetupLogger()

	tests := []struct {
		name    string
		runTest func(*testing.T)
	}{
		{
			name: "generates valid password",
			runTest: func(t *testing.T) {
				password, err := GeneratePassword()
				assert.NoError(t, err)
				assert.NotEmpty(t, password)

				// Validate the generated password meets our requirements
				isValid, errors := ValidatePasswordStrength(password)
				assert.True(t, isValid, "Generated password should be valid, errors: %v", errors)
			},
		},
		{
			name: "generates unique passwords",
			runTest: func(t *testing.T) {
				passwords := make(map[string]bool)
				const numPasswords = 10

				for i := 0; i < numPasswords; i++ {
					password, err := GeneratePassword()
					assert.NoError(t, err)
					assert.False(t, passwords[password], "Password should be unique: %s", password)
					passwords[password] = true
				}
			},
		},
		{
			name: "generates passwords with correct length",
			runTest: func(t *testing.T) {
				password, err := GeneratePassword()
				assert.NoError(t, err)
				assert.Equal(t, 14, len(password), "Password should be 14 characters long")
			},
		},
		{
			name: "generates passwords with all required character types",
			runTest: func(t *testing.T) {
				password, err := GeneratePassword()
				assert.NoError(t, err)

				hasLower := false
				hasUpper := false
				hasDigit := false
				hasSpecial := false

				for _, char := range password {
					if unicode.IsLower(char) {
						hasLower = true
					} else if unicode.IsUpper(char) {
						hasUpper = true
					} else if unicode.IsDigit(char) {
						hasDigit = true
					} else {
						hasSpecial = true
					}
				}

				assert.True(t, hasLower, "Password should contain lowercase letters")
				assert.True(t, hasUpper, "Password should contain uppercase letters")
				assert.True(t, hasDigit, "Password should contain digits")
				assert.True(t, hasSpecial, "Password should contain special characters")
			},
		},
		{
			name: "generates passwords without obvious patterns",
			runTest: func(t *testing.T) {
				password, err := GeneratePassword()
				assert.NoError(t, err)

				// Check for common patterns that should not exist
				assert.False(t, strings.Contains(password, "123"), "Password should not contain sequential digits")
				assert.False(t, strings.Contains(password, "abc"), "Password should not contain sequential letters")
				assert.False(t, strings.Contains(password, "password"), "Password should not contain 'password'")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.runTest)
	}
}

func TestGeneratePassword_Performance(t *testing.T) {
	testutils.SetupLogger()

	// Test that password generation completes within reasonable time
	const numTests = 100
	for i := 0; i < numTests; i++ {
		password, err := GeneratePassword()
		assert.NoError(t, err)
		assert.NotEmpty(t, password)
	}
}

func TestGeneratePassword_Internal(t *testing.T) {
	testutils.SetupLogger()

	tests := []struct {
		name    string
		runTest func(*testing.T)
	}{
		{
			name: "internal generatePassword produces correct length",
			runTest: func(t *testing.T) {
				password, err := generatePassword()
				assert.NoError(t, err)
				assert.Equal(t, 14, len(password))
			},
		},
		{
			name: "internal generatePassword contains required character types",
			runTest: func(t *testing.T) {
				password, err := generatePassword()
				assert.NoError(t, err)

				// Count different character types
				lowerCount := 0
				upperCount := 0
				digitCount := 0
				specialCount := 0

				for _, char := range password {
					if unicode.IsLower(char) {
						lowerCount++
					} else if unicode.IsUpper(char) {
						upperCount++
					} else if unicode.IsDigit(char) {
						digitCount++
					} else {
						specialCount++
					}
				}

				// Check minimum requirements based on the algorithm
				assert.GreaterOrEqual(t, lowerCount, 1, "Should have at least 1 lowercase letter")
				assert.GreaterOrEqual(t, upperCount, 1, "Should have at least 1 uppercase letter")
				assert.GreaterOrEqual(t, digitCount, 1, "Should have at least 1 digit")
				assert.GreaterOrEqual(t, specialCount, 1, "Should have at least 1 special character")
			},
		},
		{
			name: "internal generatePassword is different on multiple calls",
			runTest: func(t *testing.T) {
				password1, err1 := generatePassword()
				password2, err2 := generatePassword()

				assert.NoError(t, err1)
				assert.NoError(t, err2)
				assert.NotEqual(t, password1, password2, "Consecutive password generations should be different")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.runTest)
	}
}

func TestRandomChar(t *testing.T) {
	tests := []struct {
		name      string
		charSet   string
		expectErr bool
	}{
		{
			name:      "valid character set",
			charSet:   "abcdefghij",
			expectErr: false,
		},
		{
			name:      "single character set",
			charSet:   "a",
			expectErr: false,
		},
		{
			name:      "special characters",
			charSet:   "!@#$%^&*()",
			expectErr: false,
		},
		{
			name:      "mixed character set",
			charSet:   "abcDEF123!@#",
			expectErr: false,
		},
		{
			name:      "empty character set",
			charSet:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			char, err := randomChar(tt.charSet)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Equal(t, byte(0), char)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, tt.charSet, string(char), "Returned character should be from the provided set")
			}
		})
	}
}

func TestRandomChar_Distribution(t *testing.T) {
	const charSet = "abc"
	const numTests = 1000
	counts := make(map[byte]int)

	// Generate many random characters and check distribution
	for i := 0; i < numTests; i++ {
		char, err := randomChar(charSet)
		assert.NoError(t, err)
		counts[char]++
	}

	// Each character should appear at least some times (rough distribution check)
	for _, expectedChar := range charSet {
		count := counts[byte(expectedChar)]
		assert.Greater(t, count, 0, "Character '%c' should appear at least once", expectedChar)

		// Rough check that distribution isn't completely skewed (allowing for randomness)
		// Each character should appear at least 10% of expected frequency
		minExpected := numTests / (len(charSet) * 10)
		assert.Greater(t, count, minExpected, "Character '%c' should have reasonable distribution", expectedChar)
	}
}

func TestGeneratePassword_EdgeCases(t *testing.T) {
	testutils.SetupLogger()

	// Test multiple consecutive calls to ensure no state issues
	passwords := make([]string, 20)
	for i := 0; i < 20; i++ {
		password, err := GeneratePassword()
		assert.NoError(t, err)
		passwords[i] = password

		// Validate each password
		isValid, _ := ValidatePasswordStrength(password)
		assert.True(t, isValid, "Password %d should be valid: %s", i+1, password)
	}

	// Ensure all passwords are unique
	passwordSet := make(map[string]bool)
	for i, password := range passwords {
		assert.False(t, passwordSet[password], "Password %d should be unique: %s", i+1, password)
		passwordSet[password] = true
	}
}

func TestGeneratePassword_SpecialCharacterSafety(t *testing.T) {
	testutils.SetupLogger()

	password, err := GeneratePassword()
	assert.NoError(t, err)

	// Check that only safe special characters are used
	safeSpecialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?~"
	for _, char := range password {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			assert.Contains(t, safeSpecialChars, string(char), "Special character should be from safe set: %c", char)
		}
	}
}

func TestGeneratePassword_ConsistentValidation(t *testing.T) {
	testutils.SetupLogger()

	// Generate multiple passwords and ensure they all pass validation
	const numTests = 50
	for i := 0; i < numTests; i++ {
		password, err := GeneratePassword()
		assert.NoError(t, err)

		isValid, errors := ValidatePasswordStrength(password)
		assert.True(t, isValid, "Generated password should always be valid. Password: %s, Errors: %v", password, errors)
	}
}
