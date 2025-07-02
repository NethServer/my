package testutils

import (
	"github.com/nethesis/my/backend/models"
)

// Test JWT tokens for different scenarios
const (
	ValidJWTToken     = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJleHAiOjk5OTk5OTk5OTksInVzZXJfaWQiOiJ0ZXN0LXVzZXIiLCJvcmdhbml6YXRpb25faWQiOiJ0ZXN0LW9yZyJ9.test-signature"
	ExpiredJWTToken   = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJleHAiOjEsInVzZXJfaWQiOiJ0ZXN0LXVzZXIifQ.test-signature"
	InvalidJWTToken   = "invalid.jwt.token"
	MalformedJWTToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid-payload.signature"
)

// MockUser creates a test user with specified roles and permissions
func MockUser(userID, orgID string, userRoles, orgRoles []string, userPerms, orgPerms []string) *models.User {
	return &models.User{
		ID:               userID,
		Username:         "test-user",
		Email:            "test@example.com",
		Name:             "Test User",
		OrganizationID:   orgID,
		OrganizationName: "Test Organization",
		UserRoles:        userRoles,
		UserPermissions:  userPerms,
		OrgRole:          orgRoles[0], // First role as primary
		OrgPermissions:   orgPerms,
	}
}

// MockOwnerUser creates a user with Owner role and all permissions
func MockOwnerUser() *models.User {
	return MockUser(
		"owner-user-id",
		"demo-org",
		[]string{"Admin"},
		[]string{"Owner"},
		[]string{"manage:systems", "manage:accounts", "manage:organizations"},
		[]string{"manage:all"},
	)
}

// MockDistributorAdmin creates a distributor admin user
func MockDistributorAdmin() *models.User {
	return MockUser(
		"dist-admin-id",
		"distributor-org",
		[]string{"Admin"},
		[]string{"Distributor"},
		[]string{"manage:systems", "manage:accounts"},
		[]string{"manage:resellers", "manage:customers"},
	)
}

// MockResellerAdmin creates a reseller admin user
func MockResellerAdmin() *models.User {
	return MockUser(
		"reseller-admin-id",
		"reseller-org",
		[]string{"Admin"},
		[]string{"Reseller"},
		[]string{"manage:systems", "manage:accounts"},
		[]string{"manage:customers"},
	)
}

// MockCustomerAdmin creates a customer admin user
func MockCustomerAdmin() *models.User {
	return MockUser(
		"customer-admin-id",
		"customer-org",
		[]string{"Admin"},
		[]string{"Customer"},
		[]string{"manage:systems"},
		[]string{"view:systems"},
	)
}

// MockSupportUser creates a support user with limited permissions
func MockSupportUser() *models.User {
	return MockUser(
		"support-user-id",
		"customer-org",
		[]string{"Support"},
		[]string{"Customer"},
		[]string{"view:systems", "view:logs"},
		[]string{"view:systems"},
	)
}

// MockCreateAccountRequest creates a test account creation request
func MockCreateAccountRequest() map[string]interface{} {
	return map[string]interface{}{
		"username":         "new-user",
		"email":            "newuser@example.com",
		"name":             "New User",
		"phone":            "+1234567890",
		"password":         "SecurePassword123!",
		"userRole":         "Admin",
		"organizationId":   "test-org-id",
		"organizationRole": "Customer",
		"metadata": map[string]interface{}{
			"department": "IT",
			"location":   "Milan",
		},
	}
}

// LogtoMockResponses contains mock responses for Logto API
var LogtoMockResponses = struct {
	UserInfo      string
	Roles         string
	Organizations string
}{
	UserInfo: `{
		"sub": "test-user-id",
		"username": "test-user",
		"email": "test@example.com",
		"name": "Test User",
		"custom_data": {
			"organizationId": "test-org-id"
		}
	}`,
	Roles: `{
		"data": [
			{
				"id": "admin-role-id",
				"name": "Admin",
				"description": "Administrator role"
			},
			{
				"id": "support-role-id",
				"name": "Support",
				"description": "Support role"
			}
		]
	}`,
	Organizations: `{
		"data": [
			{
				"id": "test-org-id",
				"name": "Test Organization",
				"custom_data": {
					"role": "Customer"
				}
			}
		]
	}`,
}
