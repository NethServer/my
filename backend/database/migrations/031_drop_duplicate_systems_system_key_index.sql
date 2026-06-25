-- Migration 031: Drop the duplicate UNIQUE index on systems(system_key)
--
-- The DB carries two identical unique indexes on systems(system_key):
-- idx_systems_system_key (tracked in schema.sql) and idx_systems_system_key_new
-- (an untracked orphan, never in schema.sql or any migration). Every write to
-- systems maintains both. Drop the orphan; idx_systems_system_key remains.

DROP INDEX IF EXISTS idx_systems_system_key_new;
