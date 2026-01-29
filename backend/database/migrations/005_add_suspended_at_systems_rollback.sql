-- Remove suspension support from systems table
ALTER TABLE systems DROP COLUMN IF EXISTS suspended_by_org_id;
ALTER TABLE systems DROP COLUMN IF EXISTS suspended_at;
