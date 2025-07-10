# Backend API Documentation

REST API for Nethesis Operation Center with business hierarchy management and RBAC.

## 📖 Interactive Documentation
**Complete API Reference:** https://bump.sh/nethesis/doc/my

## Development Commands
```bash
# Validate OpenAPI spec before commit
make validate-docs
```

## Base URL
```
http://localhost:8080/api
```

## Overview

### Authentication
All endpoints require JWT token from token exchange (except `/auth/exchange`).
```
Authorization: Bearer {JWT_TOKEN}
Content-Type: application/json
```

### API Features
- **Pagination**: All list endpoints support `page` and `page_size` parameters
- **Filtering**: Search and field-specific filters available
- **Hierarchical Authorization**: Business hierarchy controls access to resources
- **Unified Error Responses**: Consistent error format across all endpoints

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

## Endpoints

### Authentication
- `POST /auth/exchange` - Exchange Logto token for custom JWT
- `POST /auth/refresh` - Refresh expired tokens
- `GET /auth/me` - Current user information

### Business Hierarchy
- `GET|POST|PUT|DELETE /distributors` - Distributor management (Owner only)
- `GET|POST|PUT|DELETE /resellers` - Reseller management (Owner + Distributor)
- `GET|POST|PUT|DELETE /customers` - Customer management (Owner + Distributor + Reseller)

### Account Management
- `GET|POST|PUT|DELETE /accounts` - User account management with hierarchical validation
- `GET /roles` - Available user roles
- `GET /organization-roles` - Available organization roles
- `GET /organizations` - Organizations accessible to current user

### Applications
- `GET /applications` - Third-party applications filtered by user access permissions

### System Management
- `GET /stats` - System statistics (requires `manage:distributors` permission)
- `GET /health` - Health check

## Standard Response Format
```json
{
  "code": 200,
  "message": "operation completed successfully",
  "data": {}
}
```

## Error Responses
```json
{
  "code": 400,
  "message": "descriptive error message",
  "data": {
    "type": "validation_error|external_api_error",
    "errors": [
      {
        "key": "fieldName",
        "message": "error_code",
        "value": "rejected_value"
      }
    ]
  }
}
```

## Quick Start
1. Exchange Logto token: `POST /auth/exchange`
2. Use returned JWT for all subsequent requests
3. Check permissions with `GET /auth/me`

---

**🔗 Related Links**
- [Complete API Documentation](https://bump.sh/nethesis/doc/my) - Interactive reference with examples
- [Project Overview](../README.md) - Main project documentation
- [sync CLI](../sync/README.md) - RBAC configuration management tool
