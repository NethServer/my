-- Migration 051: soft tier limits. An add-on sold in tiers without seats
-- (Threat Shield: devices on a firewall, nodes of a cluster) is never
-- blocked past its tier: collect measures the usage at every inventory and
-- Nethesis sees the systems beyond it. The tier limit is the grant's seats.

ALTER TABLE entitlement_catalog ADD COLUMN IF NOT EXISTS usage_metric VARCHAR(20) NOT NULL DEFAULT '';

ALTER TABLE entitlement_catalog DROP CONSTRAINT IF EXISTS entitlement_catalog_usage_metric_check;
ALTER TABLE entitlement_catalog ADD CONSTRAINT entitlement_catalog_usage_metric_check
    CHECK (usage_metric IN ('', 'arp_macs', 'instances')
           AND (usage_metric = '' OR NOT seated)
           AND (usage_metric <> 'instances' OR applies_to <> ''));

COMMENT ON COLUMN entitlement_catalog.usage_metric IS 'What the tier limit (system_entitlements.seats) is measured against, for add-ons without seats: arp_macs (devices seen by the firewall, inventory facts.arp_macs) | instances (application instances of applies_to) | empty = not measured';

ALTER TABLE system_entitlements ADD COLUMN IF NOT EXISTS usage INTEGER;
ALTER TABLE system_entitlements ADD COLUMN IF NOT EXISTS usage_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN system_entitlements.usage    IS 'Usage measured by collect at the last inventory, for add-ons with a usage_metric; NULL = not measured';
COMMENT ON COLUMN system_entitlements.usage_at IS 'When usage was measured';
COMMENT ON COLUMN system_entitlements.seats    IS 'Seats bought, for seated add-ons; tier limit compared with usage, for add-ons with a usage_metric (NULL = no limit); set by activate from the shop tier';

CREATE INDEX IF NOT EXISTS idx_system_entitlements_over_limit
    ON system_entitlements(entitlement) WHERE usage > seats;

UPDATE entitlement_catalog SET usage_metric = 'arp_macs', updated_at = NOW()
WHERE id = 'nsec-blacklist' AND usage_metric = '';

UPDATE entitlement_catalog SET usage_metric = 'instances', updated_at = NOW()
WHERE id = 'crowdsec-advanced-threat-shield' AND usage_metric = '' AND applies_to <> '' AND NOT seated;

-- Grants bought before the shop sent the limit carry it in the tier label
-- only: "1-15 device" -> 15, "Singolo Nodo" -> 1.
UPDATE system_entitlements e
SET seats = CASE
                WHEN e.variant->>'label' ~* '\d+\s*device' THEN substring(e.variant->>'label' from '(\d+)\s*[Dd]evice')::int
                WHEN e.variant->>'label' ~* 'singolo' THEN 1
            END
FROM entitlement_catalog c
WHERE c.id = e.entitlement
  AND c.usage_metric <> ''
  AND e.seats IS NULL
  AND (e.variant->>'label' ~* '\d+\s*device' OR e.variant->>'label' ~* 'singolo');
