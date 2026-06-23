-- Rollback migration 027: Drop api_key_audit table

DROP INDEX IF EXISTS idx_api_key_audit_event;
DROP INDEX IF EXISTS idx_api_key_audit_key;
DROP INDEX IF EXISTS idx_api_key_audit_org;
DROP INDEX IF EXISTS idx_api_key_audit_user;
DROP TABLE IF EXISTS api_key_audit;
