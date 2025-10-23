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
go install github.com/nethesis/sync/cmd/sync@latest
```

## Quick Start

### Prerequisites
- Go 1.23+
- Logto instance with M2M app configured

### Setup

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

# Database connection (required for pull command)
DATABASE_URL=postgresql://user:password@localhost:5432/database
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
   - Start your backend: `cd backend && make dev`

### Manual RBAC Sync

```bash
# Create configuration
cp configs/config.yml my-config.yml

# Preview changes
./build/sync sync -c my-config.yml --dry-run --verbose

# Apply configuration
./build/sync sync -c my-config.yml
```

## Usage

### Basic Commands

```bash
# Complete Logto initialization
sync init --tenant-id your-tenant-id --backend-app-id your-backend-app-id --backend-app-secret your-secret --logto-domain your-domain.com --app-url https://your-app.com --owner-username owner --owner-email owner@example.com

# RBAC sync with default config
sync sync

# Sync with specific config file
sync sync -c configs/config.yml

# Dry run to preview changes
sync sync --dry-run --verbose

# Pull from Logto to local database (disaster recovery)
sync pull --verbose

# Clean up all test data from Logto and database
sync prune

# Output results in JSON format
sync sync --output json
```

### Init Command

Zero-to-production setup:

```bash
# Basic initialization
sync init \
  --tenant-id your-tenant-id \
  --backend-app-id your-backend-app-id \
  --backend-app-secret your-secret-here \
  --logto-domain your-domain.com \
  --app-url https://your-app.com

# Complete setup with user details
sync init \
  --tenant-id your-tenant-id \
  --backend-app-id your-backend-app-id \
  --backend-app-secret your-secret-here \
  --logto-domain your-domain.com \
  --app-url https://your-app.com \
  --owner-username owner \
  --owner-email owner@example.com \
  --owner-name "System Owner"

# Output formats
sync init --output json > setup-result.json
sync init --force  # Force re-initialization
```

**What Init Does:**
1. Creates custom domain in Logto
2. Verifies backend M2M application
3. Creates frontend SPA application
4. Creates owner user with secure password
5. Sets up complete RBAC system
6. Configures Multi-Factor Authentication (MFA) with TOTP as mandatory
7. Generates all environment variables

### Sync Command

RBAC configuration management:

```bash
# Basic sync
sync sync -c config.yml

# Dry run with verbose output
sync sync -c config.yml --dry-run --verbose

# Cleanup unused resources
sync sync -c config.yml --cleanup --dry-run
sync sync -c config.yml --cleanup

# Skip specific operations
sync sync --skip-resources --skip-roles
```

### Pull Command

Disaster recovery and reverse synchronization:

```bash
# Basic pull from Logto to local database
sync pull

# Pull with verbose output
sync pull --verbose

# Dry run to see what would be pulled
sync pull --dry-run --verbose

# Output results in JSON format
sync pull --output json
```

**What Pull Does:**
1. Fetches organizations from Logto (excludes Owner)
2. Creates/updates distributors, resellers, and customers in local database
3. Fetches users from Logto (excludes Owner users)
4. Creates/updates users with correct organization assignments
5. Acknowledges roles, resources, and permissions (managed in Logto)

**Use Cases:**
- **Disaster Recovery**: Restore local database from Logto after data loss
- **Development Setup**: Populate local database with production data
- **Environment Sync**: Keep local database consistent with Logto

### Prune Command

Complete cleanup of test data from both Logto and local database:

```bash
# Interactive mode - prompts for confirmations
sync prune

# Preview what would be deleted without making changes
sync prune --dry-run

# Force mode - skip all confirmations (DANGEROUS)
sync prune --force
```

**What Prune Does:**
1. Fetches all organizations from Logto (across all pages)
2. Fetches all users from Logto (across all pages)
3. Deletes all organizations and users from Logto
   - Prompts for confirmation before deleting Owner organization
   - Prompts for confirmation before deleting owner user
4. Queries local database for active records
5. Permanently deletes all records from database tables:
   - distributors
   - resellers
   - customers
   - users

**Special Handling:**
- **Owner Organization**: Prompts for confirmation (or skipped in dry-run)
- **owner User**: Prompts for confirmation (or skipped in dry-run)
- **Force Mode**: Deletes everything without prompts
- **Dry Run**: Shows what would be deleted without making changes

**Use Cases:**
- **Test Cleanup**: Remove fake organizations and users after testing
- **Environment Reset**: Clean up development/staging environments
- **Pre-Production Setup**: Clear test data before production deployment

**Warning:** This operation is **IRREVERSIBLE**. All deleted data cannot be recovered. After pruning Owner/owner, run `sync init` to reinitialize.

### Global Flags

**Common Flags:**
- `--env-file`: Environment file to load (default: .env)
- `-v, --verbose`: Enable verbose output
- `-o, --output`: Output format (text, json, yaml)

**Init Command:**
- `--tenant-id`: Logto tenant identifier (required)
- `--backend-app-id`: M2M application ID (required)
- `--backend-app-secret`: M2M application secret (required)
- `--logto-domain`: Custom domain for Logto authentication (required, e.g., auth.yourcompany.com)
- `--app-url`: Frontend application URL (required, e.g., https://app.yourcompany.com)
- `--owner-username`: Owner user username (default: "owner")
- `--owner-email`: Owner user email (default: "owner@example.com")
- `--owner-name`: Owner user display name (default: "System Owner")
- `--force`: Force re-initialization

**Sync Command:**
- `-c, --config`: Configuration file path
- `--dry-run`: Preview changes only
- `--cleanup`: Remove undefined resources
- `--skip-resources`: Skip resource sync
- `--skip-roles`: Skip role sync

**Pull Command:**
- `--dry-run`: Preview changes only
- `DATABASE_URL`: PostgreSQL connection string (required)

**Prune Command:**
- `--dry-run`: Preview changes only (no deletions)
- `--force`: Skip all confirmations (DANGEROUS)
- `DATABASE_URL`: PostgreSQL connection string (optional, for database cleanup)

## Configuration

Simple YAML format with business/technical separation:

```yaml
metadata:
  name: "nethesis-rbac"
  version: "1.0.0"

