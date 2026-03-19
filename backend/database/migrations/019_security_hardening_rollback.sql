-- Rollback Migration 018: Security hardening for support service

DROP INDEX IF EXISTS idx_support_sessions_reconnect_token;
ALTER TABLE support_sessions DROP COLUMN IF EXISTS reconnect_token;

DROP INDEX IF EXISTS idx_systems_support_enabled;
ALTER TABLE systems DROP COLUMN IF EXISTS support_enabled;

DELETE FROM schema_migrations WHERE migration_number = 18;
