-- Rollback 017: Restore inventory_diffs FK constraints to CASCADE

ALTER TABLE inventory_diffs
    DROP CONSTRAINT inventory_diffs_previous_id_fkey,
    ADD CONSTRAINT inventory_diffs_previous_id_fkey
        FOREIGN KEY (previous_id) REFERENCES inventory_records(id) ON DELETE CASCADE;

ALTER TABLE inventory_diffs
    DROP CONSTRAINT inventory_diffs_current_id_fkey,
    ADD CONSTRAINT inventory_diffs_current_id_fkey
        FOREIGN KEY (current_id) REFERENCES inventory_records(id) ON DELETE CASCADE;
