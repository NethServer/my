-- Migration 017: Change inventory_diffs FK constraints from CASCADE to SET NULL
-- Diffs must survive when inventory_records are pruned by exponential retention cleanup.
-- previous_value/current_value are stored in the diff itself, so diffs are self-contained.

ALTER TABLE inventory_diffs
    DROP CONSTRAINT inventory_diffs_previous_id_fkey,
    ADD CONSTRAINT inventory_diffs_previous_id_fkey
        FOREIGN KEY (previous_id) REFERENCES inventory_records(id) ON DELETE SET NULL;

ALTER TABLE inventory_diffs
    DROP CONSTRAINT inventory_diffs_current_id_fkey,
    ADD CONSTRAINT inventory_diffs_current_id_fkey
        FOREIGN KEY (current_id) REFERENCES inventory_records(id) ON DELETE SET NULL;
