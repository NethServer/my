-- Rollback 014: Re-add Argon2id system_secret column
ALTER TABLE systems ADD COLUMN IF NOT EXISTS system_secret VARCHAR(512);
CREATE INDEX IF NOT EXISTS idx_systems_system_secret ON systems(system_secret);
