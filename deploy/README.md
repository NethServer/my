# Deploy Guide

## Automated Deploy Workflow

### QA Environment
- **URL**: `qa.my.nethesis.it`
- **Trigger**: Every commit to `main` branch
- **Action**: Automatic deployment via Render auto-deploy
- **Services**: `my-backend-qa` + `my-redis-qa`

### Pull Request Previews
- **URL**: `pr-{number}.my-backend-qa.onrender.com`
- **Trigger**: Every PR opened/updated
- **Action**: Automatic temporary environment creation
- **Lifecycle**: Created on PR open, destroyed on PR close

### Production Environment
- **URL**: `my.nethesis.it`
- **Trigger**: Manual GitHub Actions workflow
- **Action**: Blueprint sync (Infrastructure as Code)
- **Pipeline**: Verify images → Update render.yaml → Render autodeploy

## Setup Instructions

### 1. Render Configuration
1. **Connect Repository**: Link GitHub repository to Render
2. **Deploy Blueprint**: Use `render.yaml` to create all services automatically
3. **Configure Environment Variables**: Set production-specific values in Render dashboard

#### Services Created
##### Production
- `my-redis-prod` (Production Redis) - Private Service
- `my-postgres-prod` (Production PostgreSQL) - Private Service
- `my-backend-prod` (Production Backend) - **Private Service** (accessible only via proxy)
- `my-collect-prod` (Production Collect) - **Private Service** (accessible only via proxy)
- `my-frontend-prod` (Production Frontend) - **Private Service** (accessible only via proxy)
- `my-proxy-prod` (Production Proxy) - **Public Service** (single entry point)
##### QA
- `my-redis-qa` (QA Redis) - Private Service
- `my-postgres-qa` (QA PostgreSQL) - Private Service
- `my-backend-qa` (QA Backend) - **Private Service** (accessible only via proxy)
- `my-collect-qa` (QA Collect) - **Private Service** (accessible only via proxy)
- `my-frontend-qa` (QA Frontend) - **Private Service** (accessible only via proxy)
- `my-proxy-qa` (QA Proxy) - **Public Service** (single entry point)

### 2. GitHub Permissions
The deploy workflow uses Infrastructure as Code via blueprint sync.

**Required**: Repository must have write access to commit updated `render.yaml`
- The workflow uses `GITHUB_TOKEN` (automatically provided)
- No additional Render API keys or service IDs needed

### 3. DNS Configuration
Point your domains to Render services:
```bash
qa.my.nethesis.it   -> my-proxy-qa.onrender.com
my.nethesis.it      -> my-proxy-prod.onrender.com
```

## Environment Variables

Configure environment variables in Render dashboard for each service:
- **Complete Variable List**: See [backend/README.md](../backend/README.md) for all required variables
- **Environment-Specific Values**: Adjust `LOGTO_AUDIENCE`, `JWT_ISSUER`, and domain-specific values
- **Secrets Management**: Use Render dashboard for sensitive values (`sync: false` in `render.yaml`)

### Production Release Process
```bash
# 1. Create and push release
./release.sh patch  # Creates v0.1.1 tag and Docker images

# 2. Deploy to production (Manual GitHub Action)
# Go to: GitHub → Actions → Deploy Production → Run workflow
# Input: v0.1.1
```

## Production Deployment Pipeline

The deployment uses **Infrastructure as Code** via blueprint sync:

### 1. Checkout Tag (`checkout-tag` job)
- **Action**: Checkout the specific version tag
- **Verification**: Ensure tag exists and contains expected commit
- **Failure**: Stops entire pipeline

### 2. Verify Images (`verify-images` job)
- **Action**: Check all Docker images exist in registry
- **Images**: `backend`, `collect`, `frontend`, `proxy`
- **Failure**: Stops deployment (images not available)

### 3. Update Configuration (`update-render-config` job)
- **Action**: Update `render.yaml` with new image tags
- **Method**: Replace version tags using sed patterns
- **Result**: Commit updated configuration to main branch

### 4. Auto-Deploy via Render
- **Trigger**: Render detects `render.yaml` changes
- **Action**: Automatic blueprint sync deployment
- **Services**: All production services updated simultaneously

### Pipeline Benefits
- **Infrastructure as Code**: Single source of truth in `render.yaml`
- **Atomic Updates**: All services updated in one blueprint sync
- **Version Control**: Configuration changes tracked in git
- **No API Dependencies**: Uses standard git workflow

## Monitoring & Status

### GitHub Actions Status
Monitor deployment progress:
- **Deploy Production**: Shows overall deployment workflow status
- **Workflow Summary**: Displays updated image versions and deployment method
- **Commit History**: Track `render.yaml` updates in git history

### Render Dashboard
- **Service Logs**: Real-time logs for each service
- **Deployment History**: Track all deployments and rollbacks
- **Performance Metrics**: CPU, memory, and request metrics
- **Health Checks**: All services have `/api/health` endpoints

### GitHub Actions
- **Workflow Status**: Track production deployment pipeline
- **Job Dependencies**: Visual representation of deployment sequence
- **Failure Notifications**: Email alerts for failed deployments

## Troubleshooting

### Failed Production Deployment
1. **Check GitHub Actions**: Review deploy workflow logs for failures
2. **Verify Images**: Ensure Docker images exist for the specified version
3. **Check render.yaml**: Verify configuration file was updated correctly
4. **Monitor Render**: Check Render dashboard for blueprint sync status and service logs

### QA Environment Issues
1. **Auto-deploy Failures**: Check Render service logs for build/runtime errors
2. **Redis Connection**: Verify `REDIS_URL` environment variable is correctly set
3. **Environment Variables**: Ensure all required variables are configured

### Common Issues
- **Database Connection Refused**: Check if Redis and PostgreSQL services are running and healthy
- **Build Failures**: Verify Go version and dependencies in `render.yaml`
- **Environment Variable Mismatch**: Ensure production vs QA values are correct
- **Blueprint Sync Failure**: Check Render dashboard for configuration validation errors
- **Version Mismatch**: Ensure Docker images exist for the specified version tag

### Service Configuration

#### Private Services Architecture
All application services use `type: pserv` (Private Service) for enhanced security:

- **Backend** (`my-backend-prod`): Private API service accessible only via internal network
- **Collect** (`my-collect-prod`): Private inventory service accessible only via internal network
- **Frontend** (`my-frontend-prod`): Private static file server accessible only via internal network
- **Proxy** (`my-proxy-prod`): **Single public entry point** that routes traffic to private services

#### Internal Communication
- **Service URLs**: Private services use internal URLs (e.g., `my-backend-prod:10000`)
- **Protocol**: HTTP for internal communication (encrypted at transport layer by Render)
- **Routing**: Proxy forwards requests to appropriate private services
  - `/backend/api/` → `my-backend-prod:10000`
  - `/collect/api/` → `my-collect-prod:10000`
  - `/` → `my-frontend-prod:10000`

#### Database Configuration
- **Redis**: Internal connections only (`ipAllowList: []`)
- **PostgreSQL**: Internal connections only (`ipAllowList: []`)
- **Backend & Collect**: Connect to databases via `fromService` configuration

#### Security Benefits
- **Reduced Attack Surface**: Only proxy service is publicly accessible
- **No Direct Access**: Backend, Collect, and Frontend cannot be reached directly from internet
- **Internal Network**: All service communication happens within Render's secure internal network