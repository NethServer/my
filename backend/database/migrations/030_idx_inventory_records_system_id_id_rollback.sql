-- Rollback migration 030: Drop the inventory_records (system_id, id) index

DROP INDEX IF EXISTS idx_inventory_records_system_id_id;
