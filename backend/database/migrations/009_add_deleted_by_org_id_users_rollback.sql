-- Rollback Migration 009: Remove deleted_by_org_id column from users table

DROP INDEX IF EXISTS idx_users_deleted_by_org_id;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_by_org_id;

-- Remove migration record
DELETE FROM schema_migrations WHERE migration_number = '009';
