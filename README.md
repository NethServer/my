# Nethesis Operation Center

[![CI](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?style=for-the-badge&label=CI)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![Backend Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=backend-tests&label=Backend%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![sync Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=sync-tests&label=sync%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)

[![Release](https://img.shields.io/github/actions/workflow/status/NethServer/my/release.yml?style=for-the-badge&label=Release)](https://github.com/NethServer/my/actions/workflows/release.yml)
[![Version](https://img.shields.io/github/v/release/NethServer/my?style=for-the-badge&color=3a3c3f&label=Version)](https://github.com/NethServer/my/releases)

A web application providing centralized authentication and management using Logto as an Identity Provider, built with a sophisticated Role-Based Access Control (RBAC) system.

## 🏗️ Project Structure

```
my/
├── backend/            # Go REST API with Logto JWT authentication
├── sync/               # CLI tool for RBAC configuration synchronization
├── DESIGN.md           # Project design documentation
└── LICENSE             # Project license
```

## 🚀 Components

### Backend API
Go-based REST API featuring:
- **Authentication**: Logto JWT validation with JWKS caching and token exchange system
- **Authorization**: Sophisticated RBAC with business hierarchy and technical capability separation
- **Logging**: Structured zerolog-based system with security-first design and automatic sensitive data redaction
- **Performance**: Thread-safe JWKS cache, optimized middleware chain, and embedded permissions
- **Security**: Comprehensive audit trail, authentication event tracking, and pattern-based credential detection
- **Account Management**: Hierarchical business rule enforcement with organizational role validation
- **Framework**: Gin web framework with middleware architecture
- **Integration**: Real-time Logto Management API data fetching for roles and permissions

### sync CLI
RBAC management tool featuring:
- **Configuration Management**: Simplified YAML-based role and permission definitions with business/technical separation
- **Logging**: Structured zerolog-based system with component isolation and security-first design
- **Safety Features**: Comprehensive dry-run mode, system entity protection, and validation workflows
- **Performance**: Optimized permission synchronization with deduplication and caching utilities
- **Security**: Automatic sensitive data redaction in logs and pattern-based credential detection
- **Monitoring**: Detailed operation tracking, API call timing, and configuration validation reporting
- **Integration**: Direct Logto Management API synchronization with error handling and retry logic

## 🛠️ Quick Start

### System Requirements

#### Runtime (Binary Execution)
- **Any 64-bit OS**: Linux, macOS, Windows
- **No dependencies** - Both components compile to statically linked binaries

#### Development & Building
- **Go 1.21+** (backend requires 1.23+) - [Download from golang.org](https://golang.org/download/)
- **Make** (for build automation):
  - **macOS**: Preinstalled with Xcode Command Line Tools (`xcode-select --install`)
  - **Linux**: Usually preinstalled, or install with package manager (`apt install build-essential`)
  - **Windows**: Install via [Git Bash](https://git-scm.com/download/win), [WSL2](https://docs.microsoft.com/en-us/windows/wsl/install), or [Chocolatey](https://chocolatey.org/) (`choco install make`)
- **golangci-lint** (optional, for linting): [Installation guide](https://golangci-lint.run/usage/install/)

#### External Dependencies
- **Logto instance** - Identity provider with RBAC configuration
- **Logto Management API** - Machine-to-Machine app with full Management API permissions
- **Custom JWT secret** - For backend token signing

### Alternative Build Methods
If Make is not available, both projects support direct Go commands:
```bash
# Backend
cd backend && go run main.go

# sync
cd sync && go run ./cmd/sync
```

### Getting Started
Each component has its own setup instructions:

- **Backend**: See [backend/README.md](./backend/README.md) for Backend server setup
- **Backend API**: See [backend/API.md](./backend/API.md) for API references
- **sync CLI**: See [sync/README.md](./sync/README.md) for sync and RBAC management

## 📝 Logging & Monitoring Architecture

Both components feature structured logging systems built on [zerolog](https://github.com/rs/zerolog):

### Security-First Design
- **Automatic Redaction**: Sensitive data (passwords, tokens, secrets) automatically sanitized from all logs
- **Pattern Matching**: Advanced regex patterns detect and redact various credential formats in request bodies
- **Audit Trail**: Complete authentication and authorization event tracking with user context
- **Security Events**: Failed authentication attempts, invalid tokens, and authorization failures

### Structured Observability
- **Component Isolation**: Separate loggers for different system components (http, auth, rbac, api-client, sync, config)
- **Performance Tracking**: HTTP request timing, API call duration, and system operation metrics
- **Operational Intelligence**: Configuration validation, sync operation status, and health monitoring
- **Stream Separation**: Clean command output (stdout) separate from operational logs (stderr)

### Production Integration
```bash
# Backend structured JSON logs
GIN_MODE=release ./backend 2>&1 | jq 'select(.level == "error")'

# sync tool component monitoring
LOG_LEVEL=debug ./sync sync 2>&1 | grep "component=api-client"

# Security event monitoring
./backend 2>&1 | jq 'select(.component == "auth" and .success == false)'
```

## 🔐 Authorization Architecture

The system implements a **token exchange pattern** with real-time Management API integration:

1. **Token Exchange**: Frontend exchanges Logto access_token for custom JWT with embedded permissions
2. **Management API Integration**: Real-time fetching of user roles and organization memberships from Logto
3. **Permission Embedding**: All permissions pre-computed and embedded in custom JWT
4. **Unified Authorization**: Single middleware checks both user and organization permissions

### Example Authorization Flow
```go
// Token exchange endpoint (public)
auth.POST("/exchange", methods.ExchangeToken)

// Protected routes with custom JWT
protected := api.Group("/", middleware.CustomAuthMiddleware())
systemsGroup := protected.Group("/systems", middleware.RequirePermission("read:systems"))
systemsGroup.POST("/:id/restart", middleware.RequirePermission("manage:systems"), methods.RestartSystem)

// Account management with hierarchical validation
accountsGroup := protected.Group("/accounts")
accountsGroup.POST("", methods.CreateAccount) // Business rule validation in handler
```

## 📝 Configuration

### Backend Configuration
Environment variables for production deployment:
```bash
# Authentication (Required)
LOGTO_ISSUER=https://your-logto-instance.logto.app
LOGTO_AUDIENCE=your-api-resource-identifier
JWT_SECRET=your-custom-jwt-secret

# Management API (Required)
LOGTO_MANAGEMENT_CLIENT_ID=your-m2m-client-id
LOGTO_MANAGEMENT_CLIENT_SECRET=your-m2m-secret

# Logging & Performance (Optional)
GIN_MODE=release              # 'debug' for development
LOG_LEVEL=info                # debug, info, warn, error
LISTEN_ADDRESS=127.0.0.1:8080 # Server bind address
```

### sync Tool Configuration
Environment variables and YAML configuration:
```bash
# Logto Management API (Required)
LOGTO_BASE_URL=https://your-logto-instance.logto.app
LOGTO_CLIENT_ID=your-m2m-client-id
LOGTO_CLIENT_SECRET=your-m2m-secret

# Logging (Optional)
LOG_LEVEL=info               # debug, info, warn, error, fatal
```

### Key Requirements
- **Logto Instance**: Identity provider with RBAC configuration
- **Management API**: Machine-to-Machine app with full API permissions for both components
- **Custom JWT Secret**: Backend secret for signing custom tokens (backend only)
- **RBAC Configuration**: YAML files defining roles and permissions (sync only)

See individual README files for detailed configuration instructions and examples.

## 📚 Documentation

### Component Documentation
- **[Backend](./backend/README.md)** - Go REST API server setup, RBAC system, and development guide
- **[Backend API](./backend/API.md)** - Go REST APIs reference
- **[sync CLI](./sync/README.md)** - RBAC configuration management tool and usage instructions

### Project Documentation
- **[DESIGN.md](./DESIGN.md)** - Project architecture and design decisions

## 🤝 Contributing

1. Follow existing code patterns and conventions
2. **Pre-commit workflow**: Run `make fmt && make test` in both `backend/` and `sync/` directories
3. Always test RBAC changes with `--dry-run` before applying
4. Ensure environment variables are properly configured
5. Check that CI tests pass before submitting PRs

## 📄 License

See [LICENSE](./LICENSE) file for details.

---

**🔗 Related Links**
- [Logto Documentation](https://docs.logto.io/) - Identity provider documentation
