-- Migration 018: Add alert_history table
-- Stores resolved/inactive alerts received from Alertmanager webhooks

CREATE TABLE IF NOT EXISTS alert_history (
    id             BIGSERIAL PRIMARY KEY,

    -- System identification
    system_key     VARCHAR(255) NOT NULL,

    -- Alert identity
    alertname      VARCHAR(255) NOT NULL,
    severity       VARCHAR(50),
    status         VARCHAR(50)  NOT NULL,  -- resolved
    fingerprint    VARCHAR(255) NOT NULL,

    -- Timing
    starts_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at        TIMESTAMP WITH TIME ZONE,

    -- Human-readable summary (from annotations.summary)
    summary        TEXT,

    -- Raw labels and annotations from the alert
    labels         JSONB NOT NULL DEFAULT '{}',
    annotations    JSONB NOT NULL DEFAULT '{}',

    -- Alertmanager receiver that handled the alert
    receiver       VARCHAR(255),

    -- Timestamps
    created_at     TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE alert_history IS 'Resolved and inactive alerts received from Alertmanager webhooks';
COMMENT ON COLUMN alert_history.system_key IS 'System key extracted from alert labels.system_key';
COMMENT ON COLUMN alert_history.fingerprint IS 'Alert fingerprint from Alertmanager for deduplication';
COMMENT ON COLUMN alert_history.status IS 'Alert status at time of receipt: resolved';
COMMENT ON COLUMN alert_history.ends_at IS 'NULL when end time is the zero time (0001-01-01)';

CREATE INDEX IF NOT EXISTS idx_alert_history_system_key_created_at ON alert_history(system_key, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alert_history_starts_at ON alert_history(starts_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_alert_history_fingerprint_system_key_starts_at ON alert_history(fingerprint, system_key, starts_at);
