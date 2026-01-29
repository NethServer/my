-- Rollback migration 004: Remove name and source columns from applications table

ALTER TABLE applications DROP COLUMN IF EXISTS name;
ALTER TABLE applications DROP COLUMN IF EXISTS source;
