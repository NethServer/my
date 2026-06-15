/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package cron

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
)

// AlertsTotalsRefresher periodically fans out to Mimir's Alertmanager API
// across every active tenant and writes per-organization severity/muted counts
// into alerts_totals_by_org. The /api/alerts/totals endpoint on the backend
// reads from that table with a single SUM query instead of paying the fan-out
// cost on every user request.
type AlertsTotalsRefresher struct {
	db               *sql.DB
	mimirURL         string
	httpClient       *http.Client
	checkIntervalSec int
	fanoutTimeout    time.Duration
	concurrency      int
}

// NewAlertsTotalsRefresher wires the refresher with the same tuned HTTP
// transport the backend used for its in-line fan-out, so a single tenant
// stall cannot cascade into connection starvation across the rest.
func NewAlertsTotalsRefresher() *AlertsTotalsRefresher {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	return &AlertsTotalsRefresher{
		db:       database.DB,
		mimirURL: configuration.Config.MimirURL,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
		checkIntervalSec: 60,
		fanoutTimeout:    30 * time.Second,
		concurrency:      50,
	}
}

// Start runs the refresh loop until ctx is cancelled. Blocks; call as a goroutine.
func (r *AlertsTotalsRefresher) Start(ctx context.Context) {
	logger.Info().
		Int("check_interval_seconds", r.checkIntervalSec).
		Msg("Starting alerts totals refresher cron job")

	ticker := time.NewTicker(time.Duration(r.checkIntervalSec) * time.Second)
	defer ticker.Stop()

	r.refresh(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Alerts totals refresher stopped")
			return
		case <-ticker.C:
			r.refresh(ctx)
		}
	}
}

// refresh executes one full cycle: load tenant list, fan out to Mimir, upsert
// counts. A per-cycle ctx with fanoutTimeout caps the worst case so a stalled
// Mimir cannot delay the next tick.
//
// Per-tenant Mimir errors are NOT logged individually — that would flood the
// log with N warnings per cycle when Mimir is down (e.g. in local dev). They
// are collected and summarised as a single Warn at the end of the cycle, with
// a sample error so the operator knows the root cause without grep.
func (r *AlertsTotalsRefresher) refresh(parent context.Context) {
	start := time.Now()

	orgIDs, err := r.loadOrgIDs(parent)
	if err != nil {
		logger.Error().Err(err).Msg("alerts totals refresher: failed to load organization IDs")
		return
	}
	if len(orgIDs) == 0 {
		logger.Debug().Msg("alerts totals refresher: no organizations to refresh")
		return
	}

	ctx, cancel := context.WithTimeout(parent, r.fanoutTimeout)
	defer cancel()

	counts, failures, sampleErr := r.fanOut(ctx, orgIDs)

	if err := r.upsertCounts(parent, counts); err != nil {
		logger.Error().Err(err).Msg("alerts totals refresher: failed to upsert counts")
		return
	}

	if err := r.purgeStaleRows(parent); err != nil {
		logger.Warn().Err(err).Msg("alerts totals refresher: failed to purge stale rows")
	}

	// One aggregate warn per cycle when there were per-tenant failures; the
	// sample error carries the root cause. Operators can correlate the failure
	// count against the org total to tell "Mimir down" (failures == len(orgIDs))
	// from "isolated tenant trouble" (failures << len(orgIDs)).
	if failures > 0 {
		logger.Warn().
			Int("failures", failures).
			Int("orgs_total", len(orgIDs)).
			Int("orgs_refreshed", len(counts)).
			Str("sample_error", sampleErr).
			Dur("elapsed", time.Since(start)).
			Msg("alerts totals refresher: cycle had per-tenant mimir failures")
		return
	}

	logger.Debug().
		Int("orgs_refreshed", len(counts)).
		Dur("elapsed", time.Since(start)).
		Msg("alerts totals refresher: cycle completed")
}

// orgCounts is the per-organization aggregation written to alerts_totals_by_org.
type orgCounts struct {
	active   int
	critical int
	warning  int
	info     int
	muted    int
}

// loadOrgIDs returns every active organization ID from the unified view.
// Soft-deleted orgs are excluded by the view definition.
func (r *AlertsTotalsRefresher) loadOrgIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT logto_id FROM unified_organizations WHERE logto_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("query unified_organizations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan logto_id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate logto_id rows: %w", err)
	}
	return ids, nil
}

