-- Rollback 035: remove the on_behalf_of flag from creator snapshots.
--
-- The organization_name refresh is NOT reverted: the pre-refresh (stale)
-- names are not recorded anywhere, and the refreshed values are the current
-- organization names, which older code renders identically.

UPDATE resellers
SET custom_data = custom_data #- '{createdByUser,on_behalf_of}'
WHERE custom_data->'createdByUser' ? 'on_behalf_of';

UPDATE customers
SET custom_data = custom_data #- '{createdByUser,on_behalf_of}'
WHERE custom_data->'createdByUser' ? 'on_behalf_of';

UPDATE systems
SET created_by = created_by #- '{on_behalf_of}'
WHERE created_by ? 'on_behalf_of';

COMMENT ON COLUMN systems.created_by IS 'JSON object: {user_id, username, organization_id} who created the system';
