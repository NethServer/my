-- Migration 033: Anchor owner API keys to the Logto ID
--
-- Personal API keys are anchored to a local users row (user_id FK) so the live
-- suspend check can resolve the key's owner. The Nethesis owner super-admin has
-- no local users row by design (users cannot be created in the Owner
-- organization), so it could not hold keys. Owner keys (myo_ prefix) are now
-- anchored to the Logto ID instead:
--
--   - user_id becomes nullable; a row with user_id IS NULL is an owner key
--     and must carry logto_id (enforced by the CHECK below);
--   - regular keys (myk_) keep the NOT NULL-in-practice user_id anchor and are
--     untouched: their auth path still JOINs users, which naturally excludes
--     owner rows;
--   - the owner suspend check moves to the Logto profile (isSuspended),
--     cached briefly; revocation stays instant via revoked_at.

ALTER TABLE user_api_keys ALTER COLUMN user_id DROP NOT NULL;

ALTER TABLE user_api_keys ADD COLUMN IF NOT EXISTS logto_id VARCHAR(255);

ALTER TABLE user_api_keys
    ADD CONSTRAINT user_api_keys_anchor_check
    CHECK (user_id IS NOT NULL OR logto_id IS NOT NULL);

COMMENT ON COLUMN user_api_keys.logto_id IS 'Anchor for owner keys (myo_): Logto user ID; regular keys anchor on user_id';

CREATE INDEX IF NOT EXISTS idx_user_api_keys_logto_id ON user_api_keys(logto_id) WHERE logto_id IS NOT NULL;
