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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/constants"
	"github.com/nethesis/my/sync/internal/logger"
)

var (
	initForce               bool
	initDomain              string
	initTenantID            string
	initBackendClientID     string
	initBackendClientSecret string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Logto configuration with basic setup",
	Long: `Initialize Logto with basic configuration required for Nethesis Operation Center.

This command will:
1. Create custom domain in Logto (e.g., your-domain.com)
2. Create backend and frontend applications in Logto
3. Create a god@nethesis.it account with generated password
4. Synchronize basic RBAC configuration
5. Output environment variables and setup instructions

Requirements:
- A Machine-to-Machine application in Logto with Management API access

Two modes available:

Mode 1 - Environment Variables:
  TENANT_ID=your-tenant-id
  BACKEND_CLIENT_ID=your-backend-client-id
  BACKEND_CLIENT_SECRET=your-secret
  TENANT_DOMAIN=your-domain.com

  sync init

Mode 2 - CLI Flags:
  sync init --tenant-id your-tenant-id --backend-client-id your-backend-client-id --backend-client-secret your-secret --domain your-domain.com

Output formats:
  sync init --output json   # JSON output for automation/CI-CD
  sync init --output text   # Human-readable output (default)

Note: CLI flags take precedence over environment variables. If any CLI flag is provided, all must be provided.`,
	RunE: runInit,
}

type InitResult struct {
	BackendApp      Application `json:"backend_app"`
	FrontendApp     Application `json:"frontend_app"`
	GodUser         User        `json:"god_user"`
	CustomDomain    string      `json:"custom_domain"`
	GeneratedSecret string      `json:"generated_jwt_secret"`
	AlreadyInit     bool        `json:"already_initialized"`
	TenantInfo      TenantInfo  `json:"tenant_info"`
	NextSteps       []string    `json:"next_steps"`
}

type TenantInfo struct {
	TenantID string `json:"tenant_id"`
	BaseURL  string `json:"base_url"`
	Mode     string `json:"mode"` // "env" or "cli"
}

