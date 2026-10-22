-- Rollback migration 051: drop customer transfers audit and renewal stop columns.

ALTER TABLE system_entitlements DROP COLUMN IF EXISTS renewal_stopped_reason;
ALTER TABLE system_entitlements DROP COLUMN IF EXISTS renewal_stopped_at;

DROP TABLE IF EXISTS customer_org_transfers;
