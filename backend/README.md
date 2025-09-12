# Backend API

Go REST API server for My Nethesis with Logto JWT authentication and Role-Based Access Control (RBAC).

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

# Run database migrations
make db-migrate

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

# App URL configuration (frontend application URL)
APP_URL=https://your-app-domain.com

# Logto Management API (from your M2M app)
BACKEND_APP_ID=your-management-api-app-id
BACKEND_APP_SECRET=your-management-api-app-secret

# Custom JWT for resilient offline operation
JWT_SECRET=your-super-secret-jwt-signing-key-min-32-chars

# PostgreSQL connection string (shared 'noc' database)
DATABASE_URL=postgresql://backend:backend@localhost:5432/noc?sslmode=disable

# Redis connection URL
REDIS_URL=redis://localhost:6379

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
SMTP_FROM_NAME=Nethesis Operation Center
SMTP_TLS=true
```

**Auto-derived URLs:**
- `LOGTO_ISSUER` = `https://{TENANT_ID}.logto.app`
- `LOGTO_AUDIENCE` = `https://{TENANT_DOMAIN}/api`
- `JWKS_ENDPOINT` = `https://{TENANT_ID}.logto.app/oidc/jwks`
- `LOGTO_MANAGEMENT_BASE_URL` = `https://{TENANT_ID}.logto.app/api`
- `JWT_ISSUER` = `{TENANT_DOMAIN}`

## Email Configuration

The backend automatically sends welcome emails to newly created users with their temporary password and login instructions. Email functionality is optional and degrades gracefully if not configured.

### SMTP Setup

Configure SMTP settings in your environment:

```bash
# SMTP server details
SMTP_HOST=smtp.gmail.com          # Your SMTP server hostname
SMTP_PORT=587                     # SMTP port (587 for TLS, 465 for SSL, 25 for plain)
SMTP_USERNAME=your-email@gmail.com # SMTP authentication username
SMTP_PASSWORD=your-app-password    # SMTP authentication password
SMTP_FROM=noreply@yourdomain.com   # From email address for outgoing emails
SMTP_FROM_NAME=Your Company Name   # Display name for sender
SMTP_TLS=true                     # Enable TLS encryption (recommended)
```

### Supported Providers

**Gmail/Google Workspace:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password  # Use App Password, not account password
SMTP_TLS=true
```

**AWS SES:**
```bash
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USERNAME=your-ses-smtp-username
SMTP_PASSWORD=your-ses-smtp-password
SMTP_TLS=true
```

**SendGrid:**
```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=your-sendgrid-api-key
SMTP_TLS=true
```

### Email Features

- **Automatic Delivery**: Welcome emails sent when users are created via API
- **Modern Templates**: Responsive HTML and text templates with dark/light mode support
- **Secure Password Delivery**: Auto-generated temporary passwords sent securely
- **Smart Links**: Login URLs point directly to password change page
- **Graceful Degradation**: User creation succeeds even if email delivery fails
- **Security**: Passwords never logged, email content is sanitized

### Template Customization

Email templates are located in `services/email/templates/`:
- `welcome.html` - Modern HTML template with dark mode support
- `welcome.txt` - Plain text fallback template

Templates support Go template syntax with variables:
- `{{.UserName}}` - User's full name
- `{{.UserEmail}}` - User's email address
- `{{.OrganizationName}}` - Organization name
- `{{.OrganizationType}}` - Organization type (Owner, Distributor, etc.)
- `{{.UserRoles}}` - Array of user role names
- `{{.TempPassword}}` - Generated temporary password
- `{{.LoginURL}}` - Direct link to password change page
- `{{.SupportEmail}}` - Support contact email
- `{{.CompanyName}}` - Company name from SMTP configuration

## Callbacks

The backend supports OAuth-style system creation callbacks for external applications that need to integrate system provisioning into their workflows. This feature enables a seamless "one-click" system creation experience similar to GitHub CLI token acquisition.

### Key Features

- **Time-based Expiration**: State tokens expire after 1 hour
- **One-shot Protection**: Each state token can only be used once (24-hour blacklist)
- **Dual Callback Methods**: URL parameters + PostMessage for maximum compatibility
- **CSRF Protection**: State validation prevents cross-site request forgery

### Complete Documentation

**For detailed integration guides, security details, and working examples:**
**[examples/callbacks/README.md](./examples/callbacks/README.md)**

**Example Available:**
- `test-callback-page.html` - External application simulation with complete callback integration

**API Reference:** See [OpenAPI documentation](./openapi.yaml) for detailed schemas.

## Architecture

### Two-Layer Authorization
1. **Base Authentication**: All routes require valid Logto JWT
2. **Access Control**: Routes check permissions from user roles OR organization roles

### Permission Sources
- **User Roles** (technical capabilities): Admin, Support
- **Organization Roles** (business hierarchy): Owner, Distributor, Reseller, Customer

### User Impersonation System
Secure user impersonation allowing Owner organization users to temporarily become another user:

#### Features
- **Owner-Only Access**: Only users with "Owner" organization role can impersonate others
- **Secure Tokens**: 1-hour duration impersonation JWT tokens with embedded user data
- **Permission Filtering**: All API calls filtered by impersonated user's actual permissions
- **Audit Trail**: Complete logging of all impersonation activities for security compliance
- **UI Integration**: Frontend banner shows impersonation status with easy exit mechanism

#### Security Controls
- Prevents self-impersonation and token chaining
- Strict token validation prevents regular tokens being used as impersonation tokens
- Comprehensive security checks ensure only authorized users can initiate impersonation
- Token blacklisting for immediate session termination

#### API Endpoints
- `POST /api/auth/impersonate` - Start impersonating another user
- `POST /api/auth/exit-impersonation` - Exit impersonation and return to original user

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

### API Documentation
```bash
# Validate OpenAPI documentation
make validate-docs
```

### Testing

#### Authentication Testing
```bash
# Test token exchange
curl -X POST http://localhost:8080/api/auth/exchange \
  -H "Content-Type: application/json" \
  -d '{"access_token": "YOUR_LOGTO_TOKEN"}'

# Test with custom JWT
curl -X GET http://localhost:8080/api/me \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT"
```

#### User Impersonation Testing
```bash
# Impersonate a user (Owner users only)
curl -X POST http://localhost:8080/api/auth/impersonate \
  -H "Authorization: Bearer YOUR_OWNER_JWT" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "target_user_id"}'

# Verify impersonation is active
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_IMPERSONATION_JWT"

# Exit impersonation
curl -X POST http://localhost:8080/api/auth/exit-impersonation \
  -H "Authorization: Bearer YOUR_IMPERSONATION_JWT"
```

#### Email Testing
```bash
# Test welcome email service configuration
curl -X POST http://localhost:8080/api/users \
  -H "Authorization: Bearer YOUR_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "name": "Test User",
    "userRoleIds": ["role_123"],
    "organizationId": "org_456"
  }'

# Check logs for email delivery status
docker logs backend-container 2>&1 | grep "Welcome email"
```

#### SMTP Connection Testing
The backend automatically validates SMTP configuration on startup. Check logs for:
```
{"level":"info","message":"Welcome email service configuration test successful"}
```

If SMTP is misconfigured, you'll see:
```
{"level":"warn","message":"SMTP not configured, skipping welcome email"}
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