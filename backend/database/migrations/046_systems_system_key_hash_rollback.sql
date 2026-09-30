-- Rollback migration 046: drop the learned key hash.

DROP INDEX IF EXISTS idx_systems_system_key_hash;

ALTER TABLE systems DROP COLUMN IF EXISTS system_key_hash;
