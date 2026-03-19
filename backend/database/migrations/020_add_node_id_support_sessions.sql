-- Migration 019: Add node_id to support_sessions for multi-node cluster support
-- Each node in an NS8 cluster connects its own tunnel, identified by node_id.
-- node_id is NULL for single-node (non-cluster) systems.

ALTER TABLE support_sessions ADD COLUMN node_id VARCHAR(16);

COMMENT ON COLUMN support_sessions.node_id IS 'NS8 cluster node ID (e.g., 1, 2, 3). NULL for single-node systems.';

-- Index for efficient lookups by (system_id, node_id)
CREATE INDEX idx_support_sessions_system_node ON support_sessions(system_id, node_id) WHERE status IN ('pending', 'active');
