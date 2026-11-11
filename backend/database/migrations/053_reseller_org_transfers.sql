-- Migration 053: reseller "managed by" transfers — audit log.

CREATE TABLE IF NOT EXISTS reseller_org_transfers (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reseller_id           VARCHAR(255) NOT NULL,
    from_org_id           VARCHAR(255) NOT NULL,
    to_org_id             VARCHAR(255) NOT NULL,
    actor_user_id         VARCHAR(255),
    actor_organization_id VARCHAR(255),
    actor_ip              VARCHAR(64),
    user_agent            TEXT,
    customers_count       INTEGER NOT NULL DEFAULT 0,
    systems_count         INTEGER NOT NULL DEFAULT 0,
    renewals_stopped      INTEGER NOT NULL DEFAULT 0,
    assignments_released  INTEGER NOT NULL DEFAULT 0,
    occurred_at           TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE reseller_org_transfers IS 'Append-only audit log of reseller "managed by" transfers (custom_data.createdBy changes)';
COMMENT ON COLUMN reseller_org_transfers.reseller_id IS 'Logto org id of the transferred reseller';
COMMENT ON COLUMN reseller_org_transfers.from_org_id IS 'Managing organization (distributor or Owner) the reseller left';
COMMENT ON COLUMN reseller_org_transfers.to_org_id IS 'Distributor the reseller moved to';
COMMENT ON COLUMN reseller_org_transfers.customers_count IS 'Customers of the reseller, which followed it';
COMMENT ON COLUMN reseller_org_transfers.systems_count IS 'Systems of the reseller and of its customers, which followed it';
COMMENT ON COLUMN reseller_org_transfers.renewals_stopped IS 'Add-on grants whose shop renewal was stopped because the buyer no longer manages the reseller';

CREATE INDEX IF NOT EXISTS idx_reseller_org_transfers_reseller_id_occurred_at
  ON reseller_org_transfers(reseller_id, occurred_at DESC);