# Business hierarchy (organization roles)
organization_roles:
  - id: owner
    name: "Owner"
    permissions:
      - id: create:distributors
      - id: manage:distributors

# Technical capabilities (user roles)
user_roles:
  - id: admin
    name: "Admin"
    permissions:
      - id: admin:systems
      - id: destroy:systems

# Available resources
resources:
  - name: "systems"
    actions: ["read", "manage", "admin", "destroy"]

# Third-party applications (optional)
third_party_apps:
    - name: "example.company.com"
      description: "Example third-party application"
      display_name: "Example App"
      # OAuth redirect URIs (required for authentication flow)
      redirect_uris:
        - "https://example.company.com/callback"
        - "https://example.company.com/auth/callback"
      # Post logout redirect URIs (optional)
      post_logout_redirect_uris:
        - "https://example.company.com"
        - "https://example.company.com/logout"
      # Access control configuration
      access_control:
        organization_ids:    # Which specific organization IDs can access this app (optional)
          - "org-12345"
          - "org-67890"
        organization_roles:  # Which organization roles can access this app
          - "owner"
          - "distributor"
        user_roles:          # Which user roles can access this app
          - "admin"
          - "support"
      # Optional custom scopes (default: profile, email, roles, urn:logto:scope:organizations, urn:logto:scope:organization_roles)
      # scopes:
      #   - "profile"
      #   - "email"
      #   - "roles"
      #   - "urn:logto:scope:organizations"
      #   - "urn:logto:scope:organization_roles"
      #   - "custom:scope"

# Sign-in experience configuration (optional)
sign_in_experience:
  # Brand colors (hex format)
  colors:
    primary_color: "#0069A8"
    primary_color_dark: "#0087DB"
    dark_mode_enabled: true

  # Branding assets (relative paths from config file directory)
  branding:
    logo_path: "sign-in/logo.png"
    logo_dark_path: "sign-in/logo-dark.png"
    favicon_path: "sign-in/favicon.ico"
    favicon_dark_path: "sign-in/favicon-dark.ico"

  # Custom CSS (relative path from config file directory)
  custom_css_path: "sign-in/default.css"

  # Language configuration
  language:
    auto_detect: true
    fallback_language: "en"

  # Sign-in methods configuration
  sign_in:
    methods:
      - identifier: "email"
        password: true
        verification_code: false
        is_password_primary: true

  # Sign-up configuration (disabled by default)
  sign_up:
    identifiers: []
    password: false
    verify: false
    secondary_identifiers: []

  # Social sign-in configuration (empty by default)
  social_sign_in: {}

# Connectors configuration (optional)
connectors:
  # SMTP email connector for password reset and verification emails
  smtp:
    # SMTP server configuration
    host: "smtp.your-provider.com"
    port: 587
    username: "your-smtp-username"
    password: "your-smtp-password"
    from_email: "no-reply@your-company.com"
    from_name: "Your Company Name"
    tls: true
    secure: false
    debug: false
    logger: true
    disable_file_access: true
    disable_url_access: true
    
    # Template variable settings for dynamic content replacement
    template_settings:
      company_name: "Your Company Name"
      support_email: "support@your-company.com"
    
    # Custom headers (optional)
    custom_headers: {}
```

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

## Examples

### Complete Setup Workflow
```bash
# 1. Initialize everything (see Init Command section above)
sync init --tenant-id your-tenant-id --backend-app-id your-app-id --backend-app-secret your-secret --logto-domain your-domain.com --app-url https://your-app.com

# 2. Copy environment variables to backend/.env
# 3. Start backend: cd backend && make dev
```

### RBAC Management
```bash
# Edit configuration
vim config.yml

# Preview changes
sync sync -c config.yml --dry-run --verbose

# Apply changes
sync sync -c config.yml
```

### Disaster Recovery
```bash
# Restore local database from Logto
DATABASE_URL="postgresql://user:pass@localhost:5432/db" sync pull --verbose

# Preview what would be restored
DATABASE_URL="postgresql://user:pass@localhost:5432/db" sync pull --dry-run

# Restore with structured output
DATABASE_URL="postgresql://user:pass@localhost:5432/db" sync pull --output json
```

### Data Cleanup
```bash
# Clean up test data (interactive mode)
sync prune

# Preview what would be deleted
sync prune --dry-run

# Force cleanup without confirmations
sync prune --force

# Cleanup with database connection
DATABASE_URL="postgresql://user:pass@localhost:5432/db" sync prune
```

### Automation
```bash
# JSON output for CI/CD
sync init --output json | jq '.backend_app.environment_vars'

# Extract environment variables
sync init --output json | jq -r '.backend_app.environment_vars | to_entries[] | "\(.key)=\(.value)"' > backend/.env

# Use different environment files in CI/CD
sync init --env-file .env.staging --output json
sync sync --env-file .env.production --dry-run

# Disaster recovery in CI/CD
DATABASE_URL="$DATABASE_URL" sync pull --output json --env-file .env.production
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