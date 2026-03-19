-- Migration 017: Support sessions and access logs
-- Description: Tables for WebSocket tunnel-based support sessions

-- Support sessions track active tunnel connections from client systems
CREATE TABLE IF NOT EXISTS support_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    system_id VARCHAR(255) NOT NULL REFERENCES systems(id) ON DELETE CASCADE,
    session_token VARCHAR(64) UNIQUE NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '24 hours'),
    status VARCHAR(16) NOT NULL DEFAULT 'pending',
    closed_at TIMESTAMPTZ,
    closed_by VARCHAR(32),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT support_sessions_status_check CHECK (status IN ('pending', 'active', 'expired', 'closed'))
);

CREATE INDEX IF NOT EXISTS idx_support_sessions_system_id ON support_sessions(system_id);
CREATE INDEX IF NOT EXISTS idx_support_sessions_status ON support_sessions(status);
CREATE INDEX IF NOT EXISTS idx_support_sessions_session_token ON support_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_support_sessions_expires_at ON support_sessions(expires_at);

-- Access logs track operator interactions with support sessions
CREATE TABLE IF NOT EXISTS support_access_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES support_sessions(id) ON DELETE CASCADE,
    operator_id VARCHAR(255) NOT NULL,
    operator_name VARCHAR(255),
    access_type VARCHAR(16) NOT NULL DEFAULT 'view',
    connected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    disconnected_at TIMESTAMPTZ,
    metadata JSONB,
    CONSTRAINT support_access_logs_access_type_check CHECK (access_type IN ('view', 'ssh', 'web_terminal', 'ui_proxy'))
);

CREATE INDEX IF NOT EXISTS idx_support_access_logs_session_id ON support_access_logs(session_id);
CREATE INDEX IF NOT EXISTS idx_support_access_logs_operator_id ON support_access_logs(operator_id);

-- Record migration
INSERT INTO schema_migrations (migration_number, description) VALUES (17, 'Support sessions and access logs');
