-- Rollback migration 048: drop the re-registration counters.

ALTER TABLE systems DROP COLUMN IF EXISTS last_reregistered_at;
ALTER TABLE systems DROP COLUMN IF EXISTS reregistration_count;
