-- Migration 018: Add missing createdBy indexes on resellers and customers
-- These indexes speed up hierarchy lookups used in distributors/resellers/customers list queries

CREATE INDEX IF NOT EXISTS idx_resellers_created_by ON resellers ((custom_data->>'createdBy')) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_customers_created_by ON customers ((custom_data->>'createdBy')) WHERE deleted_at IS NULL;
