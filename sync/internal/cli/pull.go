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

	"github.com/nethesis/my/sync/internal/cli/pullcmd"
	"github.com/nethesis/my/sync/internal/cli/syncmd"
	"github.com/nethesis/my/sync/internal/database"
	"github.com/nethesis/my/sync/internal/logger"
	"github.com/nethesis/my/sync/internal/sync"
)

var pullCmd = &cobra.Command{
	Use:   "pull",
	Short: "⬇️ Pull RBAC configuration from Logto to local database",
	Long: `⬇️ Pull RBAC configuration from Logto to local database

📋 WHAT THIS COMMAND DOES:
  📥 Fetch current configuration from Logto
  🔍 Compare with local database state
  💾 Update local database with Logto configuration
  📊 Report changes made to local data

⚠️  REQUIREMENTS:
  🔧 Properly initialized Logto instance (run 'sync init' first)
  🔑 Valid environment variables (TENANT_ID, BACKEND_APP_ID, etc.)
  💾 Access to local database

📝 EXAMPLES:
  sync pull                                 # ⬇️ Standard pull
  sync pull --dry-run --verbose            # 👀 Preview changes
  sync pull --output json                  # 🤖 JSON output
  sync pull --organizations-only           # 📊 Pull organizations only
  sync pull --users-only                   # 👥 Pull users only
  sync pull --conflict-strategy overwrite  # ⚠️ Overwrite conflicts

📤 OUTPUT FORMATS:
  sync pull --output text   # 📖 Human-readable output (default)
  sync pull --output json   # 🤖 JSON output for automation
  sync pull --output yaml   # 📋 YAML output for configuration

🔄 CONFLICT STRATEGIES:
  --conflict-strategy skip      # 🚫 Skip conflicting records (default)
  --conflict-strategy overwrite # ⚠️ Overwrite local with Logto data
  --conflict-strategy merge     # 🔀 Merge changes where possible

⚠️ DANGEROUS OPTIONS:
  --overwrite-all              # 🗑️ Overwrite all local data (PERMANENT!)
  --force                      # ⚡ Skip validation checks

💡 TIP: Always use --dry-run first to preview changes before applying them.`,
	RunE: runPull,
}

func init() {
	rootCmd.AddCommand(pullCmd)

	// Pull-specific flags
	pullCmd.Flags().Bool("organizations-only", false, "pull organizations and organization roles only")
	pullCmd.Flags().Bool("users-only", false, "pull users and user roles only")
	pullCmd.Flags().Bool("resources-only", false, "pull resources and permissions only")
	pullCmd.Flags().StringVar(&pullConflictStrategy, "conflict-strategy", "skip", "conflict resolution strategy (skip, overwrite, merge)")
	pullCmd.Flags().Bool("overwrite-all", false, "overwrite all local data with Logto configuration (DANGEROUS)")
	pullCmd.Flags().Bool("force", false, "force pull even if validation fails")

	// Bind flags to viper
	_ = viper.BindPFlag("organizations-only", pullCmd.Flags().Lookup("organizations-only"))
	_ = viper.BindPFlag("users-only", pullCmd.Flags().Lookup("users-only"))
	_ = viper.BindPFlag("resources-only", pullCmd.Flags().Lookup("resources-only"))
	_ = viper.BindPFlag("conflict-strategy", pullCmd.Flags().Lookup("conflict-strategy"))
	_ = viper.BindPFlag("overwrite-all", pullCmd.Flags().Lookup("overwrite-all"))
	_ = viper.BindPFlag("force", pullCmd.Flags().Lookup("force"))
}

var pullConflictStrategy string

func runPull(cmd *cobra.Command, args []string) error {
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

	// Log tenant information for consistency with other commands
	if tenantID := os.Getenv("TENANT_ID"); tenantID != "" {
		logger.Info("Using tenant ID: %s", tenantID)
	}
	if tenantDomain := os.Getenv("TENANT_DOMAIN"); tenantDomain != "" {
		logger.Info("Using tenant domain: %s", tenantDomain)
	}

	// Validate pull-specific flags
	if err := pullcmd.ValidatePullFlags(); err != nil {
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

	// Initialize database connection
	if err := database.Init(); err != nil {
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	// Create pull engine
	pullEngine := sync.NewPullEngine(logtoClient, &sync.PullOptions{
		DryRun:            viper.GetBool("dry-run"),
		Verbose:           viper.GetBool("verbose"),
		OrganizationsOnly: viper.GetBool("organizations-only"),
		UsersOnly:         viper.GetBool("users-only"),
		ResourcesOnly:     viper.GetBool("resources-only"),
		ConflictStrategy:  viper.GetString("conflict-strategy"),
		OverwriteAll:      viper.GetBool("overwrite-all"),
		Force:             viper.GetBool("force"),
		APIBaseURL:        syncmd.GetAPIBaseURL(),
	})

	// Run pull operation
	logger.Info("Starting pull from Logto...")
	result, err := pullEngine.Pull()
	if err != nil {
		return fmt.Errorf("pull operation failed: %w", err)
	}

	// Output results
	if err := pullcmd.OutputResult(result); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	if viper.GetBool("dry-run") {
		logger.Info("Dry run completed - no changes made")
	} else {
		logger.Info("Pull operation completed successfully")
	}

	return nil
}
