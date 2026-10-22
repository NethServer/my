-- Migration 050: seated NS8 add-ons. A seated add-on is bought for the whole
-- cluster with a number of seats, and each seat is given to one application
-- instance (nethvoice1, nethvoice5, ...) on any node. An add-on without seats
-- is granted to the whole cluster and covers every instance.

ALTER TABLE entitlement_catalog ADD COLUMN IF NOT EXISTS seated BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE entitlement_catalog DROP CONSTRAINT IF EXISTS entitlement_catalog_seated_check;
ALTER TABLE entitlement_catalog ADD CONSTRAINT entitlement_catalog_seated_check
    CHECK (NOT seated OR (applies_to <> '' AND NOT scoped));

COMMENT ON COLUMN entitlement_catalog.seated IS 'TRUE = bought for the whole system with a number of seats, each seat given to one application instance of applies_to (entitlement_seats); FALSE = the grant covers the whole system';

ALTER TABLE system_entitlements ADD COLUMN IF NOT EXISTS seats INTEGER;

ALTER TABLE system_entitlements DROP CONSTRAINT IF EXISTS system_entitlements_seats_check;
ALTER TABLE system_entitlements ADD CONSTRAINT system_entitlements_seats_check
    CHECK (seats IS NULL OR seats > 0);

COMMENT ON COLUMN system_entitlements.seats IS 'Seats bought, for seated add-ons (set by activate from the shop tier); NULL for add-ons without seats';

CREATE TABLE IF NOT EXISTS entitlement_seats (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    grant_id        UUID NOT NULL REFERENCES system_entitlements(id) ON DELETE CASCADE,
    module_id       VARCHAR(255) NOT NULL,
    assigned_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    assigned_by     JSONB,
    released_at     TIMESTAMP WITH TIME ZONE,
    released_reason VARCHAR(50),
    released_by     JSONB
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_entitlement_seats_held
    ON entitlement_seats(grant_id, module_id) WHERE released_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_entitlement_seats_grant_id ON entitlement_seats(grant_id);

COMMENT ON TABLE  entitlement_seats                 IS 'Seats of a seated add-on given to application instances; checked by collect /auth/service/<id>?scope=<module_id>. Released rows are kept for audit';
COMMENT ON COLUMN entitlement_seats.module_id       IS 'Application instance holding the seat (applications.module_id, e.g. nethvoice5)';
COMMENT ON COLUMN entitlement_seats.assigned_by     IS 'Actor snapshot of the user that gave the seat';
COMMENT ON COLUMN entitlement_seats.released_at     IS 'Set when the seat is freed; NULL = held';
COMMENT ON COLUMN entitlement_seats.released_reason IS 'manual (freed by a user) | instance_removed (the instance left the inventory: removed, or moved to another node under a new id)';
COMMENT ON COLUMN entitlement_seats.released_by     IS 'Actor snapshot of the user that freed the seat; NULL when collect freed it';

-- NS8 add-ons are bought for the whole cluster, never for one instance: the
-- catalog items narrowed to single instances become cluster-wide, and so does
-- the most relevant grant of each (system, add-on) holding no cluster-wide one.
UPDATE entitlement_catalog SET scoped = FALSE, updated_at = NOW() WHERE scoped;

UPDATE system_entitlements e SET scope = '', updated_at = NOW()
WHERE e.scope <> ''
  AND e.id = (SELECT e2.id FROM system_entitlements e2
               WHERE e2.system_id = e.system_id AND e2.entitlement = e.entitlement AND e2.scope <> ''
               ORDER BY (e2.revoked_at IS NULL) DESC, e2.valid_until DESC NULLS FIRST, e2.updated_at DESC
               LIMIT 1)
  AND NOT EXISTS (SELECT 1 FROM system_entitlements e3
                   WHERE e3.system_id = e.system_id AND e3.entitlement = e.entitlement AND e3.scope = '');
