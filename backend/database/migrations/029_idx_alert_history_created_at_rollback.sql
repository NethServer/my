-- Rollback migration 029: Drop the alert_history created_at index

DROP INDEX IF EXISTS idx_alert_history_created_at;
