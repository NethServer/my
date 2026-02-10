-- Migration 008: Add deleted_by_org_id column to systems table
-- Tracks which organization caused cascade soft-deletion of systems

ALTER TABLE systems ADD COLUMN IF NOT EXISTS deleted_by_org_id VARCHAR(255);

-- Partial index for efficient lookups of cascade-deleted systems
CREATE INDEX IF NOT EXISTS idx_systems_deleted_by_org_id ON systems(deleted_by_org_id) WHERE deleted_by_org_id IS NOT NULL;

-- Track migration
INSERT INTO schema_migrations (migration_number, description)
VALUES ('008', 'Add deleted_by_org_id to systems for cascade soft-delete tracking')
ON CONFLICT (migration_number) DO NOTHING;
