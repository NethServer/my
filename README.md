# Nethesis Operation Center

A web application providing centralized authentication and management using Logto as an Identity Provider, built with a sophisticated Role-Based Access Control (RBAC) system.

## 🏗️ Project Structure

```
my/
├── backend/            # Go REST API with Logto JWT authentication
├── logto-sync/         # CLI tool for RBAC configuration synchronization
├── DESIGN.md           # Project design documentation
└── LICENSE             # Project license
```

## 🚀 Components

### Backend API
Go-based REST API featuring:
- **Authentication**: Logto JWT token validation via JWKS
- **Authorization**: Hybrid RBAC system with multiple permission layers
- **Framework**: Gin web framework with middleware architecture
- **Roles**: Support, Admin, Sales (user roles) + God, Distributor, Reseller, Customer (organization hierarchy)

### logto-sync CLI
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
- **logto-sync CLI**: See [logto-sync/README.md](./logto-sync/README.md) for RBAC management

## 🔐 Authorization Architecture

The system implements a sophisticated three-tier authorization model:

1. **Base Authentication**: JWT validation via Logto
2. **Role-Based Access**: General user roles (Support, Admin, Sales)
3. **Organization Hierarchy**: Business roles (God → Distributor → Reseller → Customer)
4. **Fine-Grained Scopes**: Specific permissions (e.g., `admin:systems`, `manage:billing`)

### Example Authorization Flow
```go
// Route with multiple authorization layers
protected := api.Group("/", middleware.LogtoAuthMiddleware())
systemsGroup := protected.Group("/systems", middleware.AutoRoleRBAC("Support"))
systemsGroup.POST("/:id/restart", middleware.RequireScope("manage:systems"), methods.RestartSystem)
```

## 📝 Configuration

Configuration details are specific to each component:
- **Backend**: Environment variables for Logto JWT validation
- **logto-sync**: Management API credentials and RBAC configuration files

See individual README files for detailed configuration instructions.

## 📚 Documentation

### Component Documentation
- **[Backend API](./backend/README.md)** - Go REST API server setup, RBAC system, and development guide
- **[logto-sync CLI](./logto-sync/README.md)** - RBAC configuration management tool and usage instructions

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
