/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// maxLegacySystemCounts bounds one push. The legacy portal has fewer than 1200
// resellers, so this leaves generous headroom while keeping a malformed or
// hostile payload from turning into an unbounded transaction.
const maxLegacySystemCounts = 5000

// ReplaceLegacySystemCounts stores how many systems each organization still has
// on the old my. It is called by the proxy_sync cron on the legacy host, with
// the owner API key, once per run.
//
// The payload is the COMPLETE picture, and it replaces the table wholesale: an
// organization missing from it is an organization with no legacy systems left.
// That is what lets the counters fall to zero as machines migrate, which an
// upsert-only endpoint could never do.
//
// Transitional by design: it exists for as long as two portals answer for the
// same partner, and goes away with the legacy decommission.
func ReplaceLegacySystemCounts(c *gin.Context) {
	var req models.ReplaceLegacySystemCountsRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, response.ValidationBadRequestMultiple(err))
		return
	}

	if len(req.Counts) > maxLegacySystemCounts {
		c.JSON(http.StatusBadRequest, response.BadRequest("too many organizations in one push", nil))
		return
	}

	counts := make(map[string]int, len(req.Counts))
	for _, entry := range req.Counts {
		if entry.OrganizationID == "" {
			c.JSON(http.StatusBadRequest, response.BadRequest("organization_id cannot be empty", nil))
			return
		}
		if entry.Total < 0 {
			c.JSON(http.StatusBadRequest, response.BadRequest("total cannot be negative", nil))
			return
		}
		counts[entry.OrganizationID] = entry.Total
	}

	repo := entities.NewLocalLegacySystemsByOrgRepository()
	if err := repo.ReplaceAll(counts); err != nil {
		logger.RequestLogger(c, "legacy_systems").Error().
			Err(err).
			Int("organizations", len(counts)).
			Msg("Failed to replace legacy systems counts")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to store legacy systems counts", nil))
		return
	}

	total := 0
	for _, n := range counts {
		total += n
	}

	logger.RequestLogger(c, "legacy_systems").Info().
		Int("organizations", len(counts)).
		Int("systems", total).
		Msg("Legacy systems counts replaced")

	c.JSON(http.StatusOK, response.OK("legacy systems counts stored", gin.H{
		"organizations": len(counts),
		"systems":       total,
	}))
}
