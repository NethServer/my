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
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
	"github.com/nethesis/my/sync/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize configuration with Logto",
	Long: `Synchronize RBAC configuration from YAML file to Logto.

This command will:
1. Load the configuration from the specified YAML file
2. Connect to Logto using environment variables
3. Synchronize resources, roles, and permissions
4. Report any changes made

Examples:
  sync sync -c config.yml
  sync sync --dry-run --verbose
  sync sync --output json`,
	RunE: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)

	// Sync-specific flags
	syncCmd.Flags().Bool("skip-resources", false, "skip synchronizing resources")
	syncCmd.Flags().Bool("skip-roles", false, "skip synchronizing roles")
	syncCmd.Flags().Bool("skip-permissions", false, "skip synchronizing permissions")
	syncCmd.Flags().Bool("force", false, "force synchronization even if validation fails")
	syncCmd.Flags().Bool("cleanup", false, "remove resources/roles/scopes not defined in config (DANGEROUS)")

	// Bind flags to viper
	_ = viper.BindPFlag("skip-resources", syncCmd.Flags().Lookup("skip-resources"))
	_ = viper.BindPFlag("skip-roles", syncCmd.Flags().Lookup("skip-roles"))
	_ = viper.BindPFlag("skip-permissions", syncCmd.Flags().Lookup("skip-permissions"))
	_ = viper.BindPFlag("force", syncCmd.Flags().Lookup("force"))
	_ = viper.BindPFlag("cleanup", syncCmd.Flags().Lookup("cleanup"))
}

func runSync(cmd *cobra.Command, args []string) error {
	// Validate environment variables
	if err := validateEnvironment(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	// Load configuration
	configFile := cfgFile
	if configFile == "" {
		configFile = viper.ConfigFileUsed()
	}

	if configFile == "" {
		return fmt.Errorf("no configuration file specified or found")
	}

	logger.Info("Loading configuration from: %s", configFile)
	cfg, err := config.LoadFromFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Log configuration loading with structured data
	resourceCount := len(cfg.Hierarchy.Resources)
	roleCount := len(cfg.Hierarchy.UserRoles) + len(cfg.Hierarchy.OrganizationRoles)

	// Validate configuration
	if err := cfg.Validate(); err != nil && !viper.GetBool("force") {
		logger.LogConfigLoad(configFile, resourceCount, roleCount, false)
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	logger.LogConfigLoad(configFile, resourceCount, roleCount, true)

	// Create Logto client using derived base URL
	tenantID := os.Getenv("TENANT_ID")
	baseURL := fmt.Sprintf("https://%s.logto.app", tenantID)

	logtoClient := client.NewLogtoClient(
		baseURL,
		os.Getenv("BACKEND_CLIENT_ID"),
		os.Getenv("BACKEND_CLIENT_SECRET"),
	)

	// Test connection
	logger.Info("Testing connection to Logto...")
	if err := logtoClient.TestConnection(); err != nil {
		return fmt.Errorf("failed to connect to Logto: %w", err)
	}

	// Check if Logto is properly initialized for Nethesis Operation Center
	if initialized, err := checkLogtoInitialization(logtoClient); err != nil {
		logger.Warn("Could not check initialization status: %v", err)
	} else if !initialized {
		logger.Warn("Logto does not appear to be initialized for Nethesis Operation Center")
		logger.Info("Run 'sync init' first to set up applications and users")
		return fmt.Errorf("initialization required - run 'sync init' first")
	}

	// Create sync engine
	syncEngine := sync.NewEngine(logtoClient, &sync.Options{
		DryRun:          viper.GetBool("dry-run"),
		Verbose:         viper.GetBool("verbose"),
		SkipResources:   viper.GetBool("skip-resources"),
		SkipRoles:       viper.GetBool("skip-roles"),
		SkipPermissions: viper.GetBool("skip-permissions"),
		Cleanup:         viper.GetBool("cleanup"),
		APIBaseURL:      getAPIBaseURL(),
	})

	// Run synchronization
	logger.Info("Starting synchronization...")
	result, err := syncEngine.Sync(cfg)
	if err != nil {
		return fmt.Errorf("synchronization failed: %w", err)
	}

	// Output results
	if err := outputResult(result); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	if viper.GetBool("dry-run") {
		logger.Info("Dry run completed - no changes were made")
	} else {
		logger.Info("Synchronization completed successfully")
	}

	return nil
}

func getAPIBaseURL() string {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}
	return apiBaseURL
}

func outputResult(result *sync.Result) error {
	format := viper.GetString("output")

	switch format {
	case "json":
		return result.OutputJSON(os.Stdout)
	case "yaml":
		return result.OutputYAML(os.Stdout)
	default:
		return result.OutputText(os.Stdout)
	}
}

func checkLogtoInitialization(client *client.LogtoClient) (bool, error) {
	// Check if backend and frontend applications exist
	backendClientID := os.Getenv("BACKEND_CLIENT_ID")

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
		if name, ok := app["name"].(string); ok && name == "frontend" {
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

	return backendExists && frontendExists && ownerExists, nil
}
