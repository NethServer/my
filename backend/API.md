# Backend API Documentation

REST API for Nethesis Operation Center with business hierarchy management and RBAC.

## Base URL
```
http://localhost:8080/api
```

## API Features

### Pagination
All list endpoints support pagination with query parameters:
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20, max: 100)

All paginated responses include a `pagination` object with metadata.

### Filtering & Search
List endpoints support various filtering options:
- `search`: General search in relevant fields
- Specific field filters (e.g., `name`, `email`, `created_by`)
- Server-side search when supported by Logto
- Client-side filtering for custom data fields

## Authentication
All endpoints require JWT token from token exchange (except `/auth/exchange`).

```
Authorization: Bearer {JWT_TOKEN}
Content-Type: application/json
```

---

## 🔐 Authentication

### Exchange Logto Token
**POST** `/auth/exchange`

```bash
curl -X POST "http://localhost:8080/api/auth/exchange" \
  -H "Content-Type: application/json" \
  -d '{"access_token": "YOUR_LOGTO_ACCESS_TOKEN"}'
```

Returns custom JWT with embedded permissions and 7-day refresh token.

### Refresh Token
**POST** `/auth/refresh`

```bash
curl -X POST "http://localhost:8080/api/auth/refresh" \
  -H "Content-Type: application/json" \
  -d '{"refresh_token": "YOUR_REFRESH_TOKEN"}'
```

Returns new access token (24h) and refresh token (7d) with fresh user data.

### Get Current User
**GET** `/auth/me`

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
**Authorization:** Owner only

### Get Single Distributor
**GET** `/distributors/:id`

**Response:**
```json
{
  "code": 200,
  "message": "distributor retrieved successfully",
  "data": {
    "id": "org_123456789",
    "name": "ACME Distribution SpA",
    "description": "Main distributor for Italian and Swiss markets",
    "customData": {
      "email": "contact@acme-distribution.com",
      "contactPerson": "John Smith",
      "region": "Italy"
    },
    "isMfaRequired": false
  }
}
```

### List Distributors
**GET** `/distributors`

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20)
- `search`: Search in name or ID
- `name`: Exact name match
- `type`: Filter by type (distributor)
- `created_by`: Filter by creator organization ID

```bash
# Examples
GET /distributors?page=1&page_size=10
GET /distributors?search=acme
GET /distributors?created_by=org_owner_123
```

```json
{
  "code": 200,
  "message": "distributors retrieved successfully",
  "data": {
    "distributors": [
      {
        "id": "org_123456789",
        "name": "ACME Distribution SpA",
        "description": "Main distributor for Italian and Swiss markets",
        "customData": {
          "email": "contact@acme-distribution.com",
          "contactPerson": "John Smith",
          "region": "Italy",
          "phone": "+39 02 1234567"
        },
        "isMfaRequired": false
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_count": 25,
      "total_pages": 2,
      "has_next": true,
      "has_prev": false,
      "next_page": 2
    }
  }
}
```

### Create Distributor
**POST** `/distributors`

```json
{
  "name": "ACME Distribution SpA",
  "description": "Main distributor for Italian and Swiss markets",
  "customData": {
    "email": "contact@acme-distribution.com",
    "contactPerson": "John Smith",
    "region": "Italy",
    "phone": "+39 02 1234567"
  },
  "isMfaRequired": false
}
```

### Update Distributor
**PUT** `/distributors/{id}`

```json
{
  "name": "ACME Distribution SpA (Updated)",
  "description": "Updated main distributor for European market",
  "customData": {
    "email": "contact.new@acme-distribution.com",
    "contactPerson": "John Smith",
    "region": "Europe",
    "phone": "+39 02 9876543"
  },
  "isMfaRequired": true
}
```

**Response:**
```json
{
  "code": 200,
  "message": "distributor updated successfully",
  "data": {
    "id": "org_123456789",
    "name": "ACME Distribution SpA (Updated)",
    "description": "Updated main distributor for European market",
    "customData": {
      "email": "contact.new@acme-distribution.com",
      "contactPerson": "John Smith",
      "region": "Europe",
      "phone": "+39 02 9876543"
    },
    "isMfaRequired": true
  }
}
```

