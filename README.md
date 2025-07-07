# Nethesis Operation Center

[![CI](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?style=for-the-badge&label=CI)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![Backend Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=backend-tests&label=Backend%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![sync Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=sync-tests&label=sync%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)

[![Release](https://img.shields.io/github/actions/workflow/status/NethServer/my/release.yml?style=for-the-badge&label=Release)](https://github.com/NethServer/my/actions/workflows/release.yml)
[![Version](https://img.shields.io/github/v/release/NethServer/my?style=for-the-badge&color=3a3c3f&label=Version)](https://github.com/NethServer/my/releases)

Web application providing centralized authentication and management using Logto as an Identity Provider with sophisticated Role-Based Access Control.

## 🏗️ Components

- **[backend/](./backend/)** - Go REST API with Logto JWT authentication and RBAC
- **[sync/](./sync/)** - CLI tool for RBAC configuration synchronization

## 🚀 Quick Start

### Requirements
- **Runtime**: Any 64-bit OS (Linux, macOS, Windows) - statically linked binaries
- **Development**: Go 1.21+ (backend requires 1.23+), Make
- **External**: Logto instance with M2M app and Management API permissions

### Getting Started
1. **Backend**: See [backend/README.md](./backend/README.md) for server setup
2. **sync CLI**: See [sync/README.md](./sync/README.md) for RBAC management

## 🔐 Authorization Architecture

Token exchange pattern with real-time Management API integration:

1. Frontend exchanges Logto access_token for custom JWT with embedded permissions
2. Real-time fetching of user roles and organization memberships from Logto
3. All permissions pre-computed and embedded in custom JWT
4. Single middleware checks both user and organization permissions

## 📝 Configuration

### Backend
```bash
# Logto Authentication
LOGTO_ISSUER=https://your-tenant-id.logto.app
LOGTO_AUDIENCE=https://your-domain.com/api

# Custom JWT (for legacy endpoints)
JWT_SECRET=your-super-secret-jwt-signing-key

# Management API
BACKEND_APP_ID=your-management-api-app-id
BACKEND_APP_SECRET=your-management-api-app-secret

# Server
LISTEN_ADDRESS=127.0.0.1:8080
```

### sync Tool
```bash
# Required
TENANT_ID=your-tenant-id
BACKEND_APP_ID=your-backend-m2m-app-id
BACKEND_APP_SECRET=your-backend-m2m-app-secret

# For 'sync init' command only
TENANT_DOMAIN=your-domain.com
```

## 📚 Documentation

- **[Backend](./backend/README.md)** - Go REST API server setup and development
- **[Backend API](./backend/API.md)** - Complete API reference
- **[sync CLI](./sync/README.md)** - RBAC configuration management
- **[DESIGN.md](./DESIGN.md)** - Architecture and design decisions

## 🤝 Contributing

1. Follow existing code patterns and conventions
2. **Pre-commit**: Run `make fmt && make test` in both directories
3. Test RBAC changes with `--dry-run` before applying
4. Ensure CI tests pass before submitting PRs

## 📄 License

See [LICENSE](./LICENSE) file for details.
