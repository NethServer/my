/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/response"
)

// GetSystemInfo returns information about the authenticated system
func GetSystemInfo(c *gin.Context) {
	systemID, exists := c.Get("system_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	systemKey, exists := c.Get("system_key")
	if !exists {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}

	query := `
		SELECT s.id, s.system_key, s.name, s.type, s.fqdn, s.status,
		       s.suspended_at, s.deleted_at, s.registered_at, s.created_at,
		       s.organization_id,
		       COALESCE(d.id, r.id, c.id, s.organization_id) AS org_db_id,
		       COALESCE(d.logto_id, r.logto_id, c.logto_id, s.organization_id) AS org_logto_id,
		       COALESCE(d.name, r.name, c.name, 'Owner') AS org_name,
		       CASE
		           WHEN d.logto_id IS NOT NULL THEN 'distributor'
		           WHEN r.logto_id IS NOT NULL THEN 'reseller'
		           WHEN c.logto_id IS NOT NULL THEN 'customer'
		           ELSE 'owner'
		       END AS org_type,
		       COALESCE(d.suspended_at, r.suspended_at, c.suspended_at) AS org_suspended_at
		FROM systems s
		LEFT JOIN distributors d ON (s.organization_id = d.logto_id OR s.organization_id = d.id) AND d.deleted_at IS NULL
		LEFT JOIN resellers r ON (s.organization_id = r.logto_id OR s.organization_id = r.id) AND r.deleted_at IS NULL
		LEFT JOIN customers c ON (s.organization_id = c.logto_id OR s.organization_id = c.id) AND c.deleted_at IS NULL
		WHERE s.id = $1
	`

	var (
		sysID          string
		sysKey         string
		name           string
		sysType        *string
		fqdn           *string
		status         string
		suspendedAt    *time.Time
		deletedAt      *time.Time
		registeredAt   *time.Time
		createdAt      time.Time
		organizationID string
		orgDBID        *string
		orgLogtoID     *string
		orgName        string
		orgType        string
		orgSuspendedAt *time.Time
	)

	err := database.DB.QueryRow(query, systemID.(string)).Scan(
		&sysID, &sysKey, &name, &sysType, &fqdn, &status,
		&suspendedAt, &deletedAt, &registeredAt, &createdAt,
		&organizationID, &orgDBID, &orgLogtoID, &orgName, &orgType, &orgSuspendedAt,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}
	if err != nil {
		logger.Error().
			Str("component", "system_info").
			Str("operation", "database_query").
			Str("system_id", systemID.(string)).
			Str("system_key", systemKey.(string)).
			Err(err).
			Msg("failed to query system info")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to retrieve system info", nil))
		return
	}

	// Return 404 if system is deleted
	if deletedAt != nil {
		c.JSON(http.StatusNotFound, response.NotFound("system not found", nil))
		return
	}

	info := models.SystemInfo{
		SystemID:     sysID,
		SystemKey:    sysKey,
		Name:         name,
		Type:         sysType,
		FQDN:         fqdn,
		Status:       status,
		Suspended:    suspendedAt != nil,
		SuspendedAt:  suspendedAt,
		Deleted:      false,
		DeletedAt:    nil,
		Registered:   registeredAt != nil,
		RegisteredAt: registeredAt,
		CreatedAt:    createdAt,
		Organization: models.SystemInfoOrg{
			ID:          derefString(orgDBID),
			LogtoID:     derefString(orgLogtoID),
			Name:        orgName,
			Type:        orgType,
			Suspended:   orgSuspendedAt != nil,
			SuspendedAt: orgSuspendedAt,
		},
	}

	c.JSON(http.StatusOK, response.OK("system info retrieved successfully", info))
}

// derefString returns the value of a string pointer or empty string if nil
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
