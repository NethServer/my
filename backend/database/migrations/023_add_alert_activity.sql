-- Migration 023: Add alert_activity table
-- Append-only timeline of operator actions performed on a single alert
-- (silence created/updated/deleted). The UI renders this in the alert-detail
-- drawer ("Activity" section). Per-alert scoped via (organization_id,
-- fingerprint). Operator "notes" are not a separate concept: they are stored
-- as the comment of the silence, so a note edit is recorded here as a
-- silence_updated event whose details payload includes the comment change.

CREATE TABLE IF NOT EXISTS alert_activity (
    id              BIGSERIAL PRIMARY KEY,

    organization_id VARCHAR(255) NOT NULL,
    fingerprint     VARCHAR(255) NOT NULL,

    -- Action identifier. Open-ended so new event types don't require a schema
    -- change; current values: 'silenced', 'silence_updated', 'unsilenced'.
    action          VARCHAR(50)  NOT NULL,

    -- Actor identity (denormalized for cheap render). user_id may be NULL for
    -- system-driven events (none today, kept for future).
    actor_user_id   VARCHAR(255),
    actor_name      VARCHAR(255),

    -- Optional silence reference, set on silence-related actions so the
    -- DELETE handler can resolve the originating fingerprint without a
    -- separate mapping table.
    silence_id      VARCHAR(255),

    -- Free-form structured payload (e.g. comment, end_at, note excerpt).
    details         JSONB        NOT NULL DEFAULT '{}',

    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  alert_activity              IS 'Append-only audit timeline of operator actions on individual alerts';
COMMENT ON COLUMN alert_activity.fingerprint  IS 'Alertmanager fingerprint (hex hash of labels) of the alert the action targets';
COMMENT ON COLUMN alert_activity.action       IS 'Event kind: silenced | silence_updated | unsilenced. Note changes are silence_updated events.';
COMMENT ON COLUMN alert_activity.silence_id   IS 'Silence ID associated with the event. Lets DELETE silence resolve the fingerprint.';

CREATE INDEX IF NOT EXISTS idx_alert_activity_org_fp_created_at ON alert_activity(organization_id, fingerprint, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_activity_silence_lookup    ON alert_activity(organization_id, silence_id) WHERE silence_id IS NOT NULL;
