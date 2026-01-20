-- Migration: 001_add_applications
-- Description: Add applications table for tracking NS8 cluster applications

-- Applications table - extracted from inventory, with organization assignment
CREATE TABLE IF NOT EXISTS applications (
    id VARCHAR(255) PRIMARY KEY,

    -- Relationship to system (source of the application)
    system_id VARCHAR(255) NOT NULL,

    -- Identity from inventory (cluster_module_domain_table)
    module_id VARCHAR(255) NOT NULL,        -- Inventory identifier (e.g., "nethvoice1", "webtop3", "mail1")
    instance_of VARCHAR(100) NOT NULL,      -- Application type from inventory (e.g., "nethvoice", "webtop", "mail")

    -- Display name (for UI customization)
    display_name VARCHAR(255),              -- Custom name like "Milan Office PBX" (nullable, falls back to module_id)

    -- From inventory
    node_id INTEGER,                        -- Cluster node ID where the app runs
    domain_id VARCHAR(255),                 -- User domain associated with the app (can be null)
    version VARCHAR(100),                   -- Application version (when available from inventory)

    -- Organization assignment (core business requirement)
    organization_id VARCHAR(255),           -- FK to org (NULL = unassigned)
    organization_type VARCHAR(50),          -- owner, distributor, reseller, customer (denormalized for queries)

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'unassigned',  -- unassigned, assigned, error

    -- Flexible JSONB for type-specific data from inventory
    inventory_data JSONB,                   -- All raw data from cluster_module_domain_table entry
    backup_data JSONB,                      -- Backup status from inventory (when available)
    services_data JSONB,                    -- Services health status (when available)

    -- App URL (extracted from traefik name_module_map or configured manually)
    url VARCHAR(500),

    -- Notes/description
    notes TEXT,

    -- Flags
    is_user_facing BOOLEAN NOT NULL DEFAULT TRUE,  -- FALSE for system components like traefik, loki

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_inventory_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE      -- Soft delete
);

-- Comment for applications table
COMMENT ON TABLE applications IS 'Applications extracted from NS8 cluster inventory with organization assignment';

-- Comments for key columns
COMMENT ON COLUMN applications.module_id IS 'Unique module identifier from inventory (e.g., nethvoice1, webtop3)';
COMMENT ON COLUMN applications.instance_of IS 'Application type from inventory (e.g., nethvoice, webtop, mail, nextcloud)';
COMMENT ON COLUMN applications.display_name IS 'Custom display name for UI. Falls back to module_id if NULL';
COMMENT ON COLUMN applications.node_id IS 'Cluster node ID where the application runs';
COMMENT ON COLUMN applications.domain_id IS 'User domain associated with the application (from inventory)';
COMMENT ON COLUMN applications.organization_id IS 'Assigned organization ID. NULL means unassigned';
COMMENT ON COLUMN applications.organization_type IS 'Denormalized organization type for efficient filtering';
COMMENT ON COLUMN applications.status IS 'Application status: unassigned (no org), assigned (has org), error (has issues)';
COMMENT ON COLUMN applications.inventory_data IS 'Raw application data from cluster_module_domain_table';
COMMENT ON COLUMN applications.backup_data IS 'Backup status information from inventory';
COMMENT ON COLUMN applications.services_data IS 'Services health status from inventory';
COMMENT ON COLUMN applications.is_user_facing IS 'FALSE for system components (traefik, loki) that should be hidden in UI';
COMMENT ON COLUMN applications.deleted_at IS 'Soft delete timestamp. NULL means active';

-- Unique constraint: one application per module_id per system
CREATE UNIQUE INDEX IF NOT EXISTS idx_applications_system_module
    ON applications(system_id, module_id) WHERE deleted_at IS NULL;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_applications_system_id ON applications(system_id);
CREATE INDEX IF NOT EXISTS idx_applications_organization_id ON applications(organization_id);
CREATE INDEX IF NOT EXISTS idx_applications_instance_of ON applications(instance_of);
CREATE INDEX IF NOT EXISTS idx_applications_status ON applications(status);
CREATE INDEX IF NOT EXISTS idx_applications_version ON applications(version);
CREATE INDEX IF NOT EXISTS idx_applications_is_user_facing ON applications(is_user_facing);
CREATE INDEX IF NOT EXISTS idx_applications_deleted_at ON applications(deleted_at);
CREATE INDEX IF NOT EXISTS idx_applications_created_at ON applications(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_applications_node_id ON applications(node_id);
CREATE INDEX IF NOT EXISTS idx_applications_domain_id ON applications(domain_id);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_applications_org_type_status
    ON applications(organization_id, instance_of, status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_applications_system_user_facing
    ON applications(system_id, is_user_facing) WHERE deleted_at IS NULL;

-- Foreign key to systems
ALTER TABLE applications
ADD CONSTRAINT applications_system_id_fkey
FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;

-- Status validation
ALTER TABLE applications ADD CONSTRAINT chk_applications_status
    CHECK (status IN ('unassigned', 'assigned', 'error'));

-- Organization type validation
ALTER TABLE applications ADD CONSTRAINT chk_applications_org_type
    CHECK (organization_type IS NULL OR organization_type IN ('owner', 'distributor', 'reseller', 'customer'));