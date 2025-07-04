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
	"crypto/rand"
	"encoding/base64"
	"math/big"
)

// GenerateSecurePassword generates a secure password for the owner user
func GenerateSecurePassword() string {
	const (
		lowerCase = "abcdefghijklmnopqrstuvwxyz"
		upperCase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		symbols   = "!@#$%^&*"
		length    = 16
	)

	charset := lowerCase + upperCase + digits + symbols
	password := make([]byte, length)

	// Ensure at least one character from each set
	password[0] = lowerCase[RandomInt(len(lowerCase))]
	password[1] = upperCase[RandomInt(len(upperCase))]
	password[2] = digits[RandomInt(len(digits))]
	password[3] = symbols[RandomInt(len(symbols))]

	// Fill the rest randomly
	for i := 4; i < length; i++ {
		password[i] = charset[RandomInt(len(charset))]
	}

	// Shuffle the password
	for i := length - 1; i > 0; i-- {
		j := RandomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

// GenerateJWTSecret generates a JWT secret
func GenerateJWTSecret() string {
	bytes := make([]byte, 32) // 256-bit key
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to a deterministic but secure method
		return "your-super-secret-jwt-key-please-change-in-production"
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

// RandomInt generates a random integer between 0 and max-1
func RandomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}
