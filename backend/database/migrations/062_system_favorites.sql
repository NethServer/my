-- Migration 062: favorite systems
--
-- One row per system a user starred. Keyed on the Logto user ID rather than
-- users(id): the Owner account has no local users row. A destroyed system
-- takes its rows with it; a destroyed user is cleaned up by the backend.

CREATE TABLE IF NOT EXISTS system_favorites (
    user_logto_id VARCHAR(255) NOT NULL,
    system_id     VARCHAR(255) NOT NULL REFERENCES systems(id) ON DELETE CASCADE,
    created_at    TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_logto_id, system_id)
);

CREATE INDEX IF NOT EXISTS idx_system_favorites_system_id ON system_favorites(system_id);

COMMENT ON TABLE  system_favorites               IS 'Systems a user starred as favorites; personal, never shared with other users';
COMMENT ON COLUMN system_favorites.user_logto_id IS 'Logto user ID of the user who starred the system (no FK: the Owner has no users row)';
