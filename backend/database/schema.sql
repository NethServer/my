-- =============================================================================
-- Nethesis Operation Center - Database Schema
-- =============================================================================
-- This schema implements a local-first approach with separate entity tables
-- for distributors, resellers, customers, users, systems, and applications.
-- All organization tables sync with Logto identity provider.
-- =============================================================================

-- =============================================================================
-- DISTRIBUTORS TABLE
-- =============================================================================
-- Top-level business partners in the hierarchy (Owner > Distributor > Reseller > Customer)
-- Synced with Logto organizations

CREATE TABLE IF NOT EXISTS distributors (
    id VARCHAR(255) PRIMARY KEY,            -- Local unique identifier

    -- Logto synchronization
    logto_id VARCHAR(255),                  -- Logto organization ID (synced from Logto)
    logto_synced_at TIMESTAMP WITH TIME ZONE,  -- Last successful sync timestamp
    logto_sync_error TEXT,                  -- Last sync error message (if any)

    -- Business information
    name VARCHAR(255) NOT NULL,             -- Display name (e.g., "Acme Distribution")
    description TEXT,                       -- Optional description

    -- Flexible metadata (VAT, address, contact, etc.)
    custom_data JSONB,                      -- {vat, address, city, contact, email, phone, language, notes, createdBy}

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Soft delete and suspension
    deleted_at TIMESTAMP WITH TIME ZONE,    -- NULL = active, non-NULL = soft deleted
    suspended_at TIMESTAMP WITH TIME ZONE   -- NULL = active, non-NULL = suspended/blocked
);

-- Table documentation
COMMENT ON TABLE distributors IS 'Top-level business partners that can have resellers and customers';
COMMENT ON COLUMN distributors.logto_id IS 'Logto organization ID for identity provider sync';
COMMENT ON COLUMN distributors.custom_data IS 'Flexible JSON: {vat, address, city, contact, email, phone, language, notes, createdBy}';
COMMENT ON COLUMN distributors.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted';
COMMENT ON COLUMN distributors.suspended_at IS 'Suspension timestamp. NULL means active, non-NULL means blocked';

