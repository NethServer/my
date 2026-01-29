-- Migration: Add rebranding tables
-- Supports per-product rebranding for organizations with asset storage

-- Rebrandable products registry (seed data for supported products)
CREATE TABLE IF NOT EXISTS rebrandable_products (
    id VARCHAR(100) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE rebrandable_products IS 'Registry of products that support rebranding';
COMMENT ON COLUMN rebrandable_products.id IS 'Product identifier (e.g., nethvoice, webtop, ns8, nsec)';
COMMENT ON COLUMN rebrandable_products.display_name IS 'Default display name for the product';
COMMENT ON COLUMN rebrandable_products.type IS 'Product type: application or system';

-- Type validation
ALTER TABLE rebrandable_products ADD CONSTRAINT chk_rebrandable_products_type
    CHECK (type IN ('application', 'system'));

-- Seed rebrandable products
INSERT INTO rebrandable_products (id, display_name, type) VALUES
    ('nethvoice', 'NethVoice', 'application'),
    ('webtop', 'NethService', 'application'),
    ('ns8', 'NS8', 'system'),
    ('nsec', 'NethSecurity', 'system')
ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, type = EXCLUDED.type;

-- Rebranding enablement per organization (Owner decides)
CREATE TABLE IF NOT EXISTS rebranding_enabled (
    organization_id VARCHAR(255) PRIMARY KEY,
    organization_type VARCHAR(50) NOT NULL,
    enabled_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE rebranding_enabled IS 'Tracks which organizations have rebranding enabled by Owner';
COMMENT ON COLUMN rebranding_enabled.organization_id IS 'Logto organization ID with rebranding enabled';
COMMENT ON COLUMN rebranding_enabled.organization_type IS 'Organization type: distributor, reseller, customer';

-- Organization type validation
ALTER TABLE rebranding_enabled ADD CONSTRAINT chk_rebranding_enabled_org_type
    CHECK (organization_type IN ('distributor', 'reseller', 'customer'));

-- Rebranding assets per organization per product
CREATE TABLE IF NOT EXISTS rebranding_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id VARCHAR(255) NOT NULL,
    product_id VARCHAR(100) NOT NULL REFERENCES rebrandable_products(id) ON DELETE CASCADE,

    -- Custom product name
    product_name VARCHAR(100),

    -- Image assets stored as binary
    logo_light_rect BYTEA,
    logo_dark_rect BYTEA,
    logo_light_square BYTEA,
    logo_dark_square BYTEA,
    favicon BYTEA,
    background_image BYTEA,

    -- MIME types for each asset
    logo_light_rect_mime VARCHAR(50),
    logo_dark_rect_mime VARCHAR(50),
    logo_light_square_mime VARCHAR(50),
    logo_dark_square_mime VARCHAR(50),
    favicon_mime VARCHAR(50),
    background_image_mime VARCHAR(50),

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- One rebranding config per organization per product
    UNIQUE(organization_id, product_id)
);

COMMENT ON TABLE rebranding_assets IS 'Rebranding assets (logos, favicon, background) per organization per product';
COMMENT ON COLUMN rebranding_assets.organization_id IS 'Logto organization ID that owns these assets';
COMMENT ON COLUMN rebranding_assets.product_id IS 'Product being rebranded (FK to rebrandable_products)';
COMMENT ON COLUMN rebranding_assets.product_name IS 'Custom product name (e.g., CustomVoice instead of NethVoice)';

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_rebranding_assets_organization_id ON rebranding_assets(organization_id);
CREATE INDEX IF NOT EXISTS idx_rebranding_assets_product_id ON rebranding_assets(product_id);
CREATE INDEX IF NOT EXISTS idx_rebranding_assets_org_product ON rebranding_assets(organization_id, product_id);
