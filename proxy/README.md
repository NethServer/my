# My Nethesis - Nginx Reverse Proxy

## Overview

This nginx reverse proxy is the only public entry point for all My Nethesis services. Backend, collect, and frontend are private services (`pserv`) accessible only via this proxy on Render's internal network.

- **Production**: `my.nethesis.it`
- **QA**: `qa.my.nethesis.it`

## Architecture

```
my.nethesis.it (Production)
├── /                    → Frontend Service (private, HTTP :10000)
├── /backend/api/        → Backend Service (private, HTTP :10000)
└── /collect/api/        → Collect Service (private, HTTP :10000)

qa.my.nethesis.it (QA)
├── /                    → Frontend Service (private, HTTP :10000)
├── /backend/api/        → Backend Service (private, HTTP :10000)
└── /collect/api/        → Collect Service (private, HTTP :10000)
```

All inter-service communication uses HTTP over Render's internal network. The proxy handles TLS termination for external clients.

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

The proxy receives service hostnames from Render's `fromService` mechanism and routes traffic to private services via HTTP on port 10000.

### DNS Resolution

The entrypoint script extracts the DNS resolver from `/etc/resolv.conf` to resolve internal Render hostnames. This is required because private services are not accessible via public DNS.

### Environment Variables

Set automatically by Render:
- `BACKEND_SERVICE_NAME` - Internal hostname of the backend service
- `COLLECT_SERVICE_NAME` - Internal hostname of the collect service
- `FRONTEND_SERVICE_NAME` - Internal hostname of the frontend service
- `RESOLVER` - DNS resolver extracted from `/etc/resolv.conf`

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

- Backend, collect, and frontend are private services, not accessible from the internet
- All inter-service communication uses HTTP over Render's internal network
- TLS termination happens at the proxy level for external clients
- Security headers added to all responses

## Performance Tuning

Current configuration supports:
- 1024 concurrent connections
- 30-second timeouts
- Gzip compression for text content
- HTTP/1.1 keep-alive connections
