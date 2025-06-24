/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
)

// CustomClaims represents our custom JWT claims structure
type CustomClaims struct {
	User models.User `json:"user"`
	jwt.RegisteredClaims
}

// GenerateCustomToken creates a JWT token with user information and permissions
func GenerateCustomToken(user models.User) (string, error) {
	// Parse expiration duration
	expDuration, err := time.ParseDuration(configuration.Config.JWTExpiration)
	if err != nil {
		expDuration = 24 * time.Hour // Default fallback
	}

	// Create custom claims
	claims := CustomClaims{
		User: user,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    configuration.Config.JWTIssuer,
			Subject:   user.ID,
			Audience:  jwt.ClaimStrings{configuration.Config.LogtoAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token with secret
	tokenString, err := token.SignedString([]byte(configuration.Config.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateCustomToken parses and validates our custom JWT token
func ValidateCustomToken(tokenString string) (*CustomClaims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configuration.Config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract and validate claims
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}