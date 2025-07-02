/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cli

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestInitCommand(t *testing.T) {
	t.Run("command structure", func(t *testing.T) {
		assert.Equal(t, "init", initCmd.Use)
		assert.Equal(t, "Initialize Logto configuration with basic setup", initCmd.Short)
		assert.Contains(t, initCmd.Long, "Initialize Logto with basic configuration")
		assert.NotNil(t, initCmd.RunE)
	})

	t.Run("init flags", func(t *testing.T) {
		flags := []string{
			"force",
			"domain",
			"tenant-id",
			"backend-client-id",
			"backend-client-secret",
			"owner-username",
			"owner-email",
			"owner-name",
		}

		for _, flagName := range flags {
			flag := initCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Flag %s should exist", flagName)
		}
	})

	t.Run("default flag values", func(t *testing.T) {
		usernameFlag := initCmd.Flags().Lookup("owner-username")
		assert.Equal(t, "owner", usernameFlag.DefValue)

		emailFlag := initCmd.Flags().Lookup("owner-email")
		assert.Equal(t, "owner@example.com", emailFlag.DefValue)

		nameFlag := initCmd.Flags().Lookup("owner-name")
		assert.Equal(t, "Company Owner", nameFlag.DefValue)
	})
}

func TestInitTypes(t *testing.T) {
	t.Run("InitResult structure", func(t *testing.T) {
		result := InitResult{
			BackendApp: Application{
				ID:       "backend-id",
				Name:     "backend",
				Type:     "MachineToMachine",
				ClientID: "client-id",
			},
			FrontendApp: Application{
				ID:       "frontend-id",
				Name:     "frontend",
				Type:     "SPA",
				ClientID: "frontend-client-id",
			},
			OwnerUser: User{
				ID:       "user-id",
				Username: "owner",
				Email:    "owner@example.com",
				Password: "test-password",
			},
			CustomDomain:    "example.com",
			GeneratedSecret: "jwt-secret",
			AlreadyInit:     false,
			TenantInfo: TenantInfo{
				TenantID: "test-tenant",
				BaseURL:  "https://test-tenant.logto.app",
				Mode:     "cli",
			},
			NextSteps: []string{"step1", "step2"},
		}

		// Verify structure is complete
		assert.NotEmpty(t, result.BackendApp.ID)
		assert.NotEmpty(t, result.FrontendApp.ID)
		assert.NotEmpty(t, result.OwnerUser.ID)
		assert.NotEmpty(t, result.CustomDomain)
		assert.NotEmpty(t, result.TenantInfo.TenantID)
		assert.Len(t, result.NextSteps, 2)
	})

	t.Run("Application structure", func(t *testing.T) {
		app := Application{
			ID:           "app-id",
			Name:         "test-app",
			Type:         "SPA",
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			EnvironmentVars: map[string]interface{}{
				"TEST_VAR": "test-value",
			},
		}

		assert.Equal(t, "app-id", app.ID)
		assert.Equal(t, "test-app", app.Name)
		assert.Equal(t, "SPA", app.Type)
		assert.Contains(t, app.EnvironmentVars, "TEST_VAR")
	})

	t.Run("User structure", func(t *testing.T) {
		user := User{
			ID:       "user-123",
			Username: "testuser",
			Email:    "test@example.com",
			Password: "secure-password",
		}

		assert.Equal(t, "user-123", user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "secure-password", user.Password)
	})
}

