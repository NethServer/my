-- Rollback migration 031: Recreate the (redundant) duplicate index

CREATE UNIQUE INDEX IF NOT EXISTS idx_systems_system_key_new ON systems(system_key);
