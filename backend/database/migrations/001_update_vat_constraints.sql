-- Migration: Update VAT constraints to be scoped by organization role
-- Date: 2024-09-04
-- Description: 
--   - Remove global VAT uniqueness constraint
--   - Add per-entity-type VAT uniqueness constraints  
--   - Distributors and Resellers: VAT unique within their entity type
--   - Customers: No VAT uniqueness constraint (allows duplicates)

-- =============================================================================
-- STEP 1: Drop existing global VAT triggers and functions
-- =============================================================================

-- Drop triggers first
DROP TRIGGER IF EXISTS trg_check_vat_distributors ON distributors;
DROP TRIGGER IF EXISTS trg_check_vat_resellers ON resellers;
DROP TRIGGER IF EXISTS trg_check_vat_customers ON customers;

-- Drop old global function
DROP FUNCTION IF EXISTS check_unique_vat();

-- =============================================================================
-- STEP 2: Create new per-entity-type VAT uniqueness functions
-- =============================================================================

-- VAT uniqueness function for distributors
CREATE OR REPLACE FUNCTION check_unique_vat_distributors()
RETURNS TRIGGER AS $$
DECLARE
    new_vat TEXT;
BEGIN
    new_vat := TRIM(NEW.custom_data->>'vat');

    IF new_vat IS NULL OR new_vat = '' OR NEW.deleted_at IS NOT NULL THEN
        RETURN NEW;
    END IF;

    -- Check in distributors only, excluding same id (for updates)
    IF EXISTS (
        SELECT 1 FROM distributors
        WHERE TRIM(custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in distributors', new_vat;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- VAT uniqueness function for resellers
CREATE OR REPLACE FUNCTION check_unique_vat_resellers()
RETURNS TRIGGER AS $$
DECLARE
    new_vat TEXT;
BEGIN
    new_vat := TRIM(NEW.custom_data->>'vat');

    IF new_vat IS NULL OR new_vat = '' OR NEW.deleted_at IS NOT NULL THEN
        RETURN NEW;
    END IF;

    -- Check in resellers only, excluding same id (for updates)  
    IF EXISTS (
        SELECT 1 FROM resellers
        WHERE TRIM(custom_data->>'vat') = new_vat
          AND deleted_at IS NULL
          AND (id IS DISTINCT FROM NEW.id)
    ) THEN
        RAISE EXCEPTION 'VAT "%" already exists in resellers', new_vat;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- VAT uniqueness function for customers (no uniqueness constraint)
CREATE OR REPLACE FUNCTION check_unique_vat_customers()
RETURNS TRIGGER AS $$
BEGIN
    -- No VAT uniqueness constraint for customers
    -- VAT is optional for customers and can be duplicate
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- STEP 3: Create new triggers using the per-entity-type functions
-- =============================================================================

-- Distributors
CREATE TRIGGER trg_check_vat_distributors
BEFORE INSERT OR UPDATE ON distributors
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_distributors();

-- Resellers
CREATE TRIGGER trg_check_vat_resellers
BEFORE INSERT OR UPDATE ON resellers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_resellers();

-- Customers
CREATE TRIGGER trg_check_vat_customers
BEFORE INSERT OR UPDATE ON customers
FOR EACH ROW
EXECUTE FUNCTION check_unique_vat_customers();

-- =============================================================================
-- STEP 4: Verify migration success
-- =============================================================================

-- Function to verify triggers are properly installed
DO $$
DECLARE
    trigger_count INTEGER;
BEGIN
    -- Check that all 3 triggers exist
    SELECT COUNT(*) INTO trigger_count
    FROM pg_trigger 
    WHERE tgname IN (
        'trg_check_vat_distributors', 
        'trg_check_vat_resellers', 
        'trg_check_vat_customers'
    );
    
    IF trigger_count != 3 THEN
        RAISE EXCEPTION 'Migration failed: Expected 3 VAT triggers, found %', trigger_count;
    END IF;
    
    RAISE NOTICE 'Migration completed successfully: % VAT triggers installed', trigger_count;
END
$$;

-- =============================================================================
-- MIGRATION COMPLETED
-- =============================================================================

-- Summary of changes:
-- ✅ Removed global VAT uniqueness constraint
-- ✅ Added per-entity-type VAT constraints:
--    - Distributors: VAT unique within distributors table
--    - Resellers: VAT unique within resellers table  
--    - Customers: No VAT uniqueness (allows duplicates)
-- ✅ Maintained soft-delete awareness (deleted_at IS NULL)
-- ✅ Maintained update exclusion (id IS DISTINCT FROM NEW.id)