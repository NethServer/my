/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package initcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveEnvironmentVariables(t *testing.T) {
	config := &InitConfig{
		TenantID:            "test-tenant",
		TenantDomain:        "example.com",
		BackendClientID:     "backend-client",
		BackendClientSecret: "backend-secret",
	}

	backendApp := &Application{
		ID:       "backend-id",
		Name:     "backend",
		Type:     "MachineToMachine",
		ClientID: "backend-client",
	}

	frontendApp := &Application{
		ID:       "frontend-id",
		Name:     "frontend",
		Type:     "SPA",
		ClientID: "frontend-client-id",
	}

	t.Run("environment variable derivation", func(t *testing.T) {
		err := DeriveEnvironmentVariables(config, backendApp, frontendApp)
		assert.NoError(t, err)

		// Check backend environment variables
		backendEnv := backendApp.EnvironmentVars
		assert.NotNil(t, backendEnv)

		expectedBackendVars := []string{
			"LOGTO_ISSUER",
			"LOGTO_AUDIENCE",
			"LOGTO_JWKS_ENDPOINT",
			"BACKEND_CLIENT_ID",
			"BACKEND_CLIENT_SECRET",
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
		frontendEnv := frontendApp.EnvironmentVars
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

	t.Run("environment variables with nil maps", func(t *testing.T) {
		// Test that the function initializes the maps if they're nil
		appWithoutEnv := &Application{
			ID:       "test-id",
			Name:     "test-app",
			Type:     "SPA",
			ClientID: "test-client-id",
		}

		frontendWithoutEnv := &Application{
			ID:       "frontend-test-id",
			Name:     "frontend-test",
			Type:     "SPA",
			ClientID: "frontend-test-client-id",
		}

		err := DeriveEnvironmentVariables(config, appWithoutEnv, frontendWithoutEnv)
		assert.NoError(t, err)

		assert.NotNil(t, appWithoutEnv.EnvironmentVars)
		assert.NotNil(t, frontendWithoutEnv.EnvironmentVars)
		assert.NotEmpty(t, appWithoutEnv.EnvironmentVars)
		assert.NotEmpty(t, frontendWithoutEnv.EnvironmentVars)
	})

	t.Run("derived URLs format", func(t *testing.T) {
		testConfig := &InitConfig{
			TenantID:     "my-tenant",
			TenantDomain: "mydomain.com",
		}

		testBackend := &Application{ClientID: "backend-id"}
		testFrontend := &Application{ClientID: "frontend-id"}

		err := DeriveEnvironmentVariables(testConfig, testBackend, testFrontend)
		assert.NoError(t, err)

		backendEnv := testBackend.EnvironmentVars

		// Test URL formatting
		assert.Equal(t, "https://my-tenant.logto.app", backendEnv["LOGTO_ISSUER"])
		assert.Equal(t, "https://mydomain.com/api", backendEnv["LOGTO_AUDIENCE"])
		assert.Equal(t, "https://my-tenant.logto.app/oidc/jwks", backendEnv["LOGTO_JWKS_ENDPOINT"])
		assert.Equal(t, "https://my-tenant.logto.app/api", backendEnv["LOGTO_MANAGEMENT_BASE_URL"])
		assert.Equal(t, "mydomain.com.api", backendEnv["JWT_ISSUER"])

		frontendEnv := testFrontend.EnvironmentVars
		assert.Equal(t, "https://my-tenant.logto.app", frontendEnv["VITE_LOGTO_ENDPOINT"])
		assert.Equal(t, "frontend-id", frontendEnv["VITE_LOGTO_APP_ID"])
		assert.Equal(t, "[\"https://mydomain.com/api\"]", frontendEnv["VITE_LOGTO_RESOURCES"])
		assert.Equal(t, "https://mydomain.com/api", frontendEnv["VITE_API_BASE_URL"])
	})
}

// Note: CreateCustomDomain, CreateApplications, and CheckIfAlreadyInitialized functions
// require a real Logto client and would need integration tests or mocking.
// For now, we focus on the functions that can be unit tested in isolation.
