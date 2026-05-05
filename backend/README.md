# Backend API

Go REST API server for My Nethesis with Logto JWT authentication and Role-Based Access Control (RBAC).

## Quick Start

### Prerequisites
- Go 1.24+
- Docker/Podman (for PostgreSQL and Redis)
- Logto instance with M2M app configured

**Note:** The Makefile automatically detects and uses Docker or Podman.

### Setup

```bash
# Setup development environment
make dev-setup

# Start PostgreSQL and Redis containers
make dev-up

# Start the application (initializes database with schema.sql on first run)
make run

# Run database migrations (applies incremental changes on top of base schema)
make db-migrate

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

# App URL configuration (frontend application URL)
APP_URL=https://your-app-domain.com

# Logto Management API (from your M2M app)
BACKEND_APP_ID=your-management-api-app-id
BACKEND_APP_SECRET=your-management-api-app-secret

# Custom JWT for resilient offline operation
JWT_SECRET=your-super-secret-jwt-signing-key-min-32-chars

# PostgreSQL connection string (shared 'noc' database)
DATABASE_URL=postgresql://noc_user:noc_password@localhost:5432/noc?sslmode=disable

# Redis connection URL
REDIS_URL=redis://localhost:6379

# ===========================================
# BACKUP STORAGE (S3-compatible)
# ===========================================
# Any S3-compatible endpoint works: DigitalOcean Spaces, AWS S3,
# Cloudflare R2, Backblaze B2, self-hosted MinIO / Garage, etc.
# Both backend and collect must point at the SAME bucket; backend
# reads and issues presigned URLs, collect writes uploads from
# appliances. See collect/README.md for the bucket layout.
BACKUP_S3_ENDPOINT=https://ams3.digitaloceanspaces.com
BACKUP_S3_REGION=ams3
BACKUP_S3_BUCKET=my-backups
BACKUP_S3_ACCESS_KEY=your-access-key
BACKUP_S3_SECRET_KEY=your-secret-key
# Set to "true" only against local emulators that do not serve
# virtual-hosted-style URLs. Spaces / S3 / R2 use the default (false).
BACKUP_S3_USE_PATH_STYLE=false
# Optional: override the endpoint used for signing download URLs so the
# browser can follow them when backend and the S3 emulator are reached
# via different hostnames (compose network vs. host ports). Empty in
# production.
#BACKUP_S3_PRESIGN_ENDPOINT=
# Lifetime of a presigned download URL (capped server-side at 15m).
#BACKUP_PRESIGN_TTL=5m

# ===========================================
# SMTP EMAIL CONFIGURATION (Optional)
# ===========================================
# SMTP server configuration for welcome emails
# If not configured, welcome emails will be skipped (user creation still succeeds)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourdomain.com
SMTP_FROM_NAME=My Nethesis
SMTP_TLS=true
```

**Auto-derived URLs:**
- `LOGTO_ISSUER` = `https://{TENANT_ID}.logto.app`
- `LOGTO_AUDIENCE` = `https://{TENANT_DOMAIN}/api`
- `JWKS_ENDPOINT` = `https://{TENANT_ID}.logto.app/oidc/jwks`
- `LOGTO_MANAGEMENT_BASE_URL` = `https://{TENANT_ID}.logto.app/api`
- `JWT_ISSUER` = `{TENANT_DOMAIN}`

## Email Configuration

Welcome emails are sent automatically when users are created via API. Email is optional and degrades gracefully if SMTP is not configured.

Configure SMTP in your `.env` (see `.env.example` for all variables):

```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=noreply@yourdomain.com
SMTP_FROM_NAME=Your Company Name
SMTP_TLS=true
```

Templates are in `services/email/templates/` and support Go template syntax.

## Architecture

### Two-Layer Authorization
1. **Base Authentication**: All routes require valid Logto JWT
2. **Access Control**: Routes check permissions from user roles OR organization roles

### Permission Sources
- **User Roles** (technical capabilities): Admin, Support
- **Organization Roles** (business hierarchy): Owner, Distributor, Reseller, Customer

### User Impersonation
Owner-only feature: temporarily act as another user with 1-hour scoped JWT tokens. Prevents self-impersonation and token chaining. All actions are logged.

- `POST /api/auth/impersonate` - Start impersonation
- `POST /api/auth/exit-impersonation` - Exit impersonation

### Redis Caching
Multiple cache layers with graceful degradation (system works without Redis):

| Cache | TTL | Keys |
|-------|-----|------|
| Statistics | 10 min | `stats:*` |
| JIT Roles | 5 min | `jit_roles:*` |
| Organization Users | 3 min | `org_users:*` |
| JWKS | 5 min | `jwks:*` |

## API Endpoints

See [openapi.yaml](openapi.yaml) for complete API specification.

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

# Run all pending database migrations (automatically skips applied ones)
make db-migrate                              # Uses .env
make db-migrate-qa                           # Uses .env.qa

# Run specific migration (for advanced users)
make db-migration MIGRATION=001 ACTION=apply              # Uses .env
make db-migration MIGRATION=001 ACTION=apply ENV=qa       # Uses .env.qa
make db-migration MIGRATION=001 ACTION=rollback
make db-migration MIGRATION=001 ACTION=status
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

### Test Tokens
```bash
# Generate JWT tokens for all 4 RBAC roles (Owner, Distributor, Reseller, Customer)
make gen-tokens

# Use tokens for manual API testing
curl -H "Authorization: Bearer $(cat token-owner)" http://localhost:8080/api/users
curl -H "Authorization: Bearer $(cat token-distributor)" http://localhost:8080/api/resellers
curl -H "Authorization: Bearer $(cat token-reseller)" http://localhost:8080/api/customers
curl -H "Authorization: Bearer $(cat token-customer)" http://localhost:8080/api/systems
```

See [openapi.yaml](openapi.yaml) for all available endpoints and expected payloads.

### API Documentation
```bash
# Validate OpenAPI documentation
make validate-docs
```

## Project Structure
```
backend/
├── main.go                  # Server entry point
├── cache/                   # Redis caching system
├── configuration/           # Environment config
├── helpers/                 # Utilities for JWT context
├── jwt/                     # Utilities for JWT claims
├── logger/                  # Structured logging
├── methods/                 # HTTP handlers
├── middleware/              # Auth and RBAC middleware
├── models/                  # Data structures
├── response/                # HTTP response helpers
├── services/                # Business logic
│   ├── email/               # Email service
│   │   ├── smtp.go          # SMTP client implementation
│   │   ├── templates.go     # Template rendering service
│   │   ├── welcome.go       # Welcome email service
│   │   └── templates/       # Email templates
│   │       ├── welcome.html # HTML email template
│   │       └── welcome.txt  # Text email template
│   ├── local/               # Local database services
│   └── logto/               # Logto API integration
└── .env.example             # Environment variables template
```


## Related
- [openapi.yaml](openapi.yaml) - API specification
- [Collect](../collect/README.md) - Collect server
- [sync CLI](../sync/README.md) - RBAC configuration tool
- [Project Overview](../README.md) - Main documentation