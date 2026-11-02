-- Migration 059: Services added by operators to a system's support sessions
-- A web UI the discovery missed (a phone, a printer, a device on the LAN)
-- is added once per system node and offered again to every later session,
-- until an operator removes it.

CREATE TABLE IF NOT EXISTS support_system_services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    system_id VARCHAR(255) NOT NULL REFERENCES systems(id) ON DELETE CASCADE,
    node_id VARCHAR(16) NOT NULL DEFAULT '',
    name VARCHAR(64) NOT NULL,
    target VARCHAR(255) NOT NULL,
    label VARCHAR(255) NOT NULL DEFAULT '',
    tls BOOLEAN NOT NULL DEFAULT false,
    created_by_id VARCHAR(255) NOT NULL,
    created_by_name VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT support_system_services_unique UNIQUE (system_id, node_id, name)
);

COMMENT ON TABLE support_system_services IS 'Services added by operators, restored on every support session of the system node';
COMMENT ON COLUMN support_system_services.node_id IS 'Cluster node the target is reached from; empty for single-node systems';
