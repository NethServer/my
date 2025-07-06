package jwt

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set test environment variables
	_ = os.Setenv("JWT_SECRET", "test-secret-key-for-testing-only")
	_ = os.Setenv("JWT_ISSUER", "test-issuer")
	_ = os.Setenv("JWT_EXPIRATION", "24h")
	_ = os.Setenv("JWT_REFRESH_EXPIRATION", "168h")
	_ = os.Setenv("LOGTO_ISSUER", "https://test-logto.example.com")
	_ = os.Setenv("LOGTO_AUDIENCE", "test-api-resource")
	_ = os.Setenv("BACKEND_CLIENT_ID", "test-client-id")
	_ = os.Setenv("BACKEND_CLIENT_SECRET", "test-client-secret")

	// Initialize configuration
	configuration.Init()

	code := m.Run()
	os.Exit(code)
}

func TestGenerateCustomToken(t *testing.T) {
	user := models.User{
		ID:               "test-user-123",
		Username:         "testuser",
		Email:            "test@example.com",
		Name:             "Test User",
		OrganizationID:   "test-org-456",
		OrganizationName: "Test Organization",
		UserRoles:        []string{"Admin"},
		UserPermissions:  []string{"manage:systems", "manage:accounts"},
		OrgRole:          "Customer",
		OrgPermissions:   []string{"view:systems"},
	}

	tests := []struct {
		name     string
		user     models.User
		wantErr  bool
		validate func(t *testing.T, tokenString string)
	}{
		{
			name:    "valid user generates valid JWT",
			user:    user,
			wantErr: false,
			validate: func(t *testing.T, tokenString string) {
				// Parse and validate the token using our validation function
				claims, err := ValidateCustomToken(tokenString)
				require.NoError(t, err)
				require.NotNil(t, claims)

				// Check user data in claims
				assert.Equal(t, "test-user-123", claims.User.ID)
				assert.Equal(t, "testuser", claims.User.Username)
				assert.Equal(t, "test@example.com", claims.User.Email)
				assert.Equal(t, "Test User", claims.User.Name)
				assert.Equal(t, "test-org-456", claims.User.OrganizationID)
				assert.Equal(t, []string{"Admin"}, claims.User.UserRoles)
				assert.Equal(t, []string{"manage:systems", "manage:accounts"}, claims.User.UserPermissions)
				assert.Equal(t, "Customer", claims.User.OrgRole)
				assert.Equal(t, []string{"view:systems"}, claims.User.OrgPermissions)

				// Check registered claims
				assert.Equal(t, "test-user-123", claims.Subject)
				assert.Equal(t, "test-issuer", claims.Issuer)
				assert.True(t, claims.ExpiresAt.After(time.Now()))
			},
		},
		{
			name: "user with empty ID still generates token",
			user: models.User{
				Username: "testuser",
				Email:    "test@example.com",
			},
			wantErr: false,
			validate: func(t *testing.T, tokenString string) {
				// Even with empty ID, token is generated successfully
				claims, err := ValidateCustomToken(tokenString)
				require.NoError(t, err)
				assert.Equal(t, "", claims.User.ID)
				assert.Equal(t, "testuser", claims.User.Username)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateCustomToken(tt.user)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)
			}

			tt.validate(t, token)
		})
	}
}

func TestValidateCustomToken(t *testing.T) {
	// Create a valid user for testing
	user := models.User{
		ID:               "test-user-123",
		Username:         "testuser",
		Email:            "test@example.com",
		Name:             "Test User",
		OrganizationID:   "test-org-456",
		OrganizationName: "Test Organization",
		UserRoles:        []string{"Admin"},
		UserPermissions:  []string{"manage:systems"},
		OrgRole:          "Customer",
		OrgPermissions:   []string{"view:systems"},
	}

	// Generate a valid token for testing
	validToken, err := GenerateCustomToken(user)
	require.NoError(t, err)

	tests := []struct {
		name        string
		tokenString string
		wantErr     bool
		wantUser    *models.User
	}{
		{
			name:        "valid token returns claims with user",
			tokenString: validToken,
			wantErr:     false,
			wantUser:    &user,
		},
		{
			name:        "empty token returns error",
			tokenString: "",
			wantErr:     true,
			wantUser:    nil,
		},
		{
			name:        "invalid token format returns error",
			tokenString: "not.a.jwt",
			wantErr:     true,
			wantUser:    nil,
		},
		{
			name:        "token with invalid signature returns error",
			tokenString: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXVzZXIiLCJleHAiOjk5OTk5OTk5OTl9.invalid-signature",
			wantErr:     true,
			wantUser:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := ValidateCustomToken(tt.tokenString)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)

				// Compare user fields in claims
				assert.Equal(t, tt.wantUser.ID, claims.User.ID)
				assert.Equal(t, tt.wantUser.Username, claims.User.Username)
				assert.Equal(t, tt.wantUser.Email, claims.User.Email)
				assert.Equal(t, tt.wantUser.OrganizationID, claims.User.OrganizationID)
				assert.Equal(t, tt.wantUser.UserRoles, claims.User.UserRoles)
				assert.Equal(t, tt.wantUser.UserPermissions, claims.User.UserPermissions)
				assert.Equal(t, tt.wantUser.OrgRole, claims.User.OrgRole)
				assert.Equal(t, tt.wantUser.OrgPermissions, claims.User.OrgPermissions)
			}
		})
	}
}

