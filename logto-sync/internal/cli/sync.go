/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nethesis/my/logto-sync/internal/client"
	"github.com/nethesis/my/logto-sync/internal/config"
	"github.com/nethesis/my/logto-sync/internal/logger"
	"github.com/nethesis/my/logto-sync/internal/sync"
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
  logto-sync sync -c hierarchy.yml
  logto-sync sync --dry-run --verbose
  logto-sync sync --output json`,
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
	viper.BindPFlag("skip-resources", syncCmd.Flags().Lookup("skip-resources"))
	viper.BindPFlag("skip-roles", syncCmd.Flags().Lookup("skip-roles"))
	viper.BindPFlag("skip-permissions", syncCmd.Flags().Lookup("skip-permissions"))
	viper.BindPFlag("force", syncCmd.Flags().Lookup("force"))
	viper.BindPFlag("cleanup", syncCmd.Flags().Lookup("cleanup"))
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

	// Validate configuration
	if err := cfg.Validate(); err != nil && !viper.GetBool("force") {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Create Logto client
	logtoClient := client.NewLogtoClient(
		os.Getenv("LOGTO_BASE_URL"),
		os.Getenv("LOGTO_CLIENT_ID"),
		os.Getenv("LOGTO_CLIENT_SECRET"),
	)

	// Test connection
	logger.Info("Testing connection to Logto...")
	if err := logtoClient.TestConnection(); err != nil {
		return fmt.Errorf("failed to connect to Logto: %w", err)
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
		apiBaseURL = "https://dev.my.nethesis.it"
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
