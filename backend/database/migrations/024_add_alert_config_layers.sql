-- Migration 024: alert_config_layers
--
-- One row per organization carrying that org's alerting configuration as a
-- flat recipient-based JSON blob. The effective per-tenant Mimir YAML is
-- the server-side merge of all rows walking up the org hierarchy from the
-- tenant to the Owner:
--
--    Owner.layer  →  Distributor.layer  →  Reseller.layer  →  Customer.layer
--
-- The merge is internal — /alerts/config exposes only the caller's own
-- row, never the merged effective view or anyone else's row.
--
-- Merge rules (additive-only for security-relevant fields):
--   - bool channel toggles (enabled.{email,webhook,telegram}): OR. A
--     descendant cannot disable a channel an ancestor enabled. Non-Owner
--     layers cannot store an explicit false (normalised to null on save).
--   - recipient lists (email/webhook/telegram): union with stable dedup.
--     Dedup keys: email→address, webhook→url, telegram→(bot_token,chat_id).
--   - per-recipient severities[]: union; if any contributor uses [] ("all
--     severities") the merged copy widens back to [].
--
-- Mimir sees a flat YAML per tenant; the layered model is server-internal
-- and invisible to Alertmanager.

CREATE TABLE IF NOT EXISTS alert_config_layers (
    organization_id  VARCHAR(255) PRIMARY KEY,

    -- Serialized AlertingConfigLayer (Go struct):
    --   {
    --     "enabled":             {"email": *bool, "webhook": *bool, "telegram": *bool},
    --     "email_recipients":    [{address, severities[], language, format}],
    --     "webhook_recipients":  [{name, url, severities[]}],
    --     "telegram_recipients": [{bot_token, chat_id, severities[]}]
    --   }
    -- Channel toggles are tri-state (null = "no opinion at this layer,
    -- inherit from above"). Per-recipient severities=[] means "all".
    config_json      JSONB NOT NULL,

    -- Audit fields. updated_by_user_id stores the logto_id of the user who
    -- last saved this layer. updated_by_name is denormalised for cheap UI
    -- rendering of "who set this".
    updated_by_user_id VARCHAR(255),
    updated_by_name    VARCHAR(255),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  alert_config_layers              IS 'Per-organization alerting config layer. Effective Mimir YAML for a tenant is the merge of all layers from Owner down to that tenant; merge is server-side only and never exposed via API.';
COMMENT ON COLUMN alert_config_layers.config_json  IS 'Serialized AlertingConfigLayer: { enabled:{email,webhook,telegram}, email_recipients[], webhook_recipients[], telegram_recipients[] }. Each recipient carries its own severities[]; email recipients additionally carry language+format. Channel toggles are nullable tri-state.';