func TestRefreshToken(t *testing.T) {
	userID := "test-user-123"

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{
			name:    "valid user ID generates refresh token",
			userID:  userID,
			wantErr: false,
		},
		{
			name:    "empty user ID still generates token",
			userID:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateRefreshToken(tt.userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Validate the refresh token
				claims, err := ValidateRefreshToken(token)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, claims.UserID)
				assert.Equal(t, tt.userID, claims.Subject)
				assert.True(t, claims.ExpiresAt.After(time.Now()))
			}
		})
	}
}

func TestTokenExpiration(t *testing.T) {
	user := models.User{
		ID:       "test-user",
		Username: "testuser",
		Email:    "test@example.com",
	}

	// Generate token
	tokenString, err := GenerateCustomToken(user)
	require.NoError(t, err)

	// Validate and check expiration
	claims, err := ValidateCustomToken(tokenString)
	require.NoError(t, err)

	// Token should expire in the future (within 24 hours)
	now := time.Now()
	expectedExpiry := now.Add(24 * time.Hour)

	assert.True(t, claims.ExpiresAt.After(now), "Token should not be expired")
	assert.True(t, claims.ExpiresAt.Before(expectedExpiry.Add(time.Minute)), "Token should expire within 24 hours")
}

func TestCustomTokenValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupToken  func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "token with wrong signing method fails",
			setupToken: func() string {
				user := models.User{ID: "test-user", Username: "testuser"}
				claims := CustomClaims{
					User: user,
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    configuration.Config.JWTIssuer,
						Subject:   user.ID,
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}
				// Use RS256 instead of HS256
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				// This will fail because we don't have RSA keys, but that's expected
				tokenString, _ := token.SignedString([]byte("fake-key"))
				return tokenString
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
		{
			name: "token signed with wrong secret fails",
			setupToken: func() string {
				user := models.User{ID: "test-user", Username: "testuser"}
				claims := CustomClaims{
					User: user,
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    configuration.Config.JWTIssuer,
						Subject:   user.ID,
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte("wrong-secret"))
				return tokenString
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
		{
			name: "expired token fails validation",
			setupToken: func() string {
				user := models.User{ID: "test-user", Username: "testuser"}
				claims := CustomClaims{
					User: user,
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    configuration.Config.JWTIssuer,
						Subject:   user.ID,
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(configuration.Config.JWTSecret))
				return tokenString
			},
			wantErr:     true,
			errContains: "failed to parse token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.setupToken()
			claims, err := ValidateCustomToken(tokenString)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestRefreshTokenValidationEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupToken  func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "refresh token with wrong signing method fails",
			setupToken: func() string {
				claims := RefreshTokenClaims{
					UserID: "test-user",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    configuration.Config.JWTIssuer,
						Subject:   "test-user",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}
				// Use RS256 instead of HS256
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
				tokenString, _ := token.SignedString([]byte("fake-key"))
				return tokenString
			},
			wantErr:     true,
			errContains: "failed to parse refresh token",
		},
		{
			name: "refresh token signed with wrong secret fails",
			setupToken: func() string {
				claims := RefreshTokenClaims{
					UserID: "test-user",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    configuration.Config.JWTIssuer,
						Subject:   "test-user",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
						IssuedAt:  jwt.NewNumericDate(time.Now()),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte("wrong-secret"))
				return tokenString
			},
			wantErr:     true,
			errContains: "failed to parse refresh token",
		},
		{
			name: "expired refresh token fails validation",
			setupToken: func() string {
				claims := RefreshTokenClaims{
					UserID: "test-user",
					RegisteredClaims: jwt.RegisteredClaims{
						Issuer:    configuration.Config.JWTIssuer,
						Subject:   "test-user",
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired
						IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
					},
				}
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, _ := token.SignedString([]byte(configuration.Config.JWTSecret))
				return tokenString
			},
			wantErr:     true,
			errContains: "failed to parse refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenString := tt.setupToken()
			claims, err := ValidateRefreshToken(tokenString)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
			}
		})
	}
}

