# Backend API

Go REST API server for the Nethesis Operation Center, providing secure authentication via Logto JWT tokens and a simplified Role-Based Access Control (RBAC) system with clear separation between business hierarchy and technical capabilities.

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
├── main.go                    # Server setup and route definitions
├── configuration/             # Environment configuration loading
├── middleware/                # Authentication and simplified RBAC middleware
│   ├── auth.go                # Logto JWT authentication with simplified claims
│   └── simplified_rbac.go     # Unified RBAC with business + technical separation
├── methods/                   # HTTP request handlers
├── models/                    # Data structures with simplified User model
├── logs/                      # Logging utilities
└── response/                  # HTTP response helpers
```

## 🔐 Simplified Authorization System

The API implements a clean two-layer authorization approach with clear separation:

### 1. Base Authentication
All protected routes require valid Logto JWT tokens:
```go
protected := api.Group("/", middleware.LogtoAuthMiddleware())
```

### 2. Permission-Based Access Control
Routes check for specific permissions from either user roles OR organization roles:
```go
// Checks both user and organization permissions
systemsGroup.POST("/:id/restart",
    middleware.RequirePermission("manage:systems"), methods.RestartSystem)
```

### 3. User Role-Based (Technical Capabilities)
Routes can be protected by technical capability roles:
```go
adminGroup := protected.Group("/admin", middleware.RequireUserRole("Admin"))
```

### 4. Organization Role-Based (Business Hierarchy)
Routes can be protected by business hierarchy roles:
```go
distributorGroup := protected.Group("/distributors",
    middleware.RequireAnyOrgRole("God", "Distributor"))
```

### Authorization Model
**Two Clear Sources of Permissions:**

#### **User Roles** (Technical Capabilities)
- **Admin**: Complete platform administration, dangerous operations
- **Support**: System management, customer troubleshooting, standard operations

#### **Organization Roles** (Business Hierarchy)
- **God**: Complete control over commercial hierarchy (Nethesis level)
- **Distributor**: Can manage resellers and customers
- **Reseller**: Can manage customers only
- **Customer**: Read-only access to own data

#### **Permission Logic**
```
Final User Permissions = User Role Permissions + Organization Role Permissions
```

Users get permissions from BOTH their technical capabilities AND their organization's business role.

## 🚀 Quick Start

### System Requirements

#### Runtime (Binary Execution)
- **Any 64-bit OS**: Linux, macOS, Windows
- **No dependencies** - Statically compiled Go binary

#### Development & Building
- **Go 1.23+** - [Download from golang.org](https://golang.org/download/)
- **Make** (for build automation):
  - **macOS**: Preinstalled with Xcode Command Line Tools (`xcode-select --install`)
  - **Linux**: Usually preinstalled, or install with package manager (`apt install build-essential`)
  - **Windows**: Install via [Git Bash](https://git-scm.com/download/win), [WSL2](https://docs.microsoft.com/en-us/windows/wsl/install), or [Chocolatey](https://chocolatey.org/) (`choco install make`)
- **golangci-lint** (optional, for linting): [Installation guide](https://golangci-lint.run/usage/install/)

#### External Dependencies
- **Logto instance** - Identity provider
- **Logto API resource** - Configured in your Logto admin console
- **Logto Management API** - Machine-to-Machine app for RBAC synchronization

### Alternative Build Methods
If Make is not available, you can use Go commands directly:
```bash
# Development
go run main.go

# Build
mkdir -p build && go build -o build/backend main.go

# Test
go test ./...
```

### Token Exchange System
This API now supports a **token exchange pattern** for enhanced security and performance:

1. **Frontend** authenticates with Logto → gets `access_token`
2. **Frontend** calls `POST /api/auth/exchange` with `access_token`
3. **Backend** validates token, enriches with roles/permissions → returns custom JWT
4. **Frontend** uses custom JWT for all subsequent API calls

### Setup
```bash
# Clone and navigate to backend directory
cd backend

