-- Migration 036: Add alert_assignments table
-- Current assignee of an active alert ("who is working on this"), keyed by
-- (organization_id, fingerprint) — one assignee per alert at a time. Self-assign
-- only: the assignee is always the authenticated caller; taking over an alert
-- already assigned to someone else replaces the row (upsert). There is no
-- manual unassign: rows are deleted by collect when the resolved webhook for
-- the fingerprint arrives (auto-release), with a sweep in the cleanup worker
-- covering partial failures. History lives in alert_activity as
-- assigned/unassigned events; this table only answers "who has it now".

CREATE TABLE IF NOT EXISTS alert_assignments (
    organization_id    VARCHAR(255) NOT NULL,
    fingerprint        VARCHAR(255) NOT NULL,

    -- Assignee identity (denormalized name + org for cheap render, same
    -- approach as alert_activity actor columns). The org is the assignee's own
    -- organization (from the JWT), not the alert's: with takeover across the
    -- hierarchy the two differ, and the UI may want to show "in charge:
    -- R1C1User (e2e2-r1c1)".
    assigned_user_id       VARCHAR(255) NOT NULL,
    assigned_user_name     VARCHAR(255),
    assigned_user_org_id   VARCHAR(255),
    assigned_user_org_name VARCHAR(255),

    assigned_at        TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    PRIMARY KEY (organization_id, fingerprint)
);

-- alert_activity grows three new event kinds: assigned / unassigned (assignment
-- lifecycle, unassigned may be system-driven with NULL actor on auto-release)
-- and note_added (free-form operator note in details.text, independent from
-- silences — notes no longer require a silence to exist).
COMMENT ON COLUMN alert_activity.action IS 'Event kind: silenced | silence_updated | unsilenced | assigned | unassigned | note_added. Silence-comment changes are silence_updated events; standalone notes are note_added events.';

COMMENT ON TABLE  alert_assignments                    IS 'Current assignee per active alert; deleted on alert resolution (auto-release)';
COMMENT ON COLUMN alert_assignments.fingerprint        IS 'Alertmanager fingerprint (hex hash of labels) of the assigned alert';
COMMENT ON COLUMN alert_assignments.assigned_user_id   IS 'Logto user id of the assignee (always the authenticated caller: self-assign only)';
COMMENT ON COLUMN alert_assignments.assigned_user_name IS 'Denormalized assignee display name for cheap render';