-- Performance indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_distributors_logto_id ON distributors(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_distributors_deleted_at ON distributors(deleted_at);
CREATE INDEX IF NOT EXISTS idx_distributors_suspended_at ON distributors(suspended_at);
CREATE INDEX IF NOT EXISTS idx_distributors_logto_synced ON distributors(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_distributors_created_at ON distributors(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_distributors_name ON distributors(name);
CREATE INDEX IF NOT EXISTS idx_distributors_vat_jsonb ON distributors((custom_data->>'vat'));

-- =============================================================================
-- RESELLERS TABLE
-- =============================================================================
-- Mid-level partners in hierarchy, belong to a distributor
-- Synced with Logto organizations

CREATE TABLE IF NOT EXISTS resellers (
    id VARCHAR(255) PRIMARY KEY,            -- Local unique identifier

    -- Logto synchronization
    logto_id VARCHAR(255),                  -- Logto organization ID (synced from Logto)
    logto_synced_at TIMESTAMP WITH TIME ZONE,  -- Last successful sync timestamp
    logto_sync_error TEXT,                  -- Last sync error message (if any)

    -- Business information
    name VARCHAR(255) NOT NULL,             -- Display name (e.g., "TechReseller Inc")
    description TEXT,                       -- Optional description

    -- Flexible metadata (VAT, address, contact, parent reference, etc.)
    custom_data JSONB,                      -- {vat, address, city, contact, email, phone, language, notes, createdBy}

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Soft delete and suspension
    deleted_at TIMESTAMP WITH TIME ZONE,    -- NULL = active, non-NULL = soft deleted
    suspended_at TIMESTAMP WITH TIME ZONE   -- NULL = active, non-NULL = suspended/blocked
);

-- Table documentation
COMMENT ON TABLE resellers IS 'Mid-level partners belonging to distributors, can have customers';
COMMENT ON COLUMN resellers.logto_id IS 'Logto organization ID for identity provider sync';
COMMENT ON COLUMN resellers.custom_data IS 'Flexible JSON: {vat, address, city, contact, email, phone, language, notes, createdBy}';
COMMENT ON COLUMN resellers.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted';
COMMENT ON COLUMN resellers.suspended_at IS 'Suspension timestamp. NULL means active, non-NULL means blocked';

-- Performance indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_resellers_logto_id ON resellers(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resellers_deleted_at ON resellers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_resellers_suspended_at ON resellers(suspended_at);
CREATE INDEX IF NOT EXISTS idx_resellers_logto_synced ON resellers(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_resellers_created_at ON resellers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_resellers_name ON resellers(name);
CREATE INDEX IF NOT EXISTS idx_resellers_vat_jsonb ON resellers((custom_data->>'vat'));

-- =============================================================================
-- CUSTOMERS TABLE
-- =============================================================================
-- End customers in hierarchy, belong to a distributor or reseller
-- Synced with Logto organizations

CREATE TABLE IF NOT EXISTS customers (
    id VARCHAR(255) PRIMARY KEY,            -- Local unique identifier

    -- Logto synchronization
    logto_id VARCHAR(255),                  -- Logto organization ID (synced from Logto)
    logto_synced_at TIMESTAMP WITH TIME ZONE,  -- Last successful sync timestamp
    logto_sync_error TEXT,                  -- Last sync error message (if any)

    -- Business information
    name VARCHAR(255) NOT NULL,             -- Display name (e.g., "Example Corp")
    description TEXT,                       -- Optional description

    -- Flexible metadata (VAT, address, contact, parent reference, etc.)
    custom_data JSONB,                      -- {vat, address, city, contact, email, phone, language, notes, createdBy}

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Soft delete and suspension
    deleted_at TIMESTAMP WITH TIME ZONE,    -- NULL = active, non-NULL = soft deleted
    suspended_at TIMESTAMP WITH TIME ZONE   -- NULL = active, non-NULL = suspended/blocked
);

-- Table documentation
COMMENT ON TABLE customers IS 'End customers belonging to distributors or resellers';
COMMENT ON COLUMN customers.logto_id IS 'Logto organization ID for identity provider sync';
COMMENT ON COLUMN customers.custom_data IS 'Flexible JSON: {vat, address, city, contact, email, phone, language, notes, createdBy}';
COMMENT ON COLUMN customers.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted';
COMMENT ON COLUMN customers.suspended_at IS 'Suspension timestamp. NULL means active, non-NULL means blocked';

-- Performance indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_logto_id ON customers(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON customers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_customers_suspended_at ON customers(suspended_at);
CREATE INDEX IF NOT EXISTS idx_customers_logto_synced ON customers(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);
CREATE INDEX IF NOT EXISTS idx_customers_vat_jsonb ON customers((custom_data->>'vat'));

-- =============================================================================
-- USERS TABLE
-- =============================================================================
-- User accounts with organization membership (1 user = 1 organization)
-- Synced with Logto users

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,            -- Local unique identifier

    -- Logto synchronization
    logto_id VARCHAR(255),                  -- Logto user ID (synced from Logto)
    logto_synced_at TIMESTAMP WITH TIME ZONE,  -- Last successful sync timestamp

    -- User identity
    username VARCHAR(255) NOT NULL,         -- Unique username
    email VARCHAR(255) NOT NULL,            -- Unique email address
    name VARCHAR(255),                      -- Display name (e.g., "John Doe")
    phone VARCHAR(20),                      -- Phone number (optional)

    -- Organization membership (1 user = 1 organization)
    organization_id VARCHAR(255),           -- Logto organization ID the user belongs to

    -- Role assignment
    user_role_ids JSONB DEFAULT '[]',       -- Array of technical role IDs (e.g., ["admin-role-id", "support-role-id"])

    -- Flexible metadata
    custom_data JSONB,                      -- Additional user metadata

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    latest_login_at TIMESTAMP WITH TIME ZONE,  -- Last login timestamp

    -- Soft delete and suspension
    deleted_at TIMESTAMP WITH TIME ZONE,    -- NULL = active, non-NULL = soft deleted
    suspended_at TIMESTAMP WITH TIME ZONE,  -- NULL = active, non-NULL = suspended/blocked
    suspended_by_org_id VARCHAR(255)        -- Organization ID that caused cascade suspension
);

-- Table documentation
COMMENT ON TABLE users IS 'User accounts with organization membership, synced with Logto';
COMMENT ON COLUMN users.logto_id IS 'Logto user ID for identity provider sync';
COMMENT ON COLUMN users.organization_id IS 'Logto organization ID the user belongs to';
COMMENT ON COLUMN users.user_role_ids IS 'Array of Logto role IDs assigned to user';
COMMENT ON COLUMN users.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted';
COMMENT ON COLUMN users.suspended_at IS 'Suspension timestamp. NULL means active, non-NULL means blocked';
COMMENT ON COLUMN users.suspended_by_org_id IS 'Organization ID that caused cascade suspension (for automatic reactivation)';

-- Performance indexes
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_logto_id ON users(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_suspended_at ON users(suspended_at);
CREATE INDEX IF NOT EXISTS idx_users_suspended_by_org_id ON users(suspended_by_org_id) WHERE suspended_by_org_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_logto_synced ON users(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_latest_login_at ON users(latest_login_at DESC);

-- =============================================================================
-- SYSTEMS TABLE
-- =============================================================================
-- NS8/NethSecurity systems registered for monitoring
-- Systems authenticate via system_key + system_secret for inventory/heartbeat

CREATE TABLE IF NOT EXISTS systems (
    id VARCHAR(255) PRIMARY KEY,            -- Local unique identifier

    -- System identity
    name VARCHAR(255) NOT NULL,             -- Display name (e.g., "Milan Office Server")
    type VARCHAR(100),                      -- System type: "ns8", "nsec" (populated by collect on first inventory)
    fqdn VARCHAR(255),                      -- Fully qualified domain name (from inventory)
    ipv4_address INET,                      -- Public IPv4 address (from inventory)
    ipv6_address INET,                      -- Public IPv6 address (from inventory)
    version VARCHAR(100),                   -- OS/system version (from inventory)

    -- Status (managed by collect service heartbeat monitor)
    status VARCHAR(50) NOT NULL DEFAULT 'unknown',  -- unknown, online, offline, deleted

    -- Organization ownership
    organization_id VARCHAR(255) NOT NULL,  -- Logto organization ID that owns this system

    -- Authentication credentials
    system_key VARCHAR(255) UNIQUE NOT NULL,     -- Unique system key for identification
    system_secret_public VARCHAR(64),            -- Public part of token (my_<public>.<secret>) for fast lookup
    system_secret VARCHAR(512) NOT NULL,         -- Argon2id hash of secret part in PHC format

    -- Metadata
    custom_data JSONB,                      -- Additional system metadata
    notes TEXT DEFAULT '',                  -- User notes/description
    created_by JSONB NOT NULL,              -- {user_id, username, organization_id} who created the system

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    registered_at TIMESTAMP WITH TIME ZONE, -- When system completed registration (NULL = not registered)

    -- Soft delete
    deleted_at TIMESTAMP WITH TIME ZONE     -- NULL = active, non-NULL = soft deleted
);

-- Table documentation
COMMENT ON TABLE systems IS 'NS8/NethSecurity systems registered for monitoring and inventory collection';
COMMENT ON COLUMN systems.type IS 'System type from inventory: ns8 (NethServer 8), nsec (NethSecurity)';
COMMENT ON COLUMN systems.status IS 'Heartbeat status: unknown (no data), online (active), offline (no heartbeat), deleted';
COMMENT ON COLUMN systems.system_key IS 'Unique system key for identification (used with secret for auth)';
COMMENT ON COLUMN systems.system_secret_public IS 'Public part of token (my_<public>.<secret>) for fast DB lookup';
COMMENT ON COLUMN systems.system_secret IS 'Argon2id hash of secret part in PHC string format';
COMMENT ON COLUMN systems.registered_at IS 'Timestamp when system first sent inventory. NULL = not yet registered';
COMMENT ON COLUMN systems.created_by IS 'JSON object: {user_id, username, organization_id} who created the system';
COMMENT ON COLUMN systems.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted';

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_systems_organization_id ON systems(organization_id);
CREATE INDEX IF NOT EXISTS idx_systems_created_by_org ON systems((created_by->>'organization_id'));
CREATE INDEX IF NOT EXISTS idx_systems_status ON systems(status);
CREATE INDEX IF NOT EXISTS idx_systems_type ON systems(type);
CREATE INDEX IF NOT EXISTS idx_systems_deleted_at ON systems(deleted_at);
CREATE INDEX IF NOT EXISTS idx_systems_registered_at ON systems(registered_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_systems_system_key ON systems(system_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_systems_system_secret_public ON systems(system_secret_public) WHERE system_secret_public IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_systems_system_secret ON systems(system_secret);
CREATE INDEX IF NOT EXISTS idx_systems_fqdn ON systems(fqdn);
CREATE INDEX IF NOT EXISTS idx_systems_ipv4_address ON systems(ipv4_address);
CREATE INDEX IF NOT EXISTS idx_systems_ipv6_address ON systems(ipv6_address);

-- Status validation constraint
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_systems_status') THEN
        ALTER TABLE systems ADD CONSTRAINT chk_systems_status
            CHECK (status IN ('unknown', 'online', 'offline', 'deleted'));
    END IF;
END $$;

-- =============================================================================
-- APPLICATIONS TABLE
-- =============================================================================
-- Applications/modules extracted from NS8 cluster inventory
-- Each row represents a module instance (e.g., nethvoice1, webtop3, mail1)
-- Can be assigned to organizations for billing/management

CREATE TABLE IF NOT EXISTS applications (
    id VARCHAR(255) PRIMARY KEY,            -- Composite key: {system_id}-{module_id}

    -- Relationship to system (source of the application)
    system_id VARCHAR(255) NOT NULL,        -- FK to systems table

    -- Identity from inventory (facts.modules[])
    module_id VARCHAR(255) NOT NULL,        -- Module ID from inventory (e.g., "nethvoice1", "webtop3", "mail1")
    instance_of VARCHAR(100) NOT NULL,      -- Module type/name (e.g., "nethvoice", "webtop", "mail", "nextcloud")
    name VARCHAR(255),                      -- Human-readable label from inventory (e.g., "Nextcloud")
    source VARCHAR(500),                    -- Image source from inventory (e.g., "ghcr.io/nethserver/nextcloud")

    -- Display name (for UI customization)
    display_name VARCHAR(255),              -- From modules[].ui_name or custom name (nullable, falls back to module_id)

    -- From inventory (facts.modules[] and facts.nodes[])
    node_id INTEGER,                        -- Cluster node ID where the app runs (from modules[].node)
    node_label VARCHAR(255),                -- Node label from nodes[].ui_name
    version VARCHAR(100),                   -- Application version (when available from inventory)

    -- Organization assignment (core business requirement)
    organization_id VARCHAR(255),           -- Logto org ID assigned to this app (NULL = unassigned)
    organization_type VARCHAR(50),          -- owner, distributor, reseller, customer (denormalized for queries)

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'unassigned',  -- unassigned, assigned, error

    -- Flexible JSONB for type-specific data from inventory
    inventory_data JSONB,                   -- Module data from facts.modules[] (excludes id, name, version, node, ui_name)
    backup_data JSONB,                      -- Backup status from inventory (when available)
    services_data JSONB,                    -- Services health status from inventory (when available)

    -- App URL (extracted from traefik or configured manually)
    url VARCHAR(500),                       -- Public URL to access the application

    -- Notes/description
    notes TEXT,                             -- User notes about the application

    -- Flags
    is_user_facing BOOLEAN NOT NULL DEFAULT TRUE,  -- FALSE for system components (traefik, loki, promtail)

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    first_seen_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),  -- When app first appeared in inventory
    last_inventory_at TIMESTAMP WITH TIME ZONE,  -- Last inventory update for this app

    -- Soft delete
    deleted_at TIMESTAMP WITH TIME ZONE     -- NULL = active, non-NULL = soft deleted (app removed from cluster)
);

-- Table documentation
COMMENT ON TABLE applications IS 'Applications/modules extracted from NS8 cluster inventory with organization assignment';
COMMENT ON COLUMN applications.id IS 'Composite key: {system_id}-{module_id} for uniqueness';
COMMENT ON COLUMN applications.module_id IS 'Unique module identifier from inventory (e.g., nethvoice1, webtop3)';
COMMENT ON COLUMN applications.instance_of IS 'Application type: nethvoice, webtop, mail, nextcloud, samba, traefik, etc.';
COMMENT ON COLUMN applications.display_name IS 'Custom display name for UI. Falls back to module_id if NULL';
COMMENT ON COLUMN applications.node_id IS 'Cluster node ID where the application runs (1=leader, 2+=workers)';
COMMENT ON COLUMN applications.node_label IS 'Human-readable node label from inventory (e.g., Leader Node, Worker Node)';
COMMENT ON COLUMN applications.organization_id IS 'Assigned organization Logto ID. NULL means unassigned';
COMMENT ON COLUMN applications.organization_type IS 'Denormalized org type for efficient filtering: owner, distributor, reseller, customer';
COMMENT ON COLUMN applications.status IS 'Application status: unassigned (no org), assigned (has org), error (has issues)';
COMMENT ON COLUMN applications.inventory_data IS 'Module-specific data from facts.modules[] with enriched user_domains from cluster';
COMMENT ON COLUMN applications.backup_data IS 'Backup status information extracted from inventory';
COMMENT ON COLUMN applications.services_data IS 'Services health status extracted from inventory';
COMMENT ON COLUMN applications.is_user_facing IS 'FALSE for system components (traefik, loki, promtail) hidden in UI';
COMMENT ON COLUMN applications.first_seen_at IS 'Timestamp when app first appeared in inventory';
COMMENT ON COLUMN applications.last_inventory_at IS 'Timestamp of last inventory update containing this app';
COMMENT ON COLUMN applications.deleted_at IS 'Soft delete timestamp. Set when app disappears from inventory';

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

-- =============================================================================
-- IMPERSONATION CONSENTS TABLE
-- =============================================================================
-- User consents for allowing impersonation by Owner users
-- Required for GDPR compliance and audit trail

CREATE TABLE IF NOT EXISTS impersonation_consents (
    id VARCHAR(255) PRIMARY KEY,            -- Unique consent ID

    -- User who gave consent
    user_id VARCHAR(255) NOT NULL,          -- FK to users table

    -- Consent validity
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,  -- When consent expires
    max_duration_minutes INTEGER NOT NULL DEFAULT 60,  -- Max impersonation session duration

    -- Status
    active BOOLEAN NOT NULL DEFAULT TRUE,   -- Whether consent is currently active

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table documentation
COMMENT ON TABLE impersonation_consents IS 'User consents for allowing impersonation by Owner users';
COMMENT ON COLUMN impersonation_consents.user_id IS 'User who granted consent for impersonation';
COMMENT ON COLUMN impersonation_consents.expires_at IS 'Timestamp when consent expires and must be renewed';
COMMENT ON COLUMN impersonation_consents.max_duration_minutes IS 'Maximum duration of impersonation session in minutes';
COMMENT ON COLUMN impersonation_consents.active IS 'Whether consent is currently active (can be revoked)';

-- Foreign key constraint
ALTER TABLE impersonation_consents
ADD CONSTRAINT impersonation_consents_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_user_id ON impersonation_consents(user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_active ON impersonation_consents(active);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_expires_at ON impersonation_consents(expires_at);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_user_active ON impersonation_consents(user_id, active);

-- =============================================================================
-- IMPERSONATION AUDIT TABLE
-- =============================================================================
-- Audit log of all impersonation activities for compliance and security

CREATE TABLE IF NOT EXISTS impersonation_audit (
    id VARCHAR(255) PRIMARY KEY,            -- Unique audit record ID

    -- Session identification
    session_id VARCHAR(255) NOT NULL,       -- Impersonation session ID

    -- Actors
    impersonator_user_id VARCHAR(255) NOT NULL,   -- Owner user doing the impersonation
    impersonator_username VARCHAR(255) NOT NULL,  -- Username for display
    impersonator_name TEXT,                       -- Display name for display
    impersonated_user_id VARCHAR(255) NOT NULL,   -- User being impersonated
    impersonated_username VARCHAR(255) NOT NULL,  -- Username for display
    impersonated_name TEXT,                       -- Display name for display

    -- Action details
    action_type VARCHAR(50) NOT NULL,       -- start, end, api_call, error
    api_endpoint VARCHAR(255),              -- API endpoint accessed (for api_call actions)
    http_method VARCHAR(10),                -- HTTP method used
    request_data TEXT,                      -- Request body (sanitized)
    response_status INTEGER,                -- HTTP response status code
    response_status_text VARCHAR(50),       -- HTTP status text

    -- Timestamps
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table documentation
COMMENT ON TABLE impersonation_audit IS 'Audit log of all impersonation activities for compliance';
COMMENT ON COLUMN impersonation_audit.session_id IS 'Unique impersonation session ID for grouping related actions';
COMMENT ON COLUMN impersonation_audit.action_type IS 'Action type: start, end, api_call, error';
COMMENT ON COLUMN impersonation_audit.api_endpoint IS 'API endpoint accessed during impersonation';
COMMENT ON COLUMN impersonation_audit.request_data IS 'Sanitized request body (sensitive data redacted)';

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_session_id ON impersonation_audit(session_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonator ON impersonation_audit(impersonator_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonated ON impersonation_audit(impersonated_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_action_type ON impersonation_audit(action_type);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonator_name ON impersonation_audit(impersonator_name);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonated_name ON impersonation_audit(impersonated_name);

-- =============================================================================
-- INVENTORY RECORDS TABLE
-- =============================================================================
-- Raw inventory snapshots from systems (collected by collect service)
-- Used for diff calculation and historical analysis

CREATE TABLE IF NOT EXISTS inventory_records (
    id BIGSERIAL PRIMARY KEY,               -- Auto-incrementing record ID

    -- System identification
    system_id VARCHAR(255) NOT NULL,        -- System that sent this inventory

    -- Inventory data
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,  -- When inventory was collected on system
    data JSONB NOT NULL,                    -- Complete raw inventory JSON
    data_hash VARCHAR(64) NOT NULL,         -- SHA-256 hash for deduplication
    data_size BIGINT NOT NULL,              -- Size in bytes

    -- Processing status
    processed_at TIMESTAMP WITH TIME ZONE,  -- When diff processing completed
    has_changes BOOLEAN NOT NULL DEFAULT FALSE,  -- Whether changes were detected vs previous
    change_count INTEGER NOT NULL DEFAULT 0,     -- Number of significant changes

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table documentation
COMMENT ON TABLE inventory_records IS 'Raw inventory snapshots from systems for diff calculation';
COMMENT ON COLUMN inventory_records.data IS 'Complete raw inventory JSON from system';
COMMENT ON COLUMN inventory_records.data_hash IS 'SHA-256 hash of data for deduplication';
COMMENT ON COLUMN inventory_records.processed_at IS 'Timestamp when diff processing completed';
COMMENT ON COLUMN inventory_records.has_changes IS 'TRUE if changes detected vs previous inventory';
COMMENT ON COLUMN inventory_records.change_count IS 'Number of significant changes detected';

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_timestamp ON inventory_records(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_records_data_hash ON inventory_records(data_hash);
CREATE INDEX IF NOT EXISTS idx_inventory_records_processed_at ON inventory_records(processed_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inventory_records_system_data_hash ON inventory_records(system_id, data_hash);

-- =============================================================================
-- INVENTORY DIFFS TABLE
-- =============================================================================
-- Computed differences between inventory snapshots
-- Categorized by type (os, hardware, network, etc.) with severity levels

CREATE TABLE IF NOT EXISTS inventory_diffs (
    id BIGSERIAL PRIMARY KEY,               -- Auto-incrementing diff ID

    -- References
    system_id VARCHAR(255) NOT NULL,        -- System this diff belongs to
    previous_id BIGINT,                     -- FK to inventory_records (previous snapshot, NULL for first)
    current_id BIGINT NOT NULL,             -- FK to inventory_records (current snapshot)

    -- Change classification
    diff_type VARCHAR(20) NOT NULL,         -- create, update, delete
    category VARCHAR(100),                  -- os, hardware, network, features, security, performance, system, nodes, modules
    severity VARCHAR(20) NOT NULL DEFAULT 'medium',  -- low, medium, high, critical

    -- Change data
    field_path VARCHAR(500),                -- JSON path of the changed field (e.g., facts.nodes.1.version)
    previous_value JSONB,                   -- Previous value (NULL for create)
    current_value JSONB,                    -- New value (NULL for delete)

    -- Notification tracking
    notification_sent BOOLEAN NOT NULL DEFAULT false,  -- Whether notification was sent for this diff

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table documentation
COMMENT ON TABLE inventory_diffs IS 'Computed differences between inventory snapshots';
COMMENT ON COLUMN inventory_diffs.previous_id IS 'Reference to previous inventory record (NULL for first inventory)';
COMMENT ON COLUMN inventory_diffs.current_id IS 'Reference to current inventory record';
COMMENT ON COLUMN inventory_diffs.diff_type IS 'Type of change: create, update, delete';
COMMENT ON COLUMN inventory_diffs.category IS 'Change category: os, hardware, network, features, security, performance, system, nodes, modules';
COMMENT ON COLUMN inventory_diffs.severity IS 'Change severity: low, medium, high, critical';
COMMENT ON COLUMN inventory_diffs.field_path IS 'JSON path of the changed field (e.g., facts.nodes.1.version)';
COMMENT ON COLUMN inventory_diffs.notification_sent IS 'Whether notification was sent for this diff';

-- Diff type validation
ALTER TABLE inventory_diffs ADD CONSTRAINT chk_inventory_diffs_diff_type
    CHECK (diff_type IN ('create', 'update', 'delete'));

-- Severity validation
ALTER TABLE inventory_diffs ADD CONSTRAINT chk_inventory_diffs_severity
    CHECK (severity IN ('low', 'medium', 'high', 'critical'));

-- Foreign key constraints
ALTER TABLE inventory_diffs
ADD CONSTRAINT inventory_diffs_previous_id_fkey
FOREIGN KEY (previous_id) REFERENCES inventory_records(id) ON DELETE CASCADE;

ALTER TABLE inventory_diffs
ADD CONSTRAINT inventory_diffs_current_id_fkey
FOREIGN KEY (current_id) REFERENCES inventory_records(id) ON DELETE CASCADE;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_system_id ON inventory_diffs(system_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_previous_id ON inventory_diffs(previous_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_current_id ON inventory_diffs(current_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_category ON inventory_diffs(category);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_diff_type ON inventory_diffs(diff_type);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_severity ON inventory_diffs(severity);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_created_at ON inventory_diffs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_notification_sent ON inventory_diffs(notification_sent) WHERE notification_sent = false;

-- =============================================================================
-- SYSTEM HEARTBEATS TABLE
-- =============================================================================
-- Tracks system liveness via heartbeat pings
-- Used by collect service to determine online/offline status

CREATE TABLE IF NOT EXISTS system_heartbeats (
    id BIGSERIAL PRIMARY KEY,               -- Auto-incrementing ID

    -- System identification
    system_id VARCHAR(255) NOT NULL UNIQUE, -- FK to systems (one heartbeat record per system)

    -- Heartbeat data
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL,  -- Last heartbeat timestamp
    status VARCHAR(20) NOT NULL DEFAULT 'online',       -- online, offline (based on heartbeat freshness)
    metadata JSONB,                         -- Additional heartbeat metadata

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table documentation
COMMENT ON TABLE system_heartbeats IS 'Tracks system liveness via heartbeat pings';
COMMENT ON COLUMN system_heartbeats.last_heartbeat IS 'Timestamp of last heartbeat received';
COMMENT ON COLUMN system_heartbeats.status IS 'Current status based on heartbeat: online, offline';
COMMENT ON COLUMN system_heartbeats.metadata IS 'Additional metadata sent with heartbeat';

-- Foreign key constraint
ALTER TABLE system_heartbeats
ADD CONSTRAINT system_heartbeats_system_id_fkey
FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_system_heartbeats_system_id ON system_heartbeats(system_id);
CREATE INDEX IF NOT EXISTS idx_system_heartbeats_last_heartbeat ON system_heartbeats(last_heartbeat DESC);
CREATE INDEX IF NOT EXISTS idx_system_heartbeats_status ON system_heartbeats(status);

-- =============================================================================
-- INVENTORY ALERTS TABLE
-- =============================================================================
-- Alerts generated from inventory changes
-- Used for notifications and monitoring

CREATE TABLE IF NOT EXISTS inventory_alerts (
    id BIGSERIAL PRIMARY KEY,               -- Auto-incrementing alert ID

    -- References
    system_id VARCHAR(255) NOT NULL,        -- System this alert is for
    diff_id BIGINT,                         -- FK to inventory_diffs (optional)

    -- Alert details
    alert_type VARCHAR(50) NOT NULL,        -- Type of alert
    message TEXT NOT NULL,                  -- Human-readable alert message
    severity VARCHAR(50) NOT NULL,          -- critical, high, medium, low

    -- Resolution status
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table documentation
COMMENT ON TABLE inventory_alerts IS 'Alerts generated from inventory changes';
COMMENT ON COLUMN inventory_alerts.alert_type IS 'Type of alert (e.g., security_change, version_update)';
COMMENT ON COLUMN inventory_alerts.severity IS 'Alert severity: critical, high, medium, low';
COMMENT ON COLUMN inventory_alerts.is_resolved IS 'Whether alert has been acknowledged/resolved';

-- Foreign key constraints
ALTER TABLE inventory_alerts
ADD CONSTRAINT inventory_alerts_system_id_fkey
FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;

ALTER TABLE inventory_alerts
ADD CONSTRAINT inventory_alerts_diff_id_fkey
FOREIGN KEY (diff_id) REFERENCES inventory_diffs(id) ON DELETE SET NULL;

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_system_id_created_at ON inventory_alerts(system_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_severity ON inventory_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_resolved ON inventory_alerts(is_resolved) WHERE is_resolved = FALSE;

-- =============================================================================
-- SCHEMA MIGRATIONS TABLE
-- =============================================================================
-- Tracks applied database migrations

CREATE TABLE IF NOT EXISTS schema_migrations (
    migration_number VARCHAR(10) PRIMARY KEY,  -- Migration identifier (001, 002, etc.)
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),  -- When migration was applied
    description TEXT,                          -- Human-readable description
    checksum VARCHAR(64)                       -- Optional checksum for validation
);

-- Table documentation
COMMENT ON TABLE schema_migrations IS 'Tracks applied database migrations for version control';

-- =============================================================================
-- VAT UNIQUENESS CONSTRAINTS
-- =============================================================================
-- Prevents duplicate VAT numbers within same organization type
-- Only distributors and resellers have VAT uniqueness; customers can have duplicates

-- VAT uniqueness function for distributors
CREATE OR REPLACE FUNCTION check_unique_vat_distributors()
RETURNS TRIGGER AS $$
DECLARE
    new_vat TEXT;
BEGIN
    new_vat := TRIM(NEW.custom_data->>'vat');

    IF new_vat IS NULL OR new_vat = '' OR NEW.deleted_at IS NOT NULL THEN
        RETURN NEW;
    END IF;

    -- Check for duplicate VAT in active distributors (excluding self for updates)
    IF EXISTS (
        SELECT 1 FROM distributors
        WHERE TRIM(custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in distributors', new_vat;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- VAT uniqueness function for resellers
CREATE OR REPLACE FUNCTION check_unique_vat_resellers()
RETURNS TRIGGER AS $$
DECLARE
    new_vat TEXT;
BEGIN
    new_vat := TRIM(NEW.custom_data->>'vat');

    IF new_vat IS NULL OR new_vat = '' OR NEW.deleted_at IS NOT NULL THEN
        RETURN NEW;
    END IF;

    -- Check for duplicate VAT in active resellers (excluding self for updates)
    IF EXISTS (
        SELECT 1 FROM resellers
        WHERE TRIM(custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in resellers', new_vat;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- VAT function for customers (no uniqueness constraint)
CREATE OR REPLACE FUNCTION check_unique_vat_customers()
RETURNS TRIGGER AS $$
BEGIN
    -- No VAT uniqueness constraint for customers
    -- VAT is optional and can be duplicate
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers
DROP TRIGGER IF EXISTS trg_check_vat_distributors ON distributors;
CREATE TRIGGER trg_check_vat_distributors
BEFORE INSERT OR UPDATE ON distributors
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_distributors();

DROP TRIGGER IF EXISTS trg_check_vat_resellers ON resellers;
CREATE TRIGGER trg_check_vat_resellers
BEFORE INSERT OR UPDATE ON resellers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_resellers();

DROP TRIGGER IF EXISTS trg_check_vat_customers ON customers;
CREATE TRIGGER trg_check_vat_customers
BEFORE INSERT OR UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_customers();
