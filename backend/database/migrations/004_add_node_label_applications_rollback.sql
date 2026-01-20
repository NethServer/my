-- Rollback Migration 004: Remove node_label column from applications table

-- Remove node_label column
ALTER TABLE applications DROP COLUMN IF EXISTS node_label;

-- Remove migration record
DELETE FROM schema_migrations WHERE migration_number = '004';
