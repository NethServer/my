# sync

CLI tool for complete Logto setup and RBAC synchronization. Provides zero-to-production deployment and ongoing management of Role-Based Access Control (RBAC) configuration.

## Key Features

### Complete Zero-to-Production Setup
- **`sync init`**: Complete Logto initialization from scratch
- **Auto-Configuration**: Generates all environment variables automatically
- **Full Setup**: Custom domains, applications, users, and complete RBAC
- **Security First**: Secure password generation and JIT provisioning

### RBAC Synchronization
- **Simplified RBAC Sync**: Clear separation between business hierarchy and technical capabilities
- **Business Hierarchy**: Organization roles (Owner, Distributor, Reseller, Customer)
- **Technical Capabilities**: User roles (Admin, Support)
- **Third-Party Apps**: Automatic creation and management of external applications
- **Dry Run Mode**: Preview changes before applying
- **Cleanup Mode**: Remove resources/roles not in config

### Disaster Recovery
- **`sync pull`**: Reverse sync from Logto to local database
- **Owner Exclusion**: Automatically excludes Owner organizations and users (Logto-only)
- **Database Restoration**: Restores organizations and users from Logto for disaster recovery

### Data Cleanup
- **`sync prune`**: Complete cleanup of test data from Logto and database
- **Interactive Mode**: Prompts for confirmation before deleting Owner/owner
- **Force Mode**: Skip all confirmations for automated cleanup
- **Dry Run**: Preview what would be deleted without making changes

### Enterprise Features
- **Multiple Output Formats**: Text, JSON, and YAML
- **Safe Operations**: Preserves system entities and validates configurations
- **CI/CD Ready**: Structured output for automation

## Installation

```bash
# From source
make build

# Using Go
go install github.com/nethesis/my/sync/cmd/sync@latest
```

## Quick Start

### Prerequisites
- Go 1.24+
- Logto instance with M2M app configured

### Setup

> **Note:** The `pull` and `prune` commands require PostgreSQL running (start with `cd backend && make dev-up`).

```bash
# Setup development environment
make dev-setup

# Build the sync tool
make build

# Run sync commands
./build/sync --help
```

### Environment Configuration

The sync tool loads environment variables from a `.env` file by default. You can specify a different environment file using the `--env-file` flag.

```bash
# Use default .env file
./build/sync sync

# Use custom environment file
./build/sync sync --env-file .env.production
./build/sync init --env-file .env.staging
./build/sync pull --env-file .env.production
```

#### Required Environment Variables
```bash
# Logto tenant configuration
TENANT_ID=your-tenant-id
TENANT_DOMAIN=your-domain.com

# App URL configuration (required for init command)
APP_URL=https://your-app-domain.com

# Logto Management API (M2M app credentials)
BACKEND_APP_ID=your-backend-m2m-app-id
BACKEND_APP_SECRET=your-backend-m2m-app-secret

# Database connection (required for pull and prune commands)
DATABASE_URL=postgresql://noc_user:noc_password@localhost:5432/noc?sslmode=disable
```

### Complete Setup (Recommended)

1. **Create M2M Application in Logto**
   - Go to Logto Admin Console → Applications → Machine-to-Machine
   - Create app with full Management API access
   - Copy App ID and Secret

2. **Run Complete Initialization**
   ```bash
   make build

   ./build/sync init \
     --tenant-id your-tenant-id \
     --backend-app-id your-backend-app-id \
     --backend-app-secret your-secret-here \
     --logto-domain your-domain.com \
     --app-url https://your-app.com \
     --owner-username owner \
     --owner-email owner@example.com \
     --owner-name "System Owner"
   ```

3. **Copy Environment Variables**
   - Copy the auto-generated environment variables to your backend `.env`
   - Start your backend: `cd backend && make run`

### Manual RBAC Sync

```bash
# Create configuration
cp configs/config.yml my-config.yml

# Preview changes
./build/sync sync -c my-config.yml --dry-run --verbose

# Apply configuration
./build/sync sync -c my-config.yml
```

## Commands

All commands support `--dry-run`, `--verbose`, and `--output json|yaml|text` flags.
Run `./build/sync <command> --help` for the full flag reference.

### init

Complete Logto initialization (custom domain, M2M app, frontend SPA, owner user, RBAC, MFA):

```bash
./build/sync init \
  --tenant-id your-tenant-id \
  --backend-app-id your-backend-app-id \
  --backend-app-secret your-secret-here \
  --logto-domain your-domain.com \
  --app-url https://your-app.com \
  --owner-username owner \
  --owner-email owner@example.com

# Force re-initialization
./build/sync init --force ...
```

### sync

Push RBAC configuration from YAML to Logto:

```bash
./build/sync sync                                    # Default config
./build/sync sync -c configs/config.yml --dry-run    # Preview changes
./build/sync sync --cleanup                          # Remove undefined resources
```

### pull

Reverse sync from Logto to local database (disaster recovery):

```bash
./build/sync pull --verbose
./build/sync pull --dry-run
```

### prune

Delete all test data from Logto and local database:

```bash
./build/sync prune              # Interactive (prompts for Owner/owner)
./build/sync prune --dry-run    # Preview only
./build/sync prune --force      # Skip all confirmations (DANGEROUS, IRREVERSIBLE)
```

After pruning Owner/owner, run `sync init` to reinitialize.

## Configuration

RBAC configuration uses a YAML file with business/technical separation.
See [`configs/config.yml`](configs/config.yml) for the full annotated example.

Key sections:
- `organization_roles` - Business hierarchy (Owner, Distributor, Reseller, Customer)
- `user_roles` - Technical capabilities (Admin, Support)
- `resources` - API resources and actions
- `third_party_apps` - External application access control (optional)
- `sign_in_experience` - Branding, colors, sign-in methods (optional)
- `connectors` - SMTP email connector for password reset (optional)

## Development

### Development Commands
```bash
# Setup development environment
make dev-setup

# Build and run
make build
make run init
make run sync

# Code quality
make fmt
make lint

# Testing
make test
make test-coverage

# Test specific package
go test ./internal/config
```

## CI/CD Integration

```bash
# JSON output for automation
./build/sync init --output json | jq '.backend_app.environment_vars'

# Extract environment variables directly
./build/sync init --output json | jq -r '.backend_app.environment_vars | to_entries[] | "\(.key)=\(.value)"' > backend/.env

# Use different environment files
./build/sync sync --env-file .env.production --dry-run
```

## Project Structure
```
sync/
├── cmd/sync/           # CLI entry point
├── internal/
│   ├── cli/            # Command implementations
│   ├── client/         # Logto API client
│   ├── config/         # Configuration loading
│   ├── logger/         # Structured logging
│   └── sync/           # Synchronization engine
├── configs/            # Example configurations
└── Makefile           # Build automation
```

## Related
- [openapi.yaml](../backend/openapi.yaml) - API specification
- [Backend](../backend/README.md) - API server
- [Collect](../collect/README.md) - Collect server
- [Project Overview](../README.md) - Main documentation