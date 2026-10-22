-- Migration 051: customer "managed by" transfers — audit log and add-on renewal stop.

CREATE TABLE IF NOT EXISTS customer_org_transfers (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id           VARCHAR(255) NOT NULL,
    from_org_id           VARCHAR(255) NOT NULL,
    to_org_id             VARCHAR(255) NOT NULL,
    actor_user_id         VARCHAR(255),
    actor_organization_id VARCHAR(255),
    actor_ip              VARCHAR(64),
    user_agent            TEXT,
    systems_count         INTEGER NOT NULL DEFAULT 0,
    renewals_stopped      INTEGER NOT NULL DEFAULT 0,
    silences_cleared      INTEGER NOT NULL DEFAULT 0,
    assignments_released  INTEGER NOT NULL DEFAULT 0,
    occurred_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE customer_org_transfers IS 'Append-only audit log of customer "managed by" transfers (custom_data.createdBy changes)';
COMMENT ON COLUMN customer_org_transfers.customer_id IS 'Logto org id of the transferred customer';
COMMENT ON COLUMN customer_org_transfers.from_org_id IS 'Managing organization (reseller/distributor) the customer left';
COMMENT ON COLUMN customer_org_transfers.to_org_id IS 'Managing organization (reseller/distributor) the customer moved to';
COMMENT ON COLUMN customer_org_transfers.renewals_stopped IS 'Add-on grants whose shop renewal was stopped because the buyer no longer manages the customer';

CREATE INDEX IF NOT EXISTS idx_customer_org_transfers_customer_id_occurred_at
  ON customer_org_transfers(customer_id, occurred_at DESC);

ALTER TABLE system_entitlements ADD COLUMN IF NOT EXISTS renewal_stopped_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE system_entitlements ADD COLUMN IF NOT EXISTS renewal_stopped_reason VARCHAR(50);

COMMENT ON COLUMN system_entitlements.renewal_stopped_at IS 'The shop must not renew this grant any more: it stays valid until valid_until. Cleared by an activation from a buyer that manages the system';
COMMENT ON COLUMN system_entitlements.renewal_stopped_reason IS 'Why the renewal stopped: customer_transferred (the buyer organization no longer manages the customer). NULL when renewing normally';
