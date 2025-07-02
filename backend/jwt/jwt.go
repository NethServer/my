/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// CustomClaims represents our custom JWT claims structure
type CustomClaims struct {
	User models.User `json:"user"`
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents the claims for refresh tokens
type RefreshTokenClaims struct {
	UserID string `json:"user_id"`
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
		logger.ComponentLogger("jwt").Error().
			Err(err).
			Str("operation", "token_sign_failed").
			Str("user_id", user.ID).
			Str("username", user.Username).
			Msg("Failed to sign custom JWT token")
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	logger.ComponentLogger("jwt").Info().
		Str("operation", "token_generated").
		Str("user_id", user.ID).
		Str("username", user.Username).
		Str("organization_id", user.OrganizationID).
		Str("org_role", user.OrgRole).
		Strs("user_roles", user.UserRoles).
		Time("expires_at", time.Now().Add(expDuration)).
		Dur("duration", expDuration).
		Msg("Custom JWT token generated successfully")

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
		logger.ComponentLogger("jwt").Warn().
			Err(err).
			Str("operation", "token_validation_failed").
			Str("error_type", "parse_failed").
			Msg("Failed to parse custom JWT token")
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract and validate claims
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		logger.ComponentLogger("jwt").Debug().
			Str("operation", "token_validation_success").
			Str("user_id", claims.User.ID).
			Str("username", claims.User.Username).
			Str("organization_id", claims.User.OrganizationID).
			Str("org_role", claims.User.OrgRole).
			Time("expires_at", claims.ExpiresAt.Time).
			Msg("Custom JWT token validated successfully")
		return claims, nil
	}

	logger.ComponentLogger("jwt").Warn().
		Str("operation", "token_validation_failed").
		Str("error_type", "invalid_claims").
		Bool("token_valid", token.Valid).
		Msg("Invalid custom JWT token claims")

	return nil, fmt.Errorf("invalid token claims")
}

// GenerateRefreshToken creates a refresh token for the given user
func GenerateRefreshToken(userID string) (string, error) {
	// Parse expiration duration from config
	expDuration, err := time.ParseDuration(configuration.Config.JWTRefreshExpiration)
	if err != nil {
		expDuration = 7 * 24 * time.Hour // Default fallback: 7 days
	}

	// Create refresh token claims
	claims := RefreshTokenClaims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    configuration.Config.JWTIssuer,
			Subject:   userID,
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
		logger.ComponentLogger("jwt").Error().
			Err(err).
			Str("operation", "refresh_token_sign_failed").
			Str("user_id", userID).
			Msg("Failed to sign refresh token")
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	logger.ComponentLogger("jwt").Info().
		Str("operation", "refresh_token_generated").
		Str("user_id", userID).
		Time("expires_at", time.Now().Add(expDuration)).
		Dur("duration", expDuration).
		Msg("Refresh token generated successfully")

	return tokenString, nil
}

// ValidateRefreshToken parses and validates a refresh token
func ValidateRefreshToken(tokenString string) (*RefreshTokenClaims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &RefreshTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configuration.Config.JWTSecret), nil
	})

	if err != nil {
		logger.ComponentLogger("jwt").Warn().
			Err(err).
			Str("operation", "refresh_token_validation_failed").
			Str("error_type", "parse_failed").
			Msg("Failed to parse refresh token")
		return nil, fmt.Errorf("failed to parse refresh token: %w", err)
	}

	// Extract and validate claims
	if claims, ok := token.Claims.(*RefreshTokenClaims); ok && token.Valid {
		logger.ComponentLogger("jwt").Debug().
			Str("operation", "refresh_token_validation_success").
			Str("user_id", claims.UserID).
			Time("expires_at", claims.ExpiresAt.Time).
			Msg("Refresh token validated successfully")
		return claims, nil
	}

	logger.ComponentLogger("jwt").Warn().
		Str("operation", "refresh_token_validation_failed").
		Str("error_type", "invalid_claims").
		Bool("token_valid", token.Valid).
		Msg("Invalid refresh token claims")

	return nil, fmt.Errorf("invalid refresh token claims")
}
