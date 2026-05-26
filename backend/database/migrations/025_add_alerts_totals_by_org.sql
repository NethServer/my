-- Migration 025: Add alerts_totals_by_org table
-- Pre-aggregated per-organization counts of active alerts (by severity and
-- muted state) maintained by the collect service. Lets /api/alerts/totals
-- answer in a single SUM query instead of fanning out to Mimir per tenant.

CREATE TABLE IF NOT EXISTS alerts_totals_by_org (
    organization_id VARCHAR(255) PRIMARY KEY,

    -- Counts of currently-active alerts in Alertmanager for this organization.
    -- 'active' is the total; the per-severity columns and 'muted' partition it
    -- on different axes (severity AND muted are independent: a critical alert
    -- can also be muted, contributing to both 'critical' and 'muted').
    active   INTEGER NOT NULL DEFAULT 0,
    critical INTEGER NOT NULL DEFAULT 0,
    warning  INTEGER NOT NULL DEFAULT 0,
    info     INTEGER NOT NULL DEFAULT 0,
    muted    INTEGER NOT NULL DEFAULT 0,

    -- Last refresh time. Read by /totals to surface staleness when the
    -- background refresher is lagging or down.
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE alerts_totals_by_org IS 'Per-organization active alert counts, refreshed by collect''s AlertsTotalsRefresher cron';
COMMENT ON COLUMN alerts_totals_by_org.muted IS 'Active alerts that have at least one matching Alertmanager silence';
COMMENT ON COLUMN alerts_totals_by_org.updated_at IS 'Last successful refresh for this org; stale rows indicate the refresher is lagging';