# Setup development environment (creates .env from template)
make dev-setup

# Edit .env with your Logto configuration
# Required:
# - LOGTO_ISSUER=https://your-logto-instance.logto.app
# - LOGTO_AUDIENCE=your-api-resource-identifier
```

### Development
```bash
# Install dependencies and start development server
make dev

# The server starts on http://127.0.0.1:8080 by default
```

### Building
```bash
# Build for production
make build

# Run the binary
./build/backend

# Or build for multiple platforms
make build-all
```

## 📝 Configuration

### Environment Variables

#### Required
- `LOGTO_ISSUER`: Your Logto instance URL (e.g., `https://your-logto.logto.app`)
- `LOGTO_AUDIENCE`: API resource identifier configured in Logto
- `JWT_SECRET`: Secret key for signing custom JWT tokens (required for token exchange)
- `LOGTO_MANAGEMENT_CLIENT_ID`: Management API machine-to-machine app client ID
- `LOGTO_MANAGEMENT_CLIENT_SECRET`: Management API machine-to-machine app secret

#### Optional
- `JWKS_ENDPOINT`: JWT verification endpoint (auto-derived from issuer if not set)
- `LISTEN_ADDRESS`: Server bind address (default: `127.0.0.1:8080`)
- `JWT_ISSUER`: Custom JWT issuer (default: `my.nethesis.it`)
- `JWT_EXPIRATION`: Custom JWT expiration time (default: `24h`)
- `LOGTO_MANAGEMENT_BASE_URL`: Management API base URL (auto-derived from issuer if not set)
- `GIN_MODE`: Gin framework mode (`debug`, `release`, `test`)

### Logto Setup

#### 1. API Resource Setup
1. Create an API resource in your Logto instance
2. Configure the resource identifier as `LOGTO_AUDIENCE`
3. Ensure JWKS endpoint is accessible for token validation

#### 2. Management API Setup
1. Create a **Machine-to-Machine** application in Logto admin console
2. Configure API scopes: **Grant all Management API permissions**
3. Copy the Client ID and Client Secret to your environment variables
4. This enables the backend to fetch real user roles and permissions

#### 3. RBAC Configuration
1. Configure user roles and organization roles in Logto admin console
2. Assign permissions to roles using the simplified RBAC structure
3. Use `sync` tool to synchronize your RBAC configuration
4. Users will automatically get the correct permissions in their custom JWT tokens

## 🛠️ API Endpoints

### Public Endpoints
- `GET /api/health` - Health check endpoint
- `POST /api/auth/exchange` - Exchange Logto access_token for custom JWT

### Protected Endpoints (Custom JWT Required)
- `GET /api/profile` - User profile with full context
- `GET /api/protected` - Protected resource example

#### Systems Management
- `GET /api/systems` - List systems (requires `read:systems`)
- `POST /api/systems` - Create system (requires `manage:systems`)
- `PUT /api/systems/:id` - Update system (requires `manage:systems`)
- `DELETE /api/systems/:id` - Delete system (requires `admin:systems`)
- `POST /api/systems/:id/restart` - Restart system (requires `manage:systems`)
- `DELETE /api/systems/:id/destroy` - Destroy system (requires `destroy:systems`)

#### Account Management
- `GET /api/accounts` - List accounts with hierarchical filtering (requires custom auth)
- `POST /api/accounts` - Create account with hierarchical validation (requires custom auth)
- `PUT /api/accounts/:id` - Update account (requires custom auth)
- `DELETE /api/accounts/:id` - Delete account (requires custom auth)

#### Business Hierarchy Management
- `GET /api/distributors` - List distributors (requires `create:distributors`)
- `POST /api/distributors` - Create distributor (requires `create:distributors`)
- `PUT /api/distributors/:id` - Update distributor (requires `manage:distributors`)

