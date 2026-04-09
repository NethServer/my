/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package methods

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/response"
)

// ReceiveAlertHistory handles POST /api/alert_history.
// It persists resolved alerts from Alertmanager webhook payloads.
// Firing alerts are ignored; only resolved alerts contain valid startsAt/endsAt.
func ReceiveAlertHistory(c *gin.Context) {
	var payload models.AlertmanagerWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}

	saved := 0
	for _, alert := range payload.Alerts {
		if strings.ToLower(alert.Status) != "resolved" {
			continue
		}

		systemKey := alert.Labels["system_key"]
		if systemKey == "" {
			logger.Warn().
				Str("fingerprint", alert.Fingerprint).
				Msg("alertmanager history: skipping alert without system_key label")
			continue
		}

		alertname := alert.Labels["alertname"]
		severity := alert.Labels["severity"]
		summary := alert.Annotations["summary"]

		labelsJSON, err := json.Marshal(alert.Labels)
		if err != nil {
			logger.Error().Err(err).Str("fingerprint", alert.Fingerprint).Msg("alertmanager history: failed to marshal labels")
			continue
		}

		annotationsJSON, err := json.Marshal(alert.Annotations)
		if err != nil {
			logger.Error().Err(err).Str("fingerprint", alert.Fingerprint).Msg("alertmanager history: failed to marshal annotations")
			continue
		}

		_, err = database.DB.Exec(
			`INSERT INTO alert_history
				(system_key, alertname, severity, status, fingerprint, starts_at, ends_at, summary, labels, annotations, receiver)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			systemKey,
			alertname,
			nullableString(severity),
			"resolved",
			alert.Fingerprint,
			alert.StartsAt,
			alert.EndsAt,
			nullableString(summary),
			labelsJSON,
			annotationsJSON,
			nullableString(payload.Receiver),
		)
		if err != nil {
			logger.Error().
				Err(err).
				Str("fingerprint", alert.Fingerprint).
				Str("system_key", systemKey).
				Msg("alertmanager history: failed to insert alert")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert history", nil))
			return
		}

		saved++
	}

	logger.Info().
		Str("receiver", payload.Receiver).
		Str("status", payload.Status).
		Int("total_alerts", len(payload.Alerts)).
		Int("saved", saved).
		Msg("alertmanager history: processed webhook payload")

	c.Status(http.StatusNoContent)
}

// nullableString returns nil for empty strings so they are stored as NULL.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
