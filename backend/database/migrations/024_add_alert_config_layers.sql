-- Migration 024: Add alert_config_layers table
-- Stores ONE alerting configuration "layer" per organization. The effective
-- per-tenant Mimir config is derived at render time as a merge of all the
-- layers walking up the hierarchy from the tenant to the Owner:
--
--    Owner.layer  →  Distributor.layer  →  Reseller.layer  →  Customer.layer
--
-- Merge rules (additive-only for security-relevant fields):
--   - bool channel toggles (mail_enabled, ...): OR — a descendant cannot
--     disable a channel an ancestor enabled. Normalised at write time so
--     non-Owner layers cannot store an explicit false.
--   - list fields (mail_addresses, webhook_receivers, telegram_receivers,
--     severities, systems): union with dedup — a descendant can only ADD.
--   - email_template_lang: deepest non-empty wins (per-tenant rendering
--     preference, not a behavior modification of ancestors' recipients).
--
-- Mimir continues to see a flat YAML per tenant; the layered model is a
-- backend-side concept invisible to Alertmanager itself.

CREATE TABLE IF NOT EXISTS alert_config_layers (
    organization_id  VARCHAR(255) PRIMARY KEY,

    -- Serialized AlertingConfigLayer (boolean fields are tri-state via
    -- *bool / null = "no opinion at this layer, inherit from above").
    config_json      JSONB NOT NULL,

    -- Audit fields. updated_by_user_id stores the logto_id of the user who
    -- last saved this layer. updated_by_name is denormalised for cheap UI
    -- rendering of "who set this".
    updated_by_user_id VARCHAR(255),
    updated_by_name    VARCHAR(255),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  alert_config_layers              IS 'Per-organization alerting config layer. Effective Mimir YAML for a tenant is the merge of all layers from Owner down to that tenant.';
COMMENT ON COLUMN alert_config_layers.config_json  IS 'Serialized AlertingConfigLayer with *bool channel flags so "not set" is distinguishable from "explicit false" during merge';
