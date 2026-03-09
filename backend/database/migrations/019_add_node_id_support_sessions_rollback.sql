-- Rollback migration 019: Remove node_id from support_sessions

DROP INDEX IF EXISTS idx_support_sessions_system_node;
ALTER TABLE support_sessions DROP COLUMN IF EXISTS node_id;
