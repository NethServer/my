-- Rollback Migration 030: Remove ephemeral support users columns from support_sessions

ALTER TABLE support_sessions DROP COLUMN IF EXISTS users;
ALTER TABLE support_sessions DROP COLUMN IF EXISTS users_at;
