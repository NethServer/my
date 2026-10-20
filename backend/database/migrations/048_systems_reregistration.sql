-- Migration 048: track repeated registrations of an already-registered system.

ALTER TABLE systems ADD COLUMN IF NOT EXISTS reregistration_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE systems ADD COLUMN IF NOT EXISTS last_reregistered_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN systems.reregistration_count IS 'Times the system registered again with its valid secret after the first registration (e.g. reinstall + restore)';
COMMENT ON COLUMN systems.last_reregistered_at IS 'Timestamp of the latest repeated registration. NULL = never re-registered';
