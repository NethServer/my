-- Rollback Migration 008: Remove deleted_by_org_id column from systems table

DROP INDEX IF EXISTS idx_systems_deleted_by_org_id;
ALTER TABLE systems DROP COLUMN IF EXISTS deleted_by_org_id;

-- Remove migration record
DELETE FROM schema_migrations WHERE migration_number = '008';
