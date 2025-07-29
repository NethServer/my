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
	"fmt"
	"strings"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/constants"
	"github.com/nethesis/my/sync/internal/logger"
)

// Application represents an application structure
type Application struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	ClientID        string                 `json:"client_id"`
	ClientSecret    string                 `json:"client_secret,omitempty"`
	EnvironmentVars map[string]interface{} `json:"environment_vars,omitempty"`
}

// CreateCustomDomain creates a custom domain in Logto
func CreateCustomDomain(client *client.LogtoClient, config *InitConfig) error {
	logger.Info("Creating custom domain...")

	// Check if domain already exists
	domains, err := client.GetDomains()
	if err != nil {
		return fmt.Errorf("failed to get existing domains: %w", err)
	}

	// Check if our domain already exists
	for _, domain := range domains {
		if domainName, ok := domain["domain"].(string); ok && domainName == config.TenantDomain {
			logger.Info("Custom domain already exists: %s", config.TenantDomain)
			return nil
		}
	}

	// Create the custom domain
	domainData := map[string]interface{}{
		"domain": config.TenantDomain,
	}

	createdDomain, err := client.CreateDomain(domainData)
	if err != nil {
		// Check for domain limit or already exists errors
		errStr := err.Error()
		if strings.Contains(errStr, "limit_to_one_domain") {
			logger.Warn("Tenant already has a custom domain (limit: 1 per tenant)")
			logger.Info("Using requested domain name for configuration: %s", config.TenantDomain)
			return nil
		}
		if strings.Contains(errStr, "hostname_already_exists") || strings.Contains(errStr, "already exists") {
			logger.Warn("Domain %s already exists in Logto (possibly in another tenant)", config.TenantDomain)
			logger.Info("Continuing with initialization - domain creation is optional")
			return nil
		}
		return fmt.Errorf("failed to create custom domain: %w", err)
	}

	if domainName, ok := createdDomain["domain"].(string); ok {
		logger.Info("Created custom domain: %s", domainName)
	} else {
		logger.Info("Created custom domain: %s", config.TenantDomain)
	}

	return nil
}

// CreateApplications creates or verifies the backend and frontend applications
func CreateApplications(client *client.LogtoClient, config *InitConfig) (*Application, *Application, error) {
	logger.Info("Creating Logto applications...")

	// Verify backend M2M application exists (must be created manually first)
	backendAppID := config.BackendAppID
	backendAppSecret := config.BackendAppSecret

	apps, err := client.GetApplications()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check applications: %w", err)
	}

	var backendApp *Application
	for _, app := range apps {
		if appID, ok := app["id"].(string); ok && appID == backendAppID {
			if name, ok := app["name"].(string); ok {
				backendApp = &Application{
					ID:           appID,
					Name:         name,
					Type:         "MachineToMachine",
					ClientID:     appID,
					ClientSecret: backendAppSecret,
				}
				logger.Info("Verified backend M2M application: %s (%s)", name, appID)
				break
			}
		}
	}

	if backendApp == nil {
		return nil, nil, fmt.Errorf("backend M2M application with ID '%s' not found in Logto.\n"+
			"Please create a Machine-to-Machine application named 'backend' and update your .env file", backendAppID)
	}

	logger.Info("Backend M2M app ready: %s", backendApp.Name)

	// Check if frontend application already exists
	var frontendApp *Application
	for _, app := range apps {
		if name, ok := app["name"].(string); ok && name == constants.FrontendAppName {
			if appType, ok := app["type"].(string); ok && appType == constants.AppTypeSPA {
				frontendApp = &Application{
					ID:       app["id"].(string),
					Name:     name,
					Type:     constants.AppTypeSPA,
					ClientID: app["id"].(string),
				}
				logger.Info("Found existing frontend SPA application: %s (%s)", name, frontendApp.ID)
				break
			}
		}
	}

	// Create frontend application only if it doesn't exist
	if frontendApp == nil {
		frontendAppData := map[string]interface{}{
			"name":        constants.FrontendAppName,
			"type":        constants.AppTypeSPA,
			"description": "Single Page Application for My Nethesis",
			"oidcClientMetadata": map[string]interface{}{
				"redirectUris":            []string{"http://localhost:5173/login-redirect", fmt.Sprintf("https://%s/login-redirect", config.TenantDomain)},
				"postLogoutRedirectUris":  []string{"http://localhost:5173/login", fmt.Sprintf("https://%s/login", config.TenantDomain)},
				"corsAllowedOrigins":      []string{"http://localhost:5173", fmt.Sprintf("https://%s", config.TenantDomain)},
				"refreshTokenTtlInDays":   30,
				"alwaysIssueRefreshToken": true,
				"requireAuthTime":         false,
				"requirePkce":             true,
			},
		}

		createdFrontend, err := client.CreateApplication(frontendAppData)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create frontend application: %w", err)
		}

		frontendApp = &Application{
			ID:       createdFrontend["id"].(string),
			Name:     createdFrontend["name"].(string),
			Type:     "SPA",
			ClientID: createdFrontend["id"].(string),
		}

		logger.Info("Created frontend application: %s", frontendApp.ID)
	}

	return backendApp, frontendApp, nil
}

