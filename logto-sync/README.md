# logto-sync

A robust CLI tool for synchronizing Role-Based Access Control (RBAC) configuration with Logto identity provider.

## Features

- 🔄 **Complete RBAC Synchronization**: Resources, roles, permissions, and scopes
- 🏢 **Hierarchical Organizations**: Support for organization roles and user roles
- 🔍 **Dry Run Mode**: Preview changes before applying them with detailed analysis
- 🧹 **Cleanup Mode**: Remove resources/roles/scopes not defined in config (opt-in)
- 📊 **Multiple Output Formats**: Text, JSON, and YAML
- 🛡️ **Safe Operations**: Preserves system entities and validates configurations
- 🔧 **Flexible Configuration**: YAML-based configuration with validation

## Installation

### From Source

```bash
git clone https://github.com/nethesis/logto-sync.git
cd logto-sync
make build
```

### Using Go Install

```bash
go install github.com/nethesis/logto-sync/cmd/logto-sync@latest
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
logto-sync sync -c my-config.yml --dry-run --verbose
```

4. **Apply Configuration**

```bash
logto-sync sync -c my-config.yml
```

## Usage

### Basic Commands

```bash
# Show help
logto-sync --help

# Show version
logto-sync --version

# Sync with default config
logto-sync sync

# Sync with specific config file
logto-sync sync -c configs/hierarchy.yml

# Dry run to preview changes
logto-sync sync --dry-run --verbose

# Output results in JSON format
logto-sync sync --output json

# Skip specific sync phases
logto-sync sync --skip-resources --skip-roles
```

### Advanced Operations

#### **🔍 Dry Run Mode**

Preview what would be changed without making any modifications:

```bash
# Basic dry run
logto-sync sync -c hierarchy.yml --dry-run

# Verbose dry run with detailed logs
logto-sync sync -c hierarchy.yml --dry-run --verbose

# Dry run with JSON output for analysis
logto-sync sync -c hierarchy.yml --dry-run --output json | jq .

# Preview cleanup operations
logto-sync sync -c hierarchy.yml --cleanup --dry-run --verbose
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
logto-sync sync -c hierarchy.yml --cleanup --dry-run

# Perform cleanup (removes items not in config)
logto-sync sync -c hierarchy.yml --cleanup

# Cleanup with verbose logging
logto-sync sync -c hierarchy.yml --cleanup --verbose
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

The configuration file uses YAML format with the following structure:

```yaml
metadata:
  name: "my-rbac-config"
  version: "1.0.0"
  description: "RBAC configuration for my application"

hierarchy:
  organization_roles:
    - id: admin
      name: "Administrator"
      type: user
      priority: 0
      permissions:
        - id: "admin:systems"
          name: "Administer systems"

  user_roles:
    - id: support
      name: "Support"
      type: user
      priority: 1
      permissions:
        - id: "read:systems"
          name: "Read systems"

  resources:
    - name: "systems"
      actions: ["create", "read", "update", "delete", "admin"]
```

## Development

### Prerequisites

- Go 1.21+
- Make
- golangci-lint (for linting)

### Setup Development Environment

```bash
make dev-setup
```

### Available Make Targets

```bash
make help                 # Show all available targets
make build               # Build the binary
make test                # Run tests
make test-coverage       # Run tests with coverage
make lint                # Run linter
make fmt                 # Format code
make clean               # Clean build artifacts
make run ARGS="--help"   # Run with arguments
make run-example         # Run with example config
```

### Project Structure

```
├── cmd/logto-sync/        # CLI entry point
├── internal/
│   ├── cli/              # CLI commands and flags
│   ├── client/           # Logto API client
│   ├── config/           # Configuration loading and validation
│   ├── sync/             # Synchronization engine
│   └── logger/           # Logging utilities
├── pkg/version/          # Version information
├── configs/              # Example configurations
└── Makefile             # Build automation
```

## Examples

### Basic Workflow

```bash
# 1. Edit your configuration
vim hierarchy.yml

# 2. Preview changes
logto-sync sync -c hierarchy.yml --dry-run --verbose

# 3. Apply changes
logto-sync sync -c hierarchy.yml --verbose

# 4. Monitor results
logto-sync sync -c hierarchy.yml --output json | jq .summary
```

### Resource Management

```bash
# Add new resources to hierarchy.yml, then:
logto-sync sync -c hierarchy.yml --dry-run  # Preview
logto-sync sync -c hierarchy.yml            # Apply

# Remove resources from hierarchy.yml, then:
logto-sync sync -c hierarchy.yml --cleanup --dry-run  # Preview cleanup
logto-sync sync -c hierarchy.yml --cleanup            # Apply cleanup
```

### Troubleshooting Workflow

```bash
# Test connection
logto-sync sync --dry-run --verbose

# Check configuration validity
logto-sync sync -c hierarchy.yml --dry-run

# Force sync past validation errors
logto-sync sync -c hierarchy.yml --force

# Detailed JSON output for debugging
logto-sync sync --output json | jq .operations
```

### Selective Synchronization

```bash
# Only sync roles and permissions, skip resources
logto-sync sync --skip-resources

# Only sync resources, skip everything else
logto-sync sync --skip-roles --skip-permissions

# Sync everything except permissions
logto-sync sync --skip-permissions
```

### Output Formats

```bash
# Human-readable text output (default)
logto-sync sync

# JSON output for programmatic use
logto-sync sync --output json

# YAML output
logto-sync sync --output yaml

# Pipe JSON to jq for analysis
logto-sync sync --output json | jq '.summary'
logto-sync sync --output json | jq '.operations[] | select(.success == false)'
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
logto-sync sync -c production.yml --output json > sync-results.json

# Monitor for failures
logto-sync sync --output json | jq '.success'

# Count changes
logto-sync sync --dry-run --output json | jq '.summary'
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