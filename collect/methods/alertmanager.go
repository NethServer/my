/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package methods

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"

	collectalerting "github.com/nethesis/my/collect/alerting"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/response"
)

// alertHistoryRequestTimeout caps the per-webhook DB work. Mimir Alertmanager
// times out webhooks client-side around 10s, so we keep our budget below that
// and let the bulk path (3 round-trips total) finish well within it.
const alertHistoryRequestTimeout = 8 * time.Second

// alertHistoryMaxBodyBytes caps the JSON body size for alertmanager webhook
// payloads. Alertmanager sends JSON arrays of resolved alerts; legitimate
// payloads are at most a few KB. Cap at 1 MiB so a holder of the webhook
// secret cannot exhaust memory or DB connections by shipping a giant body.
const alertHistoryMaxBodyBytes = 1 << 20 // 1 MiB

// alertHistoryMaxAlertsPerPayload caps the number of alerts processed in a
// single webhook payload. The bulk path scales linearly with array size on
// the wire, but we still keep an upper bound so a malicious payload can't
// blow up memory while building parameter arrays.
const alertHistoryMaxAlertsPerPayload = 1000

// zeroTime is Alertmanager's sentinel for "no end time" on firing alerts.
var zeroTime = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)

// pendingAlert is the per-alert state carried from JSON parsing through to
// the bulk DB stage. labels/annotations are serialized once and reused.
type pendingAlert struct {
	alert           models.AlertmanagerAlert
	systemKey       string
	organizationID  string
	alertname       string
	severity        string
	summary         string
	labelsJSON      []byte
	annotationsJSON []byte
	endsAtValid     bool
	endsAt          time.Time
}

// ReceiveAlertHistory handles POST /api/alert_history.
// It persists resolved alerts from Alertmanager webhook payloads.
// Firing alerts are ignored; only resolved alerts contain valid startsAt/endsAt.
//
// The DB path is fully batched: one SELECT to resolve organization_ids for
// every system_key in the payload, one bulk UPDATE for LinkFailed refreshes,
// and one bulk INSERT for everything else. A 50-alert burst that used to
// take 100+ round-trips now takes 3.
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

	pending, systemKeys := preparePendingAlerts(payload.Alerts)
	if len(pending) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	orgByKey, err := resolveOrganizationIDs(ctx, systemKeys)
	if err != nil {
		logger.Error().Err(err).Msg("alertmanager history: failed to resolve organization_ids")
		c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert history", nil))
		return
	}

	deliverable := pending[:0]
	for _, p := range pending {
		org, ok := orgByKey[p.systemKey]
		if !ok {
			logger.Warn().
				Str("fingerprint", p.alert.Fingerprint).
				Str("system_key", p.systemKey).
				Msg("alertmanager history: skipping alert for unknown system_key")
			continue
		}
		p.organizationID = org
		deliverable = append(deliverable, p)
	}
	if len(deliverable) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	var linkFailed, others []pendingAlert
	for _, p := range deliverable {
		if p.alertname == collectalerting.LinkFailedAlert {
			linkFailed = append(linkFailed, p)
		} else {
			others = append(others, p)
		}
	}

	insertBatch := others
	if len(linkFailed) > 0 {
		notUpdated, err := bulkUpdateLinkFailed(ctx, linkFailed, payload.Receiver)
		if err != nil {
			logger.Error().Err(err).Msg("alertmanager history: bulk update LinkFailed failed")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert history", nil))
			return
		}
		insertBatch = append(insertBatch, notUpdated...)
	}

	if len(insertBatch) > 0 {
		if err := bulkInsertAlertHistory(ctx, insertBatch, payload.Receiver); err != nil {
			logger.Error().Err(err).Msg("alertmanager history: bulk insert failed")
			c.JSON(http.StatusInternalServerError, response.InternalServerError("failed to save alert history", nil))
			return
		}
	}

	releaseAlertAssignments(ctx, deliverable)

	logger.Debug().
		Str("receiver", payload.Receiver).
		Str("status", payload.Status).
		Int("total_alerts", len(payload.Alerts)).
		Int("saved", len(deliverable)).
		Msg("alertmanager history: processed webhook payload")

	c.Status(http.StatusNoContent)
}

// preparePendingAlerts filters the payload to resolved alerts with a usable
// system_key and pre-serializes labels/annotations. The DB is authoritative
// for organization_id; we never trust a value coming from the payload.
func preparePendingAlerts(alerts []models.AlertmanagerAlert) ([]pendingAlert, []string) {
	pending := make([]pendingAlert, 0, len(alerts))
	systemKeys := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		if !strings.EqualFold(alert.Status, "resolved") {
			continue
		}
		sk := alert.Labels["system_key"]
		if sk == "" {
			logger.Warn().
				Str("fingerprint", alert.Fingerprint).
				Msg("alertmanager history: skipping alert without system_key label")
			continue
		}
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
		p := pendingAlert{
			alert:           alert,
			systemKey:       sk,
			alertname:       alert.Labels["alertname"],
			severity:        alert.Labels["severity"],
			summary:         alert.Annotations["summary"],
			labelsJSON:      labelsJSON,
			annotationsJSON: annotationsJSON,
		}
		if !alert.EndsAt.Equal(zeroTime) {
			p.endsAtValid = true
			p.endsAt = alert.EndsAt
		}
		pending = append(pending, p)
		systemKeys = append(systemKeys, sk)
	}
	return pending, systemKeys
}

