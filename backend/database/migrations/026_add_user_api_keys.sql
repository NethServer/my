-- Migration 026: Add user_api_keys table
--
-- Personal API keys let a user authenticate non-interactive integrations
-- (CRM, ERP, ...) without the interactive Logto + 2FA login. The key is an
-- opaque token `myk_<public>.<secret>`: only the public part is stored in clear
-- (indexed for O(1) lookup) and the secret part is kept as a salted SHA-256
-- hash, mirroring the systems registration token scheme.
--
-- A key never carries its own permissions. On each request the owner's current
-- effective permissions are resolved live and masked down to the key's mode
-- (read / write), so suspending or down-scoping the owner takes effect at once.

CREATE TABLE IF NOT EXISTS user_api_keys (
    id                VARCHAR(255) PRIMARY KEY,
    user_id           VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    organization_id   VARCHAR(255),
    name              VARCHAR(255) NOT NULL,

    -- Opaque token split: public part for lookup, secret part hashed.
    key_public        VARCHAR(64)  NOT NULL,            -- public part of myk_<public>.<secret>
    key_secret_sha256 VARCHAR(128) NOT NULL,            -- salted SHA-256 of secret part (hex_salt:hex_hash)

    mode              VARCHAR(10)  NOT NULL CHECK (mode IN ('read', 'write')),

    expires_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    last_used_at      TIMESTAMP WITH TIME ZONE,
    last_used_ip      VARCHAR(64),
    revoked_at        TIMESTAMP WITH TIME ZONE,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE user_api_keys IS 'Personal API keys for non-interactive integrations; permissions resolved live and masked to mode';
COMMENT ON COLUMN user_api_keys.key_public IS 'Public part of token myk_<public>.<secret> for fast DB lookup';
COMMENT ON COLUMN user_api_keys.key_secret_sha256 IS 'Salted SHA-256 of the secret part (hex_salt:hex_hash)';
COMMENT ON COLUMN user_api_keys.mode IS 'read = read:* only; write = read:* + manage:* (destroy/impersonate/config excluded)';

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_api_keys_public ON user_api_keys(key_public);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_active ON user_api_keys(user_id) WHERE revoked_at IS NULL;
