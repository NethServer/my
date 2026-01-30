-- Add suspension support to systems table
ALTER TABLE systems ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE systems ADD COLUMN IF NOT EXISTS suspended_by_org_id VARCHAR(255);

-- Add comments
COMMENT ON COLUMN systems.suspended_at IS 'Suspension timestamp: NULL = active, non-NULL = suspended';
COMMENT ON COLUMN systems.suspended_by_org_id IS 'Organization that caused cascade suspension (for targeted reactivation)';
