-- Migration 040: align system_heartbeats with the schema it is declared with.
--
-- This table is the one piece of the schema no migration ever created: it only
-- ever existed in schema.sql, so every database bootstrapped from that file got
-- the full definition, while the older ones kept the two-column shape they were
-- created with. Nothing surfaced the difference until GET /systems/:id started
-- reading created_at as the first-contact marker, which on the lagging shape
-- fails with "column h.created_at does not exist" -- and it fails for every
-- endpoint that resolves a system by id (detail, inventory, alerts, backups),
-- not just the one that displays the new field.
--
-- id is deliberately left out. Where it is missing, the primary key sits on
-- system_id instead of a surrogate, so adding it would mean dropping and
-- recreating the constraint for a column no query reads; the heartbeat upsert
-- conflicts on system_id, which is unique in both shapes. That residual
-- difference is invisible to the code.

ALTER TABLE system_heartbeats
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS status     VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS metadata   JSONB;

COMMENT ON COLUMN system_heartbeats.created_at IS 'First contact: written by the insert of the first beat, preserved by every later upsert';
