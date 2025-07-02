# sync

A comprehensive CLI tool for **complete Logto setup** and **RBAC synchronization**. Provides zero-to-production deployment and ongoing management of simplified Role-Based Access Control (RBAC) configuration with Logto identity provider.

## 🚀 Key Features

### **Complete Zero-to-Production Setup**
- 🚀 **`sync init`**: Complete Logto initialization from scratch
- ⚡ **Auto-Configuration**: Generates all environment variables automatically
- 🏗️ **Full Setup**: Custom domains, applications, users, and complete RBAC
- 🔐 **Security First**: Secure password generation and JIT provisioning
- 📋 **Multiple Modes**: CLI flags, environment variables, or JSON output

### **RBAC Synchronization**
- 🔄 **Simplified RBAC Sync**: Clear separation between business hierarchy and technical capabilities
- 🏢 **Business Hierarchy**: Organization roles (God, Distributor, Reseller, Customer) for commercial logic
- 👥 **Technical Capabilities**: User roles (Admin, Support) for skills
- 🔗 **Backend Integration**: Powers real-time Management API data fetching in backend
- 🔍 **Dry Run Mode**: Preview changes before applying them with detailed analysis
- 🧹 **Cleanup Mode**: Remove resources/roles/permissions not defined in config (opt-in)

### **Enterprise Features**
- 📊 **Multiple Output Formats**: Text, JSON, and YAML with structured environment variables
- 🛡️ **Safe Operations**: Preserves system entities and validates configurations
- 🔧 **Simplified Configuration**: YAML-based with clear business vs technical separation
- 📝 **Structured Logging**: Zerolog-based logging with component isolation
- 🔍 **Security Features**: Automatic sensitive data redaction in logs
- 🎯 **Output Separation**: Clean command output (stdout) separate from logging (stderr)
- 🏗️ **CI/CD Ready**: Structured JSON/YAML output with organized environment variables

## System Requirements

### Runtime (Binary Execution)
- **Any 64-bit OS**: Linux, macOS, Windows
- **No dependencies** - Statically compiled Go binary

