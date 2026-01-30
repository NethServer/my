-- Migration 004: Add name and source columns to applications table
-- name: human-readable label from inventory (e.g., "Nextcloud")
-- source: image source from inventory (e.g., "ghcr.io/nethserver/nextcloud")

ALTER TABLE applications ADD COLUMN IF NOT EXISTS name VARCHAR(255);
ALTER TABLE applications ADD COLUMN IF NOT EXISTS source VARCHAR(500);
