-- Rollback Migration 006: Remove suspended_by_org_id from resellers and customers tables

DROP INDEX IF EXISTS idx_resellers_suspended_by_org_id;
ALTER TABLE resellers DROP COLUMN IF EXISTS suspended_by_org_id;

DROP INDEX IF EXISTS idx_customers_suspended_by_org_id;
ALTER TABLE customers DROP COLUMN IF EXISTS suspended_by_org_id;