// fanOut queries Mimir's Alertmanager once per tenant with bounded concurrency.
// Every input org gets an entry in the output, including those with no alerts
// (counts all zero) — so the upsert can refresh rows that just dropped to zero
// instead of leaving them stale.
//
// Per-tenant errors are surfaced to the caller as an aggregate count plus a
// sample message; individual failures are NOT logged here to avoid flooding
// the log when Mimir is down (the caller's single per-cycle warn covers it).
// On failure, the tenant is dropped from `result` so its previous successful
// counts in the table aren't overwritten by a zero placeholder.
func (r *AlertsTotalsRefresher) fanOut(ctx context.Context, orgIDs []string) (map[string]orgCounts, int, string) {
	result := make(map[string]orgCounts, len(orgIDs))
	for _, id := range orgIDs {
		result[id] = orgCounts{}
	}

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		failures  int
		sampleErr string
		sem       = make(chan struct{}, r.concurrency)
	)

	for _, orgID := range orgIDs {
		wg.Add(1)
		go func(orgID string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			counts, err := r.fetchOrgCounts(ctx, orgID)
			if err != nil {
				mu.Lock()
				failures++
				if sampleErr == "" {
					sampleErr = err.Error()
				}
				delete(result, orgID)
				mu.Unlock()
				return
			}

			mu.Lock()
			result[orgID] = counts
			mu.Unlock()
		}(orgID)
	}
	wg.Wait()
	return result, failures, sampleErr
}

// fetchOrgCounts pulls the active+silenced+inhibited alert list for one tenant
// from Mimir's Alertmanager and tallies it by severity and muted state.
// Returns an error on HTTP or parse failure; the caller aggregates these and
// skips writing the tenant for the cycle.
func (r *AlertsTotalsRefresher) fetchOrgCounts(ctx context.Context, orgID string) (orgCounts, error) {
	url := r.mimirURL + "/alertmanager/api/v2/alerts?active=true&silenced=true&inhibited=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return orgCounts{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", orgID)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return orgCounts{}, fmt.Errorf("mimir request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Mimir returns 200 with empty array for unknown tenants; non-2xx is a real failure.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return orgCounts{}, fmt.Errorf("mimir returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return orgCounts{}, fmt.Errorf("read body: %w", err)
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return orgCounts{}, fmt.Errorf("parse response: %w", err)
	}

	counts := orgCounts{active: len(alerts)}
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		switch sev, _ := labels["severity"].(string); sev {
		case "critical":
			counts.critical++
		case "warning":
			counts.warning++
		case "info":
			counts.info++
		}
		// An alert is muted when Alertmanager has at least one active silence
		// matching it (status.silencedBy non-empty).
		if status, ok := alert["status"].(map[string]interface{}); ok {
			if sb, ok := status["silencedBy"].([]interface{}); ok && len(sb) > 0 {
				counts.muted++
			}
		}
	}
	return counts, nil
}

// upsertCounts writes the fan-out result in a single multi-VALUES INSERT with
// ON CONFLICT DO UPDATE. Builds parameter arrays so we round-trip exactly once
// regardless of tenant count.
func (r *AlertsTotalsRefresher) upsertCounts(ctx context.Context, counts map[string]orgCounts) error {
	if len(counts) == 0 {
		return nil
	}

	orgIDs := make([]string, 0, len(counts))
	active := make([]int, 0, len(counts))
	critical := make([]int, 0, len(counts))
	warning := make([]int, 0, len(counts))
	info := make([]int, 0, len(counts))
	muted := make([]int, 0, len(counts))

	for id, c := range counts {
		orgIDs = append(orgIDs, id)
		active = append(active, c.active)
		critical = append(critical, c.critical)
		warning = append(warning, c.warning)
		info = append(info, c.info)
		muted = append(muted, c.muted)
	}

	// unnest expands the parallel arrays into rows. ON CONFLICT updates the
	// existing row in place; new orgs get inserted.
	query := `
		INSERT INTO alerts_totals_by_org (
			organization_id, active, critical, warning, info, muted, updated_at
		)
		SELECT unnest($1::text[]),
		       unnest($2::int[]),
		       unnest($3::int[]),
		       unnest($4::int[]),
		       unnest($5::int[]),
		       unnest($6::int[]),
		       NOW()
		ON CONFLICT (organization_id) DO UPDATE SET
			active     = EXCLUDED.active,
			critical   = EXCLUDED.critical,
			warning    = EXCLUDED.warning,
			info       = EXCLUDED.info,
			muted      = EXCLUDED.muted,
			updated_at = EXCLUDED.updated_at
	`
	if _, err := r.db.ExecContext(ctx, query,
		pq.Array(orgIDs),
		pq.Array(active),
		pq.Array(critical),
		pq.Array(warning),
		pq.Array(info),
		pq.Array(muted),
	); err != nil {
		return fmt.Errorf("upsert alerts_totals_by_org: %w", err)
	}
	return nil
}

// purgeStaleRows removes rows for organizations that no longer exist (or were
// soft-deleted) in unified_organizations. Cosmetic: the /totals query filters
// by the caller's hierarchy so deleted orgs would never sum into a response,
// but keeping the table tight makes debugging easier.
func (r *AlertsTotalsRefresher) purgeStaleRows(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM alerts_totals_by_org
		WHERE organization_id NOT IN (
			SELECT logto_id FROM unified_organizations WHERE logto_id IS NOT NULL
		)
	`)
	return err
}
