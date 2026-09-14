-- Migration 044: how many systems an organization still has on the old my.
--
-- After the cutover two portals answer for the same partner: this one, and
-- legacy.my.nethesis.it for everything the new my does not manage (NethServer 7
-- and earlier, plus anything the import never carried over). A reseller looking
-- at its dashboard should see its whole estate, not just the half that already
-- lives here.
--
-- The count is PUSHED from the legacy side by proxy_sync (the same 4h cron that
-- imports systems), not pulled: the new my never opens a connection to the old
-- MySQL. That keeps the dashboard independent from a machine that is on its way
-- out — when the legacy is down or slow the number goes stale instead of the
-- page breaking — and makes the decommission a matter of dropping this table.
--
-- One row per reseller/distributor organization, keyed by the Logto org id the
-- legacy VAT maps to. The hierarchy is NOT precomputed: callers sum the rows of
-- the organizations in their RBAC scope, exactly like alerts_totals_by_org, so a
-- distributor picks up its resellers for free.
--
-- Counts exclude systems deleted or disabled on the legacy, mirroring what that
-- portal shows. They are NOT attributable to a customer: 83% of these systems
-- belong to clients that were never created here, so the number lives at
-- reseller level and the UI sends people to the legacy portal for the detail.

CREATE TABLE IF NOT EXISTS legacy_systems_by_org (
    organization_id VARCHAR(255) PRIMARY KEY,
    total           INTEGER NOT NULL DEFAULT 0,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE legacy_systems_by_org IS 'Per-organization count of systems still living on the old my (legacy.my.nethesis.it), pushed by the proxy_sync cron on the legacy host. Transitional: drop at decommission';
COMMENT ON COLUMN legacy_systems_by_org.organization_id IS 'Logto org id of the reseller/distributor the legacy VAT maps to';
COMMENT ON COLUMN legacy_systems_by_org.total IS 'Systems on the old my with no counterpart here, excluding deleted and disabled ones';
COMMENT ON COLUMN legacy_systems_by_org.updated_at IS 'Last successful push for this org; stale rows mean the legacy sync is lagging or stopped';
