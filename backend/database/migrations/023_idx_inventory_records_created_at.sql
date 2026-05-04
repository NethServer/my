-- Migration 023: Index inventory_records on (system_id, created_at)
--
-- Supports the exponential-retention cleanup in
-- collect/workers/cleanup_worker.go which filters and partitions by
-- (system_id, created_at). The existing
-- idx_inventory_records_system_id_timestamp indexes a different column
-- (timestamp) and does not serve this query.

CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_created_at
  ON inventory_records(system_id, created_at);
