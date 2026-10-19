-- Rollback migration 047: drop update_available.

ALTER TABLE systems DROP COLUMN IF EXISTS update_available;
