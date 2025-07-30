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
	"fmt"
	"math/big"

	"github.com/nethesis/my/backend/logger"
)

// GeneratePassword generates a secure temporary password that passes our validation
func GeneratePassword() (string, error) {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		password, err := generatePassword()
		if err != nil {
			return "", fmt.Errorf("failed to generate password: %w", err)
		}

		// Validate using our existing validator
		isValid, errors := ValidatePasswordStrength(password)
		if isValid {
			logger.Debug().
				Int("attempt", attempt+1).
				Msg("Generated valid temporary password")
			return password, nil
		}

		logger.Debug().
			Int("attempt", attempt+1).
			Strs("validation_errors", errors).
			Msg("Generated password failed validation, retrying")
	}

	return "", fmt.Errorf("failed to generate valid password after %d attempts", maxAttempts)
}

// generatePassword creates a password that should meet our validation requirements
func generatePassword() (string, error) {
	// Generate a 14-character password (exceeds minimum requirement of 12)
	const length = 14

	// Character sets that match our validator requirements
	const (
		lowerChars  = "abcdefghijklmnopqrstuvwxyz"
		upperChars  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		numberChars = "0123456789"
		// Use only safe special characters that are in our validator regex
		specialChars = "!@#$%^&*()_+-=[]{}|;:,.<>?~"
	)

	password := make([]byte, length)

	// Ensure at least 2 characters from each required category (for better distribution)
	requirements := []struct {
		chars string
		count int
	}{
		{lowerChars, 3},   // At least 3 lowercase
		{upperChars, 3},   // At least 3 uppercase
		{numberChars, 2},  // At least 2 digits
		{specialChars, 2}, // At least 2 special chars
	}

	pos := 0

	// Fill required characters
	for _, req := range requirements {
		for i := 0; i < req.count && pos < length; i++ {
			char, err := randomChar(req.chars)
			if err != nil {
				return "", err
			}
			password[pos] = char
			pos++
		}
	}

	// Fill remaining positions with random mix
	allChars := lowerChars + upperChars + numberChars + specialChars
	for pos < length {
		char, err := randomChar(allChars)
		if err != nil {
			return "", err
		}
		password[pos] = char
		pos++
	}

	// Shuffle the password to avoid predictable patterns
	for i := range password {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(len(password))))
		if err != nil {
			return "", fmt.Errorf("failed to shuffle password: %w", err)
		}
		password[i], password[j.Int64()] = password[j.Int64()], password[i]
	}

	return string(password), nil
}

// randomChar returns a random character from the given character set
func randomChar(charSet string) (byte, error) {
	if len(charSet) == 0 {
		return 0, fmt.Errorf("character set cannot be empty")
	}

	index, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
	if err != nil {
		return 0, err
	}

	return charSet[index.Int64()], nil
}
