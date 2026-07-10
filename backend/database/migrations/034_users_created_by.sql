-- Migration 034: Creator snapshot on users
--
-- Users gain a point-in-time snapshot of who created them (same shape as
-- systems.created_by: user_id, username, name, email, organization_id,
-- organization_name) so the users list can show and filter by "Created by",
-- like systems and organizations already do.
--
-- A dedicated column is used instead of custom_data: the users update path
-- replaces local custom_data wholesale from the request, which would wipe an
-- embedded snapshot, while a column is untouched by those writes.
--
-- Existing rows are left NULL here; the backfill (owner as creator, plus a
-- handful of known exceptions) is applied manually per environment.

ALTER TABLE users ADD COLUMN IF NOT EXISTS created_by JSONB;

COMMENT ON COLUMN users.created_by IS 'Creator snapshot (user_id, username, name, email, organization_id, organization_name), set at creation; display/filter only, not used for RBAC';
