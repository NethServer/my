/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/middleware"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/response"
)

// UnregisterSystem handles the POST /api/systems/unregister endpoint: an
// appliance announces that it is giving up its credentials, and collect stops
// accepting them from that moment on. Whoever kept a copy of the pair loses
// heartbeat, inventory, backups, the alert proxy and the enterprise feeds along
// with the machine that gave them up.
//
// The state is terminal. A system key is one-shot, so this row cannot be
// registered again: it stays as the record of a spent key, still counted
// against the licences, until someone deletes the system on my. A machine that
// needs a subscription again is given a new system with a new secret.
//
// Only the first call answers 200. Its own revocation makes every later request
// from the same credentials, this one included, fail authentication.
func UnregisterSystem(c *gin.Context) {
	systemID, ok := getAuthenticatedSystemID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, response.Unauthorized("authentication required", nil))
		return
	}
	systemKey, _ := getAuthenticatedSystemKey(c)

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	now := time.Now()

	// The CTE snapshots the status the row carries before the write. A system
	// reading 'inactive' has a LinkFailed alert firing that has to be cleared
	// here: the monitor refreshing it skips unregistered systems, and an alert
	// nobody refreshes hangs around until its TTL.
	var previousStatus string
	err := database.DB.QueryRowContext(ctx, `
		WITH previous AS (
			SELECT status FROM systems WHERE id = $1
		)
		UPDATE systems
		SET unregistered_at = $2, status = 'unregistered', updated_at = $2
		WHERE id = $1 AND unregistered_at IS NULL
		RETURNING (SELECT status FROM previous)
	`, systemID, now).Scan(&previousStatus)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error().
			Err(err).
			Str("system_key", systemKey).
			Str("system_id", systemID).
			Msg("failed to unregister system")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to unregister system", nil))
		return
	}

	// ErrNoRows means a concurrent call won the update. The credentials are
	// revoked either way, which is all the caller asked for.

	// Detached from the request: an appliance that hangs up right after sending
	// must not leave the revoked credentials cached until the TTL.
	revokeCtx, revokeCancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Second)
	defer revokeCancel()

	middleware.InvalidateSystemAuth(revokeCtx, systemKey)

	if previousStatus == "inactive" {
		resolveLinkFailedOnUnregister(revokeCtx, systemID, systemKey)
	}

	logger.Info().
		Str("system_key", systemKey).
		Str("system_id", systemID).
		Str("previous_status", previousStatus).
		Msg("system unregistered")

	c.JSON(http.StatusOK, response.OK("system unregistered", models.UnregisterResponse{
		SystemKey:      systemKey,
		UnregisteredAt: now,
	}))
}

// resolveLinkFailedOnUnregister clears the LinkFailed alert of a system that had
// stopped reporting before giving up its credentials. The labels are rebuilt
// from the system's row, the same path the firing alert takes, so the
// fingerprints match and Alertmanager clears it.
func resolveLinkFailedOnUnregister(ctx context.Context, systemID, systemKey string) {
	systemContext, err := collectalerting.LookupSystemAlertContext(ctx, database.DB, systemID)
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).
			Msg("unregister: failed to load alert context")
		return
	}
	if systemContext.OrganizationID == "" || systemContext.ResellerOrgID == "" {
		return
	}

	alert, err := collectalerting.BuildUnregisteredLinkFailedAlert(systemContext)
	if err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).
			Msg("unregister: failed to build resolved alert")
		return
	}

	if err := collectalerting.PostAlerts(systemContext.ResellerOrgID, []models.AlertmanagerPostAlert{alert}); err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).
			Msg("unregister: failed to resolve LinkFailed alert")
	}
}
