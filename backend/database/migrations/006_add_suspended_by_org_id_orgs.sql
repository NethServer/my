-- Migration 006: Add suspended_by_org_id to resellers and customers tables
-- Tracks which organization initiated the cascade suspension

ALTER TABLE resellers ADD COLUMN IF NOT EXISTS suspended_by_org_id VARCHAR(255);
COMMENT ON COLUMN resellers.suspended_by_org_id IS 'Organization ID that caused cascade suspension (for targeted reactivation)';
CREATE INDEX IF NOT EXISTS idx_resellers_suspended_by_org_id ON resellers(suspended_by_org_id) WHERE suspended_by_org_id IS NOT NULL;

ALTER TABLE customers ADD COLUMN IF NOT EXISTS suspended_by_org_id VARCHAR(255);
COMMENT ON COLUMN customers.suspended_by_org_id IS 'Organization ID that caused cascade suspension (for targeted reactivation)';
CREATE INDEX IF NOT EXISTS idx_customers_suspended_by_org_id ON customers(suspended_by_org_id) WHERE suspended_by_org_id IS NOT NULL;