### Delete Distributor
**DELETE** `/distributors/{id}`

---

## 🏪 Reseller Management
**Authorization:** Owner + Distributor

### Get Single Reseller
**GET** `/resellers/:id`

**Response:**
```json
{
  "code": 200,
  "message": "reseller retrieved successfully",
  "data": {
    "id": "org_987654321",
    "name": "TechSolutions SRL",
    "description": "Reseller specialized in technology solutions for SMBs",
    "customData": {
      "email": "info@techsolutions.it",
      "contactPerson": "Jane Doe",
      "region": "Northern Region"
    },
    "isMfaRequired": true
  }
}
```

### List Resellers
**GET** `/resellers`

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20)
- `search`: Search in name or ID
- `name`: Exact name match
- `type`: Filter by type (reseller)
- `created_by`: Filter by creator organization ID

```bash
# Examples
GET /resellers?page=1&page_size=10
GET /resellers?search=tech
GET /resellers?created_by=org_distributor_456
```

```json
{
  "code": 200,
  "message": "resellers retrieved successfully",
  "data": {
    "resellers": [
      {
        "id": "org_987654321",
        "name": "TechSolutions SRL",
        "description": "Reseller specialized in technology solutions for SMBs",
        "customData": {
          "email": "info@techsolutions.it",
          "contactPerson": "Jane Doe",
          "region": "Northern Region",
          "phone": "+39 035 123456",
          "specialization": "Network & Security"
        },
        "isMfaRequired": true
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_count": 12,
      "total_pages": 1,
      "has_next": false,
      "has_prev": false
    }
  }
}
```

### Create Reseller
**POST** `/resellers`

```json
{
  "name": "TechSolutions SRL",
  "description": "Reseller specialized in technology solutions for SMBs",
  "customData": {
    "email": "info@techsolutions.it",
    "contactPerson": "Jane Doe",
    "region": "Northern Region",
    "phone": "+39 035 123456",
    "specialization": "Network & Security"
  },
  "isMfaRequired": true
}
```

### Update Reseller
**PUT** `/resellers/{id}`

```json
{
  "name": "TechSolutions SRL (Expanded)",
  "description": "Reseller with new cloud competencies",
  "customData": {
    "email": "info.updated@techsolutions.it",
    "contactPerson": "Jane Doe",
    "region": "Northern Region",
    "phone": "+39 02 555444",
    "specialization": "Cloud & Security"
  },
  "isMfaRequired": false
}
```

**Response:**
```json
{
  "code": 200,
  "message": "reseller updated successfully",
  "data": {
    "id": "org_987654321",
    "name": "TechSolutions SRL (Expanded)",
    "description": "Reseller with new cloud competencies",
    "customData": {
      "email": "info.updated@techsolutions.it",
      "contactPerson": "Jane Doe",
      "region": "Northern Region",
      "phone": "+39 02 555444",
      "specialization": "Cloud & Security"
    },
    "isMfaRequired": false
  }
}
```

### Delete Reseller
**DELETE** `/resellers/{id}`

---

## 🏢 Customer Management
**Authorization:** Owner + Distributor + Reseller

### Get Single Customer
**GET** `/customers/:id`

**Response:**
```json
{
  "code": 200,
  "message": "customer retrieved successfully",
  "data": {
    "id": "org_456789123",
    "name": "Modern Restaurant LLC",
    "description": "Traditional restaurant with modern IT needs",
    "customData": {
      "email": "contact@modernrestaurant.com",
      "contactPerson": "Michael Johnson",
      "tier": "basic",
      "industry": "Food & Beverage"
    },
    "isMfaRequired": false
  }
}
```

### List Customers
**GET** `/customers`

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20)
- `search`: Search in name or ID
- `name`: Exact name match
- `type`: Filter by type (customer)
- `created_by`: Filter by creator organization ID

```bash
# Examples
GET /customers?page=1&page_size=10
GET /customers?search=lep
GET /customers?created_by=org_reseller_789
```

