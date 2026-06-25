-- Migration 030: Add (system_id, id) index on inventory_records
--
-- The inventory retention cleanup computes the first/last record id per system
-- via MIN(id)/MAX(id) GROUP BY system_id (the "edge" rows it must never prune).
-- The only matching index is (system_id, created_at), which lacks id, so that
-- query was a ~50s GroupAggregate with heap fetches over ~640k rows. A composite
-- (system_id, id) index makes it an index-only scan: MIN/MAX per group are the
-- first/last entries of each system_id run.

CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_id ON inventory_records(system_id, id);
