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

	"github.com/nethesis/my/sync/internal/cli/syncmd"
	"github.com/nethesis/my/sync/internal/logger"
	"github.com/nethesis/my/sync/internal/sync"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "🔄 Synchronize RBAC configuration with Logto",
	Long: `🔄 Synchronize RBAC configuration from YAML file to Logto

📋 WHAT THIS COMMAND DOES:
  📖 Load the configuration from the specified YAML file
  🔗 Connect to Logto using environment variables
  🔄 Synchronize resources, roles, and permissions
  📊 Report any changes made

⚠️  REQUIREMENTS:
  🔧 Properly initialized Logto instance (run 'sync init' first)
  🔑 Valid environment variables (LOGTO_TENANT_ID, LOGTO_BACKEND_APP_ID, etc.)

📝 EXAMPLES:
  sync sync -c config.yml                   # 🔄 Standard sync
  sync sync --dry-run --verbose             # 👀 Preview changes
  sync sync --output json                   # 🤖 JSON output
  sync sync --force --cleanup               # ⚠️  Force sync with cleanup

📤 OUTPUT FORMATS:
  sync sync --output text   # 📖 Human-readable output (default)
  sync sync --output json   # 🤖 JSON output for automation
  sync sync --output yaml   # 📋 YAML output for configuration

🚨 DANGEROUS OPTIONS:
  --cleanup                 # 🗑️  Remove undefined resources (PERMANENT!)
  --force                   # ⚡ Skip validation checks

💡 TIP: Always use --dry-run first to preview changes before applying them.`,
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
	// Log environment file being used
	envFileRef := ".env"
	if envFile != "" {
		envFileRef = envFile
	}
	logger.Info("Using environment file: %s", envFileRef)

	// Validate environment variables
	if err := validateEnvironment(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	// Log tenant information for consistency with init command
	if tenantID := os.Getenv("LOGTO_TENANT_ID"); tenantID != "" {
		logger.Info("Using tenant ID: %s", tenantID)
	}
	if tenantDomain := os.Getenv("LOGTO_TENANT_DOMAIN"); tenantDomain != "" {
		logger.Info("Using tenant domain: %s", tenantDomain)
	}

	// Load and validate configuration
	cfg, err := syncmd.LoadAndValidateConfig(cfgFile)
	if err != nil {
		return err
	}

	// Create and test Logto client
	logtoClient, err := syncmd.CreateLogtoClient()
	if err != nil {
		return err
	}

	// Validate Logto initialization
	if err := syncmd.ValidateInitialization(logtoClient); err != nil {
		return err
	}

	// Create sync engine
	syncEngine := sync.NewEngine(logtoClient, &sync.Options{
		DryRun:          viper.GetBool("dry-run"),
		Verbose:         viper.GetBool("verbose"),
		SkipResources:   viper.GetBool("skip-resources"),
		SkipRoles:       viper.GetBool("skip-roles"),
		SkipPermissions: viper.GetBool("skip-permissions"),
		Cleanup:         viper.GetBool("cleanup"),
		APIBaseURL:      syncmd.GetAPIBaseURL(),
		ConfigFile:      cfgFile,
	})

	// Run synchronization
	logger.Info("Starting synchronization...")
	result, err := syncEngine.Sync(cfg)
	if err != nil {
		return fmt.Errorf("synchronization failed: %w", err)
	}

	// Output results
	if err := syncmd.OutputResult(result); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	if viper.GetBool("dry-run") {
		logger.Info("Dry run completed - no changes made")
	} else {
		logger.Info("Synchronization completed successfully")
	}

	return nil
}
