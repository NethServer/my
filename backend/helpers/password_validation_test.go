/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePasswordStrength_Valid(t *testing.T) {
	validPasswords := []string{
		"MySecureP4ssw9rd!",
		"Complex!@#$Pwd97",
		"Str0ng!W0rd2024",
		"ValidP4ssw9rd!",
	}

	for _, password := range validPasswords {
		t.Run("valid_password_"+password[:8], func(t *testing.T) {
			isValid, errors := ValidatePasswordStrength(password)
			assert.True(t, isValid, "Password should be valid: %s, errors: %v", password, errors)
			assert.Empty(t, errors, "Should have no validation errors")
		})
	}
}

func TestValidatePasswordStrength_TooShort(t *testing.T) {
	shortPasswords := []string{
		"Short1!",
		"Pass123!",
		"Weak!1",
	}

	for _, password := range shortPasswords {
		t.Run("short_password_"+password, func(t *testing.T) {
			isValid, errors := ValidatePasswordStrength(password)
			assert.False(t, isValid, "Password should be invalid: %s", password)
			assert.Contains(t, errors, "min_length")
		})
	}
}

func TestValidatePasswordStrength_TooLong(t *testing.T) {
	// Create a password longer than 128 characters
	longPassword := "Abcd123!" + string(make([]byte, 130))

	isValid, errors := ValidatePasswordStrength(longPassword)
	assert.False(t, isValid, "Password should be invalid")
	assert.Contains(t, errors, "max_length")
}

func TestValidatePasswordStrength_MissingUppercase(t *testing.T) {
	password := "lowercase123!password"

	isValid, errors := ValidatePasswordStrength(password)
	assert.False(t, isValid, "Password should be invalid")
	assert.Contains(t, errors, "missing_uppercase")
}

func TestValidatePasswordStrength_MissingLowercase(t *testing.T) {
	password := "UPPERCASE123!PASSWORD"

	isValid, errors := ValidatePasswordStrength(password)
	assert.False(t, isValid, "Password should be invalid")
	assert.Contains(t, errors, "missing_lowercase")
}

func TestValidatePasswordStrength_MissingDigit(t *testing.T) {
	password := "NoDigitsHere!Password"

	isValid, errors := ValidatePasswordStrength(password)
	assert.False(t, isValid, "Password should be invalid")
	assert.Contains(t, errors, "missing_digit")
}

func TestValidatePasswordStrength_MissingSpecialChar(t *testing.T) {
	password := "NoSpecialChars123Password"

	isValid, errors := ValidatePasswordStrength(password)
	assert.False(t, isValid, "Password should be invalid")
	assert.Contains(t, errors, "missing_special_char")
}

func TestValidatePasswordStrength_TooManyRepeatingChars(t *testing.T) {
	password := "ValidPass1111!Word"

	isValid, errors := ValidatePasswordStrength(password)
	assert.False(t, isValid, "Password should be invalid")
	assert.Contains(t, errors, "too_many_repeating_chars")
}

func TestValidatePasswordStrength_WeakPatterns(t *testing.T) {
	// Test passwords that have weak patterns and meet other requirements
	weakPasswords := []string{
		"MyPassword123!weak",  // 16+ chars, all requirements met except weak pattern
		"AdminUser123!Strong", // 16+ chars, all requirements met except weak pattern
		"QwertyValid123!Pass", // 16+ chars, all requirements met except weak pattern
	}

	for _, password := range weakPasswords {
		t.Run("weak_pattern_"+password[:10], func(t *testing.T) {
			isValid, errors := ValidatePasswordStrength(password)
			assert.False(t, isValid, "Password should be invalid: %s", password)
			assert.Contains(t, errors, "contains_weak_patterns")
		})
	}
}

