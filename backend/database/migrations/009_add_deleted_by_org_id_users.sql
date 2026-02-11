-- Migration 009: Add deleted_by_org_id column to users table
-- Tracks which organization caused cascade soft-deletion of users

ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_by_org_id VARCHAR(255);

-- Partial index for efficient lookups of cascade-deleted users
CREATE INDEX IF NOT EXISTS idx_users_deleted_by_org_id ON users(deleted_by_org_id) WHERE deleted_by_org_id IS NOT NULL;

-- Track migration
INSERT INTO schema_migrations (migration_number, description)
VALUES ('009', 'Add deleted_by_org_id to users for cascade soft-delete tracking')
ON CONFLICT (migration_number) DO NOTHING;
