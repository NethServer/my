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
	"math/rand"
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

	// Step 4: Try each Mimir instance starting from a random index
	urls := configuration.Config.MimirURLs
	n := len(urls)
	start := rand.Intn(n)

	for i := 0; i < n; i++ {
		base := urls[(start+i)%n]
		targetURL := fmt.Sprintf("%s%s", base, subPath)
		if rawQuery != "" {
			targetURL += "?" + rawQuery
		}

		logger.Info().Str("target", targetURL).Int("attempt", i+1).Msg("mimir proxy: trying instance")

		req, err := http.NewRequest(c.Request.Method, targetURL, bytes.NewReader(bodyBytes))
		if err != nil {
			logger.Warn().Err(err).Str("target", targetURL).Msg("mimir proxy: failed to create upstream request")
			continue
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
			logger.Warn().Err(err).Str("target", targetURL).Msg("mimir proxy: network error, trying next instance")
			continue
		}

		// Return 4xx immediately — client errors are not retried
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
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
			return
		}

		// Retry on 5xx
		if resp.StatusCode >= 500 {
			if err := resp.Body.Close(); err != nil {
				logger.Error().Err(err).Msg("mimir proxy: failed to close upstream response body")
			}
			logger.Warn().Str("target", targetURL).Int("status", resp.StatusCode).Msg("mimir proxy: 5xx response, trying next instance")
			continue
		}

		// Success: stream response back to client
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
		return
	}

	// All instances failed
	logger.Error().Int("instances_tried", n).Msg("mimir proxy: all instances failed")
	c.JSON(http.StatusBadGateway, response.InternalServerError("all mimir instances are unavailable", nil))
}
