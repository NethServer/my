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

### Complete Setup from Zero to Working Example

#### Step 1: Create Vue.js Project
```bash
# Create new Vue project
npm create vue@latest logto-test
cd logto-test

# Select options (use defaults, just press Enter for all)
# Install dependencies
npm install

# Install Logto SDK
npm install @logto/vue
```

#### Step 2: Configure Logto Settings
Before running the code, you need these values from your Logto admin console:
- `endpoint`: Your Logto instance URL (e.g., `https://y4uj0v.logto.app`)
- `appId`: Your application ID from Logto admin console
- `resources`: Your API resource identifier - **MUST be absolute URI** (e.g., `https://api.my.nethesis.it`)

#### Step 3: Install Logto Plugin

Edit `src/main.ts` to install the Logto plugin:

```typescript
import { createApp } from 'vue'
import { createLogto } from '@logto/vue'
import App from './App.vue'

// Logto configuration - UPDATE THESE VALUES!
const config = {
  endpoint: 'https://your-logto-instance.logto.app',     // Your Logto URL
  appId: 'your-app-id',                                  // Your App ID from Logto admin
  resources: [],                                         // Must be ABSOLUTE URI (https://...), can be empty
  scopes: ['openid', 'profile', 'email'],                // Required scopes
}

const app = createApp(App)

// Install Logto plugin
app.use(createLogto, config)

app.mount('#app')
```

#### Step 4: Configure Redirect URI in Logto Admin

⚠️ **IMPORTANT**: In your Logto admin console, add this redirect URI to your application:
```
http://localhost:5173/callback
```

**Steps in Logto Admin:**
1. Go to **Applications** → Your App
2. Find **Redirect URIs** section
3. Add: `http://localhost:5173/callback`
4. **Save** the configuration

#### Step 5: Replace App.vue Content

Replace the entire content of `src/App.vue` with:

```vue
<template>
  <div style="padding: 20px; font-family: Arial;">
    <!-- Callback Processing -->
    <div v-if="isCallback" style="text-align: center;">
      <h2>🔄 Processing login...</h2>
      <p>Please wait while we complete your authentication.</p>
    </div>

    <!-- Main App -->
    <div v-else>
      <h1>Logto + Backend Token Exchange Test</h1>

      <div v-if="!isAuthenticated">
        <h2>👤 Login Required</h2>
        <button @click="handleSignIn" style="padding: 10px; font-size: 16px;">
          Sign In with Logto
        </button>
      </div>

      <div v-else>
        <h2>✅ Welcome, {{ user?.name }}!</h2>
        <button @click="exchangeToken" style="padding: 10px; font-size: 16px; margin: 10px;">
          🔄 Get Custom JWT
        </button>
        <button @click="handleSignOut" style="padding: 10px; font-size: 16px;">
          🚪 Sign Out
        </button>

        <div v-if="customJWT" style="margin-top: 20px;">
          <h3>🎉 Custom JWT Response:</h3>
          <pre style="background: #f5f5f5; padding: 10px; border-radius: 5px; overflow-x: auto;">{{ JSON.stringify(customJWT, null, 2) }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { useLogto } from '@logto/vue';
import { ref } from 'vue';

// Logto composable (config is already set in main.ts)
const { isAuthenticated, user, signIn, signOut, getAccessToken, error, handleSignInCallback } = useLogto();

// Check if current page is callback
const isCallback = ref(window.location.pathname === '/callback');

// Debug errors
if (error.value) {
  console.error('🚨 Logto Error:', error.value);
}

// Debug configuration on mounted
import { onMounted } from 'vue';
onMounted(async () => {
  console.log('🔧 Logto loaded, isAuthenticated:', isAuthenticated.value);
  console.log('👤 Current user:', user?.value);
  console.log('📍 Current path:', window.location.pathname);

  // Handle callback if on callback page
  if (isCallback.value) {
    try {
      console.log('🔄 Processing callback...');
      await handleSignInCallback(window.location.href);
      console.log('✅ Callback processed successfully');
      // Redirect back to main app
      window.history.replaceState({}, '', '/');
      isCallback.value = false;
    } catch (error) {
      console.error('❌ Callback processing failed:', error);
    }
  }
});

// Custom JWT state
const customJWT = ref(null);

// Handle sign in with explicit redirect URI
const handleSignIn = async () => {
  try {
    await signIn({
      redirectUri: 'http://localhost:5173/callback'
    });
  } catch (error) {
    console.error('🚨 Sign in failed:', error);
  }
};

// Handle sign in with explicit redirect URI
const handleSignOut = async () => {
  try {
    await signOut('http://localhost:5173/');
  } catch (error) {
    console.error('🚨 Sign out failed:', error);
  }
};

// Exchange Logto token for custom JWT
const exchangeToken = async () => {
  try {
    console.log('🚀 Starting token exchange...');

    // 1. Get Logto access token
    const logtoToken = await getAccessToken(); // Same as resources[0]
    console.log('✅ Logto Access Token obtained:', logtoToken);

    // 2. Exchange for custom JWT
    const response = await fetch('http://localhost:8080/api/auth/exchange', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        access_token: logtoToken
      })
    });

    if (response.ok) {
      const result = await response.json();
      customJWT.value = result;

      // 3. Log the enriched token data
      const data = result.data;
      console.log('🎉 SUCCESS! Custom JWT obtained:', data);
      console.log('📋 User permissions:', data.user.user_permissions);
      console.log('🏢 Organization role:', data.user.org_role);
      console.log('🔑 Custom Access Token:', data.token);
      console.log('🔄 Refresh Token:', data.refresh_token);

      // Store tokens for future API calls
      localStorage.setItem('access_token', data.token);
      localStorage.setItem('refresh_token', data.refresh_token);

    } else {
      const errorText = await response.text();
      console.error('❌ Token exchange failed:', errorText);
      console.error('Response status:', response.status);
    }
  } catch (error) {
    console.error('💥 Error during token exchange:', error);
  }
};
</script>
```

