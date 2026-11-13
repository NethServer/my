-- Rollback migration 053: drop the reseller transfers audit log.

DROP INDEX IF EXISTS idx_reseller_org_transfers_reseller_id_occurred_at;
DROP TABLE IF EXISTS reseller_org_transfers;
