/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type JWKSet struct {
	Keys []JWK `json:"keys"`
}

var jwksCache map[string]*rsa.PublicKey
var jwksCacheTime time.Time
var jwksCacheMutex sync.RWMutex

func LogtoAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("authorization header required", nil))
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, response.Unauthorized("bearer token required", nil))
			c.Abort()
			return
		}

		user, err := validateLogtoToken(tokenString)
		if err != nil {
			logger.LogAuthFailure(c, "auth", "logto_jwt", "token_validation_failed", err)
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid token", err.Error()))
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user", user)
		c.Set("user_id", user.ID)

		logger.LogAuthSuccess(c, "auth", "logto_jwt", user.ID, user.OrganizationID)
		c.Next()
	}
}

func validateLogtoToken(tokenString string) (*models.User, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		publicKey, err := getPublicKey(kid)
		if err != nil {
			return nil, fmt.Errorf("failed to get public key: %w", err)
		}

		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Validate issuer
	if iss, ok := claims["iss"].(string); !ok || iss != configuration.Config.LogtoIssuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	// Validate audience
	if aud, ok := claims["aud"].(string); !ok || aud != configuration.Config.LogtoAudience {
		return nil, fmt.Errorf("invalid audience")
	}

	// Validate expiration
	if exp, ok := claims["exp"].(float64); !ok || int64(exp) < time.Now().Unix() {
		return nil, fmt.Errorf("token expired")
	}

	// Validate not before (nbf) if present
	if nbf, ok := claims["nbf"].(float64); ok && int64(nbf) > time.Now().Unix() {
		return nil, fmt.Errorf("token not yet valid")
	}

	// Extract user information
	user := &models.User{}

	if sub, ok := claims["sub"].(string); ok {
		user.ID = sub
	}

	// Extract user roles (technical capabilities) from custom claims
	if userRoles, ok := claims["user_roles"].([]interface{}); ok {
		for _, role := range userRoles {
			if roleStr, ok := role.(string); ok {
				user.UserRoles = append(user.UserRoles, roleStr)
			}
		}
	}

	// Extract user permissions from custom claims
	if userPerms, ok := claims["user_permissions"].([]interface{}); ok {
		for _, perm := range userPerms {
			if permStr, ok := perm.(string); ok {
				user.UserPermissions = append(user.UserPermissions, permStr)
			}
		}
	}

	// Extract organization role (business hierarchy) from custom claims
	if orgRole, ok := claims["org_role"].(string); ok {
		user.OrgRole = orgRole
	}

	// Extract organization permissions from custom claims
	if orgPerms, ok := claims["org_permissions"].([]interface{}); ok {
		for _, perm := range orgPerms {
			if permStr, ok := perm.(string); ok {
				user.OrgPermissions = append(user.OrgPermissions, permStr)
			}
		}
	}

	// Extract organization info from custom claims
	if orgId, ok := claims["organization_id"].(string); ok {
		user.OrganizationID = orgId
	}

	if orgName, ok := claims["organization_name"].(string); ok {
		user.OrganizationName = orgName
	}

	return user, nil
}

func getPublicKey(kid string) (*rsa.PublicKey, error) {
	// Check cache (5 minutes TTL)
	jwksCacheMutex.RLock()
	if jwksCache != nil && time.Since(jwksCacheTime) < 5*time.Minute {
		if key, exists := jwksCache[kid]; exists {
			jwksCacheMutex.RUnlock()
			logger.ComponentLogger("auth").Debug().
				Str("operation", "jwks_cache_hit").
				Str("kid", kid).
				Time("cache_time", jwksCacheTime).
				Dur("cache_age", time.Since(jwksCacheTime)).
				Msg("JWKS cache hit")
			return key, nil
		}
		logger.ComponentLogger("auth").Debug().
			Str("operation", "jwks_cache_miss").
			Str("kid", kid).
			Time("cache_time", jwksCacheTime).
			Int("cached_keys", len(jwksCache)).
			Msg("JWKS cache miss - key not found")
	}
	jwksCacheMutex.RUnlock()

	if jwksCache == nil || time.Since(jwksCacheTime) >= 5*time.Minute {
		logger.ComponentLogger("auth").Debug().
			Str("operation", "jwks_cache_expired").
			Str("kid", kid).
			Time("cache_time", jwksCacheTime).
			Dur("cache_age", time.Since(jwksCacheTime)).
			Msg("JWKS cache expired or empty")
	}

	// Fetch JWKS
	logger.ComponentLogger("auth").Info().
		Str("operation", "jwks_fetch_start").
		Str("endpoint", configuration.Config.JWKSEndpoint).
		Str("kid", kid).
		Msg("Fetching JWKS from endpoint")

	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(configuration.Config.JWKSEndpoint)
	if err != nil {
		logger.ComponentLogger("auth").Error().
			Err(err).
			Str("operation", "jwks_fetch_failed").
			Str("endpoint", configuration.Config.JWKSEndpoint).
			Dur("duration", time.Since(start)).
			Msg("Failed to fetch JWKS")
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // 1MB max
		logger.ComponentLogger("auth").Error().
			Str("operation", "jwks_fetch_bad_status").
			Int("status_code", resp.StatusCode).
			Str("endpoint", configuration.Config.JWKSEndpoint).
			Str("response_body", string(body)).
			Dur("duration", time.Since(start)).
			Msg("JWKS endpoint returned error status")
		return nil, fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKSet
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1024*1024)).Decode(&jwks); err != nil {
		logger.ComponentLogger("auth").Error().
			Err(err).
			Str("operation", "jwks_decode_failed").
			Int("status_code", resp.StatusCode).
			Dur("duration", time.Since(start)).
			Msg("Failed to decode JWKS response")
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Update cache
	jwksCacheMutex.Lock()
	jwksCache = make(map[string]*rsa.PublicKey)
	jwksCacheTime = time.Now()

	keysProcessed := 0
	keysSuccessful := 0

	for _, jwk := range jwks.Keys {
		keysProcessed++
		if jwk.Kty == "RSA" && jwk.Use == "sig" {
			key, err := jwkToRSAPublicKey(jwk)
			if err != nil {
				logger.ComponentLogger("auth").Warn().
					Err(err).
					Str("operation", "convert_jwk_to_rsa").
					Str("kid", jwk.Kid).
					Msg("Failed to convert JWK to RSA key")
				continue
			}
			jwksCache[jwk.Kid] = key
			keysSuccessful++
		}
	}

	logger.ComponentLogger("auth").Info().
		Str("operation", "jwks_cache_updated").
		Int("keys_processed", keysProcessed).
		Int("keys_successful", keysSuccessful).
		Int("total_cached", len(jwksCache)).
		Dur("fetch_duration", time.Since(start)).
		Time("cache_time", jwksCacheTime).
		Msg("JWKS cache updated successfully")

	key, exists := jwksCache[kid]
	jwksCacheMutex.Unlock()

	if exists {
		logger.ComponentLogger("auth").Debug().
			Str("operation", "jwks_key_found").
			Str("kid", kid).
			Msg("Requested key found in updated cache")
		return key, nil
	}

	return nil, fmt.Errorf("key not found for kid: %s", kid)
}

func jwkToRSAPublicKey(jwk JWK) (*rsa.PublicKey, error) {
	// Decode base64url encoded modulus (n)
	nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
	if err != nil {
		return nil, fmt.Errorf("failed to decode modulus: %w", err)
	}

	// Decode base64url encoded exponent (e)
	eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
	if err != nil {
		return nil, fmt.Errorf("failed to decode exponent: %w", err)
	}

	// Convert bytes to big integers
	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{N: n, E: e}, nil
}