// resolveOrganizationIDs returns a system_key → organization_id map for the
// requested keys, in a single round-trip. Soft-deleted and unknown systems
// are simply absent from the map.
func resolveOrganizationIDs(ctx context.Context, systemKeys []string) (map[string]string, error) {
	rows, err := database.DB.QueryContext(ctx,
		`SELECT system_key, organization_id
		 FROM systems
		 WHERE system_key = ANY($1) AND deleted_at IS NULL`,
		pq.Array(systemKeys),
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]string, len(systemKeys))
	for rows.Next() {
		var k, org string
		if err := rows.Scan(&k, &org); err != nil {
			return nil, err
		}
		out[k] = org
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// bulkUpdateLinkFailed refreshes the latest resolved LinkFailed history row
// for each (system_key, starts_at) pair in batch, and returns the alerts
// that had no existing row to refresh so the caller can route them to a
// bulk insert.
//
// LinkFailed monitor emits multiple resolved webhooks for the same outage
// (starts_at is stable across the outage) so we collapse them in place
// instead of accumulating duplicates.
func bulkUpdateLinkFailed(ctx context.Context, batch []pendingAlert, receiver string) ([]pendingAlert, error) {
	n := len(batch)
	systemKeys := make([]string, n)
	startsAt := make([]time.Time, n)
	severity := make([]string, n)
	fingerprint := make([]string, n)
	endsAtTexts := make([]string, n) // empty string → NULL
	summary := make([]string, n)
	labels := make([]string, n)
	annotations := make([]string, n)
	orgIDs := make([]string, n)

	for i, p := range batch {
		systemKeys[i] = p.systemKey
		startsAt[i] = p.alert.StartsAt
		severity[i] = p.severity
		fingerprint[i] = p.alert.Fingerprint
		if p.endsAtValid {
			endsAtTexts[i] = p.endsAt.UTC().Format(time.RFC3339Nano)
		}
		summary[i] = p.summary
		labels[i] = string(p.labelsJSON)
		annotations[i] = string(p.annotationsJSON)
		orgIDs[i] = p.organizationID
	}

	// The CTE walks the input in array order, picks the most recent
	// resolved LinkFailed row per (system_key, starts_at), and updates it
	// in place. RETURNING idx tells the caller which inputs were absorbed.
	const query = `
WITH inputs AS (
    SELECT
        idx,
        system_key,
        starts_at,
        NULLIF(severity, '')             AS severity,
        fingerprint,
        NULLIF(ends_at_text, '')::timestamptz AS ends_at,
        NULLIF(summary, '')              AS summary,
        labels::jsonb                    AS labels,
        annotations::jsonb               AS annotations,
        NULLIF($11, '')                  AS receiver,
        organization_id
    FROM unnest(
        $1::text[],
        $2::timestamptz[],
        $3::text[],
        $4::text[],
        $5::text[],
        $6::text[],
        $7::text[],
        $8::text[],
        $9::text[]
    ) WITH ORDINALITY AS t(
        system_key, starts_at, severity, fingerprint, ends_at_text,
        summary, labels, annotations, organization_id, idx
    )
),
matched AS (
    SELECT DISTINCT ON (i.idx)
        ah.id, i.idx, i.severity, i.fingerprint, i.ends_at,
        i.summary, i.labels, i.annotations, i.receiver, i.organization_id
    FROM inputs i
    JOIN alert_history ah
      ON ah.system_key = i.system_key
     AND ah.alertname  = $10
     AND ah.starts_at  = i.starts_at
     AND ah.status     = 'resolved'
    ORDER BY i.idx, ah.created_at DESC, ah.id DESC
),
updated AS (
    UPDATE alert_history a
    SET severity        = m.severity,
        status          = 'resolved',
        fingerprint     = m.fingerprint,
        ends_at         = m.ends_at,
        summary         = m.summary,
        labels          = m.labels,
        annotations     = m.annotations,
        receiver        = m.receiver,
        organization_id = m.organization_id,
        created_at      = NOW()
    FROM matched m
    WHERE a.id = m.id
    RETURNING m.idx
)
SELECT idx FROM updated`

	rows, err := database.DB.QueryContext(ctx, query,
		pq.Array(systemKeys),
		pq.Array(startsAt),
		pq.Array(severity),
		pq.Array(fingerprint),
		pq.Array(endsAtTexts),
		pq.Array(summary),
		pq.Array(labels),
		pq.Array(annotations),
		pq.Array(orgIDs),
		collectalerting.LinkFailedAlert,
		receiver,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	updated := make(map[int]struct{}, n)
	for rows.Next() {
		var idx int
		if err := rows.Scan(&idx); err != nil {
			return nil, err
		}
		updated[idx] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	notUpdated := make([]pendingAlert, 0, n-len(updated))
	for i, p := range batch {
		// unnest WITH ORDINALITY is 1-based.
		if _, ok := updated[i+1]; ok {
			continue
		}
		notUpdated = append(notUpdated, p)
	}
	return notUpdated, nil
}

// releaseAlertAssignments auto-releases the assignment of every alert in the
// just-resolved batch: there is no manual unassign in the product, resolution
// is what frees an alert from its assignee. Each released row also appends a
// system-driven `unassigned` event (NULL actor, details.reason=resolved) to
// the alert's activity timeline. Best-effort by design: a failure here must
// not fail the webhook (history is already persisted); the cleanup worker
// sweep catches anything missed.
func releaseAlertAssignments(ctx context.Context, batch []pendingAlert) {
	n := len(batch)
	orgIDs := make([]string, n)
	fingerprints := make([]string, n)
	for i, p := range batch {
		orgIDs[i] = p.organizationID
		fingerprints[i] = p.alert.Fingerprint
	}

	const query = `
WITH released AS (
    DELETE FROM alert_assignments a
    USING (
        SELECT DISTINCT organization_id, fingerprint
        FROM unnest($1::text[], $2::text[]) AS t(organization_id, fingerprint)
    ) t
    WHERE a.organization_id = t.organization_id
      AND a.fingerprint     = t.fingerprint
    RETURNING a.organization_id, a.fingerprint, a.assigned_user_id, a.assigned_user_name, a.assigned_user_org_id, a.assigned_user_org_name
)
INSERT INTO alert_activity (organization_id, fingerprint, action, details)
SELECT organization_id, fingerprint, 'unassigned',
       jsonb_build_object(
           'reason', 'resolved',
           'assigned_user_id', assigned_user_id,
           'assigned_user_name', assigned_user_name,
           'assigned_user_org_id', assigned_user_org_id,
           'assigned_user_org_name', assigned_user_org_name
       )
FROM released`

	result, err := database.DB.ExecContext(ctx, query, pq.Array(orgIDs), pq.Array(fingerprints))
	if err != nil {
		logger.Warn().Err(err).Msg("alertmanager history: failed to auto-release alert assignments (non-fatal)")
		return
	}
	if released, err := result.RowsAffected(); err == nil && released > 0 {
		logger.Info().
			Int64("released", released).
			Msg("alertmanager history: auto-released assignments for resolved alerts")
	}
}

// bulkInsertAlertHistory writes every alert in the batch with a single
// INSERT … SELECT FROM unnest(…). Empty optional strings become SQL NULLs;
// invalid endsAt becomes NULL.
func bulkInsertAlertHistory(ctx context.Context, batch []pendingAlert, receiver string) error {
	n := len(batch)
	systemKeys := make([]string, n)
	orgIDs := make([]string, n)
	alertnames := make([]string, n)
	severity := make([]string, n)
	fingerprint := make([]string, n)
	startsAt := make([]time.Time, n)
	endsAtTexts := make([]string, n)
	summary := make([]string, n)
	labels := make([]string, n)
	annotations := make([]string, n)

	for i, p := range batch {
		systemKeys[i] = p.systemKey
		orgIDs[i] = p.organizationID
		alertnames[i] = p.alertname
		severity[i] = p.severity
		fingerprint[i] = p.alert.Fingerprint
		startsAt[i] = p.alert.StartsAt
		if p.endsAtValid {
			endsAtTexts[i] = p.endsAt.UTC().Format(time.RFC3339Nano)
		}
		summary[i] = p.summary
		labels[i] = string(p.labelsJSON)
		annotations[i] = string(p.annotationsJSON)
	}

	const query = `
INSERT INTO alert_history
    (system_key, organization_id, alertname, severity, status, fingerprint,
     starts_at, ends_at, summary, labels, annotations, receiver)
SELECT
    system_key,
    organization_id,
    alertname,
    NULLIF(severity, ''),
    'resolved',
    fingerprint,
    starts_at,
    NULLIF(ends_at_text, '')::timestamptz,
    NULLIF(summary, ''),
    labels::jsonb,
    annotations::jsonb,
    NULLIF($11, '')
FROM unnest(
    $1::text[],
    $2::text[],
    $3::text[],
    $4::text[],
    $5::text[],
    $6::timestamptz[],
    $7::text[],
    $8::text[],
    $9::text[],
    $10::text[]
) AS t(
    system_key, organization_id, alertname, severity, fingerprint,
    starts_at, ends_at_text, summary, labels, annotations
)`

	_, err := database.DB.ExecContext(ctx, query,
		pq.Array(systemKeys),
		pq.Array(orgIDs),
		pq.Array(alertnames),
		pq.Array(severity),
		pq.Array(fingerprint),
		pq.Array(startsAt),
		pq.Array(endsAtTexts),
		pq.Array(summary),
		pq.Array(labels),
		pq.Array(annotations),
		receiver,
	)
	return err
}
