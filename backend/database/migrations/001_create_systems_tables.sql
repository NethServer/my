-- Systems and System Credentials Tables
-- This file should be executed when initializing the database

-- Systems table - stores all managed systems
CREATE TABLE IF NOT EXISTS systems (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'offline',
    ip_address INET NOT NULL,
    version VARCHAR(100),
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    metadata JSONB,
    organization_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) NOT NULL
);

-- System credentials table - stores hashed secrets for collect service authentication
-- This table is shared between backend and collect services
CREATE TABLE IF NOT EXISTS system_credentials (
    system_id VARCHAR(255) PRIMARY KEY REFERENCES systems(id) ON DELETE CASCADE,
    secret_hash VARCHAR(64) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_systems_organization_id ON systems(organization_id);
CREATE INDEX IF NOT EXISTS idx_systems_created_by ON systems(created_by);
CREATE INDEX IF NOT EXISTS idx_systems_status ON systems(status);
CREATE INDEX IF NOT EXISTS idx_systems_type ON systems(type);
CREATE INDEX IF NOT EXISTS idx_systems_last_seen ON systems(last_seen DESC);

CREATE INDEX IF NOT EXISTS idx_system_credentials_active ON system_credentials(is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_system_credentials_last_used ON system_credentials(last_used DESC);

-- Constraints
ALTER TABLE systems ADD CONSTRAINT chk_systems_status 
    CHECK (status IN ('online', 'offline', 'maintenance', 'error'));

ALTER TABLE systems ADD CONSTRAINT chk_systems_type 
    CHECK (type IN ('linux', 'windows', 'nethsecurity', 'nethserver', 'other'));