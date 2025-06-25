# Nethesis Operation Center

A web application providing centralized authentication and management using Logto as an Identity Provider, built with a sophisticated Role-Based Access Control (RBAC) system.

## 🏗️ Project Structure

```
my/
├── backend/            # Go REST API with Logto JWT authentication
├── sync/              # CLI tool for RBAC configuration synchronization
├── DESIGN.md           # Project design documentation
└── LICENSE             # Project license
```

## 🚀 Components

### Backend API
Go-based REST API featuring:
- **Authentication**: Token exchange system with custom JWT generation
- **Data Integration**: Real-time Management API integration for roles/permissions
- **Authorization**: Simplified RBAC system with embedded permissions
- **Account Management**: Hierarchical account creation and management with business rule enforcement
- **Framework**: Gin web framework with middleware architecture
- **Roles**: Admin, Support (user roles) + God, Distributor, Reseller, Customer (organization hierarchy)

### sync CLI
Management tool for synchronizing RBAC configuration:
- **Configuration Management**: YAML-based role and permission definitions
- **Custom JWT Claims**: JavaScript-based claim customization
- **Dry-run Support**: Safe configuration preview before applying changes
- **Logto Integration**: Direct Management API synchronization

## 🛠️ Quick Start

### Prerequisites
- Go 1.23+
- Logto instance configured
- Valid Logto Management API credentials

### Getting Started
Each component has its own setup instructions:

- **Backend API**: See [backend/README.md](./backend/README.md) for API server setup
- **sync CLI**: See [sync/README.md](./sync/README.md) for RBAC management

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

Configuration details are specific to each component:
- **Backend**: Environment variables for Logto integration and Management API credentials
- **sync**: Management API credentials and RBAC configuration files

### Key Requirements
- **Logto Instance**: Identity provider with RBAC configuration
- **Management API**: Machine-to-Machine app with full API permissions
- **Custom JWT**: Backend secret for signing custom tokens

See individual README files for detailed configuration instructions.

## 📚 Documentation

### Component Documentation
- **[Backend API](./backend/README.md)** - Go REST API server setup, RBAC system, and development guide
- **[sync CLI](./sync/README.md)** - RBAC configuration management tool and usage instructions

### Project Documentation
- **[CLAUDE.md](./CLAUDE.md)** - Comprehensive development guidance for AI assistance
- **[DESIGN.md](./DESIGN.md)** - Project architecture and design decisions

## 🤝 Contributing

1. Follow existing code patterns and conventions
2. Use the provided development commands for formatting and testing
3. Always test RBAC changes with `--dry-run` before applying
4. Ensure environment variables are properly configured

## 📄 License

See [LICENSE](./LICENSE) file for details.

---

**🔗 Related Links**
- [my.nethesis.it](https://my.nethesis.it) - Production application
- [Logto Documentation](https://docs.logto.io/) - Identity provider documentation