func TestValidateAndGetConfig(t *testing.T) {
	// Save original environment and flags
	originalEnvs := make(map[string]string)
	envVars := []string{"TENANT_ID", "BACKEND_CLIENT_ID", "BACKEND_CLIENT_SECRET", "TENANT_DOMAIN"}
	for _, env := range envVars {
		originalEnvs[env] = os.Getenv(env)
	}

	originalFlags := map[string]string{
		"tenant-id":             initTenantID,
		"domain":                initDomain,
		"backend-client-id":     initBackendClientID,
		"backend-client-secret": initBackendClientSecret,
	}

	defer func() {
		// Restore original environment
		for env, val := range originalEnvs {
			if val == "" {
				_ = os.Unsetenv(env)
			} else {
				_ = os.Setenv(env, val)
			}
		}
		// Restore original flags
		initTenantID = originalFlags["tenant-id"]
		initDomain = originalFlags["domain"]
		initBackendClientID = originalFlags["backend-client-id"]
		initBackendClientSecret = originalFlags["backend-client-secret"]
	}()

	t.Run("CLI mode - all flags provided", func(t *testing.T) {
		// Clear environment
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		// Set CLI flags
		initTenantID = "cli-tenant"
		initDomain = "cli-domain.com"
		initBackendClientID = "cli-client"
		initBackendClientSecret = "cli-secret"

		config, err := validateAndGetConfig()
		require.NoError(t, err)

		assert.Equal(t, "cli-tenant", config.TenantID)
		assert.Equal(t, "cli-domain.com", config.TenantDomain)
		assert.Equal(t, "cli-client", config.BackendClientID)
		assert.Equal(t, "cli-secret", config.BackendClientSecret)
		assert.Equal(t, "cli", config.Mode)
	})

	t.Run("CLI mode - missing flags", func(t *testing.T) {
		// Clear environment
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		// Set only some CLI flags
		initTenantID = "cli-tenant"
		initDomain = ""
		initBackendClientID = ""
		initBackendClientSecret = ""

		config, err := validateAndGetConfig()
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "all must be provided")
	})

	t.Run("environment mode - all vars set", func(t *testing.T) {
		// Clear CLI flags
		initTenantID = ""
		initDomain = ""
		initBackendClientID = ""
		initBackendClientSecret = ""

		// Set environment variables
		_ = os.Setenv("TENANT_ID", "env-tenant")
		_ = os.Setenv("TENANT_DOMAIN", "env-domain.com")
		_ = os.Setenv("BACKEND_CLIENT_ID", "env-client")
		_ = os.Setenv("BACKEND_CLIENT_SECRET", "env-secret")

		config, err := validateAndGetConfig()
		require.NoError(t, err)

		assert.Equal(t, "env-tenant", config.TenantID)
		assert.Equal(t, "env-domain.com", config.TenantDomain)
		assert.Equal(t, "env-client", config.BackendClientID)
		assert.Equal(t, "env-secret", config.BackendClientSecret)
		assert.Equal(t, "env", config.Mode)
	})

	t.Run("environment mode - missing vars", func(t *testing.T) {
		// Clear CLI flags
		initTenantID = ""
		initDomain = ""
		initBackendClientID = ""
		initBackendClientSecret = ""

		// Clear environment
		for _, env := range envVars {
			_ = os.Unsetenv(env)
		}

		config, err := validateAndGetConfig()
		assert.Error(t, err)
		assert.Nil(t, config)
		assert.Contains(t, err.Error(), "required environment variables missing")
	})
}

func TestGenerateSecurePassword(t *testing.T) {
	t.Run("password length", func(t *testing.T) {
		password := generateSecurePassword()
		assert.Equal(t, 16, len(password), "Password should be 16 characters long")
	})

	t.Run("password character sets", func(t *testing.T) {
		password := generateSecurePassword()

		hasLower := false
		hasUpper := false
		hasDigit := false
		hasSymbol := false

		lowerCase := "abcdefghijklmnopqrstuvwxyz"
		upperCase := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits := "0123456789"
		symbols := "!@#$%^&*"

		for _, char := range password {
			charStr := string(char)
			if strings.Contains(lowerCase, charStr) {
				hasLower = true
			}
			if strings.Contains(upperCase, charStr) {
				hasUpper = true
			}
			if strings.Contains(digits, charStr) {
				hasDigit = true
			}
			if strings.Contains(symbols, charStr) {
				hasSymbol = true
			}
		}

		assert.True(t, hasLower, "Password should contain lowercase letters")
		assert.True(t, hasUpper, "Password should contain uppercase letters")
		assert.True(t, hasDigit, "Password should contain digits")
		assert.True(t, hasSymbol, "Password should contain symbols")
	})

	t.Run("password uniqueness", func(t *testing.T) {
		passwords := make(map[string]bool)

		// Generate multiple passwords and check they're unique
		for i := 0; i < 100; i++ {
			password := generateSecurePassword()
			assert.False(t, passwords[password], "Passwords should be unique")
			passwords[password] = true
		}
	})
}

