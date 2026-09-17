-- Rollback migration 045: the per-distributor portal list goes away.
--
-- Dropping the column makes GET /third-party-applications fall back to the
-- role filter alone: every portal admitted by access_control shows for every
-- hierarchy again. The lists are lost; the Owner organization re-enters them
-- from the distributor drawer.

ALTER TABLE distributors DROP COLUMN IF EXISTS third_party_apps;