- `GET /api/resellers` - List resellers (requires `create:resellers`)
- `POST /api/resellers` - Create reseller (requires `create:resellers`)
- `PUT /api/resellers/:id` - Update reseller (requires `manage:resellers`)

- `GET /api/customers` - List customers (requires `create:customers`)
- `POST /api/customers` - Create customer (requires `create:customers`)
- `PUT /api/customers/:id` - Update customer (requires `manage:customers`)

#### Statistics
- `GET /api/stats` - System statistics (requires `create:distributors`)

### Simplified RBAC Examples

```go
// ✅ Unified permission checking (checks both user AND org permissions)
systemsGroup.POST("/:id/restart",
    middleware.RequirePermission("manage:systems"), methods.RestartSystem)

// ✅ Permission-based authorization for business hierarchy
distributorsGroup := protected.Group("/distributors",
    middleware.RequirePermission("create:distributors"))

// ✅ Granular permissions for different operations
systemsGroup.GET("", methods.GetSystems) // Base permission from group
systemsGroup.POST("", middleware.RequirePermission("manage:systems"), methods.CreateSystem)
systemsGroup.DELETE("/:id", middleware.RequirePermission("admin:systems"), methods.DeleteSystem)
```

### Real-World Use Cases

```go
// Marco (ACME Reseller + Admin) can:
// ✅ admin:systems, destroy:systems (from Admin user role)
// ✅ create:customers (from Reseller org role)

// Edoardo (Nethesis Distributor + Support) can:
// ✅ manage:systems (from Support user role)
// ✅ create:resellers (from Distributor org role)
// ✅ create:customers (from Distributor org role)
```

## 👥 Hierarchical Account Management

The API implements sophisticated hierarchical account management that follows business rules and organizational hierarchy:

### Authorization Rules

#### **Hierarchical Account Creation**
- **God (Nethesis)**: Can create accounts for any organization type
- **Distributor**: Can create accounts for Reseller and Customer organizations + own organization (if Admin)
- **Reseller**: Can create accounts for Customer organizations + own organization (if Admin)
- **Customer**: Can create accounts only for own organization (if Admin)

#### **Same-Organization Rule**
Only Admin users can create accounts for colleagues within the same organization.

### Account Management Endpoints

#### **POST /api/accounts**
Creates a new account with hierarchical validation:

```json
{
  "username": "mario.rossi",
  "email": "mario@acme.com",
  "name": "Mario Rossi",
  "phone": "+393334455667",
  "password": "SecurePassword123!",
  "userRole": "Admin",
  "organizationId": "org_acme_12345",
  "organizationRole": "Reseller",
  "avatar": "https://example.com/avatar.jpg",
  "metadata": {
    "department": "IT",
    "location": "Milan"
  }
}
```

**Response:**
```json
{
  "code": 201,
  "message": "account created successfully",
  "data": {
    "id": "user_generated_id",
    "username": "mario.rossi",
    "email": "mario@acme.com",
    "name": "Mario Rossi",
    "phone": "+393334455667",
    "userRole": "Admin",
    "organizationId": "org_acme_12345",
    "organizationName": "ACME S.r.l.",
    "organizationRole": "Reseller",
    "isSuspended": false,
    "createdAt": "2025-01-15T10:30:00Z",
    "updatedAt": "2025-01-15T10:30:00Z",
    "metadata": {
      "department": "IT",
      "location": "Milan"
    }
  }
}
```

#### **GET /api/accounts**
Retrieves accounts with hierarchical filtering:

- **God**: Sees all accounts across all organizations
- **Distributor**: Sees accounts from organizations they created + sub-organizations
- **Reseller**: Sees accounts from Customer organizations they created
- **Customer**: Cannot access this endpoint

**Query Parameters:**
- `organizationId`: Filter accounts by specific organization

#### **PUT /api/accounts/:id**
Updates account information with role and organization changes.

#### **DELETE /api/accounts/:id**
Deletes an account from the system.

### Hierarchical Data Visibility

