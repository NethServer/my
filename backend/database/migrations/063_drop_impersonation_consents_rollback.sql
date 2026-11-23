-- Rollback 063: recreate impersonation_consents, empty
--
-- The consent records dropped by 063 are not restored: users grant consent
-- again from their account page.

CREATE TABLE IF NOT EXISTS impersonation_consents (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    max_duration_minutes INTEGER NOT NULL DEFAULT 60,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE impersonation_consents IS 'User consents for allowing impersonation by Owner users';
COMMENT ON COLUMN impersonation_consents.user_id IS 'User who granted consent for impersonation';
COMMENT ON COLUMN impersonation_consents.expires_at IS 'Timestamp when consent expires and must be renewed';
COMMENT ON COLUMN impersonation_consents.max_duration_minutes IS 'Maximum duration of impersonation session in minutes';
COMMENT ON COLUMN impersonation_consents.active IS 'Whether consent is currently active (can be revoked)';

ALTER TABLE impersonation_consents
ADD CONSTRAINT impersonation_consents_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_impersonation_consents_user_id ON impersonation_consents(user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_active ON impersonation_consents(active);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_expires_at ON impersonation_consents(expires_at);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_user_active ON impersonation_consents(user_id, active);
