-- Migration 061: drop the duplicate UNIQUE constraint on systems(system_key)
--
-- schema.sql declared system_key both UNIQUE inline (systems_system_key_key)
-- and with idx_systems_system_key, so every write to systems maintained two
-- identical unique indexes. Keep idx_systems_system_key, which every
-- environment has.

ALTER TABLE systems DROP CONSTRAINT IF EXISTS systems_system_key_key;
