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

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/response"
)

// ProxyMimir handles ANY /api/services/mimir/* — the BasicAuthMiddleware has
// already validated system credentials and placed system_id in the context.
// This handler resolves the organization_id, sets X-Scope-OrgID, and
// reverse-proxies the request to Mimir with HA support across multiple instances.
func ProxyMimir(c *gin.Context) {
	// Step 1: Get system_id from context (set by BasicAuthMiddleware)
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		logger.Warn().Str("reason", "missing system_id in context").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	// Step 2: Query organization_id for this system
	var organizationID string
	err := database.DB.QueryRow(
		`SELECT organization_id FROM systems WHERE id = $1`,
		systemID,
	).Scan(&organizationID)

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

	// Step 3: Buffer request body once so it can be replayed across retry attempts
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Msg("mimir proxy: failed to read request body")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	subPath := c.Param("path")
	rawQuery := c.Request.URL.RawQuery

	// Step 4: Forward request to Mimir
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, subPath)
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	logger.Info().Str("target", targetURL).Msg("mimir proxy: forwarding request")

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
	req.Header.Set("X-Scope-OrgID", organizationID)

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
