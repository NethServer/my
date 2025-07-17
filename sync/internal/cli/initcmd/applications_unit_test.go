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
		TenantID:         "test-tenant",
		TenantDomain:     "example.com",
		BackendAppID:     "backend-client",
		BackendAppSecret: "backend-secret",
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
			"TENANT_ID",
			"TENANT_DOMAIN",
			"BACKEND_APP_ID",
			"BACKEND_APP_SECRET",
			"JWT_SECRET",
			"DATABASE_URL",
			"REDIS_URL",
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
		assert.Equal(t, "test-tenant", backendEnv["TENANT_ID"])
		assert.Equal(t, "example.com", backendEnv["TENANT_DOMAIN"])
		assert.Equal(t, "backend-client", backendEnv["BACKEND_APP_ID"])
		assert.Equal(t, "backend-secret", backendEnv["BACKEND_APP_SECRET"])
		assert.Equal(t, "postgresql://noc_user:noc_user@localhost:5432/noc?sslmode=disable", backendEnv["DATABASE_URL"])
		assert.Equal(t, "redis://localhost:6379", backendEnv["REDIS_URL"])
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

		// Test that environment variables are populated correctly
		assert.Equal(t, "my-tenant", backendEnv["TENANT_ID"])
		assert.Equal(t, "mydomain.com", backendEnv["TENANT_DOMAIN"])
		assert.Equal(t, "postgresql://noc_user:noc_user@localhost:5432/noc?sslmode=disable", backendEnv["DATABASE_URL"])
		assert.Equal(t, "redis://localhost:6379", backendEnv["REDIS_URL"])

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
