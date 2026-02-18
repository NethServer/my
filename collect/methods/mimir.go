/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

// mimirAllowedPaths is the whitelist of path prefixes that non-admin (system) callers
// may access. Everything else requires admin credentials.
// Paths are matched by prefix after stripping the leading slash.
var mimirAllowedPaths = []string{
	// Per-tenant alertmanager configuration (GET / POST / DELETE)
	"/api/v1/alerts",
	// Alertmanager v2 API – alert injection and listing (POST / GET)
	"/alertmanager/api/v2/alerts",
	// Alertmanager v2 API – silence management (GET / POST / DELETE)
	"/alertmanager/api/v2/silences",
	// Read-only build info
	"/alertmanager/api/v1/status/buildinfo",
}

// isAllowedPath returns true when subPath is on the system whitelist.
func isAllowedPath(subPath string) bool {
	for _, allowed := range mimirAllowedPaths {
		if subPath == allowed || strings.HasPrefix(subPath, allowed+"/") {
			return true
		}
	}
	return false
}

// ProxyMimir handles ANY /api/services/mimir/* — MimirAuthMiddleware has already
// validated credentials and set either "mimir_is_admin" = true (admin user) or
// "system_id" / "system_key" (authenticated system) in the context.
//
// Admin users:  all paths allowed, no X-Scope-OrgID injected.
// System users: only whitelisted alertmanager paths allowed, X-Scope-OrgID injected.
func ProxyMimir(c *gin.Context) {
	isAdmin, _ := c.Get("mimir_is_admin")

	subPath := c.Param("path")
	rawQuery := c.Request.URL.RawQuery

	var orgID string

	if isAdmin == true {
		// Admin access – no tenant scoping, no path restriction
		logger.Info().Str("sub_path", subPath).Msg("mimir proxy: admin access")
	} else {
		// System access – enforce whitelist
		if !isAllowedPath(subPath) {
			logger.Warn().
				Str("sub_path", subPath).
				Str("client_ip", c.ClientIP()).
				Msg("mimir proxy: system attempted to access restricted path")
			c.JSON(http.StatusForbidden, response.Error(http.StatusForbidden, "access to this path is not allowed", nil))
			return
		}

		// Resolve organization_id for X-Scope-OrgID injection
		systemID, ok := getAuthenticatedSystemID(c)
		if !ok {
			logger.Warn().Str("reason", "missing system_id in context").Msg("mimir proxy auth failed")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
			return
		}

		err := database.DB.QueryRow(
			`SELECT organization_id FROM systems WHERE id = $1`,
			systemID,
		).Scan(&orgID)

		if err == sql.ErrNoRows {
			logger.Warn().Str("system_id", systemID).Str("reason", "system not found").Msg("mimir proxy: system lookup failed")
			c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
			return
		}
		if err != nil {
			logger.Error().Err(err).Str("system_id", systemID).Msg("mimir proxy: db query failed")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
			return
		}
	}

	// Buffer request body once so it can be replayed across retry attempts
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("mimir proxy: failed to read request body")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	// Forward request to Mimir
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, subPath)
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	logger.Info().Str("target", targetURL).Str("org_id", orgID).Msg("mimir proxy: forwarding request")

	req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		logger.Error().Err(err).Str("target", targetURL).Msg("mimir proxy: failed to create upstream request")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	for _, header := range []string{"Content-Type", "Content-Encoding", "Accept", "User-Agent"} {
		if val := c.GetHeader(header); val != "" {
			req.Header.Set(header, val)
		}
	}
	// Remove Accept-Encoding so Mimir sends plain JSON, not gzip
	req.Header.Del("Accept-Encoding")
	if orgID != "" {
		req.Header.Set("X-Scope-OrgID", orgID)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("target", targetURL).Msg("mimir proxy: network error")
		c.JSON(http.StatusBadGateway, response.InternalServerError("mimir is unavailable", nil))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error().Err(err).Msg("mimir proxy: failed to close upstream response body")
		}
	}()

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	}
	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		logger.Error().Err(err).Msg("mimir proxy: error streaming response body")
	}
}