### Development & Building
- **Go 1.21+** - [Download from golang.org](https://golang.org/download/)
- **Make** (for build automation):
  - **macOS**: Preinstalled with Xcode Command Line Tools (`xcode-select --install`)
  - **Linux**: Usually preinstalled, or install with package manager (`apt install build-essential`)
  - **Windows**: Install via [Git Bash](https://git-scm.com/download/win), [WSL2](https://docs.microsoft.com/en-us/windows/wsl/install), or [Chocolatey](https://chocolatey.org/) (`choco install make`)
- **golangci-lint** (optional, for linting): [Installation guide](https://golangci-lint.run/usage/install/)

### External Dependencies
- **Logto instance** - Identity provider with Management API access
- **Management API credentials** - Machine-to-Machine app with full Management API permissions

## Installation

### From Source

```bash
git clone https://github.com/nethesis/sync.git
cd sync
make build
```

### Using Go Install

```bash
go install github.com/nethesis/sync/cmd/sync@latest
```

### Alternative Build Methods
If Make is not available, you can use Go commands directly:
```bash
# Development
go run ./cmd/sync

# Build
mkdir -p build && go build -o build/sync ./cmd/sync

# Test
go test ./...
```

## 🚀 Quick Start

### **Complete Setup (Recommended)**

**Zero to production in 3 steps:**

1. **Create M2M Application in Logto**
   - Go to Logto Admin Console → Applications → Machine-to-Machine
   - Create app named `backend` with full Management API access
   - Copy App ID and Secret

2. **Run Complete Initialization**
   ```bash
   make build

   ./build/sync init \
     --tenant-id your-tenant-id \
     --backend-client-id your-backend-client-id \
     --backend-client-secret your-secret-here \
     --domain your-domain.com \
     --admin-username admin \
     --admin-email admin@example.com \
     --admin-name "System Administrator"
   ```

3. **Copy Environment Variables**
   - Copy the auto-generated environment variables to your backend `.env`
   - Start your backend: `cd backend && go run main.go`
   - Done! Your system is fully configured.

### **Alternative: Environment Variables Mode**

```bash
export TENANT_ID=your-tenant-id
export BACKEND_CLIENT_ID=your-backend-client-id
export BACKEND_CLIENT_SECRET=your-secret-here
export TENANT_DOMAIN=your-domain.com

./build/sync init
```

### **Manual RBAC Sync**

If you already have a configured Logto instance:

1. **Setup Environment Variables**
   ```bash
   cp .env.example .env
   # Edit .env with your Logto configuration
   ```

2. **Create Configuration File**
   ```bash
   cp configs/config.yml my-config.yml
   # Edit my-config.yml with your RBAC configuration
   ```

3. **Test Configuration (Dry Run)**
   ```bash
   sync sync -c my-config.yml --dry-run --verbose
   ```

4. **Apply Configuration**
   ```bash
   sync sync -c my-config.yml
   ```

## Usage

### Basic Commands

```bash
# Show help
sync --help

# Show version
sync --version

# Complete Logto initialization
sync init --tenant-id your-tenant-id --backend-client-id your-backend-client-id --backend-client-secret your-secret --domain your-domain.com

# Force re-initialization
sync init --force

# JSON/YAML output for automation
sync init --output json
sync init --output yaml

# RBAC sync with default config
sync sync

# Sync with specific config file
sync sync -c configs/config.yml

# Dry run to preview changes
sync sync --dry-run --verbose

# Output results in JSON format
sync sync --output json

# Skip specific sync phases
sync sync --skip-resources --skip-roles
```

### 🚀 Init Command (Zero-to-Production Setup)

The `init` command provides complete Logto setup from scratch:

#### **Basic Usage**

```bash
# CLI flags (recommended for CI/CD)
sync init \
  --tenant-id your-tenant-id \
  --backend-client-id your-backend-client-id \
  --backend-client-secret your-secret-here \
  --domain your-domain.com

# Environment variables
export TENANT_ID=your-tenant-id
export BACKEND_CLIENT_ID=your-backend-client-id
export BACKEND_CLIENT_SECRET=your-secret-here
export TENANT_DOMAIN=your-domain.com
sync init

# JSON output for automation
sync init --output json > setup-result.json

# YAML output for automation
sync init --output yaml > setup-result.yaml
```

#### **What Init Command Does**

1. ✅ **Creates custom domain** in Logto (e.g., `your-domain.com`)
2. ✅ **Verifies backend M2M application** exists with correct permissions
3. ✅ **Creates frontend SPA application** with correct redirect URIs:
   - Development: `http://localhost:5173/callback`
   - Production: `https://your-domain.com/callback`
4. ✅ **Creates admin user** with configurable credentials and generated secure password
5. ✅ **Sets up complete RBAC system**:
   - Organization scopes (create:distributors, manage:resellers, etc.)
   - Organization roles (God, Distributor, Reseller, Customer)
   - User roles (Admin, Support)
   - JIT (Just-in-Time) provisioning
6. ✅ **Assigns roles to admin user**: Admin + God organization role
7. ✅ **Generates all environment variables** automatically

#### **JSON/YAML Output Structure**

The init command supports structured output perfect for automation and CI/CD:

```json
{
  "backend_app": {
    "id": "your-backend-app-id",
    "name": "backend",
    "type": "MachineToMachine",
    "client_id": "your-backend-app-id",
    "client_secret": "your-generated-client-secret",
    "environment_vars": {
      "LOGTO_ISSUER": "https://your-tenant-id.logto.app",
      "LOGTO_AUDIENCE": "https://your-domain.com/api",
      "LOGTO_JWKS_ENDPOINT": "https://your-tenant-id.logto.app/oidc/jwks",
      "JWT_SECRET": "your-generated-jwt-secret",
      "JWT_ISSUER": "your-domain.com.api",
      "JWT_EXPIRATION": "24h",
      "JWT_REFRESH_EXPIRATION": "168h",
      "LOGTO_MANAGEMENT_CLIENT_ID": "your-backend-app-id",
      "LOGTO_MANAGEMENT_CLIENT_SECRET": "your-generated-client-secret",
      "LOGTO_MANAGEMENT_BASE_URL": "https://your-tenant-id.logto.app/api",
      "LISTEN_ADDRESS": "127.0.0.1:8080"
    }
  },
  "frontend_app": {
    "id": "your-frontend-app-id",
    "name": "frontend",
    "type": "SPA",
    "client_id": "your-frontend-app-id",
    "environment_vars": {
      "VITE_LOGTO_ENDPOINT": "https://your-tenant-id.logto.app",
      "VITE_LOGTO_APP_ID": "your-frontend-app-id",
      "VITE_LOGTO_RESOURCES": "[\"https://your-domain.com/api\"]",
      "VITE_API_BASE_URL": "https://your-domain.com/api"
    }
  },
  "admin_user": {
    "id": "your-admin-user-id",
    "username": "admin",
    "email": "admin@example.com",
    "password": "your-generated-password"
  },
  "custom_domain": "your-domain.com",
  "generated_jwt_secret": "your-generated-jwt-secret",
  "already_initialized": false,
  "tenant_info": {
    "tenant_id": "your-tenant-id",
    "base_url": "https://your-tenant-id.logto.app",
    "mode": "env"
  },
  "next_steps": [
    "Copy the environment variables to your .env files",
    "Start your backend: cd backend && go run main.go",
    "Start your frontend with the Logto configuration",
    "Login with the admin credentials provided",
    "Use 'sync sync' to update RBAC configuration when needed"
  ]
}
```

**🚀 Automation Examples:**

```bash
# Extract backend environment variables
jq -r '.backend_app.environment_vars | to_entries[] | "\(.key)=\(.value)"' setup-result.json > backend/.env

# Extract frontend environment variables
jq -r '.frontend_app.environment_vars | to_entries[] | "\(.key)=\(.value)"' setup-result.json > frontend/.env

# Get god user credentials
jq -r '.god_user | "Username: \(.username)\nEmail: \(.email)\nPassword: \(.password)"' setup-result.json

# Check if already initialized
jq -r '.already_initialized' setup-result.json

# Get application IDs for further configuration
jq -r '.backend_app.client_id' setup-result.json
jq -r '.frontend_app.client_id' setup-result.json
```

#### **Environment Variables Generated (Text Output)**

The init command outputs all required environment variables:

```bash
# Backend configuration
LOGTO_ISSUER=https://your-tenant-id.logto.app
LOGTO_AUDIENCE=https://your-domain.com/api
LOGTO_JWKS_ENDPOINT=https://your-tenant-id.logto.app/oidc/jwks
JWT_SECRET=generated-32-char-secret
LOGTO_MANAGEMENT_CLIENT_ID=your-backend-client-id
LOGTO_MANAGEMENT_CLIENT_SECRET=your-secret-here
LOGTO_MANAGEMENT_BASE_URL=https://your-tenant-id.logto.app

# Frontend configuration
FRONTEND_LOGTO_ENDPOINT=https://your-tenant-id.logto.app
FRONTEND_LOGTO_APP_ID=generated-app-id
API_BASE_URL=https://your-domain.com/api
```

#### **Init Command Flags**

- `--tenant-id`: Logto tenant identifier (e.g., `your-tenant-id`)
- `--backend-client-id`: M2M application ID (e.g., `your-backend-client-id`)
- `--backend-client-secret`: M2M application secret
- `--domain`: Your custom domain (e.g., `your-domain.com`)
- `--force`: Force re-initialization even if already done
- `--output`: Output format (text, json, yaml) - default: text

#### **Troubleshooting Init**

```bash
# Verbose output for debugging
sync init --verbose

# Check if already initialized
sync init  # Will detect existing setup and suggest sync instead

# Force complete re-initialization
sync init --force

# JSON output for analysis
sync init --output json | jq .

# YAML output for analysis
sync init --output yaml
```

### Advanced Operations

#### **🔍 Dry Run Mode**

Preview what would be changed without making any modifications:

```bash
# Basic dry run
sync sync -c config.yml --dry-run

# Verbose dry run with detailed logs
sync sync -c config.yml --dry-run --verbose

# Dry run with JSON output for analysis
sync sync -c config.yml --dry-run --output json | jq .

# Preview cleanup operations
sync sync -c config.yml --cleanup --dry-run --verbose
```

**Dry run shows you:**
- 📊 Resources that would be created/updated/deleted
- 📊 Roles that would be created/updated
- 📊 Permissions and scopes that would be assigned/removed
- 📊 Cleanup operations that would be performed

#### **🧹 Cleanup Mode**

Remove resources, roles, and scopes that are no longer defined in your configuration:

```bash
# Preview what would be cleaned up
sync sync -c config.yml --cleanup --dry-run

# Perform cleanup (removes items not in config)
sync sync -c config.yml --cleanup

# Cleanup with verbose logging
sync sync -c config.yml --cleanup --verbose
```

**⚠️ Cleanup Safety Features:**
- **Opt-in only**: Must explicitly use `--cleanup` flag
- **System protection**: Never removes Logto system resources
- **Management API protection**: Preserves Logto Management API resources
- **Default resource protection**: Skips resources marked as `IsDefault: true`

**What cleanup removes:**
- ❌ Custom resources not in your YAML config
- ❌ User-created roles not in your config
- ❌ Organization roles not in your config
- ❌ Scopes not defined in your resources

**What cleanup preserves:**
- ✅ Logto Management API resources
- ✅ System default resources
- ✅ Built-in Logto roles and scopes

### Global Flags

#### **Init Command Flags**
- `--tenant-id`: Logto tenant identifier (required)
- `--backend-client-id`: M2M application client ID (required)
- `--backend-client-secret`: M2M application client secret (required)
- `--domain`: Custom domain for your deployment (required)
- `--admin-username`: Admin user username (default: "admin")
- `--admin-email`: Admin user email (default: "admin@example.com")
- `--admin-name`: Admin user display name (default: "System Administrator")
- `--force`: Force re-initialization even if already done
- `-o, --output`: Output format (text, json, yaml) - default: text
- `-v, --verbose`: Enable verbose output

#### **Sync Command Flags**
- `-c, --config`: Configuration file path (default: ./config.yml)
- `-v, --verbose`: Enable verbose output (equivalent to LOG_LEVEL=debug)
- `--dry-run`: Show what would be done without making changes
- `-o, --output`: Output format (text, json, yaml)
- `--skip-resources`: Skip synchronizing resources
- `--skip-roles`: Skip synchronizing roles
- `--skip-permissions`: Skip synchronizing permissions
- `--cleanup`: Remove resources/roles/scopes not defined in config (**DANGEROUS**)
- `--force`: Force synchronization even if validation fails

## Logging & Output

The sync tool features structured logging with clear separation between operational logs and command results.

### Log Levels

Configure logging verbosity using environment variables or command flags:

```bash
# Environment variable (recommended)
export LOG_LEVEL=debug    # Maximum detail for debugging
export LOG_LEVEL=info     # Standard information (default)
export LOG_LEVEL=warn     # Only warnings and errors
export LOG_LEVEL=error    # Only errors
export LOG_LEVEL=fatal    # Only fatal errors

# Command flag (sets LOG_LEVEL=debug)
sync sync --verbose

# Priority: LOG_LEVEL env var > --verbose flag > info (default)
```

### Output Streams

The tool separates operational logs from command results:

**📋 Logs (stderr) - Operational Information:**
- Timestamped structured logs
- Component-specific context
- API call tracking with timing
- Configuration loading status
- Sync operation progress

**📊 Results (stdout) - Command Output:**
- Synchronization summary
- Operation statistics
- Formatted results (text/json/yaml)
- Success/failure status

### Practical Examples

```bash
# Only see results (hide logs)
sync sync --dry-run 2>/dev/null

# Only see logs (hide results)
sync sync --dry-run >/dev/null

# Save logs to file, show results on screen
sync sync --dry-run 2>sync.log

# Separate logs and results to different files
sync sync --dry-run >results.txt 2>logs.txt

# Debug mode with structured output
LOG_LEVEL=debug sync sync --output json | jq '.summary'

# Production monitoring (error logs only)
LOG_LEVEL=error sync sync 2>>production.log
```

### Log Features

**🔒 Security:**
- Automatic redaction of sensitive data (tokens, passwords, secrets)
- Safe logging of API endpoints and request bodies
- Pattern-based detection of credentials

**📊 Structured Data:**
- Component isolation (api-client, sync, config)
- API call tracking with HTTP status and timing
- Sync operation results with success/failure status
- Configuration validation with resource/role counts

**🎯 Structured Output:**
- RFC3339 timestamps
- Consistent field naming
- Machine-readable format
- Human-friendly console display

### Sample Structured Logs

```
2025-06-28T21:48:45+02:00 INF Configuration loaded component=config path=hierarchy.yml resources=4 roles=6 valid=true service=sync-tool
2025-06-28T21:48:45+02:00 DBG API call completed component=api-client duration=156ms endpoint=/api/resources method=GET service=sync-tool status_code=200
2025-06-28T21:48:46+02:00 INF Sync operation completed action=create entity=api-users operation=resource service=sync-tool success=true
```

### Sync Command Flags

- `--skip-resources`: Skip synchronizing resources
- `--skip-roles`: Skip synchronizing roles
- `--skip-permissions`: Skip synchronizing permissions
- `--cleanup`: Remove resources/roles/scopes not defined in config (**DANGEROUS**)
- `--force`: Force synchronization even if validation fails

## Configuration Format

The configuration file uses simplified YAML format with clear separation between business and technical roles:

```yaml
metadata:
  name: "nethesis-rbac"
  version: "1.0.0"
  description: "Nethesis Role-Based Authentication with clear separation between business hierarchy and technical capabilities"

hierarchy:
  # BUSINESS HIERARCHY (Organization Types)
  # Users inherit these based on their organization's role in the commercial chain
  organization_roles:
    - id: god
      name: "God"
      priority: 1
      type: user
      permissions:
        - id: create:distributors
        - id: manage:distributors
        - id: create:resellers
        - id: manage:resellers
        - id: create:customers
        - id: manage:customers

    - id: distributor
      name: "Distributor"
      priority: 2
      type: user
      permissions:
        - id: create:resellers
        - id: manage:resellers
        - id: create:customers
        - id: manage:customers

  # TECHNICAL CAPABILITIES (User Skills)
  # Independent of business hierarchy - define technical operations
  user_roles:
    - id: admin
      name: "Admin"
      priority: 1
      type: user
      permissions:
        - id: admin:systems
        - id: manage:systems
        - id: destroy:systems
        - id: destroy:systems

    - id: support
      name: "Support"
      priority: 2
      type: user
      permissions:
        - id: manage:systems
        - id: read:systems

  # Resources and their available actions
  resources:
    - name: "systems"
      actions: ["read", "manage", "admin", "destroy"]
    - name: "distributors"
      actions: ["create", "manage"]
    - name: "resellers"
      actions: ["create", "manage"]
    - name: "customers"
      actions: ["create", "manage"]

```

## Development

### Prerequisites

See [System Requirements](#system-requirements) section above for detailed information about:
- Go 1.21+ installation
- Make setup per operating system
- Optional golangci-lint for code quality

### Setup Development Environment

```bash
make dev-setup
```

### Available Make Targets

```bash
make help                # Show all available targets
make build               # Build the binary
make test                # Run tests
make test-coverage       # Run tests with coverage
make lint                # Run linter
make fmt                 # Format code
make clean               # Clean build artifacts
make run ARGS="--help"   # Run with arguments
make run-example         # Run with example config
```

### Testing

```bash
# Primary commands (recommended)
make test                          # Run all tests
make test-coverage                 # Run tests with coverage report
make fmt                           # Format code (required for CI)

# Direct Go commands for specific needs
go test ./internal/config          # Test config package only
go test ./internal/sync            # Test sync engine only
go test -v ./...                   # Verbose test output
go test -race ./...                # Race condition detection
```

Coverage reports are generated in `coverage.out` and uploaded as GitHub Actions artifacts for CI tracking.

### Pre-commit Workflow

Before committing changes, always run these commands to ensure CI passes:

```bash
make fmt                           # Format code (required for CI)
make test                          # Run all tests
make lint                          # Run linter (if golangci-lint installed)

# Then commit your changes
git add .
git commit -m "your commit message"
```

**Note**: The CI pipeline will fail if code is not properly formatted with `gofmt -s`.

### Project Structure

```
├── cmd/sync/             # CLI entry point
├── internal/
│   ├── cli/              # CLI commands and flags
│   ├── client/           # Logto API client with structured logging
│   ├── config/           # Configuration loading and validation
│   ├── constants/        # Shared constants (timeouts, TTL values)
│   ├── logger/           # Zerolog-based structured logging
│   └── sync/             # Synchronization engine
│       ├── engine.go     # Main sync orchestration
│       ├── utils.go      # Common utilities (mappings, system detection)
│       ├── roles.go      # User role permission synchronization
│       ├── organization.go # Organization role synchronization
│       └── resources.go  # Resource and scope synchronization
├── pkg/version/          # Version information
├── configs/              # Example configurations
└── Makefile              # Build automation
```


## Examples

### 🚀 Complete Setup Workflow (Recommended)

```bash
# 1. Complete zero-to-production setup
sync init \
  --tenant-id your-tenant-id \
  --backend-client-id your-backend-client-id \
  --backend-client-secret your-secret-here \
  --domain your-domain.com

# 2. Copy environment variables from output to backend/.env

# 3. Start backend
cd backend && go run main.go

# 4. Optional: Make RBAC changes later
vim config.yml
sync sync -c config.yml --dry-run --verbose
sync sync -c config.yml
```

### RBAC Workflow

```bash
# 1. Edit your configuration
vim config.yml

# 2. Preview changes
sync sync -c config.yml --dry-run --verbose

# 3. Apply changes
sync sync -c config.yml --verbose

# 4. Monitor results
sync sync -c config.yml --output json | jq .summary
```

### Init Command Examples

```bash
# Basic initialization
sync init --tenant-id abc123 --backend-client-id xyz --backend-client-secret secret --domain my.domain.com

# Environment variables mode
TENANT_ID=abc123 BACKEND_CLIENT_ID=xyz BACKEND_CLIENT_SECRET=secret TENANT_DOMAIN=my.domain.com sync init

# Force re-initialization
sync init --force --tenant-id abc123 --backend-client-id xyz --backend-client-secret secret --domain my.domain.com

# JSON output for automation
sync init --output json --tenant-id abc123 --backend-client-id xyz --backend-client-secret secret --domain my.domain.com

# YAML output for automation
sync init --output yaml --tenant-id abc123 --backend-client-id xyz --backend-client-secret secret --domain my.domain.com

# Automation with jq
sync init --output json | jq '.backend_app.environment_vars'
sync init --output json | jq '.frontend_app.environment_vars'
sync init --output json | jq '.god_user.password'
```

### Resource Management

```bash
# Add new resources to config.yml, then:
sync sync -c config.yml --dry-run  # Preview
sync sync -c config.yml            # Apply

# Remove resources from config.yml, then:
sync sync -c config.yml --cleanup --dry-run  # Preview cleanup
sync sync -c config.yml --cleanup            # Apply cleanup
```

### Troubleshooting Workflow

```bash
# Test connection with detailed logs
LOG_LEVEL=debug sync sync --dry-run

# Check configuration validity
sync sync -c config.yml --dry-run

# Force sync past validation errors
sync sync -c config.yml --force

# Detailed JSON output for debugging
sync sync --output json | jq .operations

# Debug API calls and performance
LOG_LEVEL=debug sync sync --dry-run 2>debug.log
grep "API call completed" debug.log  # Check API performance
grep "component=config" debug.log    # Check config loading
```

### Advanced Logging Examples

```bash
# Production monitoring - only errors to syslog
LOG_LEVEL=error sync sync 2>&1 | logger -t sync-tool

# Development debugging with component isolation
LOG_LEVEL=debug sync sync 2>&1 | grep "component=api-client"  # API calls only
LOG_LEVEL=debug sync sync 2>&1 | grep "component=sync"       # Sync operations only

# Performance analysis
LOG_LEVEL=debug sync sync --dry-run 2>&1 | grep "duration=" | sort -k4 -nr

# Separate monitoring streams
sync sync --output json > results.json 2> >(LOG_LEVEL=warn cat > errors.log)
```

### Selective Synchronization

```bash
# Only sync roles and permissions, skip resources
sync sync --skip-resources

# Only sync resources, skip everything else
sync sync --skip-roles --skip-permissions

# Sync everything except permissions
sync sync --skip-permissions
```

### Output Formats

```bash
# Human-readable text output (default)
sync sync

# JSON output for programmatic use
sync sync --output json

# YAML output
sync sync --output yaml

# Pipe JSON to jq for analysis
sync sync --output json | jq '.summary'
sync sync --output json | jq '.operations[] | select(.success == false)'
```

## Error Handling

The tool provides detailed error messages and suggestions:

```bash
# Configuration validation error
Error: configuration validation failed: invalid permission reference 'invalid:permission' in user role 'support'

# Connection error
Error: failed to connect to Logto: failed to authenticate: 401 Unauthorized

# Environment variable missing
Error: required environment variable LOGTO_BASE_URL is not set

# Cleanup safety warning
Error: cleanup would remove 5 resources - use --cleanup flag to confirm
```

## Best Practices

### 🔄 **Recommended Workflow**
1. **Always dry-run first**: `--dry-run --verbose`
2. **Review changes carefully**: Check the operations list
3. **Apply incrementally**: Use selective sync flags for large changes
4. **Monitor results**: Use JSON output for automation
5. **Use cleanup carefully**: Always preview with `--cleanup --dry-run`

### 🛡️ **Safety Guidelines**
- ✅ Test changes in a development environment first
- ✅ Use `--dry-run` before applying to production
- ✅ Keep backups of your configuration files
- ✅ Review cleanup operations carefully before running
- ✅ Use version control for your configuration files

### 📊 **Monitoring & Integration**
```bash
# CI/CD integration
sync sync -c production.yml --output json > sync-results.json

# Monitor for failures
sync sync --output json | jq '.success'

# Count changes
sync sync --dry-run --output json | jq '.summary'
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make test lint fmt`
6. Submit a pull request

## License

This project is licensed under the GPL-2.0 License - see the [LICENSE](LICENSE) file for details.