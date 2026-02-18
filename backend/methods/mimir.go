/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/response"
)

// ProxyMimir handles ANY /api/mimir/* — authenticates system credentials,
// adds X-Scope-OrgID, and reverse-proxies to Mimir.
func ProxyMimir(c *gin.Context) {
	// Step 1: Parse Authorization: Basic <base64> header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Basic ") {
		logger.Warn().Str("reason", "missing or invalid authorization header").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
	if err != nil {
		logger.Warn().Str("reason", "failed to decode basic auth").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		logger.Warn().Str("reason", "malformed basic auth credentials").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	username := parts[0]
	password := parts[1]

	// Step 2: Split password on first '.' to extract publicPart and secretPart
	if !strings.HasPrefix(password, "my_") {
		logger.Warn().Str("reason", "password missing my_ prefix").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	dotParts := strings.SplitN(password, ".", 2)
	if len(dotParts) != 2 {
		logger.Warn().Str("reason", "password missing dot separator").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	publicPart := strings.TrimPrefix(dotParts[0], "my_")
	secretPart := dotParts[1]

	if publicPart == "" || secretPart == "" {
		logger.Warn().Str("reason", "empty public or secret part").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	// Step 3: Query DB for the system
	var dbID int
	var dbSystemKey string
	var dbSystemSecret string
	var dbOrganizationID string

	err = database.DB.QueryRow(
		`SELECT id, system_key, system_secret, organization_id
		 FROM systems
		 WHERE system_secret_public = $1
		   AND deleted_at IS NULL
		   AND suspended_at IS NULL`,
		publicPart,
	).Scan(&dbID, &dbSystemKey, &dbSystemSecret, &dbOrganizationID)

	if err == sql.ErrNoRows {
		logger.Warn().Str("reason", "system not found").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}
	if err != nil {
		logger.Error().Err(err).Msg("mimir proxy: db query failed")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	// Step 4: Verify system_key matches username
	if dbSystemKey != username {
		logger.Warn().Str("reason", "system_key mismatch").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	// Step 5: Verify secret
	ok, err := helpers.VerifySystemSecret(secretPart, dbSystemSecret)
	if err != nil {
		logger.Error().Err(err).Msg("mimir proxy: secret verification error")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}
	if !ok {
		logger.Warn().Str("reason", "secret verification failed").Msg("mimir proxy auth failed")
		c.JSON(http.StatusUnauthorized, response.Unauthorized("unauthorized", nil))
		return
	}

	// Step 6: Build target URL
	subPath := c.Param("path")
	targetURL := fmt.Sprintf("%s%s", configuration.Config.MimirURL, subPath)
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	// Step 7: Create upstream request
	req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		logger.Error().Err(err).Str("target", targetURL).Msg("mimir proxy: failed to create upstream request")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}

	// Step 8: Copy relevant request headers
	for _, header := range []string{"Content-Type", "Content-Encoding", "Accept", "Accept-Encoding", "User-Agent"} {
		if val := c.GetHeader(header); val != "" {
			req.Header.Set(header, val)
		}
	}

	// Step 9: Set X-Scope-OrgID
	req.Header.Set("X-Scope-OrgID", dbOrganizationID)

	// Step 10: Execute upstream request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error().Err(err).Str("target", targetURL).Msg("mimir proxy: upstream request failed")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("internal server error", nil))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error().Err(err).Msg("mimir proxy: failed to close upstream response body")
		}
	}()

	// Step 11: Copy response headers, status, and stream body
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Header("Content-Type", ct)
	}
	if ce := resp.Header.Get("Content-Encoding"); ce != "" {
		c.Header("Content-Encoding", ce)
	}

	c.Status(resp.StatusCode)

	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		logger.Error().Err(err).Msg("mimir proxy: error streaming response body")
	}
}
