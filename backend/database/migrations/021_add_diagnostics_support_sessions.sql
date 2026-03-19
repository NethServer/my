-- Migration 021: add diagnostics columns to support_sessions
ALTER TABLE support_sessions
    ADD COLUMN IF NOT EXISTS diagnostics JSONB,
    ADD COLUMN IF NOT EXISTS diagnostics_at TIMESTAMPTZ;

COMMENT ON COLUMN support_sessions.diagnostics IS 'Diagnostic report collected by tunnel-client at connect time (JSON)';
COMMENT ON COLUMN support_sessions.diagnostics_at IS 'Timestamp when diagnostics were last received from the tunnel-client';
