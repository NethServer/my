-- Migration 023: Index inventory_records on (system_id, created_at)
--
-- Supports the exponential-retention cleanup in
-- collect/workers/cleanup_worker.go. The pre-refactor single-shot query
-- OOM-killed postgres on Render's basic_256mb tier (2026-05-03) because the
-- ROW_NUMBER OVER PARTITION BY system_id + double NOT IN (DISTINCT ON
-- (system_id) ORDER BY created_at) had no covering index and seq-scanned the
-- full ~1.8GB table for every cleanup tick.
--
-- The existing idx_inventory_records_system_id_timestamp covers timestamp,
-- not created_at, so it could not serve this query.

CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_created_at
  ON inventory_records(system_id, created_at);