func TestGenerateJWTSecret(t *testing.T) {
	t.Run("secret length", func(t *testing.T) {
		secret := generateJWTSecret()

		// Base64 encoded 32 bytes should be longer than 32 characters
		assert.Greater(t, len(secret), 32, "JWT secret should be longer than 32 characters")
	})

	t.Run("secret is base64", func(t *testing.T) {
		secret := generateJWTSecret()

		// Should be valid base64
		decoded, err := base64.URLEncoding.DecodeString(secret)
		if err != nil {
			// If it's not valid base64, it might be the fallback
			assert.Equal(t, "your-super-secret-jwt-key-please-change-in-production", secret)
		} else {
			// If it's valid base64, it should decode to 32 bytes
			assert.Equal(t, 32, len(decoded), "Decoded secret should be 32 bytes")
		}
	})

	t.Run("secret uniqueness", func(t *testing.T) {
		secrets := make(map[string]bool)

		// Generate multiple secrets and check they're unique
		for i := 0; i < 10; i++ {
			secret := generateJWTSecret()
			// Allow the fallback secret to appear multiple times
			if secret != "your-super-secret-jwt-key-please-change-in-production" {
				assert.False(t, secrets[secret], "JWT secrets should be unique")
				secrets[secret] = true
			}
		}
	})
}

func TestRandomInt(t *testing.T) {
	t.Run("range validation", func(t *testing.T) {
		max := 10
		results := make(map[int]bool)

		// Generate multiple random numbers
		for i := 0; i < 100; i++ {
			n := randomInt(max)
			assert.GreaterOrEqual(t, n, 0, "Random int should be >= 0")
			assert.Less(t, n, max, "Random int should be < max")
			results[n] = true
		}

		// Should have some variation (at least 3 different values out of 10 possible)
		assert.GreaterOrEqual(t, len(results), 3, "Should generate varied random numbers")
	})

	t.Run("edge cases", func(t *testing.T) {
		// Test with max = 1
		for i := 0; i < 10; i++ {
			n := randomInt(1)
			assert.Equal(t, 0, n, "randomInt(1) should always return 0")
		}
	})
}

