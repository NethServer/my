-- Migration 028: Make inventory_diffs.current_id nullable
--
-- Migration 017 changed both inventory_diffs FKs (previous_id, current_id) to
-- ON DELETE SET NULL so diffs survive when their referenced inventory_records
-- snapshot is pruned by exponential retention. previous_id was already nullable,
-- but current_id was left NOT NULL — so deleting a snapshot referenced as
-- current_id raised a not-null violation and the retention cleanup had to SKIP
-- those records, which defeated the policy and let inventory_records grow
-- unbounded (the dominant cause of the QA disk-full incident).
--
-- Dropping NOT NULL lets ON DELETE SET NULL work: a pruned snapshot simply sets
-- the diff's current_id to NULL. Diffs are self-contained (field_path,
-- previous_value, current_value), so they remain meaningful. The backend read
-- path COALESCEs current_id to 0 ("snapshot pruned") for API stability.

ALTER TABLE inventory_diffs ALTER COLUMN current_id DROP NOT NULL;

COMMENT ON COLUMN inventory_diffs.current_id IS 'Reference to current inventory record; NULL once that snapshot is pruned by retention (FK ON DELETE SET NULL)';
