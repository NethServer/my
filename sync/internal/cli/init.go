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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nethesis/my/sync/internal/cli/initcmd"
	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

var (
	initForce            bool
	initDomain           string
	initTenantID         string
	initBackendAppID     string
	initBackendAppSecret string
	// Owner user configuration
	initOwnerUsername    string
	initOwnerEmail       string
	initOwnerDisplayName string
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "🚀 Initialize Logto configuration with complete setup",
	Long: `🚀 Initialize Logto with complete configuration for Operation Center

📋 WHAT THIS COMMAND DOES:
  🌐 Create custom domain in Logto (e.g., your-domain.com)
  🔧 Create backend and frontend applications in Logto
  👤 Create an owner account with secure credentials
  🔐 Synchronize basic RBAC configuration  
  📄 Output environment variables and setup instructions

⚠️  REQUIREMENTS:
  🔑 A Machine-to-Machine application in Logto with Management API access

📝 USAGE MODES:

🔤 Mode 1 - Environment Variables:
  TENANT_ID=your-tenant-id
  BACKEND_APP_ID=your-backend-app-id
  BACKEND_APP_SECRET=your-secret
  TENANT_DOMAIN=your-domain.com

  sync init

🚩 Mode 2 - CLI Flags:
  sync init \
    --tenant-id your-tenant-id \
    --backend-app-id your-backend-app-id \
    --backend-app-secret your-secret \
    --domain your-domain.com \
    --owner-username owner \
    --owner-email owner@example.com \
    --owner-name "Company Owner"

📤 OUTPUT FORMATS:
  sync init --output json   # 🤖 JSON output for automation/CI-CD
  sync init --output yaml   # 📋 YAML output for configuration
  sync init --output text   # 📖 Human-readable output (default)

💡 NOTE: CLI flags take precedence over environment variables. 
    If any CLI flag is provided, all required flags must be provided.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Init-specific flags
	initCmd.Flags().BoolVar(&initForce, "force", false, "force re-initialization even if already done")
	initCmd.Flags().StringVar(&initDomain, "domain", "", "tenant domain (e.g., your-domain.com)")
	initCmd.Flags().StringVar(&initTenantID, "tenant-id", "", "Logto tenant ID (e.g., your-tenant-id)")
	initCmd.Flags().StringVar(&initBackendAppID, "backend-app-id", "", "backend M2M application ID")
	initCmd.Flags().StringVar(&initBackendAppSecret, "backend-app-secret", "", "backend M2M application secret")
	// Owner user configuration flags
	initCmd.Flags().StringVar(&initOwnerUsername, "owner-username", "owner", "Owner user username")
	initCmd.Flags().StringVar(&initOwnerEmail, "owner-email", "owner@example.com", "Owner user email")
	initCmd.Flags().StringVar(&initOwnerDisplayName, "owner-name", "Company Owner", "Owner user display name")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine configuration mode and validate parameters
	config, err := initcmd.ValidateAndGetConfig(initTenantID, initBackendAppID, initBackendAppSecret, initDomain)
	if err != nil {
		return err
	}

	logger.Info("Using tenant domain: %s", config.TenantDomain)
	logger.Info("Using tenant ID: %s", config.TenantID)

	// Create Logto client using configuration
	baseURL := fmt.Sprintf("https://%s.logto.app", config.TenantID)

	logtoClient := client.NewLogtoClient(
		baseURL,
		config.BackendAppID,
		config.BackendAppSecret,
	)

	// Test connection
	logger.Info("Testing connection to Logto...")
	if err := logtoClient.TestConnection(); err != nil {
		return fmt.Errorf("failed to connect to Logto: %w", err)
	}

	// Check if already initialized
	alreadyInit, err := initcmd.CheckIfAlreadyInitialized(logtoClient, config)
	if err != nil {
		return fmt.Errorf("failed to check initialization status: %w", err)
	}

	if alreadyInit && !initForce {
		logger.Info("Logto appears to already be initialized for Operation Center")
		logger.Info("Use 'sync sync' to synchronize configuration changes")
		logger.Info("Use 'sync init --force' to re-run initialization")
		return nil
	}

	result := &initcmd.InitResult{
		AlreadyInit:  alreadyInit,
		CustomDomain: config.TenantDomain,
		TenantInfo: initcmd.TenantInfo{
			TenantID: config.TenantID,
			BaseURL:  fmt.Sprintf("https://%s.logto.app", config.TenantID),
			Mode:     config.Mode,
		},
		NextSteps: []string{
			"Copy the environment variables to your .env files",
			"Start your backend: cd backend && go run main.go",
			"Start your frontend with the Logto configuration",
			"Login with the owner credentials provided",
			"Use 'sync sync' to update RBAC configuration when needed",
		},
		EnvFile: envFile,
	}

	logger.Info("Starting Logto initialization...")

	// Step 1: Create custom domain
	if err := initcmd.CreateCustomDomain(logtoClient, config); err != nil {
		return fmt.Errorf("failed to create custom domain: %w", err)
	}

	// Step 2: Create applications
	backendApp, frontendApp, err := initcmd.CreateApplications(logtoClient, config)
	if err != nil {
		return fmt.Errorf("failed to create applications: %w", err)
	}
	result.BackendApp = *backendApp
	result.FrontendApp = *frontendApp

	// Step 3: Derive environment variables and populate applications
	if err := initcmd.DeriveEnvironmentVariables(config, backendApp, frontendApp); err != nil {
		return fmt.Errorf("failed to derive environment variables: %w", err)
	}
	// Update result with populated environment variables
	result.BackendApp = *backendApp
	result.FrontendApp = *frontendApp

	// Step 4: Create owner user
	ownerUser, err := initcmd.CreateOwnerUser(logtoClient, initOwnerUsername, initOwnerEmail, initOwnerDisplayName)
	if err != nil {
		return fmt.Errorf("failed to create owner user: %w", err)
	}
	result.OwnerUser = *ownerUser

	// Step 5: Generate JWT secret
	result.GeneratedSecret = initcmd.GenerateJWTSecret()
	if result.BackendApp.EnvironmentVars != nil {
		result.BackendApp.EnvironmentVars["JWT_SECRET"] = result.GeneratedSecret
	}

	// Step 6: Sync basic RBAC configuration
	if err := initcmd.SyncBasicConfiguration(logtoClient, initOwnerUsername); err != nil {
		return fmt.Errorf("failed to sync basic configuration: %w", err)
	}

	// Step 7: Output setup instructions
	if err := initcmd.OutputSetupInstructions(result); err != nil {
		return fmt.Errorf("failed to output setup instructions: %w", err)
	}

	logger.Info("Logto initialization completed successfully!")
	return nil
}
