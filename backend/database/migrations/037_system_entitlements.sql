-- Migration 037: entitlement catalog + system_entitlements (granular add-on licensing)
-- Replaces the transitional grant-all /auth broker.
--
-- entitlement_catalog: the set of sellable/grantable add-ons, DB-driven so the
-- owner can add types (and, in fase 3, map shop products) without a deploy.
-- `scoped` marks add-ons that can be granted to a single application instance
-- of a system (e.g. the "chat" module of nethvoice5 on an NS8 cluster).
--
-- system_entitlements: one row grants one add-on to one system, optionally
-- narrowed to one application instance via `scope` ('' = whole system).
-- Collect's native /auth/service/<id>[?scope=<app>] answers 200/403 from
-- here: a system-wide grant also covers every instance (fallback). The
-- backend exposes CRUD under /api/systems/:id/entitlements (manual writes =
-- owner org or Super Admin only: licensing is controlled by Nethesis,
-- downstream orgs buy through the shop). Sources: 'legacy-import' (one-shot +
-- cron delta from the old my service_server), 'shop' (nethshop
-- order/subscription), 'manual'. Renewals UPDATE valid_until in place — one
-- row per (system, entitlement, scope), history stays in the audit log.
-- Active = not revoked AND (valid_until IS NULL OR valid_until > now());
-- valid_until NULL = no expiry (all legacy imports are perpetual by decision
-- of 2026-07-15).