The system implements data visibility based on organizational hierarchy and creation relationships:

#### **Visibility Rules**
- **God**: Can see all organizations and their accounts
- **Distributors**: Can see:
  - Resellers they created (`customData.createdBy = distributor.organizationId`)
  - Customers created by their resellers (transitively)
- **Resellers**: Can see:
  - Customers they created (`customData.createdBy = reseller.organizationId`)
- **Customers**: Cannot access organization management endpoints

#### **Creation Tracking**
When accounts are created, they include visibility metadata:
```json
{
  "customData": {
    "createdBy": "creating-organization-id",
    "createdAt": "2025-01-15T10:30:00Z",
    "userRole": "Admin",
    "organizationId": "target-org-id",
    "organizationRole": "Reseller"
  }
}
```

## 🧪 Testing

```bash
# Primary commands (recommended)
make test                          # Run all tests
make test-coverage                 # Run tests with coverage report
make fmt                           # Format code (required for CI)

# Direct Go commands for specific needs
go test ./jwt                      # Test JWT package only  
go test ./middleware               # Test middleware package only
go test -v ./...                   # Verbose test output
go test -race ./...                # Race condition detection
go test -count=1 ./...             # Force test execution (disable cache)
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

## 🔧 Development Commands

```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Clean dependencies
make tidy

# Clean build artifacts
make clean

# Install binary globally
make install

# Show all available commands
make help
```

## 📊 Monitoring & Debugging

### Logging
The application uses structured logging via the `logs` package:
- Request/response logging in debug mode
- Authentication and authorization events
- Error tracking and debugging information

**Key Log Messages:**
```
[INFO][LOGTO] Management API token obtained, expires at ...
[INFO][LOGTO] Enriched user user-123 with 1 user roles, 4 user permissions, org role 'Distributor', 4 org permissions
[INFO][ACCOUNTS] Account created in Logto: Mario Rossi (ID: user_123) by user user_456
```

### Health Check
- `GET /ping` returns server status and can be used for health monitoring

### CORS
CORS is configured for development with permissive settings. For production, configure appropriate origins.

### Troubleshooting

#### **Token Exchange Issues**
1. **Management API Connection Fails**
   - Check `LOGTO_MANAGEMENT_CLIENT_ID` and `LOGTO_MANAGEMENT_CLIENT_SECRET`
   - Verify Machine-to-Machine app has all Management API permissions
   - Check network connectivity to Logto instance

2. **Empty Permissions in JWT**
   - Verify user has roles assigned in Logto admin console
   - Check role permissions configuration in Logto
   - Ensure user has organization membership

3. **403 Forbidden on API Calls**
   - Verify JWT contains expected permissions (decode at jwt.io)
   - Check middleware permission requirements in source code
   - Confirm RBAC configuration matches API expectations

#### **Account Management Issues**
1. **Account Creation Denied**
   - Check hierarchical authorization rules
   - Verify user has Admin role for same-organization creation
   - Confirm target organization exists and is valid

2. **Visibility Issues**
   - Check organization creation relationships in Logto
   - Verify `customData.createdBy` tracking
   - Confirm user's organization role permissions

#### **Debugging Tools**
```bash
# Test token exchange
curl -X POST http://localhost:8080/api/auth/exchange \
  -H "Content-Type: application/json" \
  -d '{"access_token": "YOUR_LOGTO_TOKEN"}'

# Check user profile with custom JWT
curl -X GET http://localhost:8080/api/profile \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT"

# Test specific permissions
curl -X GET http://localhost:8080/api/systems \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT"
```

## 🤝 Contributing

1. Follow Go best practices and conventions
2. Use the existing middleware patterns for new protected routes
3. Maintain the RBAC authorization model consistency
4. Add appropriate tests for new functionality
5. Format code with `go fmt` before committing

## 🔗 Related Documentation

- [Project Overview](../README.md) - Main project documentation
- [sync](../sync/README.md) - RBAC configuration management tool