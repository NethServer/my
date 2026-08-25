-- Rollback migration 039: back to deriving the application from the id prefix
ALTER TABLE entitlement_catalog DROP COLUMN IF EXISTS applies_to;
