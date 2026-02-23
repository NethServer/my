-- Optimize applications: materialized view + covering indexes

-- 1. Convert unified_organizations from VIEW to MATERIALIZED VIEW
DROP VIEW IF EXISTS unified_organizations;
CREATE MATERIALIZED VIEW unified_organizations AS
SELECT logto_id, id::text AS db_id, name, 'distributor' AS org_type FROM distributors WHERE deleted_at IS NULL
UNION ALL
SELECT logto_id, id::text AS db_id, name, 'reseller' AS org_type FROM resellers WHERE deleted_at IS NULL
UNION ALL
SELECT logto_id, id::text AS db_id, name, 'customer' AS org_type FROM customers WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX idx_unified_organizations_logto_id ON unified_organizations(logto_id);

-- 2. Covering index for certified user-facing application queries
CREATE INDEX IF NOT EXISTS idx_applications_system_cert_userfacing
ON applications(system_id, instance_of, status, version)
WHERE deleted_at IS NULL AND is_user_facing = TRUE
  AND (inventory_data->>'certification_level')::int IN (4, 5);

-- 3. Partial index for organization counts on certified user-facing applications
CREATE INDEX IF NOT EXISTS idx_applications_org_id_certified
ON applications(organization_id)
WHERE deleted_at IS NULL AND is_user_facing = TRUE
  AND (inventory_data->>'certification_level')::int IN (4, 5);
