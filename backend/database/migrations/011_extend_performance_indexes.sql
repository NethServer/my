-- Covering indexes for filter DISTINCT queries
CREATE INDEX IF NOT EXISTS idx_systems_type_version ON systems(type, version) WHERE deleted_at IS NULL;

-- Hierarchy lookups (distributors don't have idx yet)
CREATE INDEX IF NOT EXISTS idx_distributors_created_by ON distributors ((custom_data->>'createdBy')) WHERE deleted_at IS NULL;

-- Users organization lookup (partial index filtering deleted)
CREATE INDEX IF NOT EXISTS idx_users_organization_id_active ON users(organization_id) WHERE deleted_at IS NULL;
