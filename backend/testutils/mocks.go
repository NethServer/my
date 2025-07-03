package testutils

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/mock"
)

// MockLogtoService is a mock implementation of Logto service functions
type MockLogtoService struct {
	mock.Mock
}

// MockGetUserInfoFromLogto simulates fetching user info from Logto
func (m *MockLogtoService) GetUserInfoFromLogto(accessToken string) (*models.User, error) {
	args := m.Called(accessToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

// MockEnrichUserWithRolesAndPermissions simulates enriching user with roles and permissions
func (m *MockLogtoService) EnrichUserWithRolesAndPermissions(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

// MockLogtoHTTPServer creates a mock HTTP server that simulates Logto API responses
func MockLogtoHTTPServer(t *testing.T) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/oidc/userinfo"):
			// Mock userinfo endpoint
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(LogtoMockResponses.UserInfo))

		case strings.Contains(r.URL.Path, "/api/users") && strings.Contains(r.URL.Path, "/roles"):
			// Mock user roles endpoint
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(LogtoMockResponses.Roles))

		case strings.Contains(r.URL.Path, "/api/users") && strings.Contains(r.URL.Path, "/organizations"):
			// Mock user organizations endpoint
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(LogtoMockResponses.Organizations))

		case strings.Contains(r.URL.Path, "/oidc/token"):
			// Mock token endpoint for management API
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"access_token": "mock-management-token",
				"token_type": "Bearer",
				"expires_in": 3600
			}`))

		default:
			// Default fallback
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": "endpoint not found"}`))
		}
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

// MockEnvironment sets up test environment variables
func MockEnvironment(t *testing.T) {
	// Set test environment variables
	t.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
	t.Setenv("LOGTO_ISSUER", "https://test-logto.example.com")
	t.Setenv("LOGTO_AUDIENCE", "test-api-resource")
	t.Setenv("BACKEND_CLIENT_ID", "test-client-id")
	t.Setenv("BACKEND_CLIENT_SECRET", "test-client-secret")
	t.Setenv("LISTEN_ADDRESS", "127.0.0.1:0") // Random port for testing
}

// AuthHeaders creates authorization headers for testing
func AuthHeaders(token string) map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + token,
	}
}

// MockJWTToken generates a simple mock JWT for testing (not cryptographically valid)
func MockJWTToken(userID, orgID string, exp int64) string {
	// This is a simplified mock token for testing purposes
	// In real tests, you'd use the actual JWT library to create valid tokens
	return "mock-jwt-token-" + userID + "-" + orgID
}