```json
{
  "code": 200,
  "message": "customers retrieved successfully",
  "data": {
    "customers": [
      {
        "id": "org_456789123",
        "name": "Modern Restaurant LLC",
        "description": "Traditional restaurant with modern IT needs",
        "customData": {
          "email": "contact@modernrestaurant.com",
          "contactPerson": "Michael Johnson",
          "tier": "basic",
          "industry": "Food & Beverage",
          "city": "Rome",
          "phone": "+39 06 12345678",
          "employees": 15
        },
        "isMfaRequired": false
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_count": 43,
      "total_pages": 3,
      "has_next": true,
      "has_prev": false,
      "next_page": 2
    }
  }
}
```

### Create Customer
**POST** `/customers`

```json
{
  "name": "Modern Restaurant LLC",
  "description": "Traditional restaurant with modern IT needs",
  "customData": {
    "email": "contact@modernrestaurant.com",
    "contactPerson": "Michael Johnson",
    "tier": "basic",
    "industry": "Food & Beverage",
    "city": "Rome",
    "phone": "+39 06 12345678",
    "employees": 15
  },
  "isMfaRequired": false
}
```

### Update Customer
**PUT** `/customers/{id}`

```json
{
  "name": "Modern Restaurant (Franchise)",
  "description": "Restaurant chain with 3 locations",
  "customData": {
    "email": "franchise@modernrestaurant.com",
    "contactPerson": "Michael Johnson",
    "tier": "enterprise",
    "industry": "Food & Beverage",
    "city": "Rome",
    "phone": "+39 06 12345678",
    "employees": 45,
    "locations": 3
  },
  "isMfaRequired": true
}
```

**Response:**
```json
{
  "code": 200,
  "message": "customer updated successfully",
  "data": {
    "id": "org_456789123",
    "name": "Modern Restaurant (Franchise)",
    "description": "Restaurant chain with 3 locations",
    "customData": {
      "email": "franchise@modernrestaurant.com",
      "contactPerson": "Michael Johnson",
      "tier": "enterprise",
      "industry": "Food & Beverage",
      "city": "Rome",
      "phone": "+39 06 12345678",
      "employees": 45,
      "locations": 3
    },
    "isMfaRequired": true
  }
}
```

### Delete Customer
**DELETE** `/customers/{id}`

---

## 🎭 Role Management
Role information needed for account creation and management.

### Get All User Roles
**GET** `/roles`

Returns all available user roles with their IDs and descriptions.

**Response:**
```json
{
  "code": 200,
  "message": "roles retrieved successfully",
  "data": {
    "roles": [
      {
        "id": "rol_admin_123",
        "name": "Admin",
        "description": "Full system administration capabilities"
      },
      {
        "id": "rol_support_456",
        "name": "Support",
        "description": "Customer support and system management"
      }
    ]
  }
}
```

### Get All Organization Roles
**GET** `/organization-roles`

Returns all available organization roles with their IDs and descriptions.

**Response:**
```json
{
  "code": 200,
  "message": "organization roles retrieved successfully",
  "data": {
    "organizationRoles": [
      {
        "id": "org_rol_owner_123",
        "name": "Owner",
        "description": "Complete control over organization and business hierarchy"
      },
      {
        "id": "org_rol_distributor_456",
        "name": "Distributor",
        "description": "Can manage resellers and customers"
      },
      {
        "id": "org_rol_reseller_789",
        "name": "Reseller",
        "description": "Can manage customers"
      },
      {
        "id": "org_rol_customer_101",
        "name": "Customer",
        "description": "End customer with read-only access"
      }
    ]
  }
}
```

---

## 👥 Account Management
Users within organizations with technical capabilities and business hierarchy roles.

### Get Single Account
**GET** `/accounts/:id`

**Response:**
```json
{
  "code": 200,
  "message": "account retrieved successfully",
  "data": {
    "id": "usr_123456789",
    "username": "john.doe",
    "email": "john@example.com",
    "name": "John Doe",
    "phone": "+39 333 123456",
    "userRoleId": "rol_admin_id",
    "organizationId": "org_123456789",
    "organizationName": "ACME Corp",
    "organizationRole": "Admin",
    "isSuspended": false,
    "customData": {
      "department": "IT",
      "position": "Senior Developer"
    }
  }
}
```

