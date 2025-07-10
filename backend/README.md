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

# Start Redis container (Docker/Podman auto-detected)
make redis-up

# Edit .env with your Logto configuration
# Start development server
make run

# Stop Redis when done
make redis-down
```

### Required Environment Variables
```bash
# Authentication
LOGTO_ISSUER=https://your-logto-instance.logto.app
LOGTO_AUDIENCE=your-api-resource-identifier
JWT_SECRET=your-custom-jwt-secret

# Management API
BACKEND_APP_ID=your-m2m-app-id
BACKEND_APP_SECRET=your-m2m-secret

# Redis Configuration (Required)
REDIS_URL=redis://localhost:6379
REDIS_DB=0
REDIS_PASSWORD=
# Note: REDIS_PASSWORD can be empty for Redis without auth
```

### Optional Environment Variables
```bash
LISTEN_ADDRESS=127.0.0.1:8080
JWKS_ENDPOINT=https://your-logto-instance.logto.app/oidc/jwks
JWT_ISSUER=your-api.com
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=168h
LOGTO_MANAGEMENT_BASE_URL=https://your-logto-instance.logto.app/api

# Redis Connection Settings (Optional - defaults shown)
REDIS_MAX_RETRIES=3
REDIS_DIAL_TIMEOUT=5s
REDIS_READ_TIMEOUT=3s
REDIS_WRITE_TIMEOUT=3s
REDIS_OPERATION_TIMEOUT=5s
# Note: Omit any of these to use the default values

# Cache TTL Configuration
STATS_CACHE_TTL=10m
STATS_UPDATE_INTERVAL=5m
STATS_STALE_THRESHOLD=15m
JIT_ROLES_CACHE_TTL=5m
JIT_ROLES_CLEANUP_INTERVAL=2m
ORG_USERS_CACHE_TTL=3m
ORG_USERS_CLEANUP_INTERVAL=1m
JWKS_CACHE_TTL=5m
JWKS_HTTP_TIMEOUT=10s

# API Configuration
DEFAULT_PAGE_SIZE=100
```

## Architecture

### Two-Layer Authorization
1. **Base Authentication**: All routes require valid Logto JWT
2. **Access Control**: Routes check permissions from user roles OR organization roles

### Permission Sources
- **User Roles** (technical capabilities): Admin, Support
- **Organization Roles** (business hierarchy): Owner, Distributor, Reseller, Customer

### Redis Caching System
- **JIT Roles Cache**: User and organization roles with TTL-based expiration
- **Organization Users Cache**: Cached user lists per organization
- **JWKS Cache**: JSON Web Key Sets for token validation
- **System Statistics Cache**: Real-time system metrics and counts
- **Background Processing**: Automatic cache updates and cleanup

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

# Run with hot reload
make dev

# Test coverage
make test-coverage
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

## Background Systems

### Redis Cache System
The application uses Redis for high-performance caching with the following components:

### Statistics Cache
- **Auto-initialized**: Starts with server, updates every 5 minutes
- **Cached Data**: Organization counts, user statistics, system metrics
- **TTL**: 10 minutes
- **Redis Keys**: `stats:*`

### JIT Roles Cache
- **JIT-initialized**: Lazy loading on first access
- **Cached Data**: User roles, organization roles, permissions
- **TTL**: 5 minutes per cache entry
- **Redis Keys**: `jit_roles:*`

### Organization Users Cache
- **Cached Data**: User lists per organization
- **TTL**: 3 minutes per cache entry
- **Redis Keys**: `org_users:*`

### JWKS Cache
- **Cached Data**: JSON Web Key Sets for token validation
- **TTL**: 5 minutes
- **Redis Keys**: `jwks:*`

### Cache Management
- **Graceful Degradation**: System continues working if Redis is unavailable
- **Background Updates**: Automatic cache refresh and cleanup
- **Configurable TTLs**: All timeouts configurable via environment variables

## Related
- [API.md](API.md) - API docs reference
- [sync CLI](../sync/README.md) - RBAC configuration tool
- [Project Overview](../README.md) - Main documentation