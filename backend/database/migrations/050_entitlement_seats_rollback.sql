-- Rollback migration 050: drop seated add-ons. Cluster-wide crowdsec grants
-- stay cluster-wide.

DROP TABLE IF EXISTS entitlement_seats;

ALTER TABLE system_entitlements DROP CONSTRAINT IF EXISTS system_entitlements_seats_check;
ALTER TABLE system_entitlements DROP COLUMN IF EXISTS seats;

ALTER TABLE entitlement_catalog DROP CONSTRAINT IF EXISTS entitlement_catalog_seated_check;
ALTER TABLE entitlement_catalog DROP COLUMN IF EXISTS seated;
