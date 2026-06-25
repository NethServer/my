-- Migration 029: Add created_at index on alert_history for time-based retention
--
-- The cleanup worker now enforces a flat TTL on alert_history (delete rows older
-- than 6 months). Existing indexes are all (system_key|organization_id, created_at)
-- or (starts_at) — none can serve a bare `created_at < cutoff` scan, so the
-- batched retention DELETE would seq-scan ~773k rows every batch. This index lets
-- it find expiring rows directly.

CREATE INDEX IF NOT EXISTS idx_alert_history_created_at ON alert_history(created_at);
