-- Migration: Implement global VAT uniqueness constraint
-- This migration removes old per-creator VAT constraints and implements a global VAT uniqueness system
-- Date: 2025-07-31

-- Step 1: Remove old unique indexes that use custom_data->>'createdBy'
DROP INDEX IF EXISTS uk_distributors_vat_created_by;
DROP INDEX IF EXISTS uk_resellers_vat_created_by;
DROP INDEX IF EXISTS uk_customers_vat_created_by;

-- Step 2: Remove old indexes for createdBy field (no longer needed)
DROP INDEX IF EXISTS idx_distributors_created_by_jsonb;
DROP INDEX IF EXISTS idx_resellers_created_by_jsonb;
DROP INDEX IF EXISTS idx_customers_created_by_jsonb;

-- Step 3: Create global VAT uniqueness function
CREATE OR REPLACE FUNCTION check_unique_vat()
RETURNS TRIGGER AS $$
DECLARE
    new_vat TEXT;
BEGIN
    new_vat := TRIM(NEW.custom_data->>'vat');

    IF new_vat IS NULL OR new_vat = '' OR NEW.deleted_at IS NOT NULL THEN
        RETURN NEW;
    END IF;

    -- Check in distributors, excluding same id (for updates)
    IF EXISTS (
        SELECT 1 FROM distributors
        WHERE TRIM(custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in distributors', new_vat;
    END IF;

    -- Check in resellers, excluding same id (for updates)
    IF TG_TABLE_NAME <> 'resellers' AND EXISTS (
        SELECT 1 FROM resellers
        WHERE (custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in resellers', new_vat;
    END IF;

    -- Check in customers, excluding same id (for updates)
    IF EXISTS (
        SELECT 1 FROM customers
        WHERE TRIM(custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in customers', new_vat;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Create triggers for all three tables

-- Distributors
DROP TRIGGER IF EXISTS trg_check_vat_distributors ON distributors;
CREATE TRIGGER trg_check_vat_distributors
BEFORE INSERT OR UPDATE ON distributors
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat();

-- Resellers
DROP TRIGGER IF EXISTS trg_check_vat_resellers ON resellers;
CREATE TRIGGER trg_check_vat_resellers
BEFORE INSERT OR UPDATE ON resellers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat();

-- Customers
DROP TRIGGER IF EXISTS trg_check_vat_customers ON customers;
CREATE TRIGGER trg_check_vat_customers
BEFORE INSERT OR UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat();