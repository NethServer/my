-- Rollback migration 026: Drop user_api_keys table

DROP INDEX IF EXISTS idx_user_api_keys_active;
DROP INDEX IF EXISTS idx_user_api_keys_user_id;
DROP INDEX IF EXISTS idx_user_api_keys_public;
DROP TABLE IF EXISTS user_api_keys;
