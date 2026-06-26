-- Migration 032: Cut write amplification on the systems table
--
-- systems was getting a non-HOT UPDATE on essentially every inventory ingest
-- (each phone-home rewrote version/fqdn/ip and bumped updated_at), so it churned
-- continuously and autovacuum never stopped (idx_systems_updated_at alone was
-- ~5MB of bloat). Paired with the inventory worker now bumping updated_at only on
-- a real change, two DB changes here:
--
--   1. Drop idx_systems_updated_at. It is unused (0 scans in pg_stat_user_indexes)
--      and was never tracked in schema.sql -- an orphan that every write had to
--      maintain. Dropping it lets updated_at bumps qualify for HOT updates.
--   2. Lower fillfactor to 85 so the frequent last_inventory_at-only refresh has
--      in-page room to land as a HOT update (no index maintenance, cheap prune).
--      Applies to pages written from now on; existing bloat is reclaimed by
--      autovacuum over time.

DROP INDEX IF EXISTS idx_systems_updated_at;

ALTER TABLE systems SET (fillfactor = 85);
