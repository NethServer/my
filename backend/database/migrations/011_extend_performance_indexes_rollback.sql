-- Rollback covering indexes for filter DISTINCT queries
DROP INDEX IF EXISTS idx_systems_type_version;

-- Rollback hierarchy lookups
DROP INDEX IF EXISTS idx_distributors_created_by;

-- Rollback users organization lookup
DROP INDEX IF EXISTS idx_users_organization_id_active;