CREATE TABLE IF NOT EXISTS entitlement_catalog (
    id           VARCHAR(100) PRIMARY KEY,
    display_name VARCHAR(255) NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    scoped       BOOLEAN NOT NULL DEFAULT FALSE,
    kind         VARCHAR(20)  NOT NULL DEFAULT 'service',
    system_type  VARCHAR(50)  NOT NULL DEFAULT '',
    legacy_alias VARCHAR(100) NOT NULL DEFAULT '',
    created_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_entitlement_catalog_legacy_alias
    ON entitlement_catalog(legacy_alias) WHERE legacy_alias <> '';

COMMENT ON TABLE  entitlement_catalog              IS 'Grantable add-on types. Id convention: nsec-<service> (firewall services), ns8-<app> (app enablement on a cluster), <app>-<module> (per-app-instance modules, scoped=TRUE)';
COMMENT ON COLUMN entitlement_catalog.scoped       IS 'TRUE = grantable per application instance of a system (scope on the grant); FALSE = system-wide only';
COMMENT ON COLUMN entitlement_catalog.kind         IS 'service (nsec-<service>, firewall add-on, system-wide) | module (<app>-<module>, add-on for one application instance of an NS8 cluster) — both sellable on the shop';
COMMENT ON COLUMN entitlement_catalog.system_type  IS 'System type the add-on applies to: nsec | ns8; empty = any. Grants are refused on mismatching systems and the UI/shop only offer pertinent add-ons';
COMMENT ON COLUMN entitlement_catalog.legacy_alias IS 'Legacy wire id the appliance feeds still call on /auth/service/<id> (e.g. ng-blacklist for nsec-blacklist); collect resolves it to the canonical id';

INSERT INTO entitlement_catalog (id, display_name, description, scoped, kind, system_type, legacy_alias) VALUES
    ('nsec-blacklist', 'Advanced Threat Shield', 'Enterprise blacklist feeds (bl.nethesis.it)', FALSE, 'service', 'nsec', 'ng-blacklist'),
    ('nsec-ha',        'High Availability',      'High availability (HA)',                      FALSE, 'service', 'nsec', 'ng-ha'),
    ('nsec-sandbox',   'Sandbox',                'Sandbox',                                     FALSE, 'service', 'nsec', 'ng-sandbox')
ON CONFLICT (id) DO NOTHING;

-- entitlement_availability: OPTIONAL commercial restriction. A catalog item
-- is available to EVERYONE by default; when rules exist for an item, only
-- the matching role/orgs may buy it (org_role XOR organization_id per row).
-- This table does NOT affect /auth enforcement: an existing grant works
-- regardless.
CREATE TABLE IF NOT EXISTS entitlement_availability (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entitlement     VARCHAR(100) NOT NULL REFERENCES entitlement_catalog(id) ON DELETE CASCADE,
    org_role        VARCHAR(50)  NOT NULL DEFAULT '',
    organization_id VARCHAR(255) NOT NULL DEFAULT '',
    created_by      JSONB,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT entitlement_availability_unique UNIQUE (entitlement, org_role, organization_id),
    CONSTRAINT entitlement_availability_target CHECK ((org_role <> '') <> (organization_id <> ''))
);

CREATE INDEX IF NOT EXISTS idx_entitlement_availability_ent ON entitlement_availability(entitlement);

COMMENT ON TABLE  entitlement_availability                 IS 'Optional commercial restriction: no rows = item available to everyone; rules restrict to matching role/orgs. Does not affect /auth enforcement';
COMMENT ON COLUMN entitlement_availability.org_role        IS 'Role-wide unlock: distributor | reseller | customer (empty when organization_id is set)';
COMMENT ON COLUMN entitlement_availability.organization_id IS 'Org-specific unlock (empty when org_role is set)';

CREATE TABLE IF NOT EXISTS system_entitlements (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    system_id   VARCHAR(255) NOT NULL REFERENCES systems(id) ON DELETE CASCADE,
    entitlement VARCHAR(100) NOT NULL REFERENCES entitlement_catalog(id),
    scope       VARCHAR(255) NOT NULL DEFAULT '',
    source      VARCHAR(50)  NOT NULL DEFAULT 'manual',
    source_ref  VARCHAR(255),
    valid_from  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMP WITH TIME ZONE,
    revoked_at  TIMESTAMP WITH TIME ZONE,
    revoked_source VARCHAR(50),
    pending_ref VARCHAR(255),
    pending_since TIMESTAMP WITH TIME ZONE,
    created_by  JSONB,
    purchased_by JSONB,
    variant     JSONB,
    renewal_count INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT system_entitlements_unique UNIQUE (system_id, entitlement, scope)
);

CREATE INDEX IF NOT EXISTS idx_system_entitlements_system_id ON system_entitlements(system_id);
CREATE INDEX IF NOT EXISTS idx_system_entitlements_entitlement ON system_entitlements(entitlement);

COMMENT ON TABLE  system_entitlements             IS 'Granular add-on grants per system (ng-blacklist & co.); checked by collect /auth/service/<id>[?scope=]';
COMMENT ON COLUMN system_entitlements.entitlement IS 'Add-on id from entitlement_catalog (same ids as the legacy service table)';
COMMENT ON COLUMN system_entitlements.scope       IS 'Application instance the grant is narrowed to (e.g. nethvoice5 on an NS8 cluster); empty = whole system';
COMMENT ON COLUMN system_entitlements.source      IS 'How the grant was created: legacy-import | shop | manual';
COMMENT ON COLUMN system_entitlements.source_ref  IS 'Reference in the source system (e.g. nethshop order/subscription id, legacy service_server id)';
COMMENT ON COLUMN system_entitlements.valid_until IS 'Expiry; NULL = perpetual (legacy imports). Renewals push this forward in place';
COMMENT ON COLUMN system_entitlements.revoked_at  IS 'Set on revoke (DELETE endpoint / subscription cancelled); row kept for audit';
COMMENT ON COLUMN system_entitlements.revoked_source IS 'Who revoked: manual (admin DELETE/PUT — deliberate, not re-buyable) | shop (deactivate webhook: subscription cancelled/payment failed — re-buyable). NULL when not revoked';
COMMENT ON COLUMN system_entitlements.pending_ref  IS 'Shop order awaiting payment (set at checkout, cleared on activate/cancel). Display-only: enforcement ignores it. A never-activated pending stub has valid_until = valid_from';
COMMENT ON COLUMN system_entitlements.created_by  IS 'Actor snapshot (user/org or shop M2M) that created the grant';
COMMENT ON COLUMN system_entitlements.purchased_by IS 'Snapshot of the my user that BOUGHT the grant on the shop, resolved from the order customer email (webhook activation or legacy-import backfill): {logto_id, name, email, organization_id, organization_name, org_role, user_roles}. {email} only when the address matches no my user; NULL for manual grants and legacy rows without an order';
COMMENT ON COLUMN system_entitlements.variant IS 'Shop variation (tier) of the purchased product line: {id, sku, label} (e.g. label "16-30 device"). Display metadata only — the add-on mapping stays on the parent product and /auth enforcement ignores it. Refreshed by activate (upgrades/downgrades follow renewals); NULL for manual grants and simple products';
COMMENT ON COLUMN system_entitlements.renewal_count IS 'Paid shop orders on this grant beyond the first: incremented by activate when source_ref CHANGES (webhook retries on the same order never double-count). 0 = first period';
