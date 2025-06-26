# sync

A robust CLI tool for synchronizing simplified Role-Based Access Control (RBAC) configuration with Logto identity provider. Works in conjunction with the backend's Management API integration to provide real-time permission synchronization.

## Features

- 🔄 **Simplified RBAC Synchronization**: Clear separation between business hierarchy and technical capabilities
- 🏢 **Business Hierarchy**: Organization roles (God, Distributor, Reseller, Customer) for commercial logic
- 👥 **Technical Capabilities**: User roles (Admin, Support) for skills
- 🔗 **Backend Integration**: Powers real-time Management API data fetching in backend
- 🔍 **Dry Run Mode**: Preview changes before applying them with detailed analysis
- 🧹 **Cleanup Mode**: Remove resources/roles/permissions not defined in config (opt-in)
- 📊 **Multiple Output Formats**: Text, JSON, and YAML
- 🛡️ **Safe Operations**: Preserves system entities and validates configurations
- 🔧 **Simplified Configuration**: YAML-based with clear business vs technical separation

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

## Quick Start

1. **Setup Environment Variables**

```bash
cp .env.example .env
# Edit .env with your Logto configuration
```

Required environment variables:
- `LOGTO_BASE_URL`: Your Logto instance URL
- `LOGTO_CLIENT_ID`: Management API client ID
- `LOGTO_CLIENT_SECRET`: Management API client secret
- `API_BASE_URL`: Your API base URL (optional, defaults to https://dev.my.nethesis.it)

2. **Create Configuration File**

```bash
cp configs/hierarchy.yml my-config.yml
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

# Sync with default config
sync sync

# Sync with specific config file
sync sync -c configs/hierarchy.yml

# Dry run to preview changes
sync sync --dry-run --verbose

# Output results in JSON format
sync sync --output json

# Skip specific sync phases
sync sync --skip-resources --skip-roles
```

### Advanced Operations

#### **🔍 Dry Run Mode**

Preview what would be changed without making any modifications:

```bash
# Basic dry run
sync sync -c hierarchy.yml --dry-run

# Verbose dry run with detailed logs
sync sync -c hierarchy.yml --dry-run --verbose

# Dry run with JSON output for analysis
sync sync -c hierarchy.yml --dry-run --output json | jq .

# Preview cleanup operations
sync sync -c hierarchy.yml --cleanup --dry-run --verbose
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
sync sync -c hierarchy.yml --cleanup --dry-run

# Perform cleanup (removes items not in config)
sync sync -c hierarchy.yml --cleanup

# Cleanup with verbose logging
sync sync -c hierarchy.yml --cleanup --verbose
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

- `-c, --config`: Configuration file path (default: ./hierarchy.yml)
- `-v, --verbose`: Enable verbose output
- `--dry-run`: Show what would be done without making changes
- `-o, --output`: Output format (text, json, yaml)

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
  name: "nethesis-simplified-rbac"
  version: "2.0.0"
  description: "Simplified RBAC with business logic separation"

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
├── cmd/sync/       # CLI entry point
├── internal/
│   ├── cli/              # CLI commands and flags
│   ├── client/           # Logto API client
│   ├── config/           # Configuration loading and validation
│   ├── sync/             # Synchronization engine
│   └── logger/           # Logging utilities
├── pkg/version/          # Version information
├── configs/              # Example configurations
└── Makefile              # Build automation
```

## Examples

### Basic Workflow

```bash
# 1. Edit your configuration
vim hierarchy.yml

# 2. Preview changes
sync sync -c hierarchy.yml --dry-run --verbose

# 3. Apply changes
sync sync -c hierarchy.yml --verbose

# 4. Monitor results
sync sync -c hierarchy.yml --output json | jq .summary
```

### Resource Management

```bash
# Add new resources to hierarchy.yml, then:
sync sync -c hierarchy.yml --dry-run  # Preview
sync sync -c hierarchy.yml            # Apply

# Remove resources from hierarchy.yml, then:
sync sync -c hierarchy.yml --cleanup --dry-run  # Preview cleanup
sync sync -c hierarchy.yml --cleanup            # Apply cleanup
```

### Troubleshooting Workflow

```bash
# Test connection
sync sync --dry-run --verbose

# Check configuration validity
sync sync -c hierarchy.yml --dry-run

# Force sync past validation errors
sync sync -c hierarchy.yml --force

# Detailed JSON output for debugging
sync sync --output json | jq .operations
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