func TestTokenClaimsValidation(t *testing.T) {
	user := models.User{
		ID:               "user-123",
		Username:         "testuser",
		Email:            "test@example.com",
		Name:             "Test User",
		OrganizationID:   "org-456",
		OrganizationName: "Test Organization",
		UserRoles:        []string{"Admin", "Support"},
		UserPermissions:  []string{"manage:systems", "view:logs"},
		OrgRole:          "Distributor",
		OrgPermissions:   []string{"create:resellers", "manage:customers"},
	}

	tokenString, err := GenerateCustomToken(user)
	require.NoError(t, err)

	claims, err := ValidateCustomToken(tokenString)
	require.NoError(t, err)

	// Verify all user data is preserved in token
	assert.Equal(t, user.ID, claims.User.ID)
	assert.Equal(t, user.Username, claims.User.Username)
	assert.Equal(t, user.Email, claims.User.Email)
	assert.Equal(t, user.Name, claims.User.Name)
	assert.Equal(t, user.OrganizationID, claims.User.OrganizationID)
	assert.Equal(t, user.OrganizationName, claims.User.OrganizationName)
	assert.Equal(t, user.UserRoles, claims.User.UserRoles)
	assert.Equal(t, user.UserPermissions, claims.User.UserPermissions)
	assert.Equal(t, user.OrgRole, claims.User.OrgRole)
	assert.Equal(t, user.OrgPermissions, claims.User.OrgPermissions)

	// Verify registered claims
	assert.Equal(t, user.ID, claims.Subject)
	assert.Equal(t, configuration.Config.JWTIssuer, claims.Issuer)
	assert.Contains(t, claims.Audience, configuration.Config.LogtoAudience)
	assert.True(t, claims.ExpiresAt.After(time.Now()))
	assert.True(t, claims.IssuedAt.Before(time.Now().Add(time.Minute)))
	assert.True(t, claims.NotBefore.Before(time.Now().Add(time.Minute)))
}

func TestConfigurationFallbacks(t *testing.T) {
	// Test with invalid expiration configuration
	originalExp := configuration.Config.JWTExpiration
	originalRefreshExp := configuration.Config.JWTRefreshExpiration

	// Set invalid durations
	configuration.Config.JWTExpiration = "invalid-duration"
	configuration.Config.JWTRefreshExpiration = "invalid-duration"

	user := models.User{ID: "test-user", Username: "testuser"}

	// Test custom token generation with invalid expiration config
	tokenString, err := GenerateCustomToken(user)
	assert.NoError(t, err) // Should succeed with fallback
	assert.NotEmpty(t, tokenString)

	// Validate token to check it uses fallback expiration (24 hours)
	claims, err := ValidateCustomToken(tokenString)
	require.NoError(t, err)

	// Should expire approximately 24 hours from now (with some tolerance)
	expectedExpiry := time.Now().Add(24 * time.Hour)
	assert.True(t, claims.ExpiresAt.Before(expectedExpiry.Add(time.Minute)))
	assert.True(t, claims.ExpiresAt.After(expectedExpiry.Add(-time.Minute)))

	// Test refresh token generation with invalid expiration config
	refreshToken, err := GenerateRefreshToken("test-user")
	assert.NoError(t, err) // Should succeed with fallback
	assert.NotEmpty(t, refreshToken)

	// Validate refresh token to check it uses fallback expiration (7 days)
	refreshClaims, err := ValidateRefreshToken(refreshToken)
	require.NoError(t, err)

	// Should expire approximately 7 days from now (with some tolerance)
	expectedRefreshExpiry := time.Now().Add(7 * 24 * time.Hour)
	assert.True(t, refreshClaims.ExpiresAt.Before(expectedRefreshExpiry.Add(time.Minute)))
	assert.True(t, refreshClaims.ExpiresAt.After(expectedRefreshExpiry.Add(-time.Minute)))

	// Restore original configuration
	configuration.Config.JWTExpiration = originalExp
	configuration.Config.JWTRefreshExpiration = originalRefreshExp
}
