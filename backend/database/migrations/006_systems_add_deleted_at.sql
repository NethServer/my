-- Migration: Add deleted_at column to systems table
-- This migration adds soft delete functionality to systems using deleted_at timestamp
-- Date: 2025-07-31

-- Step 1: Add deleted_at column
ALTER TABLE systems ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Step 2: Create index for deleted_at column for performance
CREATE INDEX IF NOT EXISTS idx_systems_deleted_at 
ON systems(deleted_at);

-- Step 3: Add comment for documentation
COMMENT ON COLUMN systems.deleted_at IS 'Soft delete timestamp. NULL means active, non-NULL means deleted at that time.';