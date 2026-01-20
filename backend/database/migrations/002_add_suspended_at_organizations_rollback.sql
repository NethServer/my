-- Rollback: Remove suspended_at column from organization tables

-- Remove from distributors
DROP INDEX IF EXISTS idx_distributors_suspended_at;
ALTER TABLE distributors DROP COLUMN IF EXISTS suspended_at;

-- Remove from resellers
DROP INDEX IF EXISTS idx_resellers_suspended_at;
ALTER TABLE resellers DROP COLUMN IF EXISTS suspended_at;

-- Remove from customers
DROP INDEX IF EXISTS idx_customers_suspended_at;
ALTER TABLE customers DROP COLUMN IF EXISTS suspended_at;
