-- Rollback migration 040: back to the two-column shape.
--
-- WARNING: destructive where the columns did NOT come from 040. On a database
-- bootstrapped from schema.sql they hold real data -- created_at is the only
-- record of when each system first reported -- and there is no way to tell the
-- two cases apart from inside the migration. Run it only against a database
-- where 040 actually added the columns.

ALTER TABLE system_heartbeats
    DROP COLUMN IF EXISTS created_at,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS metadata;
