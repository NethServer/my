-- Migration 049: rename recipient scope organization_ids to managed_organization_ids.
-- The backend still reads the old key, so this only normalises stored layers.

UPDATE alert_config_layers l
SET config_json = (
    SELECT jsonb_object_agg(k, CASE
        WHEN k IN ('email_recipients', 'webhook_recipients', 'telegram_recipients') AND jsonb_typeof(v) = 'array' THEN (
            SELECT COALESCE(jsonb_agg(CASE
                WHEN r->'scope' ? 'organization_ids' THEN jsonb_set(
                    r, '{scope}',
                    ((r->'scope') - 'organization_ids') || jsonb_build_object('managed_organization_ids', r->'scope'->'organization_ids'))
                ELSE r
            END ORDER BY ord), '[]'::jsonb)
            FROM jsonb_array_elements(v) WITH ORDINALITY AS e(r, ord))
        ELSE v
    END)
    FROM jsonb_each(l.config_json) AS f(k, v)
)
WHERE l.config_json::text LIKE '%"organization_ids"%';