func TestValidatePasswordStrength_SequentialPatterns(t *testing.T) {
	sequentialPasswords := []string{
		"ValidP@ssw0rd123",
		"StrongPassword456!",
		"MySecureAbc123!Word",
	}

	for _, password := range sequentialPasswords {
		t.Run("sequential_pattern_"+password[:10], func(t *testing.T) {
			isValid, errors := ValidatePasswordStrength(password)
			assert.False(t, isValid, "Password should be invalid: %s", password)
			assert.Contains(t, errors, "contains_weak_patterns")
		})
	}
}

func TestValidatePasswordStrength_FirstErrorOnly(t *testing.T) {
	password := "weak"

	isValid, errors := ValidatePasswordStrength(password)
	assert.False(t, isValid, "Password should be invalid")

	// Should have only one error (the first one encountered)
	assert.Len(t, errors, 1, "Should have only one validation error")
	assert.Contains(t, errors, "min_length")
}

func TestHasRepeatingChars(t *testing.T) {
	testCases := []struct {
		password string
		maxCount int
		expected bool
	}{
		{"password", 3, false},
		{"passsword", 3, false},   // exactly 3 consecutive 's'
		{"passssword", 3, true},   // 4 consecutive 's'
		{"password1111", 3, true}, // 4 consecutive '1'
		{"password111", 3, false}, // exactly 3 consecutive '1'
		{"", 3, false},
		{"a", 3, false},
		{"aa", 3, false},
		{"aaa", 3, false},
		{"aaaa", 3, true},
	}

	for _, tc := range testCases {
		t.Run("repeating_"+tc.password, func(t *testing.T) {
			result := hasRepeatingChars(tc.password, tc.maxCount)
			assert.Equal(t, tc.expected, result, "Password: %s, maxCount: %d", tc.password, tc.maxCount)
		})
	}
}

func TestContainsWeakPatterns(t *testing.T) {
	testCases := []struct {
		password string
		expected bool
	}{
		{"strongpassw0rd987", false},
		{"password123", true},     // contains "password"
		{"myqwerty", true},        // contains "qwerty"
		{"abc123test", true},      // contains "abc123"
		{"adminuser", true},       // contains "admin"
		{"validpassword", true},   // contains "password"
		{"test123sequence", true}, // contains "123"
		{"myabctest", true},       // contains "abc"
		{"strongtest", false},
		{"ValidP4ssw0rd", false}, // does not contain weak patterns
	}

	for _, tc := range testCases {
		t.Run("weak_pattern_"+tc.password, func(t *testing.T) {
			result := containsWeakPatterns(tc.password)
			assert.Equal(t, tc.expected, result, "Password: %s", tc.password)
		})
	}
}

func TestGetPasswordRequirements(t *testing.T) {
	requirements := GetPasswordRequirements()

	assert.NotEmpty(t, requirements, "Should return password requirements")
	assert.GreaterOrEqual(t, len(requirements), 5, "Should have at least 5 requirements")

	// Check for key requirements
	requirementText := ""
	for _, req := range requirements {
		requirementText += req + " "
	}

	assert.Contains(t, requirementText, "12 characters")
	assert.Contains(t, requirementText, "uppercase")
	assert.Contains(t, requirementText, "lowercase")
	assert.Contains(t, requirementText, "digit")
	assert.Contains(t, requirementText, "special character")
}

func TestValidatePasswordStrength_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		password string
		isValid  bool
	}{
		{"empty_password", "", false},
		{"exactly_12_chars", "Valid!@#Pwd9", true},
		{"exactly_128_chars", "A1!" + string(make([]byte, 125)), false}, // Will be invalid due to null bytes
		{"unicode_chars", "ValidP4ssw0rd9ñ!", true},
		{"only_special_chars", "!@#$%^&*()_+", false}, // Missing other char types
		{"borderline_valid", "MyP4ssw0rd!9", true},    // Exactly meets all requirements
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid, _ := ValidatePasswordStrength(tc.password)
			assert.Equal(t, tc.isValid, isValid, "Password: %s", tc.password)
		})
	}
}
