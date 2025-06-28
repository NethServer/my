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
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logs"
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
			logs.Logs.Println("[ERROR][AUTH] token validation failed: " + err.Error())
			c.JSON(http.StatusUnauthorized, response.Unauthorized("invalid token", err.Error()))
			c.Abort()
			return
		}

		// Store user info in context
		c.Set("user", user)
		c.Set("user_id", user.ID)
		// Username field removed from User model

		logs.Logs.Println("[INFO][AUTH] authentication success for user " + user.ID + " from " + c.ClientIP())
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

	// Extract user information
	user := &models.User{}

	if sub, ok := claims["sub"].(string); ok {
		user.ID = sub
	}

	// Username field removed from User model

	// Email field removed from User model

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
	if jwksCache != nil && time.Since(jwksCacheTime) < 5*time.Minute {
		if key, exists := jwksCache[kid]; exists {
			return key, nil
		}
	}

	// Fetch JWKS
	resp, err := http.Get(configuration.Config.JWKSEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks JWKSet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Update cache
	jwksCache = make(map[string]*rsa.PublicKey)
	jwksCacheTime = time.Now()

	for _, jwk := range jwks.Keys {
		if jwk.Kty == "RSA" && jwk.Use == "sig" {
			key, err := jwkToRSAPublicKey(jwk)
			if err != nil {
				logs.Logs.Println("[WARN][AUTH] failed to convert JWK to RSA key: " + err.Error())
				continue
			}
			jwksCache[jwk.Kid] = key
		}
	}

	if key, exists := jwksCache[kid]; exists {
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
