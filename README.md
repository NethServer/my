# My Nethesis

#### Documentation
[![API Docs build](https://img.shields.io/github/actions/workflow/status/NethServer/my/docs-api.yml?style=for-the-badge&label=API%20Docs%20build)](https://github.com/NethServer/my/actions/workflows/docs-api.yml)
[![API Docs link](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=API%20Docs%20link)](https://api.my.nethesis.it/)

#### CI and Tests

CI Pipeline:

[![CI](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?style=for-the-badge&label=CI%20Pipeline)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)

Backend:

[![Backend Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=backend-tests&label=Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
[![Backend Build](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=backend-build&label=Build&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)

Collect:

[![Collect Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=collect-tests&label=Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
[![Collect Build](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=collect-build&label=Build&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)

Sync:

[![Sync Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=sync-tests&label=Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
[![Sync Build](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=sync-build&label=Build&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)

Frontend:

[![Frontend Tests](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=frontend-tests&label=Tests&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)
[![Frontend Build](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=frontend-build&label=Build&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)

Proxy:

[![Proxy Build](https://img.shields.io/github/actions/workflow/status/NethServer/my/ci-main.yml?job=proxy-build&label=Build&style=for-the-badge)](https://github.com/NethServer/my/actions/workflows/ci-main.yml)


#### Release
[![Release](https://img.shields.io/github/actions/workflow/status/NethServer/my/release-production.yml?style=for-the-badge&label=Release)](https://github.com/NethServer/my/actions/workflows/release-production.yml)
[![Version](https://img.shields.io/github/v/release/NethServer/my?style=for-the-badge&color=3a3c3f&label=Version)](https://github.com/NethServer/my/releases)


#### Production and QA
[![My](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=my.nethesis.it)](https://my-proxy-prod.onrender.com)
[![My QA](https://img.shields.io/badge/docs-available-blue?style=for-the-badge&label=qa.my.nethesis.it)](https://qa.my.nethesis.it)

Web application providing centralized authentication and management using Logto as an Identity Provider with simple Role-Based Access Control.

## 🏗️ Components

- **[frontend/](./frontend/)** - Vue.js application for UI
- **[backend/](./backend/)** - Go REST API with Logto JWT authentication and RBAC
- **[collect/](./collect/)** - Go REST API with Redis queues to handle inventories
- **[sync/](./sync/)** - CLI tool for RBAC configuration synchronization
- **[proxy/](./proxy/)** - nginx configuration as load balancer

## 🚀 Quick Start

### Requirements
- **Development**: Go 1.21+ (backend requires 1.23+), Node.js, Make
- **Containers**: Docker OR Podman (optional, for full infrastructure)
- **External**: Logto instance with M2M app and Management API permissions
- **Deploy**: Render account with GitHub integration

### Development Setup

**Choose your preferred development approach:**

#### Option 1: Full Infrastructure (Recommended)
Complete replica of the production environment with all services containerized:

```bash
# Start all services (PostgreSQL, Redis, Backend, Collect, Frontend, Proxy)
docker-compose up -d
# OR
podman-compose up -d

# Access the application at: http://localhost:9090
```

All URLs and management commands are documented in the [docker-compose.yml](./docker-compose.yml) header.

#### Option 2: Individual Components
Start services individually for targeted development:

```bash
# Start shared databases (PostgreSQL + Redis)
cd backend && make dev-up

# Start individual services in separate terminals:
cd backend && make run          # Backend API on :8080
cd collect && make run          # Collect service on :8081
cd frontend && npm run dev      # Frontend on :5173

# Use sync tool as needed:
cd sync && make run-example
```

### Getting Started
1. **RBAC Setup**: [sync/README.md](./sync/README.md) - Use `sync init` for complete Logto configuration
2. **Frontend Development**: [frontend/README.md](./frontend/README.md) - Vue.js setup and environment configuration
3. **Backend Development**: [backend/README.md](./backend/README.md) - Backend setup and environment configuration
4. **Collect Development**: [collect/README.md](./collect/README.md) - Collect setup and environment configuration
5. **Production Deploy**: Use `./deploy.sh` for automated deployment

## 🌐 Deployment Environments

### QA (`qa.my.nethesis.it`)
- **Trigger**: Every commit to `main` branch
- **Auto-deploy**: Immediate deployment via Render
- **PR Previews**: Temporary environments for pull requests

### Production (`my.nethesis.it`)
- **Trigger**: Manual deployment via `./deploy.sh` script
- **Auto-Deploy**: Render automatically deploys when `render.yaml` is updated
- **Manual Control**: Deploy only when explicitly triggered
- **Security**: Private services (Backend, Collect, Frontend) only accessible through Proxy

## 🔐 Consent-Based Impersonation System

The platform provides a privacy-friendly impersonation system that allows **Owner organization users** to temporarily access other user accounts for troubleshooting and support, with full user consent and complete audit trails.

### 🎯 Key Features
- **User-Controlled Consent**: Users explicitly enable impersonation and set duration (1-168 hours)
- **Custom Token Duration**: Impersonation tokens match user's consent duration (not fixed)
- **Complete Audit Trail**: Every API call during impersonation is logged for transparency
- **Session-Based Tracking**: Unique session IDs for complete audit organization
- **Automatic Data Sanitization**: Sensitive information redacted from audit logs
- **Owner-Only Access**: Only users with `org_role: "Owner"` can initiate impersonation

### 🔄 Consent-Based Flow

#### 1. User Enables Consent
```bash
# User calls API to enable impersonation consent
POST /api/impersonate/consent
{
  "duration_hours": 24  # 1-168 hours
}
```

#### 2. Admin Initiates Impersonation
```bash
# Owner user starts impersonation (only if consent is active)
POST /api/impersonate
{
  "user_id": "target-user-id"
}
# Returns JWT token with custom duration matching user's consent
```

#### 3. Automatic Audit Logging
- **Every API call** during impersonation is automatically logged
- **Request/response data** captured (with sensitive data redacted)
- **Session tracking** groups all actions for easy review
- **Real-time logging** with no performance impact

#### 4. Complete Transparency
```bash
# User can view complete audit of impersonation actions
GET /api/impersonate/audit/user/{user_id}

# Get audit by session for detailed review
GET /api/impersonate/audit/session/{session_id}
```

### 🛡️ Security Features
- **No Self-Impersonation**: Users cannot impersonate themselves
- **No Chaining**: Cannot impersonate while already impersonating
- **Consent Expiration**: Automatic consent expiration based on user settings
- **Session Isolation**: Each impersonation creates unique session for audit
- **Sensitive Data Protection**: Passwords, tokens, and secrets automatically redacted
- **Owner Restriction**: Only organization owners can perform impersonation

### 📊 API Integration
The impersonation system integrates seamlessly with existing user management:

```bash
# User list includes impersonation status (for Owner users only)
GET /api/users
# Response includes: "can_be_impersonated": true/false

# Complete consent management
POST /api/impersonate/consent     # Enable consent
GET /api/impersonate/consent      # Check status  
DELETE /api/impersonate/consent   # Disable consent

# Impersonation control
POST /api/impersonate            # Start impersonation
DELETE /api/impersonate          # Exit impersonation

# Audit and transparency
GET /api/impersonate/audit/user/{user_id}      # User's audit history
GET /api/impersonate/audit/session/{session}  # Session audit details
```

This system ensures complete transparency and user control while providing necessary support capabilities for troubleshooting.

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
- **Service Configuration**: Defined in `render.yaml`
- **Deployment**: Use `./deploy.sh` script for automated deployment

## 📚 Documentation

- **[frontend](./frontend/README.md)** - UI setup, environment variables, and pages
- **[backend](./backend/README.md)** - Server setup, environment variables, authorization architecture, and consent-based impersonation
- **[backend OpenAPI](./backend/openapi.yaml)** - Complete API specification with authentication
- **[collect](./collect/README.md)** - Server setup, environment variables and inventory structure
- **[sync CLI](./sync/README.md)** - RBAC configuration and `sync init` setup
- **[deploy script](./deploy.sh)** - Production deployment script for Render
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
# → my-proxy-qa-pr-123.onrender.com created
```

### Production Release
```bash
# Automated release with quality checks
./release.sh patch                       # → 0.0.5 → 0.0.6 (bug fixes)
./release.sh minor                       # → 0.0.5 → 0.1.0 (new features)
./release.sh major                       # → 0.0.5 → 1.0.0 (breaking changes)
# → Runs tests, formatting, linting → Creates tag → Pushes to GitHub
```

The release script will:
1. Run all quality checks (formatting, linting, tests)
2. Bump version in all files
3. Create git commit and tag
4. Push to GitHub
5. Trigger GitHub Actions to build and publish Docker images

### Production Deployment
```bash
# Standard deployment with image verification
./deploy.sh

# Fast deployment without image verification (less safe but faster)
./deploy.sh --skip-verify

# Show help
./deploy.sh --help
```

The deployment script will:
1. Get the latest git tag automatically
2. Show the tag and ask for confirmation
3. Verify Docker images exist on ghcr.io (unless `--skip-verify`)
4. Update `render.yaml` with new image tags
5. Commit changes with your git user info
6. Push to main branch
7. Render automatically deploys the updated services

**Example output:**
```
ℹ️  Latest git tag: v0.1.5
Do you want to deploy v0.1.5 to production? [y/N] y
✅ All Docker images verified successfully
✅ render.yaml updated successfully
✅ Changes committed and pushed to main branch
✅ Deployment initiated successfully!
```

## 🤝 Contributing

1. Follow existing code patterns and conventions
2. **Pre-commit**: Run `make pre-commit` in both directories
3. Test RBAC changes with `--dry-run` before applying
4. Ensure CI tests pass before submitting PRs

## 📄 License

See [LICENSE](./LICENSE) file for details.
