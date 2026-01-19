-- Rollback Migration: 001_add_applications
-- Description: Remove applications table

-- Drop indexes first
DROP INDEX IF EXISTS idx_applications_system_module;
DROP INDEX IF EXISTS idx_applications_system_id;
DROP INDEX IF EXISTS idx_applications_organization_id;
DROP INDEX IF EXISTS idx_applications_instance_of;
DROP INDEX IF EXISTS idx_applications_status;
DROP INDEX IF EXISTS idx_applications_version;
DROP INDEX IF EXISTS idx_applications_is_user_facing;
DROP INDEX IF EXISTS idx_applications_deleted_at;
DROP INDEX IF EXISTS idx_applications_created_at;
DROP INDEX IF EXISTS idx_applications_node_id;
DROP INDEX IF EXISTS idx_applications_domain_id;
DROP INDEX IF EXISTS idx_applications_org_type_status;
DROP INDEX IF EXISTS idx_applications_system_user_facing;

-- Drop table
DROP TABLE IF EXISTS applications;

-- Remove migration record
DELETE FROM schema_migrations WHERE migration_number = '001';
