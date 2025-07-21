-- Systems and System Credentials Tables
-- This file should be executed when initializing the database

-- Systems table - stores all managed systems
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
    customer_id VARCHAR(255) NOT NULL,
    secret_hash VARCHAR(64) NOT NULL,
    secret_hint VARCHAR(8),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by JSONB NOT NULL
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_systems_customer_id ON systems(customer_id);
CREATE INDEX IF NOT EXISTS idx_systems_status ON systems(status);
CREATE INDEX IF NOT EXISTS idx_systems_type ON systems(type);
CREATE INDEX IF NOT EXISTS idx_systems_last_seen ON systems(last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_systems_secret_hash ON systems(secret_hash);
CREATE INDEX IF NOT EXISTS idx_systems_fqdn ON systems(fqdn);
CREATE INDEX IF NOT EXISTS idx_systems_ipv4_address ON systems(ipv4_address);
CREATE INDEX IF NOT EXISTS idx_systems_ipv6_address ON systems(ipv6_address);

-- Constraints (only add if they don't already exist)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_systems_status') THEN
        ALTER TABLE systems ADD CONSTRAINT chk_systems_status
            CHECK (status IN ('online', 'offline', 'maintenance', 'error'));
    END IF;
END $$;

-- System type validation is handled in application code using SYSTEM_TYPES environment variable

-- Organization hierarchy table - caches accessible customer IDs for RBAC optimization
CREATE TABLE IF NOT EXISTS organization_hierarchy (
    user_org_id VARCHAR(255) NOT NULL,
    user_org_role VARCHAR(50) NOT NULL,
    accessible_customer_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_org_id, user_org_role, accessible_customer_id)
);

-- Indexes for optimal RBAC query performance
CREATE INDEX IF NOT EXISTS idx_org_hierarchy_user_org ON organization_hierarchy(user_org_id, user_org_role);
CREATE INDEX IF NOT EXISTS idx_org_hierarchy_customer ON organization_hierarchy(accessible_customer_id);
CREATE INDEX IF NOT EXISTS idx_org_hierarchy_updated ON organization_hierarchy(updated_at DESC);

-- Constraints
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_org_hierarchy_role') THEN
        ALTER TABLE organization_hierarchy ADD CONSTRAINT chk_org_hierarchy_role
            CHECK (user_org_role IN ('Owner', 'Distributor', 'Reseller', 'Customer'));
    END IF;
END $$;