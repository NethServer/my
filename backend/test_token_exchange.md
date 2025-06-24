# Token Exchange Flow Test

This document explains how to test the new token exchange system.

## Setup

1. **Configure Environment Variables**
```bash
cp .env.example .env
# Edit .env with your values:
# - LOGTO_ISSUER=https://your-logto.logto.app
# - LOGTO_AUDIENCE=your-api-resource
# - JWT_SECRET=your-secret-key-here
```

2. **Start the Server**
```bash
go run main.go
```

## Testing Flow

### Step 1: Get Logto Access Token
First, get an access token from your Logto instance using your frontend application or direct OAuth flow.

### Step 2: Exchange Token
```bash
curl -X POST http://localhost:8080/api/auth/exchange \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "YOUR_LOGTO_ACCESS_TOKEN_HERE"
  }'
```

Expected response:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "user": {
    "id": "user-id",
    "username": "username", 
    "email": "user@example.com",
    "user_roles": ["Admin"],
    "user_permissions": ["admin:systems", "destroy:systems", "manage:systems", "read:systems"],
    "org_role": "Distributor", 
    "org_permissions": ["create:resellers", "manage:resellers", "create:customers", "manage:customers"],
    "organization_id": "org-123",
    "organization_name": "ACME Corp"
  }
}
```

**Note**: The system now fetches **real roles and permissions** from Logto via Management API!

### Step 3: Use Custom JWT
```bash
# Get user profile using custom JWT
curl -X GET http://localhost:8080/api/profile \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT_TOKEN"

# Access systems (requires read:systems permission)
curl -X GET http://localhost:8080/api/systems \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT_TOKEN"

# Create system (requires manage:systems permission)
curl -X POST http://localhost:8080/api/systems \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test System",
    "description": "Test system created via API"
  }'

# Restart system (requires manage:systems permission)
curl -X POST http://localhost:8080/api/systems/123/restart \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT_TOKEN"

# Create distributor (requires create:distributors permission - God only)
curl -X POST http://localhost:8080/api/distributors \
  -H "Authorization: Bearer YOUR_CUSTOM_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "New Distributor",
    "description": "Created via API"
  }'
```

## Architecture Benefits

### 🎯 **Clear Separation**
- **Frontend**: Uses Logto for authentication
- **Backend**: Uses custom JWT with full user context

### 🔐 **Enhanced Security**
- Logto token only used for initial exchange
- Custom JWT contains all necessary permissions
- No need for repeated Logto API calls

### ⚡ **Performance**
- User context embedded in JWT
- No database/API lookups on each request
- Fast permission checks

### 🛠️ **Flexibility**
- Custom claims structure
- Easy permission updates
- Backward compatibility maintained

## API Endpoints

### Public Endpoints
- `POST /api/auth/exchange` - Exchange Logto token for custom JWT

### Custom JWT Protected
- `GET /api/profile` - User profile
- `GET /api/systems` - List systems (requires `read:systems`)
- `POST /api/systems` - Create system (requires `manage:systems`) 
- `PUT /api/systems/:id` - Update system (requires `manage:systems`)
- `DELETE /api/systems/:id` - Delete system (requires `admin:systems`)
- `GET /api/distributors` - List distributors (requires `create:distributors`)
- `GET /api/resellers` - List resellers (requires `create:resellers`)
- `GET /api/customers` - List customers (requires `create:customers`)

## Next Steps

1. **Implement Management API Integration**
   - Fetch real user roles from Logto
   - Get organization memberships
   - Build complete permission sets

2. **Frontend Integration**
   - Update login flow to use token exchange
   - Store custom JWT for API calls
   - Handle token refresh

3. **Production Considerations**
   - Rotate JWT secrets
   - Monitor token usage
   - Implement token refresh mechanism