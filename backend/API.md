# Nethesis Operation Center - Backend API Documentation

## Overview

This document describes the REST API for the Nethesis Operation Center backend. The API provides centralized management for the business hierarchy (Distributors → Resellers → Customers) and user accounts with sophisticated RBAC (Role-Based Access Control).

## Base URL

```
http://localhost:8080/api
```

## Authentication

All API endpoints (except `/auth/exchange`) require a JWT token obtained through the token exchange process.

### Headers Required
```
Authorization: Bearer {JWT_TOKEN}
Content-Type: application/json
```

---

## 🔐 Authentication Endpoints

### Exchange Logto Token
Converts a Logto access token to a custom JWT with embedded permissions and generates a refresh token.

**POST** `/auth/exchange`

```bash
curl -X POST "http://localhost:8080/api/auth/exchange" \
  -H "Content-Type: application/json" \
  -d '{
    "access_token": "YOUR_LOGTO_ACCESS_TOKEN"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "user": {
    "id": "user_id",
    "username": "john.doe",
    "email": "john@example.com",
    "name": "John Doe",
    "user_roles": ["Admin"],
    "user_permissions": ["manage:systems"],
    "org_role": "Distributor",
    "org_permissions": ["manage:resellers"],
    "organization_id": "org_123",
    "organization_name": "ACME Distribution"
  }
}
```

### Refresh Access Token
Refreshes an expired access token using a refresh token. Returns fresh user data from Logto.

**POST** `/auth/refresh`

```bash
curl -X POST "http://localhost:8080/api/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "user": {
    "id": "user_id",
    "username": "john.doe",
    "email": "john@example.com",
    "name": "John Doe",
    "user_roles": ["Admin"],
    "user_permissions": ["manage:systems"],
    "org_role": "Distributor",
    "org_permissions": ["manage:resellers"],
    "organization_id": "org_123",
    "organization_name": "ACME Distribution"
  }
}
```

**Notes:**
- Access tokens expire after **24 hours**
- Refresh tokens expire after **7 days**
- Each refresh operation generates **new tokens** (both access and refresh)
- User data is **refreshed from Logto** during token refresh

### Get Current User Info
Returns current user information from JWT token.

**GET** `/auth/me`

```bash
curl -s -X GET "http://localhost:8080/api/auth/me" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

**Response:**
```json
{
  "code": 200,
  "message": "user information retrieved successfully",
  "data": {
    "id": "user_id",
    "username": "john.doe",
    "email": "john@example.com",
    "name": "John Doe",
    "userRoles": ["Admin"],
    "userPermissions": ["manage:systems"],
    "orgRole": "Distributor",
    "orgPermissions": ["manage:resellers"],
    "organizationId": "org_123",
    "organizationName": "ACME Distribution"
  }
}
```

---

## 🏢 Distributor Management

**Authorization:** Only **God** role can manage distributors.

### List Distributors

**GET** `/distributors`

```bash
curl -s -X GET "http://localhost:8080/api/distributors" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

### Create Distributor

**POST** `/distributors`

```bash
curl -s -X POST "http://localhost:8080/api/distributors" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACME Distribution SpA",
    "description": "Distributore principale per il mercato italiano e svizzero",
    "customData": {
      "email": "mario.rossi@acme-distribution.com",
      "contactPerson": "Mario Rossi",
      "region": "Italy",
      "territory": ["Italy", "Switzerland", "San Marino"],
      "phone": "+39 02 1234567",
      "website": "https://acme-distribution.com",
      "city": "Milano",
      "address": "Via Roma 123",
      "partitaIva": "IT12345678901",
      "codiceFiscale": "12345678901"
    },
    "isMfaRequired": false
  }' | jq
```

### Update Distributor

**PUT** `/distributors/{id}`

```bash
curl -s -X PUT "http://localhost:8080/api/distributors/$DISTRIBUTOR_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ACME Distribution SpA (Updated)",
    "description": "Distributore principale aggiornato per mercato europeo",
    "customData": {
      "email": "mario.rossi.new@acme-distribution.com",
      "region": "Europe",
      "territory": ["Italy", "Switzerland", "Austria", "San Marino"],
      "phone": "+39 02 9876543",
      "expansion": "2025 roadmap"
    },
    "isMfaRequired": true
  }' | jq
```

