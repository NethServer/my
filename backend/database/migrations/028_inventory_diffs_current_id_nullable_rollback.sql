-- Rollback migration 028: Restore NOT NULL on inventory_diffs.current_id
--
-- NOTE: this fails if any current_id is already NULL (snapshots pruned after the
-- forward migration). Backfill or delete those rows before rolling back.

COMMENT ON COLUMN inventory_diffs.current_id IS 'Reference to current inventory record';

ALTER TABLE inventory_diffs ALTER COLUMN current_id SET NOT NULL;
