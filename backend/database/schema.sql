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
    active BOOLEAN DEFAULT TRUE
);

-- Performance indexes for distributors
CREATE UNIQUE INDEX IF NOT EXISTS idx_distributors_logto_id ON distributors(logto_id) WHERE logto_id IS NOT NULL AND active = true;
CREATE INDEX IF NOT EXISTS idx_distributors_active ON distributors(active);
CREATE INDEX IF NOT EXISTS idx_distributors_logto_synced ON distributors(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_distributors_created_at ON distributors(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_distributors_name ON distributors(name);
CREATE INDEX IF NOT EXISTS idx_distributors_created_by_jsonb ON distributors((custom_data->>'createdBy'));
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
    active BOOLEAN DEFAULT TRUE
);

-- Performance indexes for resellers
CREATE UNIQUE INDEX IF NOT EXISTS idx_resellers_logto_id ON resellers(logto_id) WHERE logto_id IS NOT NULL AND active = true;
CREATE INDEX IF NOT EXISTS idx_resellers_active ON resellers(active);
CREATE INDEX IF NOT EXISTS idx_resellers_logto_synced ON resellers(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_resellers_created_at ON resellers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_resellers_name ON resellers(name);
CREATE INDEX IF NOT EXISTS idx_resellers_created_by_jsonb ON resellers((custom_data->>'createdBy'));
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
    active BOOLEAN DEFAULT TRUE
);

-- Performance indexes for customers
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_logto_id ON customers(logto_id) WHERE logto_id IS NOT NULL AND active = true;
CREATE INDEX IF NOT EXISTS idx_customers_active ON customers(active);
CREATE INDEX IF NOT EXISTS idx_customers_logto_synced ON customers(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_customers_name ON customers(name);
CREATE INDEX IF NOT EXISTS idx_customers_created_by_jsonb ON customers((custom_data->>'createdBy'));
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
    active BOOLEAN DEFAULT TRUE
);

-- Performance indexes for users
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_logto_id ON users(logto_id) WHERE logto_id IS NOT NULL AND active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username ON users(username) WHERE active = true;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users(email) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_users_organization_id ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);
CREATE INDEX IF NOT EXISTS idx_users_logto_synced ON users(logto_synced_at);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

-- Systems table - updated to reference customers table
CREATE TABLE IF NOT EXISTS systems (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    fqdn VARCHAR(255),
    ipv4_address INET,
    ipv6_address INET,
    version VARCHAR(100),
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    custom_data JSONB,
    reseller_id VARCHAR(255) NOT NULL,
    secret_hash VARCHAR(64) NOT NULL,
    secret_hint VARCHAR(8),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by JSONB NOT NULL,

    CONSTRAINT fk_systems_reseller FOREIGN KEY (reseller_id) REFERENCES resellers(id) ON DELETE CASCADE
);

-- Performance indexes for systems
CREATE INDEX IF NOT EXISTS idx_systems_reseller_id ON systems(reseller_id);
CREATE INDEX IF NOT EXISTS idx_systems_status ON systems(status);
CREATE INDEX IF NOT EXISTS idx_systems_type ON systems(type);
CREATE INDEX IF NOT EXISTS idx_systems_last_seen ON systems(last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_systems_secret_hash ON systems(secret_hash);
CREATE INDEX IF NOT EXISTS idx_systems_fqdn ON systems(fqdn);
CREATE INDEX IF NOT EXISTS idx_systems_ipv4_address ON systems(ipv4_address);
CREATE INDEX IF NOT EXISTS idx_systems_ipv6_address ON systems(ipv6_address);

-- System status validation
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_systems_status') THEN
        ALTER TABLE systems ADD CONSTRAINT chk_systems_status
            CHECK (status IN ('online', 'offline', 'maintenance', 'error'));
    END IF;
END $$;

-- Organization VAT uniqueness constraints within the same creator
-- This prevents the same creator from creating multiple organizations with the same VAT
-- but allows different creators to use the same VAT
-- Using unique indexes instead of constraints for JSONB expressions

-- Distributors: unique (vat, createdBy) for active records only
CREATE UNIQUE INDEX IF NOT EXISTS uk_distributors_vat_created_by
    ON distributors ((custom_data->>'vat'), (custom_data->>'createdBy')) WHERE active = true;

-- Resellers: unique (vat, createdBy) for active records only
CREATE UNIQUE INDEX IF NOT EXISTS uk_resellers_vat_created_by
    ON resellers ((custom_data->>'vat'), (custom_data->>'createdBy')) WHERE active = true;

-- Customers: unique (vat, createdBy) for active records only
CREATE UNIQUE INDEX IF NOT EXISTS uk_customers_vat_created_by
    ON customers ((custom_data->>'vat'), (custom_data->>'createdBy')) WHERE active = true;
