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

-- Systems table - access control based on created_by organization
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
    secret_hash VARCHAR(64) NOT NULL,
    secret_hint VARCHAR(8),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,  -- Soft delete timestamp (NULL = active, non-NULL = deleted)
    created_by JSONB NOT NULL
);

-- Comment for systems.deleted_at
COMMENT ON COLUMN systems.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';

-- Performance indexes for systems
CREATE INDEX IF NOT EXISTS idx_systems_created_by_org ON systems((created_by->>'organization_id'));
CREATE INDEX IF NOT EXISTS idx_systems_status ON systems(status);
CREATE INDEX IF NOT EXISTS idx_systems_type ON systems(type);
CREATE INDEX IF NOT EXISTS idx_systems_last_seen ON systems(last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_systems_deleted_at ON systems(deleted_at);
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
