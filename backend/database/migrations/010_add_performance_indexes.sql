-- Performance optimization: unified organizations view and indexes
-- This view replaces the 3-way LEFT JOIN pattern used in applications queries

CREATE OR REPLACE VIEW unified_organizations AS
SELECT logto_id, id::text AS db_id, name, 'distributor' AS org_type FROM distributors WHERE deleted_at IS NULL
UNION ALL
SELECT logto_id, id::text AS db_id, name, 'reseller' AS org_type FROM resellers WHERE deleted_at IS NULL
UNION ALL
SELECT logto_id, id::text AS db_id, name, 'customer' AS org_type FROM customers WHERE deleted_at IS NULL;

-- Expression indexes for frequently filtered JSONB fields
CREATE INDEX IF NOT EXISTS idx_applications_cert_level ON applications (((inventory_data->>'certification_level')::int)) WHERE deleted_at IS NULL AND is_user_facing = TRUE;
CREATE INDEX IF NOT EXISTS idx_resellers_created_by ON resellers ((custom_data->>'createdBy')) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_created_by ON customers ((custom_data->>'createdBy')) WHERE deleted_at IS NULL;
