# Management API Integration Test

This document explains how to test the complete Management API integration for real roles and permissions fetching.

## Prerequisites

### 1. Logto Machine-to-Machine App
1. Go to your Logto admin console
2. Create a **Machine-to-Machine** application
3. Grant **all Management API permissions**
4. Copy the Client ID and Client Secret

### 2. Environment Setup
```bash
# Add to your .env file:
LOGTO_MANAGEMENT_CLIENT_ID=your-m2m-client-id
LOGTO_MANAGEMENT_CLIENT_SECRET=your-m2m-client-secret
LOGTO_MANAGEMENT_BASE_URL=https://your-logto.logto.app/api
```

### 3. RBAC Configuration
Ensure your Logto instance has the simplified RBAC structure:

#### User Roles (Technical Capabilities)
- **Admin**: `admin:systems`, `destroy:systems`, `manage:systems`, `read:systems`
- **Support**: `manage:systems`, `read:systems`

#### Organization Roles (Business Hierarchy)
- **God**: `create:distributors`, `manage:distributors`, `create:resellers`, `manage:resellers`, `create:customers`, `manage:customers`
- **Distributor**: `create:resellers`, `manage:resellers`, `create:customers`, `manage:customers`, `read:own-resellers`, `read:own-customers`
- **Reseller**: `create:customers`, `manage:customers`, `read:own-customers`
- **Customer**: `read:own-data`, `read:systems`

## Testing Flow

### Step 1: Verify Management API Connection
Start the server and check logs for Management API token acquisition:

```bash
go run main.go
```

Look for log messages like:
```
[INFO][LOGTO] Management API token obtained, expires at ...
```

### Step 2: Create Test User in Logto
1. Create a user in Logto admin console
2. Assign user roles (e.g., "Admin", "Support")
3. Add user to an organization with organization role (e.g., "Distributor")

### Step 3: Get Logto Access Token
Get an access token for your test user through your frontend app or direct OAuth flow.

### Step 4: Test Token Exchange with Real Data
```bash
curl -X POST http://localhost:8080/api/auth/exchange \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "YOUR_REAL_LOGTO_ACCESS_TOKEN"
  }'
```

Expected response with real data:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "user": {
    "id": "real-user-id",
    "username": "testuser",
    "email": "test@example.com",
    "user_roles": ["Admin"],
    "user_permissions": ["admin:systems", "destroy:systems", "manage:systems", "read:systems"],
    "org_role": "Distributor", 
    "org_permissions": ["create:resellers", "manage:resellers", "create:customers", "manage:customers"],
    "organization_id": "org-123",
    "organization_name": "Test Organization"
  }
}
```

### Step 5: Test API Access with Real Permissions
```bash
# Use the custom JWT from step 4
CUSTOM_JWT="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Test systems access (should work - user has read:systems)
curl -X GET http://localhost:8080/api/systems \
  -H "Authorization: Bearer $CUSTOM_JWT"

# Test creating a reseller (should work - user has create:resellers from org role)
curl -X POST http://localhost:8080/api/resellers \
  -H "Authorization: Bearer $CUSTOM_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Reseller",
    "description": "Created via real permissions"
  }'

# Test creating a distributor (should fail - only God role can do this)
curl -X POST http://localhost:8080/api/distributors \
  -H "Authorization: Bearer $CUSTOM_JWT" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Distributor"
  }'
# Expected: 403 Forbidden
```

## Debugging

### Check Management API Logs
Look for detailed logs about the role fetching process:

```
[INFO][LOGTO] Management API token obtained, expires at ...
[INFO][LOGTO] Enriched user user-123 with 1 user roles, 4 user permissions, org role 'Distributor', 4 org permissions
```

### Verify User Roles in Logto
1. Check user roles in Logto admin console
2. Verify organization membership and roles
3. Ensure permissions are correctly assigned to roles

### Test Individual Management API Calls
You can test the Management API client methods individually by adding debug endpoints:

```go
// Add to main.go for debugging
auth.GET("/debug/user/:id/roles", func(c *gin.Context) {
    client := services.NewLogtoManagementClient()
    roles, err := client.GetUserRoles(c.Param("id"))
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, roles)
})
```

## Error Scenarios

### Common Issues

1. **Token Exchange Fails**
   - Check LOGTO_MANAGEMENT_CLIENT_ID/SECRET
   - Verify Management API permissions
   - Check network connectivity

2. **Empty Permissions**
   - Verify user has roles assigned in Logto
   - Check role permissions configuration
   - Ensure organization membership

3. **403 Forbidden on API Calls**
   - Verify JWT contains expected permissions
   - Check middleware permission requirements
   - Confirm RBAC configuration matches API expectations

### Recovery Actions

1. **Re-sync RBAC**: Use `sync` to ensure configuration is up to date
2. **Verify Tokens**: Check JWT contents using jwt.io
3. **Check Logs**: Monitor both application logs and Logto audit logs

## Production Considerations

1. **Token Caching**: Management API tokens are cached for performance
2. **Error Handling**: API continues working even if Management API is temporarily unavailable
3. **Permissions Refresh**: User permissions update when JWT expires (24h default)
4. **Monitoring**: Log all Management API interactions for debugging