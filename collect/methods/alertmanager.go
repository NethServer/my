/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package methods

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/response"
)

// alertHistoryRequestTimeout caps the per-webhook DB work. Typical payloads
// hold 1–50 resolved alerts and each costs ~10ms of DB time, so 15s is well
// above expected and bounds the worst case.
const alertHistoryRequestTimeout = 15 * time.Second

// alertHistoryMaxBodyBytes caps the JSON body size for alertmanager webhook
// payloads. Alertmanager sends JSON arrays of resolved alerts; legitimate
// payloads are at most a few KB. Cap at 1 MiB so a holder of the webhook
// secret cannot exhaust memory or DB connections by shipping a giant body.
const alertHistoryMaxBodyBytes = 1 << 20 // 1 MiB

// alertHistoryMaxAlertsPerPayload caps the number of alerts processed in a
// single webhook payload. Each alert costs an extra DB lookup + insert, so
// a long array would still be expensive even within the byte cap.
const alertHistoryMaxAlertsPerPayload = 1000

// zeroTime is Alertmanager's sentinel for "no end time" on firing alerts.
var zeroTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

// ReceiveAlertHistory handles POST /api/alert_history.
// It persists resolved alerts from Alertmanager webhook payloads.
// Firing alerts are ignored; only resolved alerts contain valid startsAt/endsAt.
func ReceiveAlertHistory(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, alertHistoryMaxBodyBytes)
	var payload models.AlertmanagerWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("invalid request body: "+err.Error(), nil))
		return
	}
	if len(payload.Alerts) > alertHistoryMaxAlertsPerPayload {
		logger.Warn().
			Int("alerts_received", len(payload.Alerts)).
			Int("alerts_max", alertHistoryMaxAlertsPerPayload).
			Msg("alertmanager history: rejecting oversized payload")
		c.JSON(http.StatusRequestEntityTooLarge, response.Error(http.StatusRequestEntityTooLarge, "too many alerts in payload", nil))
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), alertHistoryRequestTimeout)
	defer cancel()

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

		// Resolve organization_id from the systems table. The webhook payload is
		// attacker-influenceable (labels travel through Alertmanager), so we
		// never trust a claimed organization_id from the payload — the DB is
		// authoritative. Unknown system_keys are dropped.
		var organizationID string
		err := database.DB.QueryRowContext(ctx,
			`SELECT organization_id FROM systems WHERE system_key = $1 AND deleted_at IS NULL`,
			systemKey,
		).Scan(&organizationID)
		if err == sql.ErrNoRows {
			logger.Warn().
				Str("fingerprint", alert.Fingerprint).
				Str("system_key", systemKey).
				Msg("alertmanager history: skipping alert for unknown system_key")
			continue
		}
		if err != nil {
			logger.Error().
				Err(err).
				Str("fingerprint", alert.Fingerprint).
				Str("system_key", systemKey).
				Msg("alertmanager history: failed to resolve organization_id")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert history", nil))
			return
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

		// Convert Alertmanager zero-time sentinel to NULL
		var endsAt *time.Time
		if !alert.EndsAt.Equal(zeroTime) {
			endsAt = &alert.EndsAt
		}

		err = persistResolvedAlertHistory(
			ctx,
			alert,
			systemKey,
			organizationID,
			alertname,
			severity,
			summary,
			payload.Receiver,
			labelsJSON,
			annotationsJSON,
			endsAt,
		)
		if err != nil {
			logger.Error().
				Err(err).
				Str("fingerprint", alert.Fingerprint).
				Str("system_key", systemKey).
				Msg("alertmanager history: failed to persist alert")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert history", nil))
			return
		}

		saved++
	}

	logger.Debug().
		Str("receiver", payload.Receiver).
		Str("status", payload.Status).
		Int("total_alerts", len(payload.Alerts)).
		Int("saved", saved).
		Msg("alertmanager history: processed webhook payload")

	c.Status(http.StatusNoContent)
}

// LinkFailed refreshes can emit multiple resolved webhooks for the same outage.
// starts_at stays stable for that outage, so reuse it as the history key.
func persistResolvedAlertHistory(
	ctx context.Context,
	alert models.AlertmanagerAlert,
	systemKey, organizationID, alertname, severity, summary, receiver string,
	labelsJSON, annotationsJSON []byte,
	endsAt *time.Time,
) error {
	if alertname == collectalerting.LinkFailedAlert {
		updated, err := updateExistingLinkFailedHistory(
			ctx,
			alert,
			systemKey,
			organizationID,
			alertname,
			severity,
			summary,
			receiver,
			labelsJSON,
			annotationsJSON,
			endsAt,
		)
		if err != nil {
			return err
		}
		if updated {
			return nil
		}
	}

	return insertAlertHistory(
		ctx,
		alert,
		systemKey,
		organizationID,
		alertname,
		severity,
		summary,
		receiver,
		labelsJSON,
		annotationsJSON,
		endsAt,
	)
}

func updateExistingLinkFailedHistory(
	ctx context.Context,
	alert models.AlertmanagerAlert,
	systemKey, organizationID, alertname, severity, summary, receiver string,
	labelsJSON, annotationsJSON []byte,
	endsAt *time.Time,
) (bool, error) {
	result, err := database.DB.ExecContext(ctx,
		`WITH existing AS (
			SELECT id
			FROM alert_history
			WHERE system_key = $1
			  AND alertname = $2
			  AND starts_at = $3
			  AND status = 'resolved'
			ORDER BY created_at DESC, id DESC
			LIMIT 1
		)
		UPDATE alert_history ah
		SET severity = $4,
		    status = 'resolved',
		    fingerprint = $5,
		    ends_at = $6,
		    summary = $7,
		    labels = $8,
		    annotations = $9,
		    receiver = $10,
		    organization_id = $11,
		    created_at = NOW()
		FROM existing
		WHERE ah.id = existing.id`,
		systemKey,
		alertname,
		alert.StartsAt,
		nullableString(severity),
		alert.Fingerprint,
		endsAt,
		nullableString(summary),
		labelsJSON,
		annotationsJSON,
		nullableString(receiver),
		organizationID,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowsAffected > 0, nil
}

func insertAlertHistory(
	ctx context.Context,
	alert models.AlertmanagerAlert,
	systemKey, organizationID, alertname, severity, summary, receiver string,
	labelsJSON, annotationsJSON []byte,
	endsAt *time.Time,
) error {
	_, err := database.DB.ExecContext(ctx,
		`INSERT INTO alert_history
			(system_key, organization_id, alertname, severity, status, fingerprint, starts_at, ends_at, summary, labels, annotations, receiver)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		systemKey,
		organizationID,
		alertname,
		nullableString(severity),
		"resolved",
		alert.Fingerprint,
		alert.StartsAt,
		endsAt,
		nullableString(summary),
		labelsJSON,
		annotationsJSON,
		nullableString(receiver),
	)
	return err
}

// nullableString returns nil for empty strings so they are stored as NULL.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
