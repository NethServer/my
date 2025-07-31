-- Migration: Convert resellers.active boolean to deleted_at timestamp
-- This migration converts the active boolean field to a soft delete pattern using deleted_at timestamp
-- Date: 2025-07-31

-- Step 1: Add deleted_at column
ALTER TABLE resellers ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Step 2: Migrate existing data - set deleted_at for inactive records
UPDATE resellers 
SET deleted_at = updated_at 
WHERE active = false AND deleted_at IS NULL;

-- Step 3: Drop old indexes that depend on active column
DROP INDEX IF EXISTS idx_resellers_logto_id;
DROP INDEX IF EXISTS idx_resellers_active;

-- Step 4: Create new indexes using deleted_at pattern
CREATE UNIQUE INDEX IF NOT EXISTS idx_resellers_logto_id 
ON resellers(logto_id) 
WHERE logto_id IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_resellers_deleted_at 
ON resellers(deleted_at);

-- Step 5: Drop the active column (after ensuring all data is migrated)
ALTER TABLE resellers DROP COLUMN IF EXISTS active;

-- Add comment for documentation
COMMENT ON COLUMN resellers.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';