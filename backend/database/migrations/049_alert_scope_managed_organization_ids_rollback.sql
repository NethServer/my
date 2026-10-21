-- Rollback migration 049: restore the organization_ids key in recipient scopes.

UPDATE alert_config_layers l
SET config_json = (
    SELECT jsonb_object_agg(k, CASE
        WHEN k IN ('email_recipients', 'webhook_recipients', 'telegram_recipients') AND jsonb_typeof(v) = 'array' THEN (
            SELECT COALESCE(jsonb_agg(CASE
                WHEN r->'scope' ? 'managed_organization_ids' THEN jsonb_set(
                    r, '{scope}',
                    ((r->'scope') - 'managed_organization_ids') || jsonb_build_object('organization_ids', r->'scope'->'managed_organization_ids'))
                ELSE r
            END ORDER BY ord), '[]'::jsonb)
            FROM jsonb_array_elements(v) WITH ORDINALITY AS e(r, ord))
        ELSE v
    END)
    FROM jsonb_each(l.config_json) AS f(k, v)
)
WHERE l.config_json::text LIKE '%"managed_organization_ids"%';
