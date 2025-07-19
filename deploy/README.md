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
- **Trigger**: GitHub release creation
- **Action**: Sequential deployment via GitHub Actions
- **Pipeline**: Redis → Backend → Frontend (with dependencies)

## Setup Instructions

### 1. Render Configuration
1. **Connect Repository**: Link GitHub repository to Render
2. **Deploy Blueprint**: Use `render.yaml` to create all services automatically
3. **Configure Environment Variables**: Set production-specific values in Render dashboard

#### Services Created
##### Production
- `my-redis-prod` (Production Redis)
- `my-postgres-prod` (Production PostgreSQL)
- `my-backend-prod` (Production Backend)
- `my-collect-prod` (Production Collect)
- `my-frontend-prod` (Production Frontend)
- `my-proxy-prod` (Production Proxy)
##### QA
- `my-redis-qa` (QA Redis)
- `my-postgres-qa` (QA PostgreSQL)
- `my-backend-qa` (QA Backend)
- `my-collect-qa` (QA Collect)
- `my-frontend-qa` (QA Frontend)
- `my-proxy-qa` (QA Proxy)

### 2. GitHub Secrets
Add these secrets to your repository (**Settings** → **Secrets and variables** → **Actions**):
```bash
RENDER_API_KEY=your-render-api-key
RENDER_PRODUCTION_REDIS_SERVICE_ID=red-xxxxxxxxxxxxxxxxxx
RENDER_PRODUCTION_POSTGRES_SERVICE_ID=dpg-xxxxxxxxxxxxxxxxxx
RENDER_PRODUCTION_BACKEND_SERVICE_ID=srv-xxxxxxxxxxxxxxxxxx
RENDER_PRODUCTION_COLLECT_SERVICE_ID=srv-xxxxxxxxxxxxxxxxxx
RENDER_PRODUCTION_FRONTEND_SERVICE_ID=srv-xxxxxxxxxxxxxxxxxx
RENDER_PRODUCTION_PROXY_SERVICE_ID=srv-xxxxxxxxxxxxxxxxxx
```

**Where to find Service IDs**: Render Dashboard → Service → Settings → Service ID

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
git tag v1.2.3
git push origin v1.2.3
# Create GitHub release
# → Triggers sequential deployment pipeline (details below)
```

## Production Deployment Pipeline

The GitHub Actions workflow deploys services in sequence with dependencies:

### 1. Deploy Redis (`deploy-redis` job)
- **Service**: `my-redis-prod`
- **Dependencies**: None
- **Failure**: Stops entire pipeline

### 2. Deploy PostgreSQL (`deploy-postgres` job)
- **Service**: `my-postgres-prod`
- **Dependencies**: None (parallel with Redis)
- **Failure**: Stops application deployment

### 3. Deploy Backend (`deploy-backend` job)
- **Service**: `my-backend-prod`
- **Dependencies**: `needs: [deploy-redis, deploy-postgres]`
- **Failure**: Stops application deployment

### 4. Deploy Collect (`deploy-collect` job)
- **Service**: `my-collect-prod`
- **Dependencies**: `needs: [deploy-redis, deploy-postgres]`
- **Failure**: Isolated to Collect service

### 5. Deploy Frontend (`deploy-frontend` job)
- **Service**: `my-frontend-prod`
- **Dependencies**: `needs: deploy-backend`
- **Failure**: Isolated to Frontend only

### 6. Deploy Proxy (`deploy-proxy` job)
- **Service**: `my-proxy-prod`
- **Dependencies**: `needs: [deploy-backend, deploy-frontend]`
- **Failure**: Isolated to Proxy only

### Pipeline Benefits
- **Fail-fast**: Infrastructure failures prevent application deployment
- **Consistent State**: Each service waits for its dependencies
- **Visibility**: Individual status badges for each deployment stage

## Monitoring & Status

### GitHub Status Badges
Monitor deployment status directly from README:
- **Deploy Redis**: Shows Redis deployment status
- **Deploy Backend**: Shows Backend deployment status
- **Deploy Frontend**: Shows Frontend deployment status

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
1. **Check GitHub Actions**: Review workflow logs for the specific job that failed
2. **Verify Service Dependencies**: Ensure Redis and PostgreSQL are healthy before Backend deployment
3. **Review Render Logs**: Check individual service logs in Render dashboard

### QA Environment Issues
1. **Auto-deploy Failures**: Check Render service logs for build/runtime errors
2. **Redis Connection**: Verify `REDIS_URL` environment variable is correctly set
3. **Environment Variables**: Ensure all required variables are configured

### Common Issues
- **Database Connection Refused**: Check if Redis and PostgreSQL services are running and healthy
- **Build Failures**: Verify Go version and dependencies in `render.yaml`
- **Environment Variable Mismatch**: Ensure production vs QA values are correct
- **GitHub Secrets**: Verify all service IDs and API keys are properly configured

### Service Configuration
- **Redis**: Internal connections only (`ipAllowList: []`)
- **PostgreSQL**: Internal connections only (`ipAllowList: []`)
- **Backend & Collect**: Connect to databases via `fromService` configuration
- **Proxy**: Routes traffic to Backend and Frontend services
- **Environment Variables**: Managed separately for each environment