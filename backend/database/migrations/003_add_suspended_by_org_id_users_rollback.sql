-- Rollback migration: Remove suspended_by_org_id from users table

DROP INDEX IF EXISTS idx_users_suspended_by_org_id;
ALTER TABLE users DROP COLUMN IF EXISTS suspended_by_org_id;