type Application struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`
	ClientID        string                 `json:"client_id"`
	ClientSecret    string                 `json:"client_secret,omitempty"`
	EnvironmentVars map[string]interface{} `json:"environment_vars,omitempty"`
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type InitConfig struct {
	TenantID            string
	TenantDomain        string
	BackendClientID     string
	BackendClientSecret string
	Mode                string // "env" or "cli"
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Init-specific flags
	initCmd.Flags().BoolVar(&initForce, "force", false, "force re-initialization even if already done")
	initCmd.Flags().StringVar(&initDomain, "domain", "", "tenant domain (e.g., your-domain.com)")
	initCmd.Flags().StringVar(&initTenantID, "tenant-id", "", "Logto tenant ID (e.g., your-tenant-id)")
	initCmd.Flags().StringVar(&initBackendClientID, "backend-client-id", "", "backend M2M application client ID")
	initCmd.Flags().StringVar(&initBackendClientSecret, "backend-client-secret", "", "backend M2M application client secret")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine configuration mode and validate parameters
	config, err := validateAndGetConfig()
	if err != nil {
		return err
	}

	logger.Info("Using tenant domain: %s", config.TenantDomain)
	logger.Info("Using tenant ID: %s", config.TenantID)

	// Create Logto client using configuration
	baseURL := fmt.Sprintf("https://%s.logto.app", config.TenantID)

	logtoClient := client.NewLogtoClient(
		baseURL,
		config.BackendClientID,
		config.BackendClientSecret,
	)

	// Test connection
	logger.Info("Testing connection to Logto...")
	if err := logtoClient.TestConnection(); err != nil {
		return fmt.Errorf("failed to connect to Logto: %w", err)
	}

	// Check if already initialized
	alreadyInit, err := checkIfAlreadyInitialized(logtoClient, config)
	if err != nil {
		return fmt.Errorf("failed to check initialization status: %w", err)
	}

	if alreadyInit && !initForce {
		logger.Info("Logto appears to already be initialized for Nethesis Operation Center")
		logger.Info("Use 'sync sync' to synchronize configuration changes")
		logger.Info("Use 'sync init --force' to re-run initialization")
		return nil
	}

	result := &InitResult{
		AlreadyInit:  alreadyInit,
		CustomDomain: config.TenantDomain,
		TenantInfo: TenantInfo{
			TenantID: config.TenantID,
			BaseURL:  fmt.Sprintf("https://%s.logto.app", config.TenantID),
			Mode:     config.Mode,
		},
		NextSteps: []string{
			"Copy the environment variables to your .env files",
			"Start your backend: cd backend && go run main.go",
			"Start your frontend with the Logto configuration",
			"Login with the admin credentials provided",
			"Use 'sync sync' to update RBAC configuration when needed",
		},
	}

	logger.Info("Starting Logto initialization...")

	// Step 1: Create custom domain
	if err := createCustomDomain(logtoClient, config); err != nil {
		return fmt.Errorf("failed to create custom domain: %w", err)
	}

	// Step 2: Create applications
	if err := createApplications(logtoClient, result, config); err != nil {
		return fmt.Errorf("failed to create applications: %w", err)
	}

	// Step 3: Derive environment variables and populate applications
	if err := deriveEnvironmentVariables(logtoClient, result, config); err != nil {
		return fmt.Errorf("failed to derive environment variables: %w", err)
	}

	// Step 4: Create god user
	if err := createGodUser(logtoClient, result); err != nil {
		return fmt.Errorf("failed to create god user: %w", err)
	}

	// Step 5: Generate JWT secret
	result.GeneratedSecret = generateJWTSecret()
	if result.BackendApp.EnvironmentVars != nil {
		result.BackendApp.EnvironmentVars["JWT_SECRET"] = result.GeneratedSecret
	}

	// Step 6: Sync basic RBAC configuration
	if err := syncBasicConfiguration(logtoClient); err != nil {
		return fmt.Errorf("failed to sync basic configuration: %w", err)
	}

	// Step 7: Output setup instructions
	if err := outputSetupInstructions(result); err != nil {
		return fmt.Errorf("failed to output setup instructions: %w", err)
	}

	logger.Info("Logto initialization completed successfully!")
	return nil
}

func deriveEnvironmentVariables(client *client.LogtoClient, result *InitResult, config *InitConfig) error {
	logger.Info("Deriving environment variables...")

	baseURL := fmt.Sprintf("https://%s.logto.app", config.TenantID)
	apiBaseURL := fmt.Sprintf("https://%s/api", config.TenantDomain)

	// Initialize empty backend app if not set yet
	if result.BackendApp.EnvironmentVars == nil {
		result.BackendApp.EnvironmentVars = make(map[string]interface{})
	}

	// Backend environment variables
	result.BackendApp.EnvironmentVars = map[string]interface{}{
		// Auto-derived from base URL
		"LOGTO_ISSUER":        baseURL,
		"LOGTO_AUDIENCE":      apiBaseURL,
		"LOGTO_JWKS_ENDPOINT": baseURL + "/oidc/jwks",

		// From configuration
		"LOGTO_MANAGEMENT_CLIENT_ID":     config.BackendClientID,
		"LOGTO_MANAGEMENT_CLIENT_SECRET": config.BackendClientSecret,
		"LOGTO_MANAGEMENT_BASE_URL":      baseURL + "/api",

		// From provided tenant domain
		"JWT_ISSUER":             config.TenantDomain + ".api",
		"JWT_EXPIRATION":         "24h",
		"JWT_REFRESH_EXPIRATION": "168h",
		"LISTEN_ADDRESS":         "127.0.0.1:8080",
	}

	// Initialize empty frontend app if not set yet
	if result.FrontendApp.EnvironmentVars == nil {
		result.FrontendApp.EnvironmentVars = make(map[string]interface{})
	}

	// Frontend environment variables
	result.FrontendApp.EnvironmentVars = map[string]interface{}{
		"VITE_LOGTO_ENDPOINT":  baseURL,
		"VITE_LOGTO_APP_ID":    result.FrontendApp.ClientID,
		"VITE_LOGTO_RESOURCES": fmt.Sprintf("[\"%s\"]", apiBaseURL),
		"VITE_API_BASE_URL":    apiBaseURL,
	}

	logger.Info("Using tenant domain: %s", config.TenantDomain)
	logger.Info("Derived API base URL: %s", apiBaseURL)
	logger.Info("Derived JWT issuer: %s", config.TenantDomain+".api")

	return nil
}

func checkIfAlreadyInitialized(client *client.LogtoClient, config *InitConfig) (bool, error) {
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
	backendClientID := config.BackendClientID

	apps, err := client.GetApplications()
	if err != nil {
		return false, err
	}

	backendExists := false
	frontendExists := false

	for _, app := range apps {
		if appID, ok := app["id"].(string); ok && appID == backendClientID {
			backendExists = true
		}
		if name, ok := app["name"].(string); ok && name == constants.FrontendAppName {
			frontendExists = true
		}
	}

	// Check if god user exists
	users, err := client.GetUsers()
	if err != nil {
		return false, err
	}

	godExists := false
	for _, user := range users {
		if username, ok := user["username"].(string); ok && username == "god" {
			godExists = true
			break
		}
	}

	return domainExists && backendExists && frontendExists && godExists, nil
}

func createCustomDomain(client *client.LogtoClient, config *InitConfig) error {
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

func createApplications(client *client.LogtoClient, result *InitResult, config *InitConfig) error {
	logger.Info("Creating Logto applications...")

	// Verify backend M2M application exists (must be created manually first)
	backendClientID := config.BackendClientID
	backendClientSecret := config.BackendClientSecret

	apps, err := client.GetApplications()
	if err != nil {
		return fmt.Errorf("failed to check existing applications: %w", err)
	}

	var backendApp *Application
	for _, app := range apps {
		if appID, ok := app["id"].(string); ok && appID == backendClientID {
			if name, ok := app["name"].(string); ok {
				backendApp = &Application{
					ID:           appID,
					Name:         name,
					Type:         "MachineToMachine",
					ClientID:     appID,
					ClientSecret: backendClientSecret,
				}
				logger.Info("Verified backend M2M application: %s (%s)", name, appID)
				break
			}
		}
	}

	if backendApp == nil {
		return fmt.Errorf("backend M2M application with ID '%s' not found in Logto.\n"+
			"Please create a Machine-to-Machine application named 'backend' and update your .env file", backendClientID)
	}

	result.BackendApp = *backendApp

	// JWT audience is already set in deriveEnvironmentVariables

	logger.Info("Backend M2M app ready: %s", result.BackendApp.Name)

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
			"description": "Single Page Application for Nethesis Operation Center",
			"oidcClientMetadata": map[string]interface{}{
				"redirectUris":            []string{"http://localhost:5173/callback", fmt.Sprintf("https://%s/callback", config.TenantDomain)},
				"postLogoutRedirectUris":  []string{"http://localhost:5173", fmt.Sprintf("https://%s", config.TenantDomain)},
				"corsAllowedOrigins":      []string{"http://localhost:5173", fmt.Sprintf("https://%s", config.TenantDomain)},
				"refreshTokenTtlInDays":   30,
				"alwaysIssueRefreshToken": true,
				"requireAuthTime":         false,
				"requirePkce":             true,
			},
		}

		createdFrontend, err := client.CreateApplication(frontendAppData)
		if err != nil {
			return fmt.Errorf("failed to create frontend application: %w", err)
		}

		frontendApp = &Application{
			ID:       createdFrontend["id"].(string),
			Name:     createdFrontend["name"].(string),
			Type:     "SPA",
			ClientID: createdFrontend["id"].(string),
		}

		logger.Info("Created frontend application: %s", frontendApp.ID)
	}

	result.FrontendApp = *frontendApp
	return nil
}

func createGodUser(client *client.LogtoClient, result *InitResult) error {
	logger.Info("Creating god@nethesis.it user...")

	password := generateSecurePassword()

	// Create user
	userData := map[string]interface{}{
		"username":     constants.GodUsername,
		"primaryEmail": constants.GodUserEmail,
		"name":         constants.GodUserDisplayName,
	}

	createdUser, err := client.CreateUser(userData)
	if err != nil {
		// Check if user already exists
		errStr := err.Error()
		if strings.Contains(errStr, "username_already_in_use") || strings.Contains(errStr, "already in use") {
			logger.Warn("User 'god' already exists")
			logger.Info("Using existing user for configuration (password not updated)")

			// Find existing user
			users, userErr := client.GetUsers()
			if userErr != nil {
				return fmt.Errorf("failed to get existing users: %w", userErr)
			}

			var existingUserID string
			for _, user := range users {
				if username, ok := user["username"].(string); ok && username == "god" {
					existingUserID = user["id"].(string)
					break
				}
			}

			if existingUserID == "" {
				return fmt.Errorf("could not find existing god user")
			}

			result.GodUser = User{
				ID:       existingUserID,
				Username: "god",
				Email:    "god@nethesis.it",
				Password: "[EXISTING - NOT CHANGED]",
			}

			logger.Info("Using existing god user: %s", result.GodUser.ID)
			return nil
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	userID := createdUser["id"].(string)

	// Set password
	if err := client.SetUserPassword(userID, password); err != nil {
		return fmt.Errorf("failed to set user password: %w", err)
	}

	result.GodUser = User{
		ID:       userID,
		Username: "god",
		Email:    "god@nethesis.it",
		Password: password,
	}

	logger.Info("Created god user: %s", result.GodUser.ID)
	return nil
}

func syncBasicConfiguration(client *client.LogtoClient) error {
	logger.Info("Synchronizing basic RBAC configuration...")

	// Create essential roles for god user to work immediately
	if err := createEssentialRoles(client); err != nil {
		return fmt.Errorf("failed to create essential roles: %w", err)
	}

	// Create God organization
	if err := createGodOrganization(client); err != nil {
		return fmt.Errorf("failed to create God organization: %w", err)
	}

	// Assign roles and organization to god user
	if err := assignRolesToGodUser(client); err != nil {
		return fmt.Errorf("failed to assign roles to god user: %w", err)
	}

	logger.Info("Basic RBAC configuration synchronized - god user ready")
	logger.Info("Run 'sync sync' later to create complete RBAC configuration")
	return nil
}

func createEssentialRoles(client *client.LogtoClient) error {
	logger.Info("Creating essential roles...")

	// Create organization scopes first (from config.yml)
	if err := createEssentialOrgScopes(client); err != nil {
		return fmt.Errorf("failed to create essential organization scopes: %w", err)
	}

	// Create organization role "god" (from config.yml)
	if err := createOrgRoleIfNotExists(client, constants.GodRoleID, constants.GodRoleName, constants.GodOrgDescription); err != nil {
		return fmt.Errorf("failed to create god organization role: %w", err)
	}

	// Create user role "admin" (from config.yml)
	if err := createUserRoleIfNotExists(client, constants.AdminRoleID, constants.AdminRoleName, "Admin user role - full technical control including dangerous operations"); err != nil {
		return fmt.Errorf("failed to create admin user role: %w", err)
	}

	// Assign scopes to god organization role
	if err := assignScopesToGodRole(client); err != nil {
		return fmt.Errorf("failed to assign scopes to god role: %w", err)
	}

	logger.Info("Essential roles created successfully")
	return nil
}

func createOrgRoleIfNotExists(client *client.LogtoClient, roleID, roleName, description string) error {
	// Check if organization role already exists
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing organization roles: %w", err)
	}

	for _, role := range orgRoles {
		if role.Name == roleName {
			logger.Info("Organization role '%s' already exists, skipping creation", roleName)
			return nil
		}
	}

	// Create the organization role using the simple method
	orgRoleMap := map[string]interface{}{
		"name":        roleName,
		"description": description,
	}

	err = client.CreateOrganizationRoleSimple(orgRoleMap)
	if err != nil {
		return fmt.Errorf("failed to create organization role '%s': %w", roleName, err)
	}

	logger.Info("Created organization role: %s", roleName)
	return nil
}

func createUserRoleIfNotExists(client *client.LogtoClient, roleID, roleName, description string) error {
	// Check if user role already exists
	userRoles, err := client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to get existing user roles: %w", err)
	}

	for _, role := range userRoles {
		if role.Name == roleName {
			logger.Info("User role '%s' already exists, skipping creation", roleName)
			return nil
		}
	}

	// Create the user role using the simple method
	roleData := map[string]interface{}{
		"name":        roleName,
		"description": description,
	}

	err = client.CreateRoleSimple(roleData)
	if err != nil {
		return fmt.Errorf("failed to create user role '%s': %w", roleName, err)
	}

	logger.Info("Created user role: %s", roleName)
	return nil
}

func createEssentialOrgScopes(client *client.LogtoClient) error {
	logger.Info("Creating essential organization scopes...")

	// Organization scopes from config.yml for god role
	scopes := []struct {
		name        string
		description string
	}{
		{"create:distributors", "Create distributor organizations"},
		{"manage:distributors", "Manage distributor organizations"},
		{"create:resellers", "Create reseller organizations"},
		{"manage:resellers", "Manage reseller organizations"},
		{"create:customers", "Create customer organizations"},
		{"manage:customers", "Manage customer organizations"},
	}

	for _, scope := range scopes {
		if err := createOrgScopeIfNotExists(client, scope.name, scope.description); err != nil {
			return fmt.Errorf("failed to create scope %s: %w", scope.name, err)
		}
	}

	logger.Info("Essential organization scopes created successfully")
	return nil
}

func createOrgScopeIfNotExists(client *client.LogtoClient, scopeName, description string) error {
	// Check if organization scope already exists
	scopes, err := client.GetOrganizationScopes()
	if err != nil {
		return fmt.Errorf("failed to get existing organization scopes: %w", err)
	}

	for _, scope := range scopes {
		if scope.Name == scopeName {
			logger.Info("Organization scope '%s' already exists, skipping creation", scopeName)
			return nil
		}
	}

	// Create the organization scope using the simple method
	scopeData := map[string]interface{}{
		"name":        scopeName,
		"description": description,
	}

	err = client.CreateOrganizationScopeSimple(scopeData)
	if err != nil {
		return fmt.Errorf("failed to create organization scope '%s': %w", scopeName, err)
	}

	logger.Info("Created organization scope: %s", scopeName)
	return nil
}

func assignScopesToGodRole(client *client.LogtoClient) error {
	logger.Info("Assigning scopes to God organization role...")

	// Get God organization role ID
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get organization roles: %w", err)
	}

	var godRoleID string
	for _, role := range orgRoles {
		if role.Name == "God" {
			godRoleID = role.ID
			break
		}
	}

	if godRoleID == "" {
		return fmt.Errorf("god organization role not found")
	}

	// Get organization scopes to assign
	orgScopes, err := client.GetOrganizationScopes()
	if err != nil {
		return fmt.Errorf("failed to get organization scopes: %w", err)
	}

	// Assign all god-related scopes
	godScopeNames := []string{"create:distributors", "manage:distributors", "create:resellers", "manage:resellers", "create:customers", "manage:customers"}

	for _, scopeName := range godScopeNames {
		var scopeID string
		for _, scope := range orgScopes {
			if scope.Name == scopeName {
				scopeID = scope.ID
				break
			}
		}

		if scopeID != "" {
			if err := client.AssignScopeToOrganizationRole(godRoleID, scopeID); err != nil {
				logger.Warn("Failed to assign scope %s to God role (may already be assigned): %v", scopeName, err)
			} else {
				logger.Info("Assigned scope %s to God organization role", scopeName)
			}
		}
	}

	logger.Info("Scope assignment to God role completed")
	return nil
}

func createGodOrganization(client *client.LogtoClient) error {
	logger.Info("Creating God organization...")

	// Check if God organization already exists
	organizations, err := client.GetOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get existing organizations: %w", err)
	}

	var godOrgID string
	var organizationExists bool
	for _, org := range organizations {
		if org.Name == "God" {
			godOrgID = org.ID
			organizationExists = true
			logger.Info("Organization 'God' already exists, configuring default role")
			break
		}
	}

	// Create organization if it doesn't exist
	if !organizationExists {
		// Create the God organization using the simple method
		godOrgMap := map[string]interface{}{
			"name":        "God",
			"description": "Nethesis God organization - complete control over commercial hierarchy",
		}

		err = client.CreateOrganizationSimple(godOrgMap)
		if err != nil {
			return fmt.Errorf("failed to create God organization: %w", err)
		}

		// Get the created organization ID
		orgs, err := client.GetOrganizations()
		if err != nil {
			return fmt.Errorf("failed to get organizations after creation: %w", err)
		}

		for _, org := range orgs {
			if org.Name == "God" {
				godOrgID = org.ID
				break
			}
		}

		logger.Info("Created God organization")
	}

	// Set JIT organization role for the God organization
	if err := setGodOrganizationDefaultRole(client, godOrgID); err != nil {
		return fmt.Errorf("failed to set JIT role for God organization: %w", err)
	}

	return nil
}

func setGodOrganizationDefaultRole(client *client.LogtoClient, godOrgID string) error {
	logger.Info("Setting JIT organization role for God organization...")

	// Get God organization role ID
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get organization roles: %w", err)
	}

	var godRoleID string
	for _, role := range orgRoles {
		if role.Name == "God" {
			godRoleID = role.ID
			break
		}
	}

	if godRoleID == "" {
		return fmt.Errorf("god organization role not found")
	}

	// Set the JIT organization role (Just-in-Time provisioning)
	if err := client.SetOrganizationJITRole(godOrgID, godRoleID); err != nil {
		logger.Warn("Failed to set JIT organization role (may already be set): %v", err)
	} else {
		logger.Info("Set God as JIT organization role for God organization")
	}

	return nil
}

func assignRolesToGodUser(client *client.LogtoClient) error {
	logger.Info("Assigning roles and organization to god user...")

	// Get god user ID
	users, err := client.GetUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	godUserID, found := client.FindEntityID(users, "username", constants.GodUsername)
	if !found {
		return fmt.Errorf("god user not found")
	}

	// Get God organization ID
	organizations, err := client.GetOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}

	var godOrgID string
	for _, org := range organizations {
		if org.Name == "God" {
			godOrgID = org.ID
			break
		}
	}

	if godOrgID == "" {
		return fmt.Errorf("god organization not found")
	}

	// Get user roles to assign (admin)
	userRoles, err := client.GetRoles()
	if err != nil {
		return fmt.Errorf("failed to get user roles: %w", err)
	}

	var adminRoleID string
	for _, role := range userRoles {
		if role.Name == "Admin" {
			adminRoleID = role.ID
			break
		}
	}

	// Get organization roles to assign (god)
	orgRoles, err := client.GetOrganizationRoles()
	if err != nil {
		return fmt.Errorf("failed to get organization roles: %w", err)
	}

	var godOrgRoleID string
	for _, role := range orgRoles {
		if role.Name == "God" {
			godOrgRoleID = role.ID
			break
		}
	}

	// Assign Admin user role
	if adminRoleID != "" {
		if err := client.AssignRoleToUser(godUserID, adminRoleID); err != nil {
			logger.Warn("Failed to assign Admin user role (may already be assigned): %v", err)
		} else {
			logger.Info("Assigned Admin user role to god user")
		}
	}

	// Add user to God organization
	if err := client.AddUserToOrganization(godOrgID, godUserID); err != nil {
		logger.Warn("Failed to add user to God organization (may already be member): %v", err)
	} else {
		logger.Info("Added god user to God organization")
	}

	// Assign God organization role to user in organization
	if godOrgRoleID != "" {
		logger.Info("Attempting to assign God organization role (ID: %s) to user (ID: %s) in organization (ID: %s)", godOrgRoleID, godUserID, godOrgID)
		if err := client.AssignOrganizationRoleToUser(godOrgID, godUserID, godOrgRoleID); err != nil {
			logger.Warn("Failed to assign God organization role (may already be assigned): %v", err)
		} else {
			logger.Info("Assigned God organization role to god user")
		}
	} else {
		logger.Warn("God organization role ID not found - unable to assign role")
	}

	logger.Info("Role and organization assignment completed")
	return nil
}

func generateSecurePassword() string {
	const (
		lowerCase = "abcdefghijklmnopqrstuvwxyz"
		upperCase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		digits    = "0123456789"
		symbols   = "!@#$%^&*"
		length    = 16
	)

	charset := lowerCase + upperCase + digits + symbols
	password := make([]byte, length)

	// Ensure at least one character from each set
	password[0] = lowerCase[randomInt(len(lowerCase))]
	password[1] = upperCase[randomInt(len(upperCase))]
	password[2] = digits[randomInt(len(digits))]
	password[3] = symbols[randomInt(len(symbols))]

	// Fill the rest randomly
	for i := 4; i < length; i++ {
		password[i] = charset[randomInt(len(charset))]
	}

	// Shuffle the password
	for i := length - 1; i > 0; i-- {
		j := randomInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

func generateJWTSecret() string {
	bytes := make([]byte, 32) // 256-bit key
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to a deterministic but secure method
		return "your-super-secret-jwt-key-please-change-in-production"
	}
	return base64.URLEncoding.EncodeToString(bytes)
}

func randomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

func outputSetupInstructions(result *InitResult) error {
	outputFormat := viper.GetString("output")

	switch outputFormat {
	case "json":
		return outputJSON(result)
	case "yaml":
		return outputYAML(result)
	default:
		outputText(result)
		return nil
	}
}

func outputJSON(result *InitResult) error {
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(jsonBytes))
	return nil
}

func outputYAML(result *InitResult) error {
	yamlBytes, err := yaml.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	fmt.Println(string(yamlBytes))
	return nil
}

func outputText(result *InitResult) {
	backendEnv := result.BackendApp.EnvironmentVars
	frontendEnv := result.FrontendApp.EnvironmentVars

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("🎉 LOGTO INITIALIZATION COMPLETED!")
	fmt.Println(strings.Repeat("=", 80))

	if result.AlreadyInit {
		fmt.Println("⚠️  Note: Some components were already initialized (use --force to recreate)")
		fmt.Println()
	}

	fmt.Println("📋 SETUP INSTRUCTIONS")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Tenant: %s\n", result.CustomDomain)
	if mgmtURL, ok := backendEnv["LOGTO_MANAGEMENT_BASE_URL"].(string); ok {
		fmt.Printf("Base URL: %s\n", mgmtURL)
	}

	// Backend environment variables
	fmt.Println("\n🔧 BACKEND ENVIRONMENT VARIABLES")
	fmt.Println("Copy and paste these to your backend/.env file:")
	fmt.Println()
	fmt.Printf("# Logto Authentication (auto-derived)\n")
	fmt.Printf("LOGTO_ISSUER=%v\n", backendEnv["LOGTO_ISSUER"])
	fmt.Printf("LOGTO_AUDIENCE=%v\n", backendEnv["LOGTO_AUDIENCE"])
	fmt.Printf("LOGTO_JWKS_ENDPOINT=%v\n", backendEnv["LOGTO_JWKS_ENDPOINT"])
	fmt.Printf("\n# Custom JWT Configuration (for legacy endpoints)\n")
	fmt.Printf("JWT_SECRET=%v\n", backendEnv["JWT_SECRET"])
	fmt.Printf("JWT_ISSUER=%v\n", backendEnv["JWT_ISSUER"])
	fmt.Printf("JWT_EXPIRATION=%v\n", backendEnv["JWT_EXPIRATION"])
	fmt.Printf("JWT_REFRESH_EXPIRATION=%v\n", backendEnv["JWT_REFRESH_EXPIRATION"])
	fmt.Printf("\n# Logto Management API (from your M2M app)\n")
	fmt.Printf("LOGTO_MANAGEMENT_CLIENT_ID=%v\n", backendEnv["LOGTO_MANAGEMENT_CLIENT_ID"])
	fmt.Printf("LOGTO_MANAGEMENT_CLIENT_SECRET=%v\n", backendEnv["LOGTO_MANAGEMENT_CLIENT_SECRET"])
	fmt.Printf("LOGTO_MANAGEMENT_BASE_URL=%v\n", backendEnv["LOGTO_MANAGEMENT_BASE_URL"])
	fmt.Printf("\n# Server Configuration\n")
	fmt.Printf("LISTEN_ADDRESS=%v\n", backendEnv["LISTEN_ADDRESS"])

	// Frontend environment variables
	fmt.Println("\n🌐 FRONTEND ENVIRONMENT VARIABLES")
	fmt.Println("Copy and paste these to your frontend/.env file:")
	fmt.Println()
	fmt.Printf("# Logto Configuration (auto-derived)\n")
	fmt.Printf("VITE_LOGTO_ENDPOINT=%v\n", frontendEnv["VITE_LOGTO_ENDPOINT"])
	fmt.Printf("VITE_LOGTO_APP_ID=%v\n", frontendEnv["VITE_LOGTO_APP_ID"])
	fmt.Printf("VITE_LOGTO_RESOURCES=%v\n", frontendEnv["VITE_LOGTO_RESOURCES"])
	fmt.Printf("\n# Backend API\n")
	fmt.Printf("VITE_API_BASE_URL=%v\n", frontendEnv["VITE_API_BASE_URL"])

	// Login credentials
	fmt.Println("\n👤 ADMIN CREDENTIALS")
	fmt.Println("Use these credentials to login:")
	fmt.Println()
	fmt.Printf("Username: %s\n", result.GodUser.Username)
	fmt.Printf("Email:    %s\n", result.GodUser.Email)
	fmt.Printf("Password: %s\n", result.GodUser.Password)
	fmt.Println()
	fmt.Println("⚠️  IMPORTANT: Save these credentials securely and change the password after first login!")

	// Resources created
	fmt.Println("\n📱 RESOURCES CREATED")
	fmt.Println(strings.Repeat("-", 25))
	fmt.Printf("Custom Domain: %s\n", result.CustomDomain)
	fmt.Printf("Backend M2M:   %s (ID: %s)\n", result.BackendApp.Name, result.BackendApp.ID)
	fmt.Printf("Frontend SPA:  %s (ID: %s)\n", result.FrontendApp.Name, result.FrontendApp.ID)

	// Next steps
	fmt.Println("\n📝 NEXT STEPS")
	fmt.Println(strings.Repeat("-", 20))
	fmt.Println("1. Copy the environment variables above to your .env files")
	fmt.Println("2. Start your backend: cd backend && go run main.go")
	fmt.Println("3. Start your frontend with the Logto configuration")
	fmt.Println("4. Login with the admin credentials provided above")
	fmt.Println("5. Use 'sync sync' to update RBAC configuration when needed")

	// Configuration sync reminder
	fmt.Println("\n🔄 CONFIGURATION UPDATES")
	fmt.Println("To update roles and permissions after this initial setup:")

	// Get the executable path used to run this command
	execPath := os.Args[0]
	fmt.Printf("  %s sync -c configs/config.yml --dry-run  # Preview changes\n", execPath)
	fmt.Printf("  %s sync -c configs/config.yml            # Apply changes\n", execPath)

	fmt.Println("\n" + strings.Repeat("=", 80))
}

func validateAndGetConfig() (*InitConfig, error) {
	// Check if any CLI flags are provided
	hasCLIFlags := initTenantID != "" || initBackendClientID != "" || initBackendClientSecret != "" || initDomain != ""

	if hasCLIFlags {
		// CLI mode - all flags must be provided
		if initTenantID == "" || initBackendClientID == "" || initBackendClientSecret == "" || initDomain == "" {
			return nil, fmt.Errorf("when using CLI flags, all must be provided:\n" +
				"  --tenant-id, --backend-client-id, --backend-client-secret, --domain\n" +
				"Or use environment variables: TENANT_ID, BACKEND_CLIENT_ID, BACKEND_CLIENT_SECRET, TENANT_DOMAIN")
		}

		logger.Info("Using CLI mode")
		return &InitConfig{
			TenantID:            initTenantID,
			TenantDomain:        initDomain,
			BackendClientID:     initBackendClientID,
			BackendClientSecret: initBackendClientSecret,
			Mode:                "cli",
		}, nil
	}

	// Environment mode - check all required env vars
	tenantID := os.Getenv("TENANT_ID")
	backendClientID := os.Getenv("BACKEND_CLIENT_ID")
	backendClientSecret := os.Getenv("BACKEND_CLIENT_SECRET")
	tenantDomain := os.Getenv("TENANT_DOMAIN")

	if tenantID == "" || backendClientID == "" || backendClientSecret == "" || tenantDomain == "" {
		return nil, fmt.Errorf("required environment variables missing:\n" +
			"  TENANT_ID, BACKEND_CLIENT_ID, BACKEND_CLIENT_SECRET, TENANT_DOMAIN\n" +
			"Or use CLI flags: --tenant-id, --backend-client-id, --backend-client-secret, --domain")
	}

	logger.Info("Using environment variables mode")
	return &InitConfig{
		TenantID:            tenantID,
		TenantDomain:        tenantDomain,
		BackendClientID:     backendClientID,
		BackendClientSecret: backendClientSecret,
		Mode:                "env",
	}, nil
}
