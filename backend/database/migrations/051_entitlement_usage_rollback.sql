-- Rollback migration 051: drop soft tier limits. The limits copied into
-- seats from the tier labels are cleared again.

UPDATE system_entitlements e SET seats = NULL
FROM entitlement_catalog c
WHERE c.id = e.entitlement AND c.usage_metric <> '';

DROP INDEX IF EXISTS idx_system_entitlements_over_limit;

ALTER TABLE system_entitlements DROP COLUMN IF EXISTS usage_at;
ALTER TABLE system_entitlements DROP COLUMN IF EXISTS usage;

ALTER TABLE entitlement_catalog DROP CONSTRAINT IF EXISTS entitlement_catalog_usage_metric_check;
ALTER TABLE entitlement_catalog DROP COLUMN IF EXISTS usage_metric;
