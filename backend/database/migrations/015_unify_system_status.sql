-- Migration 015: Unify system status values
-- Changes status values from online/offline to active/inactive
-- Adds last_inventory_at column to systems table

-- Drop old constraint first (must be removed before updating values)
ALTER TABLE systems DROP CONSTRAINT IF EXISTS chk_systems_status;

-- Update existing status values
UPDATE systems SET status = 'active' WHERE status = 'online';
UPDATE systems SET status = 'inactive' WHERE status = 'offline';

-- Add new constraint
ALTER TABLE systems ADD CONSTRAINT chk_systems_status
    CHECK (status IN ('unknown', 'active', 'inactive', 'deleted'));

-- Add last_inventory_at column
ALTER TABLE systems ADD COLUMN IF NOT EXISTS last_inventory_at TIMESTAMP WITH TIME ZONE;
COMMENT ON COLUMN systems.last_inventory_at IS 'Timestamp of last inventory received. NULL means no inventory received yet';

-- Update last_inventory_at from existing inventory_records
UPDATE systems s
SET last_inventory_at = sub.latest
FROM (
    SELECT system_id, MAX(timestamp) as latest
    FROM inventory_records
    GROUP BY system_id
) sub
WHERE s.id = sub.system_id;

-- Update system_heartbeats status column values
UPDATE system_heartbeats SET status = 'active' WHERE status = 'online';
UPDATE system_heartbeats SET status = 'inactive' WHERE status = 'offline';
