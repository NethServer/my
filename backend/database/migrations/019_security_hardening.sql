-- Migration 018: Security hardening for support service
-- Description: Adds support_enabled flag to systems and reconnect_token to sessions

-- #2: Require explicit opt-in before a system can connect to the support tunnel
ALTER TABLE systems ADD COLUMN IF NOT EXISTS support_enabled BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN systems.support_enabled IS 'Explicit opt-in: system can connect to support tunnel only when true';

CREATE INDEX IF NOT EXISTS idx_systems_support_enabled ON systems(support_enabled) WHERE support_enabled = true AND deleted_at IS NULL;

-- #8: Reconnect token to prevent session hijacking during grace period
ALTER TABLE support_sessions ADD COLUMN IF NOT EXISTS reconnect_token VARCHAR(64);

COMMENT ON COLUMN support_sessions.reconnect_token IS 'Token required to reconnect to a session during grace period';

CREATE INDEX IF NOT EXISTS idx_support_sessions_reconnect_token ON support_sessions(reconnect_token) WHERE reconnect_token IS NOT NULL;

-- Record migration
INSERT INTO schema_migrations (migration_number, description) VALUES (18, 'Security hardening for support service');
