-- Rollback 034: drop the users creator snapshot column.

ALTER TABLE users DROP COLUMN IF EXISTS created_by;
