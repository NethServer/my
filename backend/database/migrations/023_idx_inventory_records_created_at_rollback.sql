-- Rollback migration 023: Drop (system_id, created_at) index on inventory_records

DROP INDEX IF EXISTS idx_inventory_records_system_id_created_at;
