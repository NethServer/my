-- Rollback migration 043: systems authenticate again with a spent key.
--
-- The unregistration timestamps are dropped, so every appliance that
-- announced its departure returns to a state where its credentials are
-- accepted. Rows that read 'unregistered' fall back to 'inactive'; the
-- heartbeat monitor moves them on from there.

UPDATE systems SET status = 'inactive' WHERE status = 'unregistered';

ALTER TABLE systems DROP CONSTRAINT IF EXISTS chk_systems_status;
ALTER TABLE systems ADD CONSTRAINT chk_systems_status
    CHECK (status IN ('unknown', 'active', 'inactive', 'deleted'));

ALTER TABLE systems DROP COLUMN IF EXISTS unregistered_at;

COMMENT ON COLUMN systems.status IS 'Heartbeat status: unknown (no data), active (heartbeat recent), inactive (heartbeat stale), deleted';
