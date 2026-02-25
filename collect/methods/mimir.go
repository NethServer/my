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

// ProxyMimir forwards requests to Mimir on behalf of authenticated systems.
// BasicAuthMiddleware has already validated credentials and set "system_id" in the context.
// Route matching in main.go restricts access to /alertmanager/api/v2/alerts and
// /alertmanager/api/v2/silences; no further path checks are needed here.
// X-Scope-OrgID is always injected using the system's organization_id.
func ProxyMimir(c *gin.Context) {
	subPath := strings.TrimPrefix(c.Request.URL.Path, "/api/services/mimir")
	rawQuery := c.Request.URL.RawQuery

	// Resolve organization_id for X-Scope-OrgID injection
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		logger.Warn().Str("reason", "missing system_id in context").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	var orgID string
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
	req.Header.Set("X-Scope-OrgID", orgID)

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
