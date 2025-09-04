-- Rollback Migration: Revert VAT constraints to global uniqueness
-- Date: 2024-09-04
-- Description: 
--   - Remove per-entity-type VAT constraints
--   - Restore global VAT uniqueness constraint
--   - This rollback restores the original behavior where VAT must be unique across all entity types

-- =============================================================================
-- STEP 1: Drop current per-entity-type triggers and functions
-- =============================================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_check_vat_distributors ON distributors;
DROP TRIGGER IF EXISTS trg_check_vat_resellers ON resellers;
DROP TRIGGER IF EXISTS trg_check_vat_customers ON customers;

-- Drop per-entity-type functions
DROP FUNCTION IF EXISTS check_unique_vat_distributors();
DROP FUNCTION IF EXISTS check_unique_vat_resellers();
DROP FUNCTION IF EXISTS check_unique_vat_customers();

-- =============================================================================
-- STEP 2: Restore original global VAT uniqueness function
-- =============================================================================

-- Global VAT uniqueness function
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
    IF EXISTS (
        SELECT 1 FROM resellers
        WHERE TRIM(custom_data->>'vat') = new_vat
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

-- =============================================================================
-- STEP 3: Restore original triggers using global function
-- =============================================================================

-- Distributors
CREATE TRIGGER trg_check_vat_distributors
BEFORE INSERT OR UPDATE ON distributors
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat();

-- Resellers
CREATE TRIGGER trg_check_vat_resellers
BEFORE INSERT OR UPDATE ON resellers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat();

-- Customers
CREATE TRIGGER trg_check_vat_customers
BEFORE INSERT OR UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat();

-- =============================================================================
-- STEP 4: Verify rollback success
-- =============================================================================

-- Function to verify rollback completed successfully
DO $$
DECLARE
    trigger_count INTEGER;
    function_exists BOOLEAN;
BEGIN
    -- Check that all 3 triggers exist
    SELECT COUNT(*) INTO trigger_count
    FROM pg_trigger 
    WHERE tgname IN (
        'trg_check_vat_distributors', 
        'trg_check_vat_resellers', 
        'trg_check_vat_customers'
    );
    
    -- Check that global function exists
    SELECT EXISTS (
        SELECT 1 FROM pg_proc WHERE proname = 'check_unique_vat'
    ) INTO function_exists;
    
    IF trigger_count != 3 THEN
        RAISE EXCEPTION 'Rollback failed: Expected 3 VAT triggers, found %', trigger_count;
    END IF;
    
    IF NOT function_exists THEN
        RAISE EXCEPTION 'Rollback failed: Global check_unique_vat function not found';
    END IF;
    
    RAISE NOTICE 'Rollback completed successfully: % triggers and global function restored', trigger_count;
END
$$;

-- =============================================================================
-- ROLLBACK COMPLETED
-- =============================================================================

-- Summary of rollback:
-- ✅ Removed per-entity-type VAT constraints
-- ✅ Restored global VAT uniqueness constraint
-- ✅ VAT must now be unique across all entity types (distributors, resellers, customers)
-- ✅ Maintained soft-delete awareness (deleted_at IS NULL)
-- ✅ Maintained update exclusion (id IS DISTINCT FROM NEW.id)