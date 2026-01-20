-- Migration: Add suspended_by_org_id to users table
-- Tracks cascade suspensions when an organization is suspended

ALTER TABLE users ADD COLUMN IF NOT EXISTS suspended_by_org_id VARCHAR(255);

-- Index for fast lookups when reactivating an organization
CREATE INDEX IF NOT EXISTS idx_users_suspended_by_org_id ON users(suspended_by_org_id) WHERE suspended_by_org_id IS NOT NULL;

COMMENT ON COLUMN users.suspended_by_org_id IS 'Organization ID that caused this user to be suspended (for cascade reactivation)';
