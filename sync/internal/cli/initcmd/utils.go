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
	"fmt"
	"math/big"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

// GenerateSecurePassword generates a secure password for the owner user
func GenerateSecurePassword() (string, error) {
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
	sets := []string{lowerCase, upperCase, digits, symbols}
	for i, set := range sets {
		idx, err := RandomInt(len(set))
		if err != nil {
			return "", fmt.Errorf("failed to generate password character: %w", err)
		}
		password[i] = set[idx]
	}

	// Fill the rest randomly
	for i := 4; i < length; i++ {
		idx, err := RandomInt(len(charset))
		if err != nil {
			return "", fmt.Errorf("failed to generate password character: %w", err)
		}
		password[i] = charset[idx]
	}

	// Shuffle the password
	for i := length - 1; i > 0; i-- {
		j, err := RandomInt(i + 1)
		if err != nil {
			return "", fmt.Errorf("failed to shuffle password: %w", err)
		}
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// GenerateJWTSecret generates a JWT secret
func GenerateJWTSecret() (string, error) {
	bytes := make([]byte, 32) // 256-bit key
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// RandomInt generates a random integer between 0 and max-1
func RandomInt(max int) (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0, fmt.Errorf("failed to generate random integer: %w", err)
	}
	return int(n.Int64()), nil
}

// ConfigureMFA configures Multi-Factor Authentication with TOTP (Authenticator app OTP)
func ConfigureMFA(logtoClient *client.LogtoClient) error {
	logger.Info("Configuring MFA with TOTP (Authenticator app OTP)...")

	// Configure MFA with mandatory policy and TOTP factor
	if err := logtoClient.UpdateSignInExperienceMFA("Mandatory", []string{"Totp"}); err != nil {
		return err
	}

	logger.Info("MFA configured successfully - all users will be required to use TOTP")
	return nil
}
