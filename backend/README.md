# Backend API

Go REST API server for Nethesis Operation Center with Logto JWT authentication and Role-Based Access Control (RBAC).

## Quick Start

### Prerequisites
- Go 1.23+
- Docker/Podman (for Redis)
- Logto instance with M2M app configured

**Note:** The Makefile automatically detects and uses Docker or Podman.

### Setup

```bash
# Setup development environment
make dev-setup

# Start PostgreSQL and Redis containers
make dev-up

# Start the application
make run

# Stop PostgreSQL and Redis when done
make dev-down
```

### Required Environment Variables
```bash
# ===========================================
# REQUIRED CONFIGURATION
# ===========================================
# Logto tenant configuration (all other URLs auto-derived)
TENANT_ID=your-tenant-id
TENANT_DOMAIN=your-domain.com

# Logto Management API (from your M2M app)
BACKEND_APP_ID=your-management-api-app-id
BACKEND_APP_SECRET=your-management-api-app-secret

# Custom JWT for resilient offline operation
JWT_SECRET=your-super-secret-jwt-signing-key-min-32-chars

# PostgreSQL connection string (shared 'noc' database)
DATABASE_URL=postgresql://backend:backend@localhost:5432/noc?sslmode=disable

# Redis connection URL
REDIS_URL=redis://localhost:6379
```

**Auto-derived URLs:**
- `LOGTO_ISSUER` = `https://{TENANT_ID}.logto.app`
- `LOGTO_AUDIENCE` = `https://{TENANT_DOMAIN}/api`
- `JWKS_ENDPOINT` = `https://{TENANT_ID}.logto.app/oidc/jwks`
- `LOGTO_MANAGEMENT_BASE_URL` = `https://{TENANT_ID}.logto.app/api`
- `JWT_ISSUER` = `{TENANT_DOMAIN}`

## Architecture

### Two-Layer Authorization
1. **Base Authentication**: All routes require valid Logto JWT
2. **Access Control**: Routes check permissions from user roles OR organization roles

### Permission Sources
- **User Roles** (technical capabilities): Admin, Support
- **Organization Roles** (business hierarchy): Owner, Distributor, Reseller, Customer

### Redis Caching System
High-performance caching system with multiple cache types:

#### Statistics Cache
- **Auto-initialized**: Starts with server, updates every 5 minutes
- **Cached Data**: Organization counts, user statistics, system metrics
- **TTL**: 10 minutes
- **Redis Keys**: `stats:*`

#### JIT Roles Cache
- **JIT-initialized**: Lazy loading on first access
- **Cached Data**: User roles, organization roles, permissions
- **TTL**: 5 minutes per cache entry
- **Redis Keys**: `jit_roles:*`

#### Organization Users Cache
- **Cached Data**: User lists per organization
- **TTL**: 3 minutes per cache entry
- **Redis Keys**: `org_users:*`

#### JWKS Cache
- **Cached Data**: JSON Web Key Sets for token validation
- **TTL**: 5 minutes
- **Redis Keys**: `jwks:*`

#### Cache Management
- **Graceful Degradation**: System continues working if Redis is unavailable
- **Background Updates**: Automatic cache refresh and cleanup
- **Configurable TTLs**: All timeouts configurable via environment variables

## API Endpoints

See [API.md](API.md) for complete API documentation.

## Development

### Basic Commands
```bash
# Run tests
make test

# Format code
make fmt

# Run linter
make lint

# Build
make build

# Run server
make run

# Run QA server (uses .env.qa)
make run-qa

# Test coverage
make test-coverage
```

### Database Commands
```bash
# Start PostgreSQL container (Docker/Podman auto-detected)
make db-up

# Stop PostgreSQL container
make db-down

# Reset PostgreSQL container (stop + start)
make db-reset

# Run database migrations
make db-migrate
```

### Redis Commands
```bash
# Start Redis container (Docker/Podman auto-detected)
make redis-up

# Stop Redis container
make redis-down

# Flush Redis cache
make redis-flush

# Connect to Redis CLI
make redis-cli
```

### API Documentation
```bash
# Validate OpenAPI documentation
make validate-docs
```

### Testing
```bash
# Test token exchange
curl -X POST http://localhost:8080/api/auth/exchange \
  -H "Content-Type: application/json" \
  -d '{"access_token": "YOUR_LOGTO_TOKEN"}'

# Test with custom JWT
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT"
```

## Project Structure
```
backend/
├── main.go                 # Server entry point
├── cache/                  # Redis caching system
├── configuration/          # Environment config
├── helpers/                # Utilities for JWT context
├── jwt/                    # Utilities for JWT claims
├── logger/                 # Structured logging
├── methods/                # HTTP handlers
├── middleware/             # Auth and RBAC middleware
├── models/                 # Data structures
├── response/               # HTTP response helpers
├── services/               # Business logic
└── .env.example            # Environment variables template
```


## Related
- [API.md](API.md) - API docs reference
- [Collect](../collect/README.md) - Collect server
- [sync CLI](../sync/README.md) - RBAC configuration tool
- [Project Overview](../README.md) - Main documentation