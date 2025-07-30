-- Migration: Add latest_login_at field to users table
-- This migration adds the latest_login_at field introduced in the email branch
-- Date: 2025-07-30

-- Add latest_login_at column to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS latest_login_at TIMESTAMP WITH TIME ZONE;

-- Add performance index for latest_login_at
CREATE INDEX IF NOT EXISTS idx_users_latest_login_at ON users(latest_login_at DESC);