**Note:** Hierarchical authorization - users can only view accounts in organizations they control.

### List Accounts
**GET** `/accounts`

**Query Parameters:**
- `page`: Page number (default: 1)
- `page_size`: Items per page (default: 20)
- `search`: Search in username, email, or name
- `username`: Exact username match
- `email`: Exact email match
- `role`: Filter by user role ID
- `organization_id`: Filter by organization ID

```bash
# Examples
GET /accounts?page=1&page_size=5
GET /accounts?search=john
GET /accounts?organization_id=org_123456789
GET /accounts?role=admin&page=2
```

```json
{
  "code": 200,
  "message": "accounts retrieved successfully",
  "data": {
    "accounts": [
      {
        "id": "usr_123456789",
        "username": "john.doe",
        "email": "john@example.com",
        "name": "John Doe",
        "phone": "+39 333 123456",
        "userRoleId": "rol_admin_id",
        "organizationId": "org_123456789",
        "organizationName": "ACME Corp",
        "organizationRole": "Admin",
        "isSuspended": false,
        "lastSignInAt": "2025-06-25T09:15:00Z",
        "createdAt": "2025-06-20T14:30:00Z",
        "updatedAt": "2025-06-21T10:45:00Z",
        "customData": {
          "createdBy": "org_owner_123",
          "createdAt": "2025-06-20T14:30:00Z"
        }
      }
    ],
    "pagination": {
      "page": 1,
      "page_size": 20,
      "total_count": 156,
      "total_pages": 8,
      "has_next": true,
      "has_prev": false,
      "next_page": 2
    }
  }
}
```

### Create Account
**POST** `/accounts`

```json
{
  "username": "john.doe",
  "email": "john.doe@acme.com",
  "name": "John Doe",
  "phone": "+39 333 445 5667",
  "password": "SecurePassword123!",
  "userRoleId": "rol_abc123def456",
  "organizationId": "org_xyz789",
  "customData": {
    "department": "IT",
    "position": "Senior Developer"
  }
}
```

**Note:** Hierarchical authorization - users can only create accounts in organizations they control.

### Update Account
**PUT** `/accounts/{id}`

```json
{
  "name": "John Doe (Updated)",
  "email": "john.updated@acme.com",
  "phone": "+39 333 999 8888",
  "userRoleId": "rol_new_role_id_here",
  "customData": {
    "department": "Sales",
    "location": "Rome"
  }
}
```

**Response:**
```json
{
  "code": 200,
  "message": "account updated successfully",
  "data": {
    "id": "usr_123456789",
    "username": "john.doe",
    "email": "john.updated@acme.com",
    "name": "John Doe (Updated)",
    "phone": "+39 333 999 8888",
    "userRoleId": "rol_new_role_id_here",
    "organizationId": "org_123456789",
    "organizationName": "ACME Corp",
    "organizationRole": "Admin",
    "isSuspended": false,
    "updatedAt": "2025-07-04T10:30:00Z",
    "customData": {
      "department": "Sales",
      "location": "Rome"
    }
  }
}
```

### Delete Account
**DELETE** `/accounts/{id}`

---

## 📊 System Statistics
**Authorization:** Requires `manage:distributors` permission

### Get System Statistics
**GET** `/stats`

**Response:**
```json
{
  "code": 200,
  "message": "system statistics",
  "data": {
    "distributors": 125,
    "resellers": 1847,
    "customers": 8934,
    "users": 24567,
    "systems": 156,
    "timestamp": "2025-07-08T14:30:00Z",
    "isStale": false
  }
}
```

---

## 📋 Data Structures

### Standard Response Format
```json
{
  "code": 200,
  "message": "operation completed successfully",
  "data": {}
}
```

