-- Migration 013: Add SHA256 hash column for system secret verification
-- Replaces Argon2id (~64MB RAM per verification) with SHA256 (negligible memory).
-- Existing systems migrate lazily on next authentication.
-- New systems store only SHA256.

-- Add SHA256 column
ALTER TABLE systems ADD COLUMN IF NOT EXISTS system_secret_sha256 VARCHAR(128);

-- Make argon2id column nullable (new systems won't have it)
ALTER TABLE systems ALTER COLUMN system_secret DROP NOT NULL;

-- Index for fast lookup during authentication
CREATE INDEX IF NOT EXISTS idx_systems_system_secret_sha256 ON systems(system_secret_sha256) WHERE system_secret_sha256 IS NOT NULL;

COMMENT ON COLUMN systems.system_secret_sha256 IS 'SHA256 hash of secret part (hex_salt:hex_hash) for fast verification';
