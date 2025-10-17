-- New Database Schema - Local-first approach with separate entity tables
-- This file replaces the old schema with a cleaner, performance-oriented structure

-- Distributors table - local mirror of distributor organizations
CREATE TABLE IF NOT EXISTS distributors (
    id VARCHAR(255) PRIMARY KEY,
    logto_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    custom_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    logto_synced_at TIMESTAMP WITH TIME ZONE,
    logto_sync_error TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE  -- Soft delete timestamp (NULL = active, non-NULL = deleted)
);

-- Comment for distributors.deleted_at
COMMENT ON COLUMN distributors.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';

-- Performance indexes for distributors
CREATE UNIQUE INDEX IF NOT EXISTS idx_distributors_logto_id ON distributors(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_distributors_deleted_at ON distributors(deleted_at);
CREATE INDEX IF NOT EXISTS idx_distributors_logto_synced ON distributors(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_distributors_created_at ON distributors(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_distributors_name ON distributors(name);
CREATE INDEX IF NOT EXISTS idx_distributors_vat_jsonb ON distributors((custom_data->>'vat'));

-- Resellers table - local mirror of reseller organizations
CREATE TABLE IF NOT EXISTS resellers (
    id VARCHAR(255) PRIMARY KEY,
    logto_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    custom_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    logto_synced_at TIMESTAMP WITH TIME ZONE,
    logto_sync_error TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE  -- Soft delete timestamp (NULL = active, non-NULL = deleted)
);

-- Comment for resellers.deleted_at
COMMENT ON COLUMN resellers.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';

-- Performance indexes for resellers
CREATE UNIQUE INDEX IF NOT EXISTS idx_resellers_logto_id ON resellers(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resellers_deleted_at ON resellers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_resellers_logto_synced ON resellers(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_resellers_created_at ON resellers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_resellers_name ON resellers(name);
CREATE INDEX IF NOT EXISTS idx_resellers_vat_jsonb ON resellers((custom_data->>'vat'));

-- Customers table - local mirror of customer organizations
CREATE TABLE IF NOT EXISTS customers (
    id VARCHAR(255) PRIMARY KEY,
    logto_id VARCHAR(255),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    custom_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    logto_synced_at TIMESTAMP WITH TIME ZONE,
    logto_sync_error TEXT,
    deleted_at TIMESTAMP WITH TIME ZONE  -- Soft delete timestamp (NULL = active, non-NULL = deleted)
);

-- Comment for customers.deleted_at
COMMENT ON COLUMN customers.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';

-- Performance indexes for customers
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_logto_id ON customers(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON customers(deleted_at);
CREATE INDEX IF NOT EXISTS idx_customers_logto_synced ON customers(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);
CREATE INDEX IF NOT EXISTS idx_customers_vat_jsonb ON customers((custom_data->>'vat'));

-- Users table - local mirror with organization membership (Approach 2)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    logto_id VARCHAR(255),
    username VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    phone VARCHAR(20),

    -- Organization membership (1 user = 1 organization)
    organization_id VARCHAR(255),
    user_role_ids JSONB DEFAULT '[]',      -- Technical role IDs (e.g., ['role1', 'role2'])
    custom_data JSONB,

    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    logto_synced_at TIMESTAMP WITH TIME ZONE,
    latest_login_at TIMESTAMP WITH TIME ZONE,
    deleted_at TIMESTAMP WITH TIME ZONE,        -- Soft delete timestamp (NULL = not deleted)
    suspended_at TIMESTAMP WITH TIME ZONE       -- Suspension timestamp (NULL = not suspended)
);

-- Performance indexes for users
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_logto_id ON users(logto_id) WHERE logto_id IS NOT NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);
CREATE INDEX IF NOT EXISTS idx_users_suspended_at ON users(suspended_at);
CREATE INDEX IF NOT EXISTS idx_users_logto_synced ON users(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_latest_login_at ON users(latest_login_at DESC);

-- Systems table - access control based on organization_id
CREATE TABLE IF NOT EXISTS systems (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100),  -- Populated by collect service on first inventory
    status VARCHAR(50) NOT NULL DEFAULT 'unknown',  -- Default: unknown, updated by collect service
    fqdn VARCHAR(255),
    ipv4_address INET,
    ipv6_address INET,
    version VARCHAR(100),
    organization_id VARCHAR(255) NOT NULL,
    custom_data JSONB,
    system_key VARCHAR(255) UNIQUE NOT NULL,
    system_secret VARCHAR(64) NOT NULL,
    notes TEXT DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,  -- Soft delete timestamp (NULL = active, non-NULL = deleted)
    created_by JSONB NOT NULL
);

-- Comment for systems.deleted_at
COMMENT ON COLUMN systems.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';

-- Comment for systems.notes
COMMENT ON COLUMN systems.notes IS 'Additional notes or description for the system';

-- Performance indexes for systems
CREATE INDEX IF NOT EXISTS idx_systems_organization_id ON systems(organization_id);
CREATE INDEX IF NOT EXISTS idx_systems_created_by_org ON systems((created_by->>'organization_id'));
CREATE INDEX IF NOT EXISTS idx_systems_status ON systems(status);
CREATE INDEX IF NOT EXISTS idx_systems_type ON systems(type);
CREATE INDEX IF NOT EXISTS idx_systems_deleted_at ON systems(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_systems_system_key ON systems(system_key);
CREATE INDEX IF NOT EXISTS idx_systems_system_secret ON systems(system_secret);
CREATE INDEX IF NOT EXISTS idx_systems_fqdn ON systems(fqdn);
CREATE INDEX IF NOT EXISTS idx_systems_ipv4_address ON systems(ipv4_address);
CREATE INDEX IF NOT EXISTS idx_systems_ipv6_address ON systems(ipv6_address);

-- System status validation
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_systems_status') THEN
        ALTER TABLE systems ADD CONSTRAINT chk_systems_status
            CHECK (status IN ('unknown', 'online', 'offline', 'deleted'));
    END IF;
END $$;

-- VAT uniqueness constraint per organization role
-- This prevents the same VAT from being used within the same organization type
-- Only applies to active records (deleted_at IS NULL)

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

    -- Check in distributors only, excluding same id (for updates)
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

    -- Check in resellers only, excluding same id (for updates)
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

-- VAT uniqueness function for customers (no uniqueness constraint)
CREATE OR REPLACE FUNCTION check_unique_vat_customers()
RETURNS TRIGGER AS $$
BEGIN
    -- No VAT uniqueness constraint for customers
    -- VAT is optional for customers and can be duplicate
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Distributors
DROP TRIGGER IF EXISTS trg_check_vat_distributors ON distributors;
CREATE TRIGGER trg_check_vat_distributors
BEFORE INSERT OR UPDATE ON distributors
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_distributors();

-- Resellers
DROP TRIGGER IF EXISTS trg_check_vat_resellers ON resellers;
CREATE TRIGGER trg_check_vat_resellers
BEFORE INSERT OR UPDATE ON resellers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_resellers();

-- Customers
DROP TRIGGER IF EXISTS trg_check_vat_customers ON customers;
CREATE TRIGGER trg_check_vat_customers
BEFORE INSERT OR UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_customers();

-- Impersonation consents table
CREATE TABLE IF NOT EXISTS impersonation_consents (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    max_duration_minutes INTEGER NOT NULL DEFAULT 60,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Foreign key constraint for impersonation_consents
ALTER TABLE impersonation_consents
ADD CONSTRAINT impersonation_consents_user_id_fkey
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- Indexes for impersonation_consents
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_user_id ON impersonation_consents(user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_active ON impersonation_consents(active);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_expires_at ON impersonation_consents(expires_at);
CREATE INDEX IF NOT EXISTS idx_impersonation_consents_user_active ON impersonation_consents(user_id, active);

-- Impersonation audit table
CREATE TABLE IF NOT EXISTS impersonation_audit (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL,
    impersonator_user_id VARCHAR(255) NOT NULL,
    impersonated_user_id VARCHAR(255) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    api_endpoint VARCHAR(255),
    http_method VARCHAR(10),
    request_data TEXT,
    response_status INTEGER,
    response_status_text VARCHAR(50),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    impersonator_username VARCHAR(255) NOT NULL,
    impersonated_username VARCHAR(255) NOT NULL,
    impersonator_name TEXT,
    impersonated_name TEXT
);

-- Indexes for impersonation_audit
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_session_id ON impersonation_audit(session_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonator ON impersonation_audit(impersonator_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonated ON impersonation_audit(impersonated_user_id);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_action_type ON impersonation_audit(action_type);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonator_name ON impersonation_audit(impersonator_name);
CREATE INDEX IF NOT EXISTS idx_impersonation_audit_impersonated_name ON impersonation_audit(impersonated_name);

-- Inventory records table
CREATE TABLE IF NOT EXISTS inventory_records (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    data JSONB NOT NULL,
    data_hash VARCHAR(64) NOT NULL,
    data_size BIGINT NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE,
    has_changes BOOLEAN NOT NULL DEFAULT FALSE,
    change_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for inventory_records
CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_timestamp ON inventory_records(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_records_data_hash ON inventory_records(data_hash);
CREATE INDEX IF NOT EXISTS idx_inventory_records_processed_at ON inventory_records(processed_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_inventory_records_system_data_hash ON inventory_records(system_id, data_hash);

-- Inventory diffs table
CREATE TABLE IF NOT EXISTS inventory_diffs (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    record_id BIGINT NOT NULL,
    category VARCHAR(100) NOT NULL,
    subcategory VARCHAR(100),
    change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('added', 'removed', 'modified')),
    old_value JSONB,
    new_value JSONB,
    path VARCHAR(500),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Foreign key constraint for inventory_diffs
ALTER TABLE inventory_diffs
ADD CONSTRAINT inventory_diffs_record_id_fkey
FOREIGN KEY (record_id) REFERENCES inventory_records(id) ON DELETE CASCADE;

-- Indexes for inventory_diffs
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_system_id ON inventory_diffs(system_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_record_id ON inventory_diffs(record_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_category ON inventory_diffs(category);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_change_type ON inventory_diffs(change_type);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_timestamp ON inventory_diffs(timestamp DESC);

-- System heartbeats table
CREATE TABLE IF NOT EXISTS system_heartbeats (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL UNIQUE,
    last_heartbeat TIMESTAMP WITH TIME ZONE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'online',
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Foreign key constraint for system_heartbeats
ALTER TABLE system_heartbeats
ADD CONSTRAINT system_heartbeats_system_id_fkey
FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;

-- Indexes for system_heartbeats
CREATE INDEX IF NOT EXISTS idx_system_heartbeats_system_id ON system_heartbeats(system_id);
CREATE INDEX IF NOT EXISTS idx_system_heartbeats_last_heartbeat ON system_heartbeats(last_heartbeat DESC);
CREATE INDEX IF NOT EXISTS idx_system_heartbeats_status ON system_heartbeats(status);

-- Inventory alerts table
CREATE TABLE IF NOT EXISTS inventory_alerts (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    diff_id BIGINT,
    alert_type VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL,
    is_resolved BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Foreign key constraints for inventory_alerts
ALTER TABLE inventory_alerts
ADD CONSTRAINT inventory_alerts_system_id_fkey
FOREIGN KEY (system_id) REFERENCES systems(id) ON DELETE CASCADE;

ALTER TABLE inventory_alerts
ADD CONSTRAINT inventory_alerts_diff_id_fkey
FOREIGN KEY (diff_id) REFERENCES inventory_diffs(id) ON DELETE SET NULL;

-- Indexes for inventory_alerts
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_system_id_created_at ON inventory_alerts(system_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_severity ON inventory_alerts(severity);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_resolved ON inventory_alerts(is_resolved) WHERE is_resolved = FALSE;

-- Schema migrations table
CREATE TABLE IF NOT EXISTS schema_migrations (
    migration_number VARCHAR(10) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT,
    checksum VARCHAR(64)
);