### Delete Distributor

**DELETE** `/distributors/{id}`

```bash
curl -s -X DELETE "http://localhost:8080/api/distributors/$DISTRIBUTOR_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

---

## 🏪 Reseller Management

**Authorization:** **God** and **Distributor** roles can manage resellers.

### List Resellers

**GET** `/resellers`

```bash
curl -s -X GET "http://localhost:8080/api/resellers" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

### Create Reseller

**POST** `/resellers`

```bash
curl -s -X POST "http://localhost:8080/api/resellers" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "TechSolutions SRL",
    "description": "Reseller specializzato in soluzioni tecnologiche per PMI",
    "customData": {
      "email": "info@techsolutions.it",
      "contactPerson": "Giulia Verdi",
      "region": "Lombardia",
      "city": "Bergamo",
      "address": "Via Garibaldi 456",
      "phone": "+39 035 123456",
      "website": "https://techsolutions.it",
      "partitaIva": "IT09876543210",
      "codiceFiscale": "09876543210",
      "specialization": "Network & Security",
      "certifications": ["Cisco Partner", "Microsoft Silver"]
    },
    "isMfaRequired": true
  }' | jq
```

### Update Reseller

**PUT** `/resellers/{id}`

```bash
curl -s -X PUT "http://localhost:8080/api/resellers/$RESELLER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "TechSolutions SRL (Expanded)",
    "description": "Reseller con nuove competenze cloud",
    "customData": {
      "email": "info.updated@techsolutions.it",
      "contactPerson": "Giulia Verdi",
      "region": "Nord Italia",
      "city": "Milano",
      "phone": "+39 02 555444",
      "certifications": ["Cisco Gold", "Microsoft Gold", "VMware Partner"],
      "newServices": ["Cloud Migration", "Cybersecurity"]
    },
    "isMfaRequired": false
  }' | jq
```

### Delete Reseller

**DELETE** `/resellers/{id}`

```bash
curl -s -X DELETE "http://localhost:8080/api/resellers/$RESELLER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

---

## 🏢 Customer Management

**Authorization:** **God**, **Distributor**, and **Reseller** roles can manage customers.

### List Customers

**GET** `/customers`

```bash
curl -s -X GET "http://localhost:8080/api/customers" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

### Create Customer

**POST** `/customers`

```bash
curl -s -X POST "http://localhost:8080/api/customers" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pizzeria Da Mario",
    "description": "Ristorante tradizionale con esigenze IT moderne",
    "customData": {
      "email": "mario@pizzeriadamario.it",
      "contactPerson": "Mario Bianchi",
      "tier": "basic",
      "industry": "Food & Beverage",
      "city": "Roma",
      "address": "Via del Corso 123",
      "phone": "+39 06 12345678",
      "website": "https://pizzeriadamario.it",
      "partitaIva": "IT11223344556",
      "codiceFiscale": "BNCMRA80A01H501Z",
      "employees": 15,
      "resellerID": "org_reseller_xyz123",
      "requirements": ["WiFi Guest", "POS Integration", "Security Cameras"]
    },
    "isMfaRequired": false
  }' | jq
```

### Update Customer

**PUT** `/customers/{id}`

```bash
curl -s -X PUT "http://localhost:8080/api/customers/$CUSTOMER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Pizzeria Da Mario (Franchise)",
    "description": "Catena di ristoranti con 3 location",
    "customData": {
      "email": "mario.franchise@pizzeriadamario.it",
      "tier": "enterprise",
      "locations": 3,
      "employees": 45,
      "newRequirements": ["Multi-site VPN", "Centralized Management"],
      "lastUpgrade": "2025-06-25"
    },
    "isMfaRequired": true
  }' | jq
```

### Delete Customer

**DELETE** `/customers/{id}`

