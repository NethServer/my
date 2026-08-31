-- Migration 043: unregistration, the terminal state of a registered system.
--
-- "Cancel subscription" on an appliance clears the credentials it holds
-- locally, but the pair itself stays valid on collect: a copy kept anywhere
-- else still authenticates, and heartbeat, inventory, backups and the
-- enterprise feeds all keep answering it until someone deletes the system.
--
-- unregistered_at closes that window. The appliance announces its own
-- departure, collect refuses the pair from that moment on, and the row
-- survives as the record of a key that has been spent.
--
-- The state is terminal by design. A system key is one-shot: a machine that
-- needs a subscription again is given a new system, and this row is deleted
-- once nobody needs its history. That is also what keeps the licence count
-- honest — the slot is consumed at creation and freed only by deletion, so
-- an unregistered system counts exactly like a registered one.

ALTER TABLE systems
    ADD COLUMN IF NOT EXISTS unregistered_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN systems.unregistered_at IS 'Timestamp when the system announced its unregistration. NULL = credentials still valid; non-NULL = credentials refused, row kept until deleted';
COMMENT ON COLUMN systems.status IS 'Heartbeat status: unknown (no data), active (heartbeat recent), inactive (heartbeat stale), unregistered (credentials refused), deleted';

ALTER TABLE systems DROP CONSTRAINT IF EXISTS chk_systems_status;
ALTER TABLE systems ADD CONSTRAINT chk_systems_status
    CHECK (status IN ('unknown', 'active', 'inactive', 'unregistered', 'deleted'));
