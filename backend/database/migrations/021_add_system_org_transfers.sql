-- Migration 021: Add system_org_transfers audit table
--
-- Records every successful cross-organization reassignment of a system. The
-- row is written synchronously inside UpdateSystem so the audit trail cannot
-- be silenced by a logging-only failure. Three downstream consumers depend
-- on it:
--
--   1. Forensics: a tamper-evident record of who moved what, when, and to
--      where — the only artefact left after a hostile insider triggers a
--      reassignment to exfiltrate backups via the legitimate download API.
--   2. GDPR purge: DestroySystem iterates every from_org_id this system has
--      ever been under and runs DeleteBackupPrefix on each, so a partial
--      cleanup failure during reassignment cannot leak ciphertext past the
--      destroy call.
--   3. UI history: future "this system was previously owned by …" surface
--      for admins.

CREATE TABLE IF NOT EXISTS system_org_transfers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    system_id     VARCHAR(255) NOT NULL,
    system_key    VARCHAR(255) NOT NULL,

    -- The org the system left and the one it landed in. Both are Logto IDs;
    -- both can outlive the orgs themselves (an org can be soft-deleted later)
    -- so neither is a foreign key.
    from_org_id   VARCHAR(255) NOT NULL,
    to_org_id     VARCHAR(255) NOT NULL,

    -- Actor identity. Captured at the moment of the change; stays valid even
    -- if the user is later deleted. user_id may match a row in users(logto_id).
    actor_user_id          VARCHAR(255),
    actor_user_email       VARCHAR(255),
    actor_organization_id  VARCHAR(255),

    -- Source IP and user agent of the request that triggered the transfer.
    actor_ip      VARCHAR(64),
    user_agent    TEXT,

    -- Outcome counters captured at completion. Best-effort steps that fail
    -- are logged separately; counts here reflect what landed in storage.
    backups_copied   INTEGER NOT NULL DEFAULT 0,
    backups_deleted  INTEGER NOT NULL DEFAULT 0,
    silences_cleared INTEGER NOT NULL DEFAULT 0,
    history_rows_reassigned INTEGER NOT NULL DEFAULT 0,
    apps_unassigned  INTEGER NOT NULL DEFAULT 0,

    occurred_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_org_transfers IS 'Append-only audit log of cross-organization system reassignments';
COMMENT ON COLUMN system_org_transfers.from_org_id IS 'Logto organization ID the system was moved from';
COMMENT ON COLUMN system_org_transfers.to_org_id IS 'Logto organization ID the system was moved to';

CREATE INDEX IF NOT EXISTS idx_system_org_transfers_system_id_occurred_at
  ON system_org_transfers(system_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_org_transfers_from_org_id
  ON system_org_transfers(from_org_id);
CREATE INDEX IF NOT EXISTS idx_system_org_transfers_to_org_id
  ON system_org_transfers(to_org_id);
