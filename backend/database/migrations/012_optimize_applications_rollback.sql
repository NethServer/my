-- Rollback: revert materialized view to regular view and drop new indexes

DROP INDEX IF EXISTS idx_applications_org_id_certified;
DROP INDEX IF EXISTS idx_applications_system_cert_userfacing;

DROP MATERIALIZED VIEW IF EXISTS unified_organizations;

-- Recreate as regular VIEW
CREATE OR REPLACE VIEW unified_organizations AS
SELECT logto_id, id::text AS db_id, name, 'distributor' AS org_type FROM distributors WHERE deleted_at IS NULL
UNION ALL
SELECT logto_id, id::text AS db_id, name, 'reseller' AS org_type FROM resellers WHERE deleted_at IS NULL
UNION ALL
SELECT logto_id, id::text AS db_id, name, 'customer' AS org_type FROM customers WHERE deleted_at IS NULL;
