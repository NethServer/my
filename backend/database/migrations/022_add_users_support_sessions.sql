-- Migration 022: Add ephemeral support users to support_sessions
-- Stores the users report from tunnel-client (JSONB) alongside the session

ALTER TABLE support_sessions ADD COLUMN IF NOT EXISTS users JSONB;
ALTER TABLE support_sessions ADD COLUMN IF NOT EXISTS users_at TIMESTAMPTZ;

COMMENT ON COLUMN support_sessions.users IS 'Ephemeral support users created by tunnel-client for this session (JSON)';
COMMENT ON COLUMN support_sessions.users_at IS 'Timestamp when users report was received from the tunnel-client';
