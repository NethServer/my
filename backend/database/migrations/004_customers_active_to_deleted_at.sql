-- Migration: Convert customers.active boolean to deleted_at timestamp
-- This migration converts the active boolean field to a soft delete pattern using deleted_at timestamp
-- Date: 2025-07-31

-- Step 1: Add deleted_at column
ALTER TABLE customers ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Step 2: Migrate existing data - set deleted_at for inactive records
UPDATE customers 
SET deleted_at = updated_at 
WHERE active = false AND deleted_at IS NULL;

-- Step 3: Drop old indexes that depend on active column
DROP INDEX IF EXISTS idx_customers_logto_id;
DROP INDEX IF EXISTS idx_customers_active;

-- Step 4: Create new indexes using deleted_at pattern
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_logto_id 
ON customers(logto_id) 
WHERE logto_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_customers_deleted_at 
ON customers(deleted_at);

-- Step 5: Drop the active column (after ensuring all data is migrated)
ALTER TABLE customers DROP COLUMN IF EXISTS active;

-- Add comment for documentation
COMMENT ON COLUMN customers.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';