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
	"encoding/json"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestInitResult(t *testing.T) {
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
			EnvFile:   ".env.production",
		}

		// Verify structure is complete
		assert.NotEmpty(t, result.BackendApp.ID)
		assert.NotEmpty(t, result.FrontendApp.ID)
		assert.NotEmpty(t, result.OwnerUser.ID)
		assert.NotEmpty(t, result.CustomDomain)
		assert.NotEmpty(t, result.TenantInfo.TenantID)
		assert.Len(t, result.NextSteps, 2)
		assert.Equal(t, ".env.production", result.EnvFile)
	})
}

func TestApplication(t *testing.T) {
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
}

func TestUser(t *testing.T) {
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

func TestTenantInfo(t *testing.T) {
	t.Run("TenantInfo structure", func(t *testing.T) {
		info := TenantInfo{
			TenantID: "test-tenant",
			BaseURL:  "https://test-tenant.logto.app",
			Mode:     "cli",
		}

		assert.Equal(t, "test-tenant", info.TenantID)
		assert.Equal(t, "https://test-tenant.logto.app", info.BaseURL)
		assert.Equal(t, "cli", info.Mode)
	})
}

func TestOutputFunctions(t *testing.T) {
	result := &InitResult{
		BackendApp: Application{
			ID:   "backend-id",
			Name: "backend",
			EnvironmentVars: map[string]interface{}{
				"LOGTO_ISSUER":              "https://example.logto.app",
				"LOGTO_AUDIENCE":            "https://api.example.com",
				"JWT_SECRET":                "test-secret",
				"BACKEND_APP_ID":            "client-id",
				"BACKEND_APP_SECRET":        "client-secret",
				"LOGTO_MANAGEMENT_BASE_URL": "https://example.logto.app/api",
				"LISTEN_ADDRESS":            "127.0.0.1:8080",
			},
		},
		FrontendApp: Application{
			ID:   "frontend-id",
			Name: "frontend",
			EnvironmentVars: map[string]interface{}{
				"VITE_LOGTO_ENDPOINT": "https://example.com",
				"VITE_LOGTO_APP_ID":   "frontend-id",
				"VITE_API_BASE_URL":   "https://app.example.com/backend/api",
			},
		},
		OwnerUser: User{
			Username: "owner",
			Email:    "owner@example.com",
			Password: "test-password",
		},
		CustomDomain: "example.com",
		TenantInfo: TenantInfo{
			TenantID: "test-tenant",
			BaseURL:  "https://test-tenant.logto.app",
		},
		EnvFile: ".env.staging",
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
		// This function outputs to stdout, so we can't easily capture it
		// We'll just verify it doesn't panic
		assert.NotPanics(t, func() {
			outputText(result)
		})
	})

	t.Run("text output with custom env file", func(t *testing.T) {
		customResult := *result
		customResult.EnvFile = ".env.production"

		assert.NotPanics(t, func() {
			outputText(&customResult)
		})
	})

	t.Run("text output with default env file", func(t *testing.T) {
		customResult := *result
		customResult.EnvFile = ""

		assert.NotPanics(t, func() {
			outputText(&customResult)
		})
	})

	t.Run("OutputSetupInstructions with different formats", func(t *testing.T) {
		// Test default (text) format
		viper.Set("output", "text")
		err := OutputSetupInstructions(result)
		assert.NoError(t, err)

		// Test JSON format
		viper.Set("output", "json")
		err = OutputSetupInstructions(result)
		assert.NoError(t, err)

		// Test YAML format
		viper.Set("output", "yaml")
		err = OutputSetupInstructions(result)
		assert.NoError(t, err)

		// Reset to default
		viper.Set("output", "text")
	})
}
