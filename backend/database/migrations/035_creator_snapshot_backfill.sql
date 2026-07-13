-- Migration 035: Creator snapshot backfill (on_behalf_of flag + stale org names)
--
-- Creator snapshots (custom_data.createdByUser on organizations, the
-- created_by JSONB column on systems and users) now carry an on_behalf_of
-- flag: true when the entity was attributed to a different organization via
-- created_by_organization_id, so the UI can render "created by <user> on
-- behalf of <org>" instead of implying the user belongs to that org.
-- New writes set the flag in code; snapshot organization_name is kept fresh
-- by the org update path from now on. This data-only migration aligns
-- existing rows:
--
--   1. Refreshes organization_name in every snapshot from the current
--      organization names (fixes renames that predate the propagation, e.g.
--      "Nethesis Diretta" -> "Clienti Diretti Nethesis").
--   2. Backfills on_behalf_of = true where the snapshot's organization
--      differs from the creator user's own organization. Users are matched
--      on their active row; snapshots whose creator user was deleted or is
--      unknown are left untouched. (Heuristic caveat: a user moved to
--      another org after creating entities would be flagged too — users are
--      1:1 with their org in this system, so this does not occur in
--      practice.) Applied to resellers, customers and systems only:
--      distributors and users cannot be created via attribution.
--
-- Idempotent: every UPDATE is guarded so re-running is a no-op.

-- ============================================================
-- 1. Refresh stale organization_name in creator snapshots
-- ============================================================

WITH org_names AS (
    SELECT DISTINCT ON (logto_id) logto_id, name
    FROM (
        SELECT logto_id, name, deleted_at FROM distributors WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM resellers WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM customers WHERE logto_id IS NOT NULL
    ) o
    ORDER BY logto_id, (deleted_at IS NOT NULL)
)
UPDATE distributors t
SET custom_data = jsonb_set(t.custom_data, '{createdByUser,organization_name}', to_jsonb(o.name))
FROM org_names o
WHERE t.custom_data->'createdByUser'->>'organization_id' = o.logto_id
  AND t.custom_data->'createdByUser'->>'organization_name' IS DISTINCT FROM o.name;

WITH org_names AS (
    SELECT DISTINCT ON (logto_id) logto_id, name
    FROM (
        SELECT logto_id, name, deleted_at FROM distributors WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM resellers WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM customers WHERE logto_id IS NOT NULL
    ) o
    ORDER BY logto_id, (deleted_at IS NOT NULL)
)
UPDATE resellers t
SET custom_data = jsonb_set(t.custom_data, '{createdByUser,organization_name}', to_jsonb(o.name))
FROM org_names o
WHERE t.custom_data->'createdByUser'->>'organization_id' = o.logto_id
  AND t.custom_data->'createdByUser'->>'organization_name' IS DISTINCT FROM o.name;

WITH org_names AS (
    SELECT DISTINCT ON (logto_id) logto_id, name
    FROM (
        SELECT logto_id, name, deleted_at FROM distributors WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM resellers WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM customers WHERE logto_id IS NOT NULL
    ) o
    ORDER BY logto_id, (deleted_at IS NOT NULL)
)
UPDATE customers t
SET custom_data = jsonb_set(t.custom_data, '{createdByUser,organization_name}', to_jsonb(o.name))
FROM org_names o
WHERE t.custom_data->'createdByUser'->>'organization_id' = o.logto_id
  AND t.custom_data->'createdByUser'->>'organization_name' IS DISTINCT FROM o.name;

WITH org_names AS (
    SELECT DISTINCT ON (logto_id) logto_id, name
    FROM (
        SELECT logto_id, name, deleted_at FROM distributors WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM resellers WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM customers WHERE logto_id IS NOT NULL
    ) o
    ORDER BY logto_id, (deleted_at IS NOT NULL)
)
UPDATE systems t
SET created_by = jsonb_set(t.created_by, '{organization_name}', to_jsonb(o.name))
FROM org_names o
WHERE t.created_by->>'organization_id' = o.logto_id
  AND t.created_by->>'organization_name' IS DISTINCT FROM o.name;

WITH org_names AS (
    SELECT DISTINCT ON (logto_id) logto_id, name
    FROM (
        SELECT logto_id, name, deleted_at FROM distributors WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM resellers WHERE logto_id IS NOT NULL
        UNION ALL SELECT logto_id, name, deleted_at FROM customers WHERE logto_id IS NOT NULL
    ) o
    ORDER BY logto_id, (deleted_at IS NOT NULL)
)
UPDATE users t
SET created_by = jsonb_set(t.created_by, '{organization_name}', to_jsonb(o.name))
FROM org_names o
WHERE t.created_by->>'organization_id' = o.logto_id
  AND t.created_by->>'organization_name' IS DISTINCT FROM o.name;

-- ============================================================
-- 2. Backfill on_behalf_of on attributed creations
-- ============================================================

UPDATE resellers t
SET custom_data = jsonb_set(t.custom_data, '{createdByUser,on_behalf_of}', 'true'::jsonb)
FROM users u
WHERE u.logto_id = t.custom_data->'createdByUser'->>'user_id'
  AND u.deleted_at IS NULL
  AND u.organization_id IS NOT NULL
  AND u.organization_id <> t.custom_data->'createdByUser'->>'organization_id'
  AND (t.custom_data->'createdByUser'->>'on_behalf_of') IS DISTINCT FROM 'true';

UPDATE customers t
SET custom_data = jsonb_set(t.custom_data, '{createdByUser,on_behalf_of}', 'true'::jsonb)
FROM users u
WHERE u.logto_id = t.custom_data->'createdByUser'->>'user_id'
  AND u.deleted_at IS NULL
  AND u.organization_id IS NOT NULL
  AND u.organization_id <> t.custom_data->'createdByUser'->>'organization_id'
  AND (t.custom_data->'createdByUser'->>'on_behalf_of') IS DISTINCT FROM 'true';

UPDATE systems t
SET created_by = jsonb_set(t.created_by, '{on_behalf_of}', 'true'::jsonb)
FROM users u
WHERE u.logto_id = t.created_by->>'user_id'
  AND u.deleted_at IS NULL
  AND u.organization_id IS NOT NULL
  AND u.organization_id <> t.created_by->>'organization_id'
  AND (t.created_by->>'on_behalf_of') IS DISTINCT FROM 'true';

COMMENT ON COLUMN systems.created_by IS 'JSON object: {user_id, username, name, email, organization_id, organization_name, on_behalf_of} who created the system; display/audit only, not used for RBAC';
