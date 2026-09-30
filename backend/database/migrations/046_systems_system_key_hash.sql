-- Migration 046: sha256(system_key:token) learned by collect, Bearer credential for /auth-hash only.

ALTER TABLE systems ADD COLUMN IF NOT EXISTS system_key_hash VARCHAR(64);

CREATE UNIQUE INDEX IF NOT EXISTS idx_systems_system_key_hash ON systems(system_key_hash) WHERE system_key_hash IS NOT NULL;

COMMENT ON COLUMN systems.system_key_hash IS 'sha256(system_key:token), learned by collect; Bearer credential for /auth-hash only';