```bash
curl -s -X DELETE "http://localhost:8080/api/customers/$CUSTOMER_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

---

## 👥 Account Management

Accounts represent users within organizations. They have both **User Roles** (technical capabilities) and **Organization Roles** (business hierarchy).

### List Accounts

**GET** `/accounts`

```bash
# All accounts visible to current user
curl -s -X GET "http://localhost:8080/api/accounts" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq

# Accounts from specific organization
curl -s -X GET "http://localhost:8080/api/accounts?organizationId=org_123" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

### Create Account

**POST** `/accounts`

```bash
curl -s -X POST "http://localhost:8080/api/accounts" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "mario.rossa",
    "email": "mario.rossa@acme.com",
    "name": "Mario Rossa",
    "phone": "+39 333 445 5667",
    "password": "SecurePassword123!",
    "userRoleId": "rol_abc123def456",
    "organizationId": "org_xyz789"
  }' | jq
```

**Note:**
- `userRoleId` must be a valid role ID from Logto (more secure than role names)
- `organizationRole` is automatically derived from the organization's JIT provisioning configuration
- **Hierarchical authorization**: Users can only create accounts in organizations they control or created

### Update Account

**PUT** `/accounts/{id}`

```bash
curl -s -X PUT "http://localhost:8080/api/accounts/$ACCOUNT_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Mario Rossa Updated",
    "email": "mario.updated@acme.com",
    "phone": "+39 333 999 8888",
    "userRoleId": "rol_new_role_id_here",
    "metadata": {
      "department": "Sales",
      "location": "Rome"
    }
  }' | jq
```

### Delete Account

**DELETE** `/accounts/{id}`

```bash
curl -s -X DELETE "http://localhost:8080/api/accounts/$ACCOUNT_ID" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq
```

---

## 📋 Data Structures

### Standard API Response Format

All API responses follow this structure:

```json
{
  "code": 200,
  "message": "operation completed successfully",
  "data": {
    // Response data here
  }
}
```

### Organization Structure (Distributor/Reseller/Customer)

```json
{
  "id": "org_123456789",
  "name": "Organization Name",
  "description": "Organization description",
  "customData": {
    "type": "distributor|reseller|customer",
    "email": "contact@organization.com",
    "contactPerson": "John Doe",
    "region": "Italy",
    "phone": "+39 02 1234567",
    "createdBy": "org_creator_id",
    "createdAt": "2025-06-25T10:30:00Z"
  },
  "isMfaRequired": false,
  "branding": {
    "logoUrl": "https://example.com/logo.png",
    "darkLogoUrl": "https://example.com/logo-dark.png"
  }
}
```

### Account Structure

```json
{
  "id": "usr_123456789",
  "username": "john.doe",
  "email": "john@example.com",
  "name": "John Doe",
  "phone": "+39 333 123456",
  "avatar": "https://example.com/avatar.jpg",
  "userRole": "rol_admin_id",
  "organizationId": "org_123456789",
  "organizationName": "ACME Corp",
  "organizationRole": "Admin",
  "isSuspended": false,
  "lastSignInAt": "2025-06-25T09:15:00Z",
  "createdAt": "2025-06-20T14:30:00Z",
  "updatedAt": "2025-06-25T10:00:00Z",
  "metadata": {
    "department": "IT",
    "location": "Milan"
  }
}
```

---

## 🔒 Authorization & Hierarchy

### Business Hierarchy

```
God (Nethesis)
  ↓
Distributor
  ↓
Reseller
  ↓
Customer
```

### Permission Matrix

| Role | Can Manage | Visibility |
|------|------------|------------|
| **God** | Everything | All organizations |
| **Distributor** | Resellers, Customers | Own org + created resellers + their customers |
| **Reseller** | Customers | Own org + created customers |
| **Customer** | Own accounts only | Own organization only |

### Account Creation Rules

- **God**: Can create accounts in any organization
- **Distributor**: Can create accounts in own org + subordinate orgs (Reseller, Customer) **they created**
- **Reseller**: Can create accounts in own org + Customer orgs **they created**
- **Customer**: Can create accounts only in own org (if Admin)
- **Admin role required**: To create colleague accounts in same organization

### Hierarchical Authorization Controls

