-- Migration 004: Add node_label column to applications table
-- This column stores the human-readable node label (e.g., "Leader Node", "Worker Node")

-- Add node_label column after node_id
ALTER TABLE applications ADD COLUMN IF NOT EXISTS node_label VARCHAR(255);

-- Add comment
COMMENT ON COLUMN applications.node_label IS 'Node label from inventory (e.g., Leader Node, Worker Node)';

-- Record migration
INSERT INTO schema_migrations (migration_number, description)
VALUES ('004', 'Add node_label column to applications table')
ON CONFLICT (migration_number) DO NOTHING;
