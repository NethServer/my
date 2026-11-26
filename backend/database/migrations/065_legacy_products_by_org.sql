-- Migration 065: per-product breakdown of the legacy systems counter.
--
-- The old my shows each reseller its "Installed products": how many systems run
-- NethService, NethSecurity and NethVoice (server.system_products). Sales still
-- needs those figures for the estate that has not migrated, so the legacy sync
-- pushes them next to the total of migration 044. A system running two products
-- counts in both columns, so they do not add up to total.

ALTER TABLE legacy_systems_by_org
    ADD COLUMN IF NOT EXISTS nethservice  INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS nethsecurity INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS nethvoice    INTEGER NOT NULL DEFAULT 0;

COMMENT ON COLUMN legacy_systems_by_org.nethservice IS 'Of total, systems running NethService (legacy system_products); a system with several products counts in each';
COMMENT ON COLUMN legacy_systems_by_org.nethsecurity IS 'Of total, systems running NethSecurity (legacy system_products)';
COMMENT ON COLUMN legacy_systems_by_org.nethvoice IS 'Of total, systems running NethVoice (legacy system_products)';