### Organization Structure
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
    "phone": "+39 02 1234567"
  },
  "isMfaRequired": false
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
  "userRoleId": "rol_admin_id",
  "organizationId": "org_123456789",
  "organizationName": "ACME Corp",
  "organizationRole": "Admin",
  "isSuspended": false
}
```

---

## 🔒 Authorization & Hierarchy

### Business Hierarchy
```
Owner (Nethesis) → Distributor → Reseller → Customer
```

### Permission Matrix
| Role | Can Manage | Visibility |
|------|------------|------------|
| **Owner** | Everything | All organizations |
| **Distributor** | Resellers, Customers | Own org + created subsidiaries |
| **Reseller** | Customers | Own org + created customers |
| **Customer** | Own accounts only | Own organization only |

### Account Creation Rules
- **Owner**: Can create accounts in any organization
- **Distributor**: Can create accounts in own org + subordinate orgs they created
- **Reseller**: Can create accounts in own org + customer orgs they created
- **Customer**: Can create accounts only in own org (if Admin)
- **Admin role required**: To create colleague accounts in same organization

### Hierarchical Authorization
All CRUD operations on accounts are protected by hierarchical authorization:
- Users can only operate on accounts in organizations they control
- Verified through `createdBy` field in organization's `customData`
- Users can always operate on their own account

### Error Responses
```json
{
  "code": 403,
  "message": "insufficient permissions to [create|update|delete] [this account|accounts for this organization]",
  "data": "distributors can only operate on accounts in organizations they created"
}
```

---

## 📋 Validation & Data Structures

### Input Validation
All API endpoints enforce strict validation:

#### Account Creation
- `username`: Required, alphanumeric + underscore, sanitized for Logto compliance
- `email`: Required, valid email format
- `name`: Required, non-empty string
- `password`: Required, minimum 8 characters
- `userRoleId`: Required, must be valid role ID
- `organizationId`: Required, must be accessible organization
- `phone`: Optional, string format
- `customData`: Optional, object with custom fields (mixed types supported)

#### Organization Management
- `name`: Required, must be unique across system
- `description`: Optional, descriptive text
- `customData`: Optional, organization metadata
- `isMfaRequired`: Optional, boolean (default: false)

#### Pagination Parameters
- `page`: Integer ≥ 1 (default: 1)
- `page_size`: Integer 1-100 (default: 20)
- `search`: String, minimum 1 character
- Field filters: Exact match strings

### Response Structures

#### Success Response Format
```json
{
  "code": 200,
  "message": "operation completed successfully",
  "data": {
    // Response data specific to endpoint
  }
}
```

#### Pagination Metadata
```json
{
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total_count": 156,
    "total_pages": 8,
    "has_next": true,
    "has_prev": false,
    "next_page": 2,
    "prev_page": null
  }
}
```

#### Error Response Format
```json
{
  "code": 400,
  "message": "validation error",
  "data": {
    "field": "email",
    "error": "invalid email format"
  }
}
```

### Common HTTP Status Codes
- `200`: Success
- `201`: Resource created
- `400`: Bad request (validation error)
- `401`: Unauthorized (invalid/missing token)
- `403`: Forbidden (insufficient permissions)
- `404`: Resource not found
- `409`: Conflict (duplicate name/username)
- `500`: Internal server error

---

## 🔧 Utilities

### Extract Organization IDs
```bash
# Get all distributor IDs
curl -s -X GET "http://localhost:8080/api/distributors" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq -r '.data.distributors[].id'

# Export first distributor ID
export DISTRIBUTOR_ID=$(curl -s -X GET "http://localhost:8080/api/distributors" \
  -H "Authorization: Bearer $JWT_TOKEN" | jq -r '.data.distributors[0].id')
```

### Health Check
```bash
curl -s -X GET "http://localhost:8080/api/health" | jq
```

### Authenticated API Helper
```javascript
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

// Usage examples
const getMyProfile = () => apiCall('/auth/me');
const listAccounts = () => apiCall('/accounts');
const createAccount = (data) => apiCall('/accounts', {
  method: 'POST',
  body: JSON.stringify(data)
});
```

---

**🔗 Related Links**
- [Project Overview](../README.md) - Main project documentation
- [sync](../sync/README.md) - RBAC configuration management tool
