# Backend API

Go REST API server for Nethesis Operation Center with Logto JWT authentication and Role-Based Access Control (RBAC).

## Quick Start

### Prerequisites
- Go 1.23+
- Logto instance with M2M app configured

### Setup
```bash
# Setup development environment
make dev-setup

# Edit .env with your Logto configuration
# Start development server
make dev
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
```

### Optional Environment Variables
```bash
LISTEN_ADDRESS=127.0.0.1:8080
JWKS_ENDPOINT=https://your-logto-instance.logto.app/oidc/jwks
JWT_ISSUER=your-api.com
JWT_EXPIRATION=24h
JWT_REFRESH_EXPIRATION=168h
LOGTO_MANAGEMENT_BASE_URL=https://your-logto-instance.logto.app/api
```

## Architecture

### Two-Layer Authorization
1. **Base Authentication**: All routes require valid Logto JWT
2. **Access Control**: Routes check permissions from user roles OR organization roles

### Permission Sources
- **User Roles** (technical capabilities): Admin, Support
- **Organization Roles** (business hierarchy): Owner, Distributor, Reseller, Customer

## API Endpoints

See [API.md](API.md) for complete API documentation.

## Development

### Commands
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
├── configuration/          # Environment config
├── middleware/             # Auth and RBAC middleware
├── methods/                # HTTP handlers
├── models/                 # Data structures
├── services/               # Business logic
├── logger/                 # Structured logging
└── response/               # HTTP response helpers
```

## Related
- [API.md](API.md) - Complete API documentation
- [sync](../sync/README.md) - RBAC configuration tool
- [Project Overview](../README.md) - Main documentation