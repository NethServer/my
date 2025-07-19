/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package validation

import (
	"regexp"
	"strings"
)

// ValidatePasswordStrength validates password strength with simple, clear rules
// Returns only the first error found to guide user step by step
func ValidatePasswordStrength(password string) (bool, []string) {
	// Minimum length: 12 characters (check first for basic requirement)
	if len(password) < 12 {
		return false, []string{"min_length"}
	}

	// Maximum length: 128 characters
	if len(password) > 128 {
		return false, []string{"max_length"}
	}

	// Must contain at least one uppercase letter
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return false, []string{"missing_uppercase"}
	}

	// Must contain at least one lowercase letter
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return false, []string{"missing_lowercase"}
	}

	// Must contain at least one digit
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return false, []string{"missing_digit"}
	}

	// Must contain at least one special character
	if !regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?~` + "`]").MatchString(password) {
		return false, []string{"missing_special_char"}
	}

	// No more than 3 consecutive identical characters
	if hasRepeatingChars(password, 3) {
		return false, []string{"too_many_repeating_chars"}
	}

	// Cannot contain common weak patterns
	if containsWeakPatterns(password) {
		return false, []string{"contains_weak_patterns"}
	}

	return true, []string{}
}

// hasRepeatingChars checks if password has more than maxCount consecutive identical characters
func hasRepeatingChars(password string, maxCount int) bool {
	if len(password) <= maxCount {
		return false
	}

	count := 1
	for i := 1; i < len(password); i++ {
		if password[i] == password[i-1] {
			count++
			if count > maxCount {
				return true
			}
		} else {
			count = 1
		}
	}
	return false
}

// containsWeakPatterns checks for common weak password patterns
func containsWeakPatterns(password string) bool {
	lower := strings.ToLower(password)

	// Common weak patterns
	weakPatterns := []string{
		"password", "123456", "qwerty", "abc123", "admin",
		"letmein", "welcome", "monkey", "dragon", "master",
	}

	for _, pattern := range weakPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Sequential patterns
	sequential := []string{"012", "123", "234", "345", "456", "567", "678", "789", "890",
		"abc", "bcd", "cde", "def", "efg", "fgh", "ghi", "hij", "ijk", "jkl", "klm",
		"lmn", "mno", "nop", "opq", "pqr", "qrs", "rst", "stu", "tuv", "uvw", "vwx", "wxy", "xyz"}

	for _, seq := range sequential {
		if strings.Contains(lower, seq) {
			return true
		}
	}

	return false
}

// GetPasswordRequirements returns a list of password requirements for user display
func GetPasswordRequirements() []string {
	return []string{
		"At least 12 characters long",
		"At least one uppercase letter (A-Z)",
		"At least one lowercase letter (a-z)",
		"At least one digit (0-9)",
		"At least one special character (!@#$%^&*...)",
		"No more than 3 consecutive identical characters",
		"Cannot contain common weak patterns",
	}
}
