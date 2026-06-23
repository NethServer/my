-- Migration 027: Add api_key_audit table
--
-- Append-only audit trail for personal API keys. It records lifecycle events
-- (created, revoked) and security-relevant failures (use of a revoked/expired
-- key, use by a suspended/deleted owner, wrong secret on a known key, rate
-- limiting). Successful per-request use is NOT recorded here (that would be
-- high volume); it is reflected by user_api_keys.last_used_at instead.
--
-- No foreign keys: the trail must survive key and user deletion (forensics).
-- Rows are scoped by user_id and organization_id for future read access.

CREATE TABLE IF NOT EXISTS api_key_audit (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    api_key_id      VARCHAR(255),            -- the key involved (null for unattributable events)
    user_id         VARCHAR(255),            -- the key owner
    organization_id VARCHAR(255),            -- owner's organization, for scoping
    event           VARCHAR(32) NOT NULL,    -- created | revoked | auth_failed | rate_limited
    reason          VARCHAR(32),             -- auth_failed detail: revoked | expired | user_inactive | invalid_secret
    key_name        VARCHAR(255),            -- snapshot of the key name at event time
    key_mode        VARCHAR(10),             -- snapshot of read|write
    ip              VARCHAR(64),
    method          VARCHAR(10),
    path            TEXT,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE api_key_audit IS 'Append-only audit of API key lifecycle and security failures; successful use tracked via user_api_keys.last_used_at';

CREATE INDEX IF NOT EXISTS idx_api_key_audit_user ON api_key_audit(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_key_audit_org ON api_key_audit(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_key_audit_key ON api_key_audit(api_key_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_api_key_audit_event ON api_key_audit(event);