#### Step 6: Start Backend Server
Make sure your backend is running:
```bash
# In your backend directory
cd backend
go run main.go
# Should show: Server starting on 127.0.0.1:8080
```

#### Step 7: Start Vue App
```bash
# In your Vue project directory
npm run dev
# Should show: Local: http://localhost:5173/
```

#### Step 8: Test the Flow
1. Open browser to `http://localhost:5173/`
2. Click "Sign In with Logto" → Complete login
3. Click "Get Custom JWT" → Check console

### 🎯 Expected Console Output
```
🚀 Starting token exchange...
✅ Logto Access Token obtained: eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...
🎉 SUCCESS! Custom JWT obtained: {
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 86400,
  "user": {
    "id": "user_123",
    "username": "john.doe",
    "user_roles": ["Admin"],
    "user_permissions": ["manage:systems"],
    "org_role": "Distributor",
    "org_permissions": ["create:resellers"]
  }
}
📋 User permissions: ["manage:systems"]
🏢 Organization role: Distributor
🔑 Custom Access Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### ✅ Success!
If you see this output, the token exchange is working perfectly!

---

## 🔧 Troubleshooting

If you encounter errors during setup or testing, check these common issues:

**1. Redirect URI Mismatch:**
```
🚨 Logto Error: redirect_uri_mismatch
```
Solution: Add `http://localhost:5173/callback` to Logto admin console

**2. Invalid Client:**
```
🚨 Logto Error: invalid_client
```
Solution: Double-check your `appId` in main.js matches Logto admin console

**3. Invalid Scope:**
```
🚨 Logto Error: invalid_scope
```
Solution: Check your API resource identifier and scopes

**4. Cannot read properties of undefined (reading 'toString'):**
```
TypeError: Cannot read properties of undefined (reading 'toString')
```
Solutions:
- Make sure your `endpoint` starts with `https://` (not `http://`)
- Verify your `appId` is exactly copied from Logto admin console
- Add explicit `redirectUri: 'http://localhost:5173/callback'` to config
- Restart the dev server: `npm run dev`

**5. Cannot destructure property 'redirectUri':**
```
Cannot destructure property 'redirectUri' of '(intermediate value)' as it is undefined
```
Solution: Use the `handleSignIn` function (already included in the code above) instead of calling `signIn` directly

**6. invalid_target - resource indicator must be an absolute URI:**
```
?error=invalid_target&error_description=resource+indicator+must+be+an+absolute+URI
```
Solution: Make sure your `resources` array contains absolute URIs starting with `https://`
```typescript
// ❌ WRONG
resources: ['your.api.domain']

// ✅ CORRECT
resources: ['https://your.api.domain']
```

**7. Invalid resource indicator / resource indicator is missing:**
```
oidc.invalid_target: Invalid resource indicator
error_description: resource indicator is missing, or unknown
```
Solution: The API resource doesn't exist in Logto. You need to:
1. Go to Logto Admin Console → **API Resources**
2. Click **Create API Resource**
3. Set identifier to `https://your.api.domain` (must match your config)
4. Add the resource to your application scopes
5. Update your configuration with the **exact** identifier from Logto

---

## 🔧 Additional Resources

### Using Custom JWT for API Calls

```javascript
// Helper function for authenticated API requests
const apiCall = async (endpoint, options = {}) => {
  const token = localStorage.getItem('access_token');

  const response = await fetch(`http://localhost:8080/api${endpoint}`, {
    ...options,
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
      ...options.headers
    }
  });

  return response.json();
};

// Example API calls
const getMyProfile = () => apiCall('/auth/me');
const listAccounts = () => apiCall('/accounts');
const createAccount = (accountData) => apiCall('/accounts', {
  method: 'POST',
  body: JSON.stringify(accountData)
});
```

---

**🔗 Related Links**
- [Project Overview](../README.md) - Main project documentation
- [sync](../sync/README.md) - RBAC configuration management tool
