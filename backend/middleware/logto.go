/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

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

		jwksManager := cache.GetJWKSCacheManager()
		publicKey, err := jwksManager.GetPublicKey(kid)
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
