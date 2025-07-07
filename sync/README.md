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
     --domain your-domain.com \
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
sync init --tenant-id your-tenant-id --backend-app-id your-backend-app-id --backend-app-secret your-secret --domain your-domain.com --owner-username owner --owner-email owner@example.com

# RBAC sync with default config
sync sync

# Sync with specific config file
sync sync -c configs/config.yml

# Dry run to preview changes
sync sync --dry-run --verbose

# Output results in JSON format
sync sync --output json
```

### Init Command

Zero-to-production setup:

```bash
# CLI flags (recommended)
sync init \
  --tenant-id your-tenant-id \
  --backend-app-id your-backend-app-id \
  --backend-app-secret your-secret-here \
  --domain your-domain.com  # Custom domain for Logto authentication

# JSON output for automation
sync init --output json > setup-result.json

# Force re-initialization
sync init --force
```

**What Init Does:**
1. Creates custom domain in Logto
2. Verifies backend M2M application
3. Creates frontend SPA application
4. Creates owner user with secure password
5. Sets up complete RBAC system
6. Generates all environment variables

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

### Global Flags

**Init Command:**
- `--tenant-id`: Logto tenant identifier (required)
- `--backend-app-id`: M2M application ID (required)
- `--backend-app-secret`: M2M application secret (required)
- `--domain`: Custom domain for Logto authentication (required, e.g., auth.yourcompany.com)
- `--owner-username`: Owner user username (default: "owner")
- `--owner-email`: Owner user email (default: "owner@example.com")
- `--owner-name`: Owner user display name (default: "System Owner")
- `--force`: Force re-initialization
- `-o, --output`: Output format (text, json, yaml)

**Sync Command:**
- `-c, --config`: Configuration file path
- `--dry-run`: Preview changes only
- `--verbose`: Enable verbose output
- `--cleanup`: Remove undefined resources
- `--skip-resources`: Skip resource sync
- `--skip-roles`: Skip role sync

## Configuration

Simple YAML format with business/technical separation:

```yaml
metadata:
  name: "nethesis-rbac"
  version: "1.0.0"

hierarchy:
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
      # Optional custom scopes (default: profile, email, roles, urn:logto:scope:organizations, urn:logto:scope:organization_roles)
      # scopes:
      #   - "profile"
      #   - "email"
      #   - "roles"
      #   - "urn:logto:scope:organizations"
      #   - "urn:logto:scope:organization_roles"
      #   - "custom:scope"
```

## Development

### Commands
```bash
# Setup development environment
make dev-setup

# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build
make build

# Run with example config
make run-example
```

### Testing
```bash
# Run all tests
make test

# Test coverage
make test-coverage

# Test specific package
go test ./internal/config
```

## Examples

### Complete Setup Workflow
```bash
# 1. Initialize everything
sync init \
  --tenant-id your-tenant-id \
  --backend-app-id your-backend-app-id \
  --backend-app-secret your-secret-here \
  --domain your-domain.com \
  --owner-username owner \
  --owner-email owner@example.com \
  --owner-name "System Owner"

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

### Automation
```bash
# JSON output for CI/CD
sync init --output json | jq '.backend_app.environment_vars'

# Extract environment variables
sync init --output json | jq -r '.backend_app.environment_vars | to_entries[] | "\(.key)=\(.value)"' > backend/.env
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
- [Backend](../backend/README.md) - API server
- [Project Overview](../README.md) - Main documentation