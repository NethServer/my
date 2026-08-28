-- Migration 042: original file names for rebranding assets, and assets tied
-- to the enablement that justifies them.
--
-- Two gaps the rebranding settings screen runs into.
--
-- The upload form lists the asset it already holds the way a file input does:
-- "logo-light.svg — 348 KB". The size comes from octet_length() and the type
-- from the *_mime columns, but the name the partner chose is nowhere, so the
-- UI can only invent one. A column per asset keeps it.
--
-- And an organization removed from rebranding kept its assets: the row in
-- rebranding_enabled went away, the blobs stayed. Re-adding the organization
-- silently restored branding the owner believed deleted, and up to ~13 MB per
-- organization and product accumulated with no endpoint left to reach it. The
-- foreign key states the real rule — assets exist only while the organization
-- is enabled — and lets the database enforce it on every path, including the
-- ones written later.

ALTER TABLE rebranding_assets
    ADD COLUMN IF NOT EXISTS logo_light_rect_filename VARCHAR(255),
    ADD COLUMN IF NOT EXISTS logo_dark_rect_filename VARCHAR(255),
    ADD COLUMN IF NOT EXISTS logo_light_square_filename VARCHAR(255),
    ADD COLUMN IF NOT EXISTS logo_dark_square_filename VARCHAR(255),
    ADD COLUMN IF NOT EXISTS favicon_filename VARCHAR(255),
    ADD COLUMN IF NOT EXISTS background_image_filename VARCHAR(255);

COMMENT ON COLUMN rebranding_assets.logo_light_rect_filename IS 'Name of the uploaded file, shown back in the upload form';

-- Assets of organizations that are no longer enabled: unreachable from the
-- product and blocked by the foreign key below.
DELETE FROM rebranding_assets ra
WHERE NOT EXISTS (
    SELECT 1 FROM rebranding_enabled re WHERE re.organization_id = ra.organization_id
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_rebranding_assets_enabled'
    ) THEN
        ALTER TABLE rebranding_assets ADD CONSTRAINT fk_rebranding_assets_enabled
            FOREIGN KEY (organization_id) REFERENCES rebranding_enabled(organization_id) ON DELETE CASCADE;
    END IF;
END $$;
