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
// across every distinct tenant (per-reseller model: a customer's alerts live
// in its managing reseller's tenant), buckets the alerts per organization via
// the organization_id label, and writes severity/muted counts into
// alerts_totals_by_org. The /api/alerts/totals endpoint on the backend reads
// from that table with a single SUM query instead of paying the fan-out cost
// on every user request.
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

	orgTenants, err := r.loadOrgTenants(parent)
	if err != nil {
		logger.Error().Err(err).Msg("alerts totals refresher: failed to load organization IDs")
		return
	}
	if len(orgTenants) == 0 {
		logger.Debug().Msg("alerts totals refresher: no organizations to refresh")
		return
	}

	ctx, cancel := context.WithTimeout(parent, r.fanoutTimeout)
	defer cancel()

	counts, failures, sampleErr := r.fanOut(ctx, orgTenants)

	if err := r.upsertCounts(parent, counts); err != nil {
		logger.Error().Err(err).Msg("alerts totals refresher: failed to upsert counts")
		return
	}

	if err := r.purgeStaleRows(parent); err != nil {
		logger.Warn().Err(err).Msg("alerts totals refresher: failed to purge stale rows")
	}

	// One aggregate warn per cycle when there were per-tenant failures; the
	// sample error carries the root cause. Operators can correlate the failure
	// count against the org total to tell "Mimir down" (failures == all tenants)
	// from "isolated tenant trouble" (failures << tenants).
	if failures > 0 {
		logger.Warn().
			Int("tenant_failures", failures).
			Int("orgs_total", len(orgTenants)).
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

// loadOrgTenants returns every active organization ID mapped to the Mimir
// tenant that holds its active alerts. With the per-reseller tenant model a
// customer's alerts live in the tenant of its managing parent (the
// reseller/distributor in custom_data.createdBy); resellers and distributors
// are their own tenant. Mirrors alerting.TenantForOrg on the backend.
// Soft-deleted orgs are excluded.
func (r *AlertsTotalsRefresher) loadOrgTenants(ctx context.Context) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT logto_id, logto_id AS tenant_id
		FROM distributors WHERE logto_id IS NOT NULL AND deleted_at IS NULL
		UNION ALL
		SELECT logto_id, logto_id
		FROM resellers WHERE logto_id IS NOT NULL AND deleted_at IS NULL
		UNION ALL
		SELECT logto_id, COALESCE(NULLIF(custom_data->>'createdBy', ''), logto_id)
		FROM customers WHERE logto_id IS NOT NULL AND deleted_at IS NULL
	`)
	if err != nil {
		return nil, fmt.Errorf("query organization tenants: %w", err)
	}
	defer func() { _ = rows.Close() }()

	orgTenants := make(map[string]string)
	for rows.Next() {
		var orgID, tenantID string
		if err := rows.Scan(&orgID, &tenantID); err != nil {
			return nil, fmt.Errorf("scan org tenant row: %w", err)
		}
		orgTenants[orgID] = tenantID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate org tenant rows: %w", err)
	}
	return orgTenants, nil
}

// fanOut queries Mimir's Alertmanager once per distinct tenant with bounded
// concurrency and buckets the returned alerts into per-organization counts via
// the organization_id label (the same label the backend's list endpoint uses
// as its isolation boundary). Every input org gets an entry in the output,
// including those with no alerts (counts all zero) — so the upsert can refresh
// rows that just dropped to zero instead of leaving them stale.
//
// Per-tenant errors are surfaced to the caller as an aggregate count plus a
// sample message; individual failures are NOT logged here to avoid flooding
// the log when Mimir is down (the caller's single per-cycle warn covers it).
// On failure, every org belonging to the failed tenant is dropped from
// `result` so its previous successful counts in the table aren't overwritten
// by a zero placeholder.
func (r *AlertsTotalsRefresher) fanOut(ctx context.Context, orgTenants map[string]string) (map[string]orgCounts, int, string) {
	result := make(map[string]orgCounts, len(orgTenants))
	tenantOrgs := make(map[string][]string)
	for orgID, tenantID := range orgTenants {
		result[orgID] = orgCounts{}
		tenantOrgs[tenantID] = append(tenantOrgs[tenantID], orgID)
	}

	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		failures  int
		sampleErr string
		sem       = make(chan struct{}, r.concurrency)
	)

	for tenantID, orgIDs := range tenantOrgs {
		wg.Add(1)
		go func(tenantID string, orgIDs []string) {
			defer wg.Done()

			dropOrgs := func(err error) {
				mu.Lock()
				failures++
				if sampleErr == "" {
					sampleErr = err.Error()
				}
				for _, orgID := range orgIDs {
					delete(result, orgID)
				}
				mu.Unlock()
			}

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				dropOrgs(ctx.Err())
				return
			}

			alerts, err := r.fetchTenantAlerts(ctx, tenantID)
			if err != nil {
				dropOrgs(err)
				return
			}

			counts := bucketAlertsByOrg(alerts)
			mu.Lock()
			// A tenant may only write rows for its own orgs: an alert whose
			// organization_id label points outside the tenant (stale routing,
			// deleted org, label mismatch) must not overwrite counts that
			// another tenant's fetch is authoritative for.
			for _, orgID := range orgIDs {
				if c, ok := counts[orgID]; ok {
					result[orgID] = c
				}
			}
			mu.Unlock()
		}(tenantID, orgIDs)
	}
	wg.Wait()
	return result, failures, sampleErr
}

// bucketAlertsByOrg tallies a tenant's alert list per organization_id label,
// by severity and muted state. Alerts without the label are skipped: the
// backend's list endpoint drops them too (filterByOrgScope), so counting them
// would make /totals disagree with the visible list.
func bucketAlertsByOrg(alerts []map[string]interface{}) map[string]orgCounts {
	counts := make(map[string]orgCounts)
	for _, alert := range alerts {
		labels, _ := alert["labels"].(map[string]interface{})
		orgID, _ := labels["organization_id"].(string)
		if orgID == "" {
			continue
		}

		c := counts[orgID]
		c.active++
		switch sev, _ := labels["severity"].(string); sev {
		case "critical":
			c.critical++
		case "warning":
			c.warning++
		case "info":
			c.info++
		}
		// An alert is muted when Alertmanager has at least one active silence
		// matching it (status.silencedBy non-empty).
		if status, ok := alert["status"].(map[string]interface{}); ok {
			if sb, ok := status["silencedBy"].([]interface{}); ok && len(sb) > 0 {
				c.muted++
			}
		}
		counts[orgID] = c
	}
	return counts
}

// fetchTenantAlerts pulls the active+silenced+inhibited alert list for one
// tenant from Mimir's Alertmanager. Returns an error on HTTP or parse failure;
// the caller aggregates these and skips writing the tenant's orgs for the cycle.
func (r *AlertsTotalsRefresher) fetchTenantAlerts(ctx context.Context, tenantID string) ([]map[string]interface{}, error) {
	url := r.mimirURL + "/alertmanager/api/v2/alerts?active=true&silenced=true&inhibited=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("X-Scope-OrgID", tenantID)
	req.Header.Set("Accept", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("mimir request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Mimir returns 200 with empty array for unknown tenants; non-2xx is a real failure.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mimir returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var alerts []map[string]interface{}
	if err := json.Unmarshal(body, &alerts); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	return alerts, nil
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
