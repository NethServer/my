-- Collect Service Database Schema
-- This file should be executed when initializing the database for collect service

-- Inventory records table - stores raw inventory data from systems
CREATE TABLE IF NOT EXISTS inventory_records (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    data JSONB NOT NULL,
    data_hash VARCHAR(64) NOT NULL,
    data_size BIGINT NOT NULL,
    compressed BOOLEAN NOT NULL DEFAULT false,
    processed_at TIMESTAMP WITH TIME ZONE,
    has_changes BOOLEAN NOT NULL DEFAULT false,
    change_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(system_id, data_hash)
);

-- Inventory diffs table - stores computed differences between inventory snapshots
CREATE TABLE IF NOT EXISTS inventory_diffs (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    previous_id BIGINT REFERENCES inventory_records(id),
    current_id BIGINT NOT NULL REFERENCES inventory_records(id),
    diff_type VARCHAR(50) NOT NULL CHECK (diff_type IN ('create', 'update', 'delete')),
    field_path TEXT NOT NULL,
    previous_value TEXT,
    current_value TEXT,
    severity VARCHAR(50) NOT NULL DEFAULT 'low' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    category VARCHAR(100) NOT NULL DEFAULT 'general',
    notification_sent BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Inventory monitoring table - stores monitoring rules and thresholds
CREATE TABLE IF NOT EXISTS inventory_monitoring (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255), -- NULL for global rules
    field_path TEXT NOT NULL,
    monitor_type VARCHAR(50) NOT NULL CHECK (monitor_type IN ('threshold', 'change', 'pattern')),
    threshold TEXT,
    pattern TEXT,
    severity VARCHAR(50) NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    is_enabled BOOLEAN NOT NULL DEFAULT true,
    description TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Inventory alerts table - stores triggered alerts based on monitoring rules
CREATE TABLE IF NOT EXISTS inventory_alerts (
    id BIGSERIAL PRIMARY KEY,
    system_id VARCHAR(255) NOT NULL,
    monitoring_id BIGINT NOT NULL REFERENCES inventory_monitoring(id),
    diff_id BIGINT REFERENCES inventory_diffs(id),
    alert_type VARCHAR(50) NOT NULL CHECK (alert_type IN ('threshold', 'change', 'pattern')),
    message TEXT NOT NULL,
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    is_resolved BOOLEAN NOT NULL DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Performance indexes for optimal query performance
CREATE INDEX IF NOT EXISTS idx_inventory_records_system_id_timestamp ON inventory_records(system_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_records_data_hash ON inventory_records(data_hash);
CREATE INDEX IF NOT EXISTS idx_inventory_records_processed_at ON inventory_records(processed_at);

CREATE INDEX IF NOT EXISTS idx_inventory_diffs_system_id_created_at ON inventory_diffs(system_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_current_id ON inventory_diffs(current_id);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_severity ON inventory_diffs(severity);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_category ON inventory_diffs(category);
CREATE INDEX IF NOT EXISTS idx_inventory_diffs_notification_sent ON inventory_diffs(notification_sent) WHERE notification_sent = false;

CREATE INDEX IF NOT EXISTS idx_inventory_monitoring_system_id ON inventory_monitoring(system_id);
CREATE INDEX IF NOT EXISTS idx_inventory_monitoring_field_path ON inventory_monitoring(field_path);
CREATE INDEX IF NOT EXISTS idx_inventory_monitoring_enabled ON inventory_monitoring(is_enabled) WHERE is_enabled = true;

CREATE INDEX IF NOT EXISTS idx_inventory_alerts_system_id_created_at ON inventory_alerts(system_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_monitoring_id ON inventory_alerts(monitoring_id);
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_resolved ON inventory_alerts(is_resolved) WHERE is_resolved = false;
CREATE INDEX IF NOT EXISTS idx_inventory_alerts_severity ON inventory_alerts(severity);

-- Note: System credentials are managed in the backend service systems table