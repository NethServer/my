-- Migration 004: Add name and source columns to applications table
-- name: human-readable label from inventory (e.g., "Nextcloud")
-- source: image source from inventory (e.g., "ghcr.io/nethserver/nextcloud")

ALTER TABLE applications ADD COLUMN name VARCHAR(255);
ALTER TABLE applications ADD COLUMN source VARCHAR(500);
