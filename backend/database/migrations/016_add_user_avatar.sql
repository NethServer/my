-- Migration 016: Add avatar columns to users table
-- Stores user profile images as binary data with MIME type

ALTER TABLE users ADD COLUMN avatar BYTEA;
ALTER TABLE users ADD COLUMN avatar_mime VARCHAR(50);
