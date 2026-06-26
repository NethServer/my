-- Rollback migration 032: restore the default fillfactor and recreate the index

ALTER TABLE systems RESET (fillfactor);

CREATE INDEX IF NOT EXISTS idx_systems_updated_at ON systems(updated_at);
