-- Rollback migration 042: assets outlive the enablement again, and the upload
-- form loses the file names it displays.
--
-- Dropping the foreign key does not bring back the assets deleted when it was
-- created: those organizations are not enabled and their branding is gone.

ALTER TABLE rebranding_assets DROP CONSTRAINT IF EXISTS fk_rebranding_assets_enabled;

ALTER TABLE rebranding_assets
    DROP COLUMN IF EXISTS logo_light_rect_filename,
    DROP COLUMN IF EXISTS logo_dark_rect_filename,
    DROP COLUMN IF EXISTS logo_light_square_filename,
    DROP COLUMN IF EXISTS logo_dark_square_filename,
    DROP COLUMN IF EXISTS favicon_filename,
    DROP COLUMN IF EXISTS background_image_filename;
