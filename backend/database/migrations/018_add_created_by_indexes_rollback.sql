-- Rollback Migration 018: Remove createdBy indexes on resellers and customers

DROP INDEX IF EXISTS idx_resellers_created_by;
DROP INDEX IF EXISTS idx_customers_created_by;
