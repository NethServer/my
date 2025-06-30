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

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/nethesis/my/sync/internal/logger"
	"github.com/nethesis/my/sync/pkg/version"
)

var (
	cfgFile      string
	verbose      bool
	dryRun       bool
	outputFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sync",
	Short: "Synchronize RBAC configuration with Logto",
	Long: `sync is a CLI tool that synchronizes role-based access control (RBAC)
configuration from YAML files to Logto identity provider.

It supports:
- Organization roles and scopes
- User roles and permissions
- Resources and scopes
- Hierarchical permission management`,
	Version: version.Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.yml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "show what would be done without making changes")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text, json, yaml)")

	// Bind flags to viper
	_ = viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
	_ = viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	_ = viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load .env file if exists
	_ = godotenv.Load()

	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in current directory
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Read in environment variables that match
	viper.AutomaticEnv()

	// Initialize logger
	err := logger.InitFromEnv("sync-tool")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize logger")
	}

	// Set log level based on flags and environment
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		if viper.GetBool("verbose") {
			logLevel = "debug"
		} else {
			logLevel = "info"
		}
	}
	logger.SetLevel(logLevel)

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil && verbose {
		logger.Info("Using config file: %s", viper.ConfigFileUsed())
	}
}

func validateEnvironment() error {
	required := []string{
		"TENANT_ID",
		"BACKEND_CLIENT_ID",
		"BACKEND_CLIENT_SECRET",
	}

	for _, env := range required {
		if os.Getenv(env) == "" {
			return fmt.Errorf("required environment variable %s is not set", env)
		}
	}

	return nil
}
