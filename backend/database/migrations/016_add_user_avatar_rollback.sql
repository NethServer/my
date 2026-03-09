-- Rollback Migration 016: Remove avatar columns from users table

ALTER TABLE users DROP COLUMN IF EXISTS avatar;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_mime;
