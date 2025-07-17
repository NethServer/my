# Nethesis Operation Center

[![Docs build](https://img.shields.io/github/actions/workflow/status/NethServer/my/docs.yml?style=for-the-badge&label=Docs%20build)](https://github.com/NethServer/my/actions/workflows/docs.yml)
[![Docs link](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=Docs%20link)](https://bump.sh/nethesis/doc/my)

[![CI](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?style=for-the-badge&label=CI)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![Backend Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=backend-tests&label=Backend%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![sync Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=sync-tests&label=sync%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)


[![Release](https://img.shields.io/github/actions/workflow/status/NethServer/my/release.yml?style=for-the-badge&label=Release)](https://github.com/NethServer/my/actions/workflows/release.yml)
[![Version](https://img.shields.io/github/v/release/NethServer/my?style=for-the-badge&color=3a3c3f&label=Version)](https://github.com/NethServer/my/releases)

[![Deploy Redis](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-redis&label=Deploy%20Redis&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Backend](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-backend&label=Deploy%20Backend&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Frontend](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-frontend&label=Deploy%20Frontend&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)

Web application providing centralized authentication and management using Logto as an Identity Provider with simple Role-Based Access Control.

## 🏗️ Components

- **[backend/](./backend/)** - Go REST API with Logto JWT authentication and RBAC
- **[sync/](./sync/)** - CLI tool for RBAC configuration synchronization

## 🚀 Quick Start

### Requirements
- **Development**: Go 1.21+ (backend requires 1.23+), Make
- **External**: Logto instance with M2M app and Management API permissions
- **Deploy**: Render account with GitHub integration

### Getting Started
1.1 **Local Development Backend**: [backend/README.md](./backend/README.md) - Backend setup and environment configuration
1.2 **Local Development Collect**: [collect/README.md](./collect/README.md) - Collect setup and environment configuration
2. **RBAC Management**: [sync/README.md](./sync/README.md) - Use `sync init` for complete setup
3. **Production Deploy**: [deploy/README.md](./deploy/README.md) - Automated deployment with Render

## 🔐 Authorization Architecture

**Token Exchange Pattern**: Frontend exchanges Logto access_token for custom JWT with embedded permissions

**Key Features**:
- Real-time role and permission fetching from Logto
- Pre-computed permissions embedded in JWT
- Combined user roles and organization roles (business hierarchy) model

**Details**: See [backend/README.md](./backend/README.md) for complete architecture documentation

## 🌐 Deployment Environments

### Development (`<custom_path>.my.nethesis.it`)
- **Environment**: Uses `.env.development` configuration
- **Usage**: Local development and testing
- **Setup**: Run components locally with default GO_ENV=development

### QA (`qa.my.nethesis.it`)
- **Environment**: Uses `.env.qa` configuration
- **Usage**: Pre-production testing and validation
- **Setup**: Set `GO_ENV=qa` before running components

### Production (`my.nethesis.it`)
- **Environment**: Uses `.env.production` configuration
- **Trigger**: Manual deployment via GitHub Actions
- **Sequential Deploy**: Redis → PostgreSQL → Backend + Collect → Frontend
- **Setup**: Set `GO_ENV=production` in production environment

## 📝 Configuration

### Environment Management
All components support multi-environment configuration:
- **Development**: `.env.development` (default for local development)
- **QA**: `.env.qa` (pre-production testing)
- **Production**: `.env.production` (production deployment)

Set `GO_ENV` variable to switch environments:
```bash
export GO_ENV=qa         # Uses .env.qa
export GO_ENV=production # Uses .env.production
# Default: development   # Uses .env.development
```

### Makefile Commands
All components support environment-specific Makefile targets:
```bash
# Development (default)
make run

# QA environment
make run-qa

# Production environment
make run-production
```

### Local Development
See individual component documentation for setup:
- **Backend**: [backend/README.md](./backend/README.md) - Environment variables and setup
- **sync CLI**: [sync/README.md](./sync/README.md) - Use `sync init` to generate all required variables

### Production Deployment
- **Environment Variables**: Configured in Render dashboard
- **GitHub Secrets**: API keys for automated deployment
- **Service Configuration**: Defined in `render.yaml`
- **Full Guide**: [deploy/README.md](./deploy/README.md) - Complete deployment setup

## 📚 Documentation

- **[backend](./backend/README.md)** - Server setup, environment variables, and authorization architecture
- **[backend API](./backend/API.md)** - Complete API reference with authentication
- **[sync CLI](./sync/README.md)** - RBAC configuration and `sync init` setup
- **[deploy](./deploy/README.md)** - Production deployment with [Render](render.yaml) and GitHub Actions
- **[DESIGN.md](./DESIGN.md)** - Architecture decisions and design patterns

### 📖 API Documentation
**Live Documentation:** https://bump.sh/nethesis/doc/my - auto-updated on every commit.

## 🤝 Development Workflow

### Standard Development
```bash
git commit -m "feat: new feature"
git push origin main                    # → dev.my.nethesis.it updates
```

### Feature Testing
```bash
git checkout -b feature/new-feature
git push origin feature/new-feature     # → Create PR
# → pr-123.my-backend-dev.onrender.com created
```

### Production Release
```bash
# 1. Create and push release tag
git tag v1.2.3
git push origin v1.2.3                  # → Create GitHub release + containers

# 2. Manual deployment trigger
# Go to: https://github.com/NethServer/my/actions/workflows/deploy.yml
# Click "Run workflow" → Enter version "v1.2.3" → Deploy
```

## 🤝 Contributing

1. Follow existing code patterns and conventions
2. **Pre-commit**: Run `make pre-commit` in both directories
3. Test RBAC changes with `--dry-run` before applying
4. Ensure CI tests pass before submitting PRs

## 📄 License

See [LICENSE](./LICENSE) file for details.