// DeriveEnvironmentVariables derives environment variables for backend and frontend applications
func DeriveEnvironmentVariables(config *InitConfig, backendApp, frontendApp *Application) error {
	logger.Info("Deriving environment variables...")

	baseURL := fmt.Sprintf("https://%s.logto.app", config.TenantID)
	apiBaseURL := fmt.Sprintf("https://%s/api", config.TenantDomain)

	// Initialize empty backend app if not set yet
	if backendApp.EnvironmentVars == nil {
		backendApp.EnvironmentVars = make(map[string]interface{})
	}

	// Backend environment variables
	backendApp.EnvironmentVars = map[string]interface{}{
		// Required configuration
		"TENANT_ID":          config.TenantID,
		"TENANT_DOMAIN":      config.TenantDomain,
		"BACKEND_APP_ID":     config.BackendAppID,
		"BACKEND_APP_SECRET": config.BackendAppSecret,
		"JWT_SECRET":         GenerateJWTSecret(),
		"DATABASE_URL":       "postgresql://noc_user:noc_password@localhost:5432/noc?sslmode=disable",
		"REDIS_URL":          "redis://localhost:6379",
	}

	// Initialize empty frontend app if not set yet
	if frontendApp.EnvironmentVars == nil {
		frontendApp.EnvironmentVars = make(map[string]interface{})
	}

	// Frontend environment variables
	frontendApp.EnvironmentVars = map[string]interface{}{
		"VITE_LOGTO_ENDPOINT":  baseURL,
		"VITE_LOGTO_APP_ID":    frontendApp.ClientID,
		"VITE_LOGTO_RESOURCES": fmt.Sprintf("[\"%s\"]", apiBaseURL),
		"VITE_API_BASE_URL":    apiBaseURL,
	}

	logger.Info("Using tenant domain: %s", config.TenantDomain)
	logger.Info("Derived API base URL: %s", apiBaseURL)
	logger.Info("Derived JWT issuer: %s", config.TenantDomain+".api")

	return nil
}

// CheckIfAlreadyInitialized checks if the Logto instance is already initialized
func CheckIfAlreadyInitialized(client *client.LogtoClient, config *InitConfig) (bool, error) {
	// Check if custom domain exists
	domains, err := client.GetDomains()
	if err != nil {
		return false, err
	}

	domainExists := false
	for _, domain := range domains {
		if domainName, ok := domain["domain"].(string); ok && domainName == config.TenantDomain {
			domainExists = true
			break
		}
	}

	// Check if backend M2M application exists (using credentials from config)
	backendAppID := config.BackendAppID

	apps, err := client.GetApplications()
	if err != nil {
		return false, err
	}

	backendExists := false
	frontendExists := false

	for _, app := range apps {
		if appID, ok := app["id"].(string); ok && appID == backendAppID {
			backendExists = true
		}
		if name, ok := app["name"].(string); ok && name == constants.FrontendAppName {
			frontendExists = true
		}
	}

	// Check if owner user exists
	users, err := client.GetUsers()
	if err != nil {
		return false, err
	}

	ownerExists := false
	for _, user := range users {
		if username, ok := user["username"].(string); ok && username == "owner" {
			ownerExists = true
			break
		}
	}

	return domainExists && backendExists && frontendExists && ownerExists, nil
}
