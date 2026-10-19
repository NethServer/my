-- Migration 047: update_available on systems, extracted by collect from each inventory.

ALTER TABLE systems ADD COLUMN IF NOT EXISTS update_available BOOLEAN;

COMMENT ON COLUMN systems.update_available IS 'Update available per the latest inventory: NS8 leader node update_available, NSEC image_updates_available. NULL = unknown';

-- Backfill from each live system's latest inventory (served by
-- idx_inventory_records_system_id_id); later inventories keep it current.
WITH latest AS (
    SELECT s.id,
           (SELECT ir.data FROM inventory_records ir WHERE ir.system_id = s.id ORDER BY ir.id DESC LIMIT 1) AS data
    FROM systems s
    WHERE s.deleted_at IS NULL
),
flags AS (
    SELECT id,
           CASE data->>'installation'
               WHEN 'nethserver' THEN COALESCE(
                   (SELECT n.value->'update_available' FROM jsonb_each(data->'facts'->'nodes') n
                     WHERE n.value->'cluster_leader' = 'true'::jsonb LIMIT 1),
                   data->'facts'->'nodes'->'1'->'update_available')
               WHEN 'nethsecurity' THEN data->'facts'->'image_updates_available'
           END AS flag
    FROM latest
    WHERE data IS NOT NULL
)
UPDATE systems s
SET update_available = (f.flag)::text::boolean
FROM flags f
WHERE s.id = f.id AND jsonb_typeof(f.flag) = 'boolean';
