# Backend API

Go REST API server for the Nethesis Operation Center, providing secure authentication via Logto JWT tokens and a sophisticated Role-Based Access Control (RBAC) system.

## 🏗️ Architecture

### Framework & Dependencies
- **Framework**: [Gin](https://github.com/gin-gonic/gin) web framework
- **Authentication**: Logto JWT token validation via JWKS endpoint
- **JWT Handling**: [golang-jwt/jwt](https://github.com/golang-jwt/jwt)
- **Configuration**: Environment variables with [godotenv](https://github.com/joho/godotenv)
- **CORS & Compression**: Built-in middleware support

### Project Structure
```
backend/
├── main.go             # Server setup and route definitions
├── configuration/      # Environment configuration loading
├── middleware/         # Authentication and RBAC middleware
│   ├── auth.go         # Logto JWT authentication
│   ├── rbac.go         # Role-based access control
│   ├── org_rbac.go     # Organization-level RBAC
│   └── user_rbac.go    # User-specific RBAC
├── methods/            # HTTP request handlers
├── models/             # Data structures and business entities
├── logs/               # Logging utilities
└── response/           # HTTP response helpers
```

## 🔐 Authorization System

The API implements a multi-layered authorization approach:

### 1. Base Authentication
All protected routes require valid Logto JWT tokens:
```go
protected := api.Group("/", middleware.LogtoAuthMiddleware())
```

### 2. Role-Based Access Control
Routes can be protected by user roles (Support, Admin, Sales):
```go
adminGroup := protected.Group("/admin", middleware.AutoRoleRBAC("Admin"))
```

### 3. Organization Hierarchy
Business logic roles following organizational hierarchy:
```go
distributorGroup := protected.Group("/distributors",
    middleware.RequireAnyOrganizationRole("God", "Distributor"))
```

### 4. Fine-Grained Scopes
Specific permissions for sensitive operations:
```go
systemsGroup.POST("/:id/restart",
    middleware.RequireScope("manage:systems"), methods.RestartSystem)
```

### Authorization Levels
- **User Roles**: Support, Admin, Sales (general application permissions)
- **Organization Roles**: God, Distributor, Reseller, Customer (business hierarchy)
- **Scopes**: Fine-grained permissions (e.g., `admin:systems`, `manage:billing`)

## 🚀 Quick Start

### Prerequisites
- Go 1.23+
- Access to a Logto instance
- Logto API resource configured

### Setup
```bash
# Clone and navigate to backend directory
cd backend

# Copy environment template
cp .env.example .env

# Edit .env with your Logto configuration
# Required:
# - LOGTO_ISSUER=https://your-logto-instance.logto.app
# - LOGTO_AUDIENCE=your-api-resource-identifier
```

### Development
```bash
# Install dependencies
go mod download

# Run development server
go run main.go

# The server starts on http://127.0.0.1:8080 by default
```

### Building
```bash
# Build for production
go build -o backend main.go

# Run the binary
./backend
```

## 📝 Configuration

### Environment Variables

#### Required
- `LOGTO_ISSUER`: Your Logto instance URL (e.g., `https://your-logto.logto.app`)
- `LOGTO_AUDIENCE`: API resource identifier configured in Logto

#### Optional
- `JWKS_ENDPOINT`: JWT verification endpoint (auto-derived from issuer if not set)
- `LISTEN_ADDRESS`: Server bind address (default: `127.0.0.1:8080`)
- `GIN_MODE`: Gin framework mode (`debug`, `release`, `test`)

### Logto Setup
1. Create an API resource in your Logto instance
2. Configure the resource identifier as `LOGTO_AUDIENCE`
3. Ensure JWKS endpoint is accessible for token validation
4. Configure user roles and organization roles in Logto admin console

## 🛠️ API Endpoints

### Public Endpoints
- `GET /ping` - Health check endpoint

### Protected Endpoints (require authentication)
- `GET /user` - Current user information
- `GET /systems` - Systems management (requires Support+ role)
- `GET /distributors` - Distributor management (requires God/Distributor role)
- `GET /resellers` - Reseller management (requires God/Distributor role)
- `GET /customers` - Customer management (requires appropriate organization role)

### RBAC Examples
```go
// Route accessible to Support, Admin, or Sales roles
systemsGroup := protected.Group("/systems", middleware.AutoRoleRBAC("Support"))

// Route accessible to God or Distributor organization roles
distributorsGroup := protected.Group("/distributors",
    middleware.RequireAnyOrganizationRole("God", "Distributor"))

// Route requiring specific scope
systemsGroup.POST("/:id/restart",
    middleware.RequireScope("manage:systems"), methods.RestartSystem)
```

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./middleware

# Test coverage
go test -cover ./...
```

## 🔧 Development Commands

```bash
# Format code
go fmt ./...

# Vet code for common errors
go vet ./...

# Clean module dependencies
go mod tidy

# Update dependencies
go get -u ./...
```

## 📊 Monitoring & Debugging

### Logging
The application uses structured logging via the `logs` package:
- Request/response logging in debug mode
- Authentication and authorization events
- Error tracking and debugging information

### Health Check
- `GET /ping` returns server status and can be used for health monitoring

### CORS
CORS is configured for development with permissive settings. For production, configure appropriate origins.

## 🤝 Contributing

1. Follow Go best practices and conventions
2. Use the existing middleware patterns for new protected routes
3. Maintain the RBAC authorization model consistency
4. Add appropriate tests for new functionality
5. Format code with `go fmt` before committing

## 🔗 Related Documentation

- [Project Overview](../README.md) - Main project documentation
- [CLAUDE.md](../CLAUDE.md) - Development guidance
- [logto-sync](../logto-sync/README.md) - RBAC configuration management tool