**All CRUD operations** (Create, Read, Update, Delete) on accounts are protected by hierarchical authorization:

#### **CREATE** (`POST /accounts`)
- Users can only create accounts in organizations under their control
- Verified through `createdBy` field in organization's `customData`

#### **READ** (`GET /accounts`)
- Users can only see accounts from organizations they have access to
- Filtered through `GetAllVisibleOrganizations()` logic
- Users always see their own account regardless of creator

#### **UPDATE** (`PUT /accounts/:id`)
- Users can only modify accounts in organizations they control
- Same hierarchical rules as CREATE apply
- Users can always modify their own account

#### **DELETE** (`DELETE /accounts/:id`)
- Users can only delete accounts in organizations they control
- Same hierarchical rules as CREATE apply
- Users can delete their own account

#### **Error Responses**
When attempting unauthorized operations:
```json
{
  "code": 403,
  "message": "insufficient permissions to [create|update|delete] [this account|accounts for this organization]",
  "data": "distributors can only operate on accounts in organizations they created"
}
```

---

## 🔧 Utility Commands

### Extract Organization IDs

```bash
# Get all distributor IDs
curl -s -X GET "http://localhost:8080/api/distributors" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq -r '.data.distributors[].id'

# Get first distributor ID and export
export DISTRIBUTOR_ID=$(curl -s -X GET "http://localhost:8080/api/distributors" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq -r '.data.distributors[0].id')
```

### Health Check

```bash
curl -s -X GET "http://localhost:8080/api/health" | jq
```

---

## 🚀 Getting Started

1. **Obtain Logto Access Token** from your frontend application
2. **Exchange for Custom JWT** using `/auth/exchange`
3. **Store both tokens** securely (access + refresh)
4. **Use JWT Token** in all subsequent API calls
5. **Handle token expiration** with refresh mechanism
6. **Check your permissions** with `/auth/me`
7. **Start managing** organizations and accounts

### Token Management Flow

```javascript
// 1. Initial authentication
const { token, refresh_token, user } = await exchangeLogtoToken(logtoAccessToken);

// 2. Store tokens securely
localStorage.setItem('access_token', token);
localStorage.setItem('refresh_token', refresh_token);

// 3. API request with automatic refresh
async function apiRequest(url, options = {}) {
  let token = localStorage.getItem('access_token');

  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json',
        ...options.headers
      }
    });

    if (response.status === 401) {
      // Token expired, try refresh
      const refreshToken = localStorage.getItem('refresh_token');
      const refreshResponse = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: refreshToken })
      });

      if (refreshResponse.ok) {
        const { token: newToken, refresh_token: newRefreshToken } = await refreshResponse.json();
        localStorage.setItem('access_token', newToken);
        localStorage.setItem('refresh_token', newRefreshToken);

        // Retry original request with new token
        return fetch(url, {
          ...options,
          headers: {
            'Authorization': `Bearer ${newToken}`,
            'Content-Type': 'application/json',
            ...options.headers
          }
        });
      } else {
        // Refresh failed, redirect to login
        window.location.href = '/login';
      }
    }

    return response;
  } catch (error) {
    console.error('API request failed:', error);
    throw error;
  }
}
```

---

## 📝 Notes

- All APIs are aligned with **Logto's Organization Management API**
- **customData** field allows flexible metadata storage
- **JIT (Just-In-Time) provisioning** automatically assigns organization roles
- **Hierarchical validation** ensures proper business rules
- **Security-first approach** with role IDs instead of names
- **Refresh tokens** provide seamless session management with automatic token rotation
- **Authorization controls** prevent cross-organization unauthorized access
- **Real-time data refresh** ensures permissions are always up-to-date

## 🔐 Security Features

- **Token Expiration**: Access tokens (24h), Refresh tokens (7d)
- **Token Rotation**: New tokens generated on each refresh
- **Hierarchical Access Control**: Users can only access organizations they created or control
- **Fresh Permission Sync**: User roles and permissions refreshed from Logto during token refresh
- **Audit Trail**: All operations logged with user context and organization hierarchy

For additional support or questions, refer to the CLAUDE.md file in the repository root.