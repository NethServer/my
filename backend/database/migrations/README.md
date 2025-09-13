# Database Migrations

This directory contains database migration scripts for the My Nethesis backend.

## Overview

Migrations are versioned SQL scripts that allow you to evolve your database schema over time while keeping track of changes. Each migration has a forward migration (apply) and a rollback migration (undo).

## Available Migrations

### Migration 001: Update VAT Constraints
**File**: `001_update_vat_constraints.sql`  
**Rollback**: `001_update_vat_constraints_rollback.sql`

**Purpose**: Changes VAT uniqueness constraints from global to per-entity-type scope.

**Changes**:
- ✅ **Distributors**: VAT unique within distributors table only
- ✅ **Resellers**: VAT unique within resellers table only  
- ✅ **Customers**: No VAT uniqueness constraint (allows duplicates)

**Before Migration**: VAT must be unique across all entity types (distributors, resellers, customers)  
**After Migration**: VAT unique only within each entity type, customers can have duplicate VATs

## Usage

### Prerequisites

1. Set your database connection string:
   ```bash
   export DATABASE_URL='postgres://username:password@host:5432/database_name'
   ```

2. Ensure Docker or Podman is installed and running:
   ```bash
   # Check Docker
   docker --version
   
   # OR check Podman
   podman --version
   ```

**Note**: You don't need PostgreSQL client (`psql`) installed locally - the script uses containers with host networking to run database commands.

### Running Migrations

#### Easy Way (Recommended)
From the backend root directory (automatically reads DATABASE_URL from .env files):
```bash
# Run all pending migrations automatically
make db-migrate                              # Uses .env
make db-migrate-qa                           # Uses .env.qa

# For specific migration management (advanced users)
make db-migration MIGRATION=001 ACTION=apply              # Uses .env
make db-migration MIGRATION=001 ACTION=apply ENV=qa       # Uses .env.qa
make db-migration MIGRATION=001 ACTION=rollback
make db-migration MIGRATION=001 ACTION=status
```

#### Direct Script Usage
From the migrations directory:
```bash
# Apply migration 001
./run_migration.sh 001 apply

# Check if migration 001 is applied
./run_migration.sh 001 status

# Rollback migration 001 (revert to previous state)
./run_migration.sh 001 rollback
```

### Migration Script Features

- ✅ **Containerized**: Uses Docker/Podman with host networking for PostgreSQL client commands
- ✅ **No Dependencies**: No need to install `psql` locally
- ✅ **Transactional**: All changes applied in database transactions
- ✅ **Idempotent**: Safe to run multiple times
- ✅ **Tracked**: Migration status recorded in `schema_migrations` table
- ✅ **Verified**: Built-in verification of successful application
- ✅ **Checksums**: File integrity verification
- ✅ **Colored Output**: Easy-to-read console output
- ✅ **Auto-Detection**: Automatically detects Docker or Podman

## Migration Tracking

The system creates a `schema_migrations` table to track applied migrations:

```sql
CREATE TABLE schema_migrations (
    migration_number VARCHAR(10) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT,
    checksum VARCHAR(64)
);
```

## Safety Notes

⚠️ **Important Considerations**:

1. **Backup First**: Always backup your database before running migrations in production
2. **Test Environment**: Test migrations in a staging environment first
3. **VAT Data**: Migration 001 doesn't migrate existing data, only changes constraints
4. **Rollback Impact**: Rolling back may re-enable old validation that could fail with current data

## Manual Migration (Alternative)

If you prefer to run migrations manually using containers:

```bash
# Using Docker
docker run --rm --network=host -v "$(pwd):/migrations:ro" postgres:16-alpine \
  psql $DATABASE_URL -f /migrations/001_update_vat_constraints.sql

# Using Podman  
podman run --rm --network=host -v "$(pwd):/migrations:ro" postgres:16-alpine \
  psql $DATABASE_URL -f /migrations/001_update_vat_constraints.sql
```

## Troubleshooting

### Common Issues

**Database Connection Error**:
- Verify `DATABASE_URL` is correctly set
- Test connection: `./run_migration.sh 001 status` (will test connection)

**Permission Error**:
- Ensure database user has necessary permissions
- Migrations require ability to create/drop functions and triggers

**Migration Already Applied**:
- Check status: `./run_migration.sh 001 status`
- Force rollback first if needed

### Migration Verification

After running migration 001, verify the changes:

```sql
-- Check that per-entity triggers exist
SELECT tgname, tgrelid::regclass 
FROM pg_trigger 
WHERE tgname LIKE 'trg_check_vat_%';

-- Should show 3 triggers:
-- trg_check_vat_distributors | distributors
-- trg_check_vat_resellers    | resellers  
-- trg_check_vat_customers    | customers

-- Test VAT uniqueness (should work - same VAT in different entity types)
INSERT INTO distributors (id, name, custom_data) VALUES ('test1', 'Test', '{"vat": "12345678901"}');
INSERT INTO customers (id, name, custom_data) VALUES ('test2', 'Test', '{"vat": "12345678901"}');
-- This should succeed after migration 001

-- Cleanup test data
DELETE FROM distributors WHERE id = 'test1';
DELETE FROM customers WHERE id = 'test2';
```

## Development

### Adding New Migrations

1. Create new migration files with incremented number:
   - `002_your_migration_name.sql`
   - `002_your_migration_name_rollback.sql`

2. Update the migration runner script if needed

3. Test thoroughly in development environment

4. Document the migration in this README