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
	"github.com/google/uuid"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// Token type constants for preventing token confusion
const (
	TokenTypeAccess        = "access"
	TokenTypeRefresh       = "refresh"
	TokenTypeImpersonation = "impersonation"
)

// CustomClaims represents our custom JWT claims structure
type CustomClaims struct {
	TokenType string      `json:"token_type"`
	User      models.User `json:"user"`
	jwt.RegisteredClaims
}

// RefreshTokenClaims represents the claims for refresh tokens
type RefreshTokenClaims struct {
	TokenType string `json:"token_type"`
	UserID    string `json:"user_id"`
	jwt.RegisteredClaims
}

// ImpersonationClaims represents the claims for impersonation tokens
type ImpersonationClaims struct {
	TokenType      string      `json:"token_type"`
	User           models.User `json:"user"`            // The user being impersonated
	ImpersonatedBy models.User `json:"impersonated_by"` // The user doing the impersonation
	IsImpersonated bool        `json:"is_impersonated"` // Flag to indicate this is an impersonation token
	SessionID      string      `json:"session_id"`      // Session ID for audit tracking
	jwt.RegisteredClaims
}

// ProxyTokenClaims represents the claims for support proxy tokens
type ProxyTokenClaims struct {
	TokenType   string `json:"token_type"`
	SessionID   string `json:"session_id"`
	ServiceName string `json:"service_name"`
	UserID      string `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateProxyToken creates a short-lived JWT for subdomain-based support proxy access
func GenerateProxyToken(sessionID, serviceName, userID string) (string, error) {
	expDuration := 8 * time.Hour

	claims := ProxyTokenClaims{
		TokenType:   "proxy",
		SessionID:   sessionID,
		ServiceName: serviceName,
		UserID:      userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    configuration.Config.JWTIssuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{configuration.Config.LogtoAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(configuration.Config.JWTSecret))
	if err != nil {
		logger.ComponentLogger("jwt").Error().
			Err(err).
			Str("operation", "proxy_token_sign_failed").
			Str("session_id", sessionID).
			Str("service_name", serviceName).
			Str("user_id", userID).
			Msg("Failed to sign proxy token")
		return "", fmt.Errorf("failed to sign proxy token: %w", err)
	}

	logger.ComponentLogger("jwt").Info().
		Str("operation", "proxy_token_generated").
		Str("session_id", sessionID).
		Str("service_name", serviceName).
		Str("user_id", userID).
		Time("expires_at", time.Now().Add(expDuration)).
		Msg("Proxy token generated successfully")

	return tokenString, nil
}

// ValidateProxyToken parses and validates a support proxy JWT token
func ValidateProxyToken(tokenString string) (*ProxyTokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &ProxyTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configuration.Config.JWTSecret), nil
	})

	if err != nil {
		logger.ComponentLogger("jwt").Warn().
			Err(err).
			Str("operation", "proxy_token_validation_failed").
			Str("error_type", "parse_failed").
			Msg("Failed to parse proxy token")
		return nil, fmt.Errorf("failed to parse proxy token: %w", err)
	}

	if claims, ok := token.Claims.(*ProxyTokenClaims); ok && token.Valid {
		if claims.TokenType != "proxy" {
			logger.ComponentLogger("jwt").Warn().
				Str("operation", "proxy_token_validation_failed").
				Str("error_type", "wrong_token_type").
				Str("token_type", claims.TokenType).
				Msg("token is not a proxy token")
			return nil, fmt.Errorf("token is not a proxy token")
		}
		if claims.SessionID == "" || claims.ServiceName == "" {
			logger.ComponentLogger("jwt").Warn().
				Str("operation", "proxy_token_validation_failed").
				Str("error_type", "missing_claims").
				Msg("Proxy token missing required claims")
			return nil, fmt.Errorf("proxy token missing required claims")
		}

		logger.ComponentLogger("jwt").Debug().
			Str("operation", "proxy_token_validation_success").
			Str("session_id", claims.SessionID).
			Str("service_name", claims.ServiceName).
			Str("user_id", claims.UserID).
			Msg("Proxy token validated successfully")
		return claims, nil
	}

	logger.ComponentLogger("jwt").Warn().
		Str("operation", "proxy_token_validation_failed").
		Str("error_type", "invalid_claims").
		Bool("token_valid", token.Valid).
		Msg("Invalid proxy token claims")

	return nil, fmt.Errorf("invalid proxy token claims")
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
		TokenType: TokenTypeAccess,
		User:      user,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
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
		// Verify token type to prevent token confusion
		if claims.TokenType != "" && claims.TokenType != TokenTypeAccess {
			return nil, fmt.Errorf("invalid token type: expected %s, got %s", TokenTypeAccess, claims.TokenType)
		}
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
		TokenType: TokenTypeRefresh,
		UserID:    userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
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
		// Verify token type to prevent token confusion
		if claims.TokenType != "" && claims.TokenType != TokenTypeRefresh {
			return nil, fmt.Errorf("invalid token type: expected %s, got %s", TokenTypeRefresh, claims.TokenType)
		}
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

// GenerateImpersonationToken creates a JWT token for user impersonation (legacy - 1 hour)
func GenerateImpersonationToken(impersonatedUser, impersonator models.User) (string, error) {
	return GenerateImpersonationTokenWithDuration(impersonatedUser, impersonator, "", 1*time.Hour)
}

// GenerateImpersonationTokenWithDuration creates a JWT token for user impersonation with custom duration and session ID
func GenerateImpersonationTokenWithDuration(impersonatedUser, impersonator models.User, sessionID string, expDuration time.Duration) (string, error) {
	// Create impersonation claims
	claims := ImpersonationClaims{
		TokenType:      TokenTypeImpersonation,
		User:           impersonatedUser,
		ImpersonatedBy: impersonator,
		IsImpersonated: true,
		SessionID:      sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    configuration.Config.JWTIssuer,
			Subject:   impersonatedUser.ID,
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
			Str("operation", "impersonation_token_sign_failed").
			Str("impersonated_user_id", impersonatedUser.ID).
			Str("impersonator_user_id", impersonator.ID).
			Msg("Failed to sign impersonation JWT token")
		return "", fmt.Errorf("failed to sign impersonation token: %w", err)
	}

	logger.ComponentLogger("jwt").Info().
		Str("operation", "impersonation_token_generated").
		Str("impersonated_user_id", impersonatedUser.ID).
		Str("impersonated_username", impersonatedUser.Username).
		Str("impersonator_user_id", impersonator.ID).
		Str("impersonator_username", impersonator.Username).
		Str("session_id", sessionID).
		Time("expires_at", time.Now().Add(expDuration)).
		Dur("duration", expDuration).
		Msg("Impersonation JWT token generated successfully")

	return tokenString, nil
}

// ValidateImpersonationToken parses and validates an impersonation JWT token
func ValidateImpersonationToken(tokenString string) (*ImpersonationClaims, error) {
	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &ImpersonationClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(configuration.Config.JWTSecret), nil
	})

	if err != nil {
		logger.ComponentLogger("jwt").Warn().
			Err(err).
			Str("operation", "impersonation_token_validation_failed").
			Str("error_type", "parse_failed").
			Msg("Failed to parse impersonation JWT token")
		return nil, fmt.Errorf("failed to parse impersonation token: %w", err)
	}

	// Extract and validate claims
	if claims, ok := token.Claims.(*ImpersonationClaims); ok && token.Valid {
		// CRITICAL: Verify this is actually an impersonation token
		// A regular token can be parsed as ImpersonationClaims but will have IsImpersonated=false
		if !claims.IsImpersonated {
			logger.ComponentLogger("jwt").Debug().
				Str("operation", "impersonation_token_validation_failed").
				Str("error_type", "not_impersonation_token").
				Bool("is_impersonated", claims.IsImpersonated).
				Msg("Token is not an impersonation token")
			return nil, fmt.Errorf("token is not an impersonation token")
		}

		// Additional validation: check that impersonator data exists
		// Note: ID might be empty for some users, so we check LogtoID or Username
		if claims.ImpersonatedBy.Username == "" && (claims.ImpersonatedBy.LogtoID == nil || *claims.ImpersonatedBy.LogtoID == "") {
			logger.ComponentLogger("jwt").Debug().
				Str("operation", "impersonation_token_validation_failed").
				Str("error_type", "missing_impersonator_data").
				Str("impersonator_id", claims.ImpersonatedBy.ID).
				Str("impersonator_username", claims.ImpersonatedBy.Username).
				Msg("Impersonation token missing impersonator data")
			return nil, fmt.Errorf("impersonation token missing impersonator data")
		}

		logger.ComponentLogger("jwt").Debug().
			Str("operation", "impersonation_token_validation_success").
			Str("impersonated_user_id", claims.User.ID).
			Str("impersonated_username", claims.User.Username).
			Str("impersonator_user_id", claims.ImpersonatedBy.ID).
			Str("impersonator_username", claims.ImpersonatedBy.Username).
			Time("expires_at", claims.ExpiresAt.Time).
			Msg("Impersonation JWT token validated successfully")
		return claims, nil
	}

	logger.ComponentLogger("jwt").Warn().
		Str("operation", "impersonation_token_validation_failed").
		Str("error_type", "invalid_claims").
		Bool("token_valid", token.Valid).
		Msg("Invalid impersonation JWT token claims")

	return nil, fmt.Errorf("invalid impersonation token claims")
}
