# My Nethesis - Nginx Reverse Proxy

## Overview

This nginx reverse proxy consolidates all My Nethesis services under a single domain with proper routing:

- **Production**: `my.nethesis.it` 
- **QA**: `qa.my.nethesis.it`

## Architecture

```
my.nethesis.it (Production)
├── /                    → Frontend Service (Vue.js)
├── /backend/api/        → Backend Service (Go REST API)
└── /collect/api/        → Collect Service (Inventory Collection)

qa.my.nethesis.it (QA)
├── /                    → Frontend Service (Vue.js)
├── /backend/api/        → Backend Service (Go REST API)
└── /collect/api/        → Collect Service (Inventory Collection)
```

## Custom Domain Setup

### 1. Configure CNAME Records in DNS

Add these CNAME records in your DNS provider:

```
# Production
my.nethesis.it       CNAME   my-proxy-prod.onrender.com

# QA  
qa.my.nethesis.it    CNAME   my-proxy-qa.onrender.com
```

### 2. Add Custom Domains in Render

1. **Go to Render Dashboard**
2. **Production Proxy Service** (`my-proxy-prod`):
   - Settings → Custom Domains
   - Add `my.nethesis.it`
   - Wait for SSL certificate provisioning

3. **QA Proxy Service** (`my-proxy-qa`):
   - Settings → Custom Domains  
   - Add `qa.my.nethesis.it`
   - Wait for SSL certificate provisioning

## Features

### Security Headers
- X-Frame-Options: SAMEORIGIN
- X-Content-Type-Options: nosniff
- X-XSS-Protection: 1; mode=block
- Referrer-Policy: strict-origin-when-cross-origin

### Performance
- Gzip compression enabled
- Proper caching headers
- Connection pooling to upstream services

### Monitoring
- Health check endpoint at `/health`
- Structured access logging
- Error logging with proper levels

## Configuration

The proxy automatically discovers service URLs based on the service names configured in `render.yaml`:

### Production Environment
- `BACKEND_SERVICE_NAME=my-backend-prod`
- `COLLECT_SERVICE_NAME=my-collect-prod`
- `FRONTEND_SERVICE_NAME=my-frontend-prod`

### QA Environment
- `BACKEND_SERVICE_NAME=my-backend-qa`
- `COLLECT_SERVICE_NAME=my-collect-qa`
- `FRONTEND_SERVICE_NAME=my-frontend-qa`

## Testing

### Health Checks
```bash
# Production
curl https://my.nethesis.it/health
curl https://my.nethesis.it/backend/api/health
curl https://my.nethesis.it/collect/api/health

# QA
curl https://qa.my.nethesis.it/health
curl https://qa.my.nethesis.it/backend/api/health
curl https://qa.my.nethesis.it/collect/api/health
```

### API Testing
```bash
# Backend API (requires authentication)
curl -X POST https://my.nethesis.it/backend/api/auth/exchange \
  -H "Content-Type: application/json" \
  -d '{"access_token": "your-logto-token"}'

# Collect API (requires basic auth)
curl -X POST https://my.nethesis.it/collect/api/systems/inventory \
  -H "Content-Type: application/json" \
  -H "Authorization: Basic base64(system_id:secret)" \
  -d '{"hostname": "test", "data": {}}'
```

## Security Notes

- All inter-service communication uses HTTPS
- SSL verification disabled for Render internal communication
- Proper headers forwarded to upstream services
- Security headers added to all responses

## Performance Tuning

Current configuration supports:
- 1024 concurrent connections
- 30-second timeouts
- Gzip compression for text content
- HTTP/1.1 keep-alive connections