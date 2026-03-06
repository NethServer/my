-- Rollback Migration 015: Revert system status unification

-- Revert status values
UPDATE systems SET status = 'online' WHERE status = 'active';
UPDATE systems SET status = 'offline' WHERE status = 'inactive';

-- Restore old constraint
ALTER TABLE systems DROP CONSTRAINT IF EXISTS chk_systems_status;
ALTER TABLE systems ADD CONSTRAINT chk_systems_status
    CHECK (status IN ('unknown', 'online', 'offline', 'deleted'));

-- Remove last_inventory_at column
ALTER TABLE systems DROP COLUMN IF EXISTS last_inventory_at;

-- Revert system_heartbeats status column values
UPDATE system_heartbeats SET status = 'online' WHERE status = 'active';
UPDATE system_heartbeats SET status = 'offline' WHERE status = 'inactive';
