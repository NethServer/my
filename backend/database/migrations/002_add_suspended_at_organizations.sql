-- Migration: Add suspended_at column to organization tables
-- This enables status filtering (Enabled/Blocked) for distributors, resellers, and customers

-- Add suspended_at to distributors
ALTER TABLE distributors ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE;
COMMENT ON COLUMN distributors.suspended_at IS 'Suspension timestamp. NULL means enabled, non-NULL means blocked/suspended at that time.';
CREATE INDEX IF NOT EXISTS idx_distributors_suspended_at ON distributors(suspended_at);

-- Add suspended_at to resellers
ALTER TABLE resellers ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE;
COMMENT ON COLUMN resellers.suspended_at IS 'Suspension timestamp. NULL means enabled, non-NULL means blocked/suspended at that time.';
CREATE INDEX IF NOT EXISTS idx_resellers_suspended_at ON resellers(suspended_at);

-- Add suspended_at to customers
ALTER TABLE customers ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMP WITH TIME ZONE;
COMMENT ON COLUMN customers.suspended_at IS 'Suspension timestamp. NULL means enabled, non-NULL means blocked/suspended at that time.';
CREATE INDEX IF NOT EXISTS idx_customers_suspended_at ON customers(suspended_at);
