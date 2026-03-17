/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// sshAuthorizeRequest represents the request body for SSH authorization
type sshAuthorizeRequest struct {
	Code string `json:"code" binding:"required"`
}

// sshNonceData matches the NonceData struct stored by the SSH gateway
type sshNonceData struct {
	SystemKey string    `json:"system_key"`
	CreatedAt time.Time `json:"created_at"`
}

// sshAuthResult is stored in Redis for the SSH gateway to consume
type sshAuthResult struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	UserEmail        string `json:"user_email"`
	OrganizationName string `json:"organization_name"`
	SessionID        string `json:"session_id"`
	SystemID         string `json:"system_id"`
	SystemName       string `json:"system_name"`
	SystemType       string `json:"system_type"`
	NodeID           string `json:"node_id,omitempty"`
}

// AuthorizeSSH handles POST /api/auth/ssh-authorize
// Called from the browser after Logto OAuth login.
// Validates the nonce, checks RBAC access to the system, and stores
// the auth result in Redis for the SSH gateway to consume.
func AuthorizeSSH(c *gin.Context) {
	_, userOrgID, userOrgRole, _ := helpers.GetUserContextExtended(c)
	user, _ := helpers.GetUserFromContext(c)

	var req sshAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body", nil))
		return
	}

	redisClient := cache.GetRedisClient()
	if redisClient == nil {
		logger.Error().Msg("Redis not available for SSH auth")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("service unavailable", nil))
		return
	}

	// Look up and consume the nonce (atomic get+delete)
	nonceKey := "ssh_auth:nonce:" + req.Code
	var nonceData sshNonceData
	if err := redisClient.GetDel(nonceKey, &nonceData); err != nil {
		logger.Warn().Str("code", req.Code).Msg("invalid or expired SSH auth code")
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid or expired auth code", nil))
		return
	}

	systemKey := nonceData.SystemKey

	// Look up system by system_key and check RBAC access
	sys, err := findSystemAndSession(systemKey, userOrgRole, userOrgID)
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("SSH auth system access denied")
		c.JSON(http.StatusForbidden, response.Forbidden("system not found or access denied", nil))
		return
	}

	// Store auth result in Redis for the SSH gateway to pick up
	userName := ""
	userEmail := ""
	userIDStr := ""
	if user != nil {
		userName = user.Name
		userEmail = user.Email
		userIDStr = user.ID
	}

	result := sshAuthResult{
		UserID:           userIDStr,
		Username:         userName,
		UserEmail:        userEmail,
		OrganizationName: sys.OrgName,
		SessionID:        sys.Session,
		SystemID:         sys.ID,
		SystemName:       sys.Name,
		SystemType:       sys.Type,
	}
	resultKey := "ssh_auth:result:" + req.Code
	if err := redisClient.Set(resultKey, result, 60*time.Second); err != nil {
		logger.Error().Err(err).Msg("failed to store SSH auth result")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to authorize", nil))
		return
	}

	logger.LogBusinessOperation(c, "auth", "ssh_authorize", "system", sys.ID, true, nil)

	c.JSON(http.StatusOK, response.OK("ssh access authorized", gin.H{
		"system_key":        systemKey,
		"system_id":         sys.ID,
		"system_name":       sys.Name,
		"system_type":       sys.Type,
		"organization_name": sys.OrgName,
		"session_id":        sys.Session,
		"key_auth_ttl":      "4h",
	}))
}

// systemInfo holds system details for the SSH authorize response
type systemInfo struct {
	ID      string
	Name    string
	Type    string
	OrgName string
	OrgID   string
	Session string
}

// findSystemAndSession looks up a system by system_key, checks RBAC access,
// and finds an active support session for it.
func findSystemAndSession(systemKey, userOrgRole, userOrgID string) (*systemInfo, error) {
	// Find system by system_key
	var systemID, orgID, systemName string
	var systemType sql.NullString
	var supportEnabled bool
	err := database.DB.QueryRow(
		`SELECT s.id, s.name, s.type, s.organization_id, s.support_enabled
		 FROM systems s
		 WHERE s.system_key = $1 AND s.deleted_at IS NULL AND s.suspended_at IS NULL`,
		systemKey,
	).Scan(&systemID, &systemName, &systemType, &orgID, &supportEnabled)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("system not found")
		}
		return nil, fmt.Errorf("failed to query system: %w", err)
	}

	if !supportEnabled {
		return nil, fmt.Errorf("support not enabled for system")
	}

	// Check RBAC access: user's org can see this system
	if !hasOrgAccess(userOrgRole, userOrgID, orgID) {
		return nil, fmt.Errorf("access denied")
	}

	// Get organization name
	var orgName string
	_ = database.DB.QueryRow(
		`SELECT name FROM unified_organizations WHERE logto_id = $1`, orgID,
	).Scan(&orgName)

	// Find active support session for this system
	var sessionID string
	err = database.DB.QueryRow(
		`SELECT id FROM support_sessions
		 WHERE system_id = $1 AND status IN ('pending', 'active')
		 ORDER BY created_at DESC LIMIT 1`,
		systemID,
	).Scan(&sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active support session for system")
		}
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	sysType := ""
	if systemType.Valid {
		sysType = systemType.String
	}

	return &systemInfo{
		ID:      systemID,
		Name:    systemName,
		Type:    sysType,
		OrgName: orgName,
		OrgID:   orgID,
		Session: sessionID,
	}, nil
}

// hasOrgAccess checks if a user with the given org role and org ID
// can access systems belonging to the target organization.
func hasOrgAccess(userOrgRole, userOrgID, targetOrgID string) bool {
	switch strings.ToLower(userOrgRole) {
	case "owner":
		return true
	case "distributor":
		if userOrgID == targetOrgID {
			return true
		}
		var exists bool
		err := database.DB.QueryRow(
			`SELECT EXISTS(
				SELECT 1 FROM resellers WHERE logto_id = $1 AND custom_data->>'createdBy' = $2 AND deleted_at IS NULL
				UNION
				SELECT 1 FROM customers WHERE logto_id = $1 AND deleted_at IS NULL AND (
					custom_data->>'createdBy' = $2 OR
					custom_data->>'createdBy' IN (
						SELECT logto_id FROM resellers WHERE custom_data->>'createdBy' = $2 AND deleted_at IS NULL
					)
				)
			)`, targetOrgID, userOrgID,
		).Scan(&exists)
		if err != nil {
			logger.Error().Err(err).
				Str("user_org_role", userOrgRole).
				Str("user_org_id", userOrgID).
				Str("target_org_id", targetOrgID).
				Msg("failed to query org access for distributor")
			return false
		}
		return exists
	case "reseller":
		if userOrgID == targetOrgID {
			return true
		}
		var exists bool
		err := database.DB.QueryRow(
			`SELECT EXISTS(
				SELECT 1 FROM customers WHERE logto_id = $1 AND custom_data->>'createdBy' = $2 AND deleted_at IS NULL
			)`, targetOrgID, userOrgID,
		).Scan(&exists)
		if err != nil {
			logger.Error().Err(err).
				Str("user_org_role", userOrgRole).
				Str("user_org_id", userOrgID).
				Str("target_org_id", targetOrgID).
				Msg("failed to query org access for reseller")
			return false
		}
		return exists
	case "customer":
		return userOrgID == targetOrgID
	default:
		return false
	}
}
