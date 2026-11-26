-- Rollback migration 065: drop the per-product breakdown, the total stays.

ALTER TABLE legacy_systems_by_org
    DROP COLUMN IF EXISTS nethservice,
    DROP COLUMN IF EXISTS nethsecurity,
    DROP COLUMN IF EXISTS nethvoice;