func TestOutputFunctions(t *testing.T) {
	result := &InitResult{
		BackendApp: Application{
			ID:   "backend-id",
			Name: "backend",
		},
		FrontendApp: Application{
			ID:   "frontend-id",
			Name: "frontend",
		},
		OwnerUser: User{
			Username: "owner",
			Email:    "owner@example.com",
		},
		CustomDomain: "example.com",
		TenantInfo: TenantInfo{
			TenantID: "test-tenant",
			BaseURL:  "https://test-tenant.logto.app",
		},
	}

	t.Run("JSON output", func(t *testing.T) {
		err := outputJSON(result)
		assert.NoError(t, err)

		// Test that the result can be marshaled to JSON
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		assert.NoError(t, err)
		assert.NotEmpty(t, jsonBytes)

		// Test that it's valid JSON
		var unmarshaled InitResult
		err = json.Unmarshal(jsonBytes, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, result.CustomDomain, unmarshaled.CustomDomain)
	})

	t.Run("YAML output", func(t *testing.T) {
		err := outputYAML(result)
		assert.NoError(t, err)

		// Test that the result can be marshaled to YAML
		yamlBytes, err := yaml.Marshal(result)
		assert.NoError(t, err)
		assert.NotEmpty(t, yamlBytes)

		// Test that it's valid YAML
		var unmarshaled InitResult
		err = yaml.Unmarshal(yamlBytes, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, result.CustomDomain, unmarshaled.CustomDomain)
	})

	t.Run("text output", func(t *testing.T) {
		// Set up environment variables for the backend app
		result.BackendApp.EnvironmentVars = map[string]interface{}{
			"LOGTO_ISSUER":                   "https://example.logto.app",
			"LOGTO_AUDIENCE":                 "https://api.example.com",
			"JWT_SECRET":                     "test-secret",
			"LOGTO_MANAGEMENT_CLIENT_ID":     "client-id",
			"LOGTO_MANAGEMENT_CLIENT_SECRET": "client-secret",
			"LOGTO_MANAGEMENT_BASE_URL":      "https://example.logto.app/api",
			"LISTEN_ADDRESS":                 "127.0.0.1:8080",
		}

		result.FrontendApp.EnvironmentVars = map[string]interface{}{
			"VITE_LOGTO_ENDPOINT": "https://example.logto.app",
			"VITE_LOGTO_APP_ID":   "frontend-id",
			"VITE_API_BASE_URL":   "https://api.example.com",
		}

		result.OwnerUser.Password = "test-password"

		// This function outputs to stdout, so we can't easily capture it
		// We'll just verify it doesn't panic
		assert.NotPanics(t, func() {
			outputText(result)
		})
	})
}

func TestInitCommandInit(t *testing.T) {
	t.Run("command is added to root", func(t *testing.T) {
		// Check that initCmd is properly added to rootCmd
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Use == "init" {
				found = true
				break
			}
		}
		assert.True(t, found, "init command should be added to root command")
	})
}

func TestDeriveEnvironmentVariables(t *testing.T) {
	config := &InitConfig{
		TenantID:            "test-tenant",
		TenantDomain:        "example.com",
		BackendClientID:     "backend-client",
		BackendClientSecret: "backend-secret",
	}

	result := &InitResult{
		FrontendApp: Application{
			ClientID: "frontend-client-id",
		},
	}

	t.Run("environment variable derivation", func(t *testing.T) {
		err := deriveEnvironmentVariables(nil, result, config)
		assert.NoError(t, err)

		// Check backend environment variables
		backendEnv := result.BackendApp.EnvironmentVars
		assert.NotNil(t, backendEnv)

		expectedBackendVars := []string{
			"LOGTO_ISSUER",
			"LOGTO_AUDIENCE",
			"LOGTO_JWKS_ENDPOINT",
			"LOGTO_MANAGEMENT_CLIENT_ID",
			"LOGTO_MANAGEMENT_CLIENT_SECRET",
			"LOGTO_MANAGEMENT_BASE_URL",
			"JWT_ISSUER",
			"JWT_EXPIRATION",
			"JWT_REFRESH_EXPIRATION",
			"LISTEN_ADDRESS",
		}

		for _, envVar := range expectedBackendVars {
			assert.Contains(t, backendEnv, envVar, "Backend env should contain %s", envVar)
		}

		// Check frontend environment variables
		frontendEnv := result.FrontendApp.EnvironmentVars
		assert.NotNil(t, frontendEnv)

		expectedFrontendVars := []string{
			"VITE_LOGTO_ENDPOINT",
			"VITE_LOGTO_APP_ID",
			"VITE_LOGTO_RESOURCES",
			"VITE_API_BASE_URL",
		}

		for _, envVar := range expectedFrontendVars {
			assert.Contains(t, frontendEnv, envVar, "Frontend env should contain %s", envVar)
		}

		// Check specific values
		assert.Equal(t, "https://test-tenant.logto.app", backendEnv["LOGTO_ISSUER"])
		assert.Equal(t, "https://example.com/api", backendEnv["LOGTO_AUDIENCE"])
		assert.Equal(t, "example.com.api", backendEnv["JWT_ISSUER"])
		assert.Equal(t, "frontend-client-id", frontendEnv["VITE_LOGTO_APP_ID"])
	})
}
