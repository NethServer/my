# Database Migrations

This directory contains database migration scripts for the My Nethesis backend.

## Overview

Migrations are versioned SQL scripts that allow you to evolve your database schema over time while keeping track of changes. Each migration has a forward migration (apply) and a rollback migration (undo).

## ⚠️ Hard rule: every migration MUST be reflected in `schema.sql`

`backend/database/schema.sql` is the cumulative head state of the database. The backend applies it on boot when it detects an empty database (`database.Init`), so a fresh dev environment starts at "head" without running any migration.

To keep that boot path correct, **whenever you add a migration here you also fold its effect into `schema.sql` in the same PR** (the CREATE TABLE / ALTER / CREATE INDEX / COMMENT statements). The migration file remains the source of truth for the delta; `schema.sql` is the source of truth for "what the DB looks like right now".

### Why this matters

The migration runner is idempotent against schema.sql via a baseline step: if it sees a populated DB with an empty `schema_migrations` table, it marks every on-disk migration as applied without running it (otherwise non-idempotent statements like `CREATE OR REPLACE VIEW` on a `MATERIALIZED VIEW` would fail — exactly what 010 does on top of 012).

**The trade-off:** if you add a migration on disk but forget to fold it into `schema.sql`, a fresh-init flow (`make dev-up` → `make run` → `make db-migrate`) will silently mark your migration as applied without ever creating the object. The bug only surfaces at runtime when something queries the missing table.

Always update both in the same PR. The check is on you — the runner cannot tell the difference between "schema.sql has it" and "schema.sql is stale".

## Available Migrations

No migrations are currently defined. When migrations are added, they will be documented here.

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
3. **Rollback Impact**: Rolling back may fail if current data violates old constraints

## Manual Migration (Alternative)

If you prefer to run migrations manually using containers:

```bash
# Using Docker
docker run --rm --network=host -v "$(pwd):/migrations:ro" postgres:16-alpine \
  psql $DATABASE_URL -f /migrations/001_your_migration.sql

# Using Podman
podman run --rm --network=host -v "$(pwd):/migrations:ro" postgres:16-alpine \
  psql $DATABASE_URL -f /migrations/001_your_migration.sql
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

## Development

### Adding New Migrations

1. Create new migration files with incremented number:
   - `001_your_migration_name.sql`
   - `001_your_migration_name_rollback.sql`

2. Update the migration runner script if needed

3. Test thoroughly in development environment

4. Document the migration in this README under "Available Migrations"

### Example Migration

**Forward migration** (`001_add_user_status.sql`):
```sql
-- Migration 001: Add status column to users
-- Description: Adds a status column to track user account state

ALTER TABLE users ADD COLUMN status VARCHAR(20) DEFAULT 'active';
CREATE INDEX idx_users_status ON users(status);
```

**Rollback migration** (`001_add_user_status_rollback.sql`):
```sql
-- Rollback Migration 001: Remove status column from users

DROP INDEX IF EXISTS idx_users_status;
ALTER TABLE users DROP COLUMN IF EXISTS status;
```