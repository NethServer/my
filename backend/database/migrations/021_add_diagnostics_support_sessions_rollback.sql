-- Rollback migration 021: remove diagnostics columns from support_sessions
ALTER TABLE support_sessions
    DROP COLUMN IF EXISTS diagnostics,
    DROP COLUMN IF EXISTS diagnostics_at;
