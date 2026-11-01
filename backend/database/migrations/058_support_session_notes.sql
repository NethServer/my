-- Migration 058: Operator notes on support sessions
-- Analyses, ticket links and chat threads written while working on a system.
-- A note belongs to the session it was written in and is listed per system,
-- so the history outlives the session.

CREATE TABLE IF NOT EXISTS support_session_notes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES support_sessions(id) ON DELETE CASCADE,
    system_id VARCHAR(255) NOT NULL REFERENCES systems(id) ON DELETE CASCADE,
    author_id VARCHAR(255) NOT NULL,
    author_name VARCHAR(255),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT support_session_notes_body_check CHECK (char_length(body) BETWEEN 1 AND 10000)
);

COMMENT ON TABLE support_session_notes IS 'Operator notes written during support sessions';
COMMENT ON COLUMN support_session_notes.author_id IS 'Logto user ID of the operator who wrote the note';

CREATE INDEX IF NOT EXISTS idx_support_session_notes_system ON support_session_notes(system_id, created_at DESC) WHERE deleted_at IS NULL;
