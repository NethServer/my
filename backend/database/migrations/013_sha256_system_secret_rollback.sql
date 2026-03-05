-- Rollback Migration 013
DROP INDEX IF EXISTS idx_systems_system_secret_sha256;
ALTER TABLE systems DROP COLUMN IF EXISTS system_secret_sha256;
-- Restore NOT NULL (only safe if all rows have a value)
-- ALTER TABLE systems ALTER COLUMN system_secret SET NOT NULL;
