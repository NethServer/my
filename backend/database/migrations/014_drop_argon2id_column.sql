-- Migration 014: Drop Argon2id system_secret column
-- All systems have been migrated to SHA256 (system_secret_sha256 column).

DROP INDEX IF EXISTS idx_systems_system_secret;
ALTER TABLE systems DROP COLUMN IF EXISTS system_secret;
