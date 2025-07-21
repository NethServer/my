# Nethesis Operation Center

##### Documentation
[![API Docs build](https://img.shields.io/github/actions/workflow/status/NethServer/my/docs.yml?style=for-the-badge&label=API%20Docs%20build)](https://github.com/NethServer/my/actions/workflows/docs.yml)
[![API Docs link](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=API%20Docs%20link)](https://bump.sh/nethesis/doc/my)

##### CI and Tests
[![CI](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?style=for-the-badge&label=CI)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![Backend Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=backend-tests&label=Backend%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![Collect Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=collect-tests&label=Collect%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)
[![sync Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci.yml?job=sync-tests&label=sync%20Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci.yml)


##### Release
[![Release](https://img.shields.io/github/actions/workflow/status/NethServer/my/release.yml?style=for-the-badge&label=Release)](https://github.com/NethServer/my/actions/workflows/release.yml)
[![Version](https://img.shields.io/github/v/release/NethServer/my?style=for-the-badge&color=3a3c3f&label=Version)](https://github.com/NethServer/my/releases)

##### Deployments
[![Deploy Redis](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-redis&label=Deploy%20Redis&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Postgres](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-postgres&label=Deploy%20Postgres&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Backend](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-backend&label=Deploy%20Backend&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Collect](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-collect&label=Deploy%20Collect&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Frontend](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-frontend&label=Deploy%20Frontend&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)
[![Deploy Proxy](https://img.shields.io/github/actions/workflow/status/NethServer/my/deploy.yml?job=deploy-proxy&label=Deploy%20Proxy&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/deploy.yml)

##### Production and QA links
[![My](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=my.nethesis.it)](https://my-proxy-prod.onrender.com)
[![My QA](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=qa.my.nethesis.it)](https://my-proxy-qa.onrender.com)

Web application providing centralized authentication and management using Logto as an Identity Provider with simple Role-Based Access Control.

## 🏗️ Components

- **[frontend/](./frontend/)** - Vue.js application for UI
- **[backend/](./backend/)** - Go REST API with Logto JWT authentication and RBAC
- **[collect/](./collect/)** - Go REST API with Redis queues to handle inventories
- **[sync/](./sync/)** - CLI tool for RBAC configuration synchronization
- **[proxy/](./proxy/)** - nginx configuration as load balancer

## 🚀 Quick Start

### Requirements
- **Development**: Go 1.21+ (backend requires 1.23+), Make
- **External**: Logto instance with M2M app and Management API permissions
- **Deploy**: Render account with GitHub integration

### Getting Started
1. **Frontend Development**: [frontend/README.md](./frontend/README.md) - Vue.js setup and environment configuration
1. **Backend Development**: [backend/README.md](./backend/README.md) - Backend setup and environment configuration
1. **Collect Development**: [collect/README.md](./collect/README.md) - Collect setup and environment configuration
2. **RBAC Management**: [sync/README.md](./sync/README.md) - Use `sync init` for complete setup
3. **Production Deploy**: [deploy/README.md](./deploy/README.md) - Automated deployment with Render

## 🌐 Deployment Environments

### QA (`qa.my.nethesis.it`)
- **Trigger**: Every commit to `main` branch
- **Auto-deploy**: Immediate deployment via Render
- **PR Previews**: Temporary environments for pull requests

### Production (`my.nethesis.it`)
- **Trigger**: Manual deployment via GitHub Actions
- **Sequential Deploy**: Redis + PostgreSQL → Backend + Collect → Frontend → Proxy
- **Manual Control**: Deploy only when explicitly triggered
- **Security**: Private services (Backend, Collect, Frontend) only accessible through Proxy

## 📝 Configuration

### Local Development
See individual component documentation for setup:
- **Fronted**: [frontend/README.md](./frontend/README.md) - Environment variables and setup for frontend
- **Backend**: [backend/README.md](./backend/README.md) - Environment variables and setup for backend
- **Collect**: [collect/README.md](./collect/README.md) - Environment variables and setup for collect
- **sync CLI**: [sync/README.md](./sync/README.md) - Use `sync init` to generate all required variables
- **proxy**: [proxy/README.md](./proxy/README.md) - nginx configuration and setup for load balancer

### Production Deployment
- **Environment Variables**: Configured in Render dashboard
- **GitHub Secrets**: API keys for automated deployment
- **Service Configuration**: Defined in `render.yaml`
- **Full Guide**: [deploy/README.md](./deploy/README.md) - Complete deployment setup

## 📚 Documentation

- **[frontend](./frontend/README.md)** - UI setup, environment variables, and pages
- **[backend](./backend/README.md)** - Server setup, environment variables, and authorization architecture
- **[backend API](./backend/API.md)** - Complete API reference with authentication
- **[collect](./collect/README.md)** - Server setup, environment variables and inventory structure
- **[sync CLI](./sync/README.md)** - RBAC configuration and `sync init` setup
- **[deploy](./deploy/README.md)** - Production deployment with [Render](render.yaml) and GitHub Actions
- **[proxy](./proxy/README.md)** - Production load balancer configuration with nginx
- **[DESIGN.md](./DESIGN.md)** - Architecture decisions and design patterns

### 📖 API Documentation
**Live Documentation:** https://bump.sh/nethesis/doc/my - auto-updated on every commit.

## 🤝 Development Workflow

### Standard Development
```bash
git commit -m "feat: new feature"
git push origin main                    # → qa.my.nethesis.it updates
```

### Feature Testing
```bash
git checkout -b feature/new-feature
git push origin feature/new-feature     # → Create PR
# → qa.my-proxy-qa-pr-123.onrender.com created
```

### Production Release
```bash
# 1. Automated release with quality checks
./release.sh patch                       # → 0.0.5 → 0.0.6 (bug fixes)
./release.sh minor                       # → 0.0.5 → 0.1.0 (new features)
./release.sh major                       # → 0.0.5 → 1.0.0 (breaking changes)
# → Runs tests, formatting, linting → Creates tag → Pushes to GitHub

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
