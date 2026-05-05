/*
 * Copyright (C) 2026 Nethesis S.r.l.
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

	"github.com/nethesis/my/sync/internal/cli/pushcmd"
	"github.com/nethesis/my/sync/internal/cli/syncmd"
	"github.com/nethesis/my/sync/internal/database"
	"github.com/nethesis/my/sync/internal/logger"
	"github.com/nethesis/my/sync/internal/sync"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "⬆️ Push local organizations and users to Logto",
	Long: `⬆️ Push local organizations and users to Logto

📋 WHAT THIS COMMAND DOES:
  📤 Fetch organizations and users from local database
  🔍 Compare with current Logto state
  ✨ Create missing entities in Logto
  🔑 Generate temporary passwords for new users
  📧 Optionally send welcome emails with temporary credentials

⚠️  REQUIREMENTS:
  🔧 Properly initialized Logto instance (run 'sync init' first)
  🔑 Valid environment variables (LOGTO_TENANT_ID, LOGTO_BACKEND_APP_ID, etc.)
  💾 Access to local database

📝 EXAMPLES:
  sync push                                    # ⬆️ Standard push
  sync push --dry-run --verbose               # 👀 Preview changes
  sync push --output json                     # 🤖 JSON output
  sync push --organizations-only              # 📊 Push organizations only
  sync push --users-only                      # 👥 Push users only
  sync push --send-email \                    # 📧 Send welcome emails
    --smtp-host smtp.example.com \
    --smtp-from noreply@example.com

📤 OUTPUT FORMATS:
  sync push --output text   # 📖 Human-readable output (default)
  sync push --output json   # 🤖 JSON output for automation
  sync push --output yaml   # 📋 YAML output for configuration

📧 EMAIL OPTIONS (flag takes precedence over env var):
  --send-email              # 📧 Send welcome email to new users (auto-enabled if SMTP_HOST is set)
  --smtp-host               # 🖥️  SMTP server hostname   (env: SMTP_HOST)
  --smtp-port               # 🔌 SMTP server port        (env: SMTP_PORT, default: 587)
  --smtp-user               # 👤 SMTP username           (env: SMTP_USERNAME)
  --smtp-password           # 🔑 SMTP password           (env: SMTP_PASSWORD)
  --smtp-from               # 📤 From address            (env: SMTP_FROM)
  --smtp-tls                # 🔒 Enable STARTTLS          (env: SMTP_TLS=true)
  --smtp-from-name          # 🏷️  Sender display name      (env: SMTP_FROM_NAME)
  --frontend-url            # 🌐 Login URL in email body  (env: APP_URL)
  --language                # 🌍 Email language: en, it   (env: LANGUAGE, default: en)

💡 TIP: Always use --dry-run first to preview changes before applying them.
💡 TIP: Temporary passwords are printed in the output even if email is not sent.`,
	RunE: runPush,
}

func init() {
	rootCmd.AddCommand(pushCmd)

	pushCmd.Flags().Bool("organizations-only", false, "push organizations only, skip users")
	pushCmd.Flags().Bool("users-only", false, "push users only, skip organizations")
	pushCmd.Flags().Bool("send-email", false, "send welcome email with temporary password to new users")
	pushCmd.Flags().String("smtp-host", "", "SMTP server hostname")
	pushCmd.Flags().Int("smtp-port", 587, "SMTP server port")
	pushCmd.Flags().String("smtp-user", "", "SMTP username")
	pushCmd.Flags().String("smtp-password", "", "SMTP password")
	pushCmd.Flags().String("smtp-from", "", "SMTP from address")
	pushCmd.Flags().Bool("smtp-tls", false, "enable STARTTLS for SMTP connection")
	pushCmd.Flags().String("smtp-from-name", "", "SMTP sender display name")
	pushCmd.Flags().String("frontend-url", "", "frontend URL included in welcome email")
	pushCmd.Flags().String("language", "", "email language: en or it (default: en)")

	_ = viper.BindPFlag("organizations-only", pushCmd.Flags().Lookup("organizations-only"))
	_ = viper.BindPFlag("users-only", pushCmd.Flags().Lookup("users-only"))
	_ = viper.BindPFlag("send-email", pushCmd.Flags().Lookup("send-email"))
	_ = viper.BindPFlag("smtp-host", pushCmd.Flags().Lookup("smtp-host"))
	_ = viper.BindPFlag("smtp-port", pushCmd.Flags().Lookup("smtp-port"))
	_ = viper.BindPFlag("smtp-user", pushCmd.Flags().Lookup("smtp-user"))
	_ = viper.BindPFlag("smtp-password", pushCmd.Flags().Lookup("smtp-password"))
	_ = viper.BindPFlag("smtp-tls", pushCmd.Flags().Lookup("smtp-tls"))
	_ = viper.BindPFlag("smtp-from", pushCmd.Flags().Lookup("smtp-from"))
	_ = viper.BindPFlag("smtp-from-name", pushCmd.Flags().Lookup("smtp-from-name"))
	_ = viper.BindPFlag("frontend-url", pushCmd.Flags().Lookup("frontend-url"))
	_ = viper.BindPFlag("language", pushCmd.Flags().Lookup("language"))
}

func runPush(cmd *cobra.Command, args []string) error {
	envFileRef := ".env"
	if envFile != "" {
		envFileRef = envFile
	}
	logger.Info("Using environment file: %s", envFileRef)

	if err := validateEnvironment(); err != nil {
		return fmt.Errorf("environment validation failed: %w", err)
	}

	if tenantID := os.Getenv("LOGTO_TENANT_ID"); tenantID != "" {
		logger.Info("Using tenant ID: %s", tenantID)
	}
	if tenantDomain := os.Getenv("LOGTO_TENANT_DOMAIN"); tenantDomain != "" {
		logger.Info("Using tenant domain: %s", tenantDomain)
	}

	if err := pushcmd.ValidatePushFlags(); err != nil {
		return err
	}

	logtoClient, err := syncmd.CreateLogtoClient()
	if err != nil {
		return err
	}

	if err := syncmd.ValidateInitialization(logtoClient); err != nil {
		return err
	}

	if err := database.Init(); err != nil {
		return fmt.Errorf("failed to initialize database connection: %w", err)
	}
	defer func() {
		_ = database.Close()
	}()

	// All SMTP options fall back to env vars (flag > env var)
	smtpHost := viper.GetString("smtp-host")
	if smtpHost == "" {
		smtpHost = os.Getenv("SMTP_HOST")
	}
	smtpPort := viper.GetInt("smtp-port")
	if smtpPort == 587 {
		if v := os.Getenv("SMTP_PORT"); v != "" {
			if _, err := fmt.Sscanf(v, "%d", &smtpPort); err != nil {
				smtpPort = 587
			}
		}
	}
	smtpUser := viper.GetString("smtp-user")
	if smtpUser == "" {
		smtpUser = os.Getenv("SMTP_USERNAME")
	}
	smtpPassword := viper.GetString("smtp-password")
	if smtpPassword == "" {
		smtpPassword = os.Getenv("SMTP_PASSWORD")
	}
	smtpFrom := viper.GetString("smtp-from")
	if smtpFrom == "" {
		smtpFrom = os.Getenv("SMTP_FROM")
	}
	smtpFromName := viper.GetString("smtp-from-name")
	if smtpFromName == "" {
		smtpFromName = os.Getenv("SMTP_FROM_NAME")
	}
	frontendURL := viper.GetString("frontend-url")
	if frontendURL == "" {
		frontendURL = os.Getenv("APP_URL")
	}
	smtpTLS := viper.GetBool("smtp-tls")
	if !smtpTLS && os.Getenv("SMTP_TLS") == "true" {
		smtpTLS = true
	}
	sendEmail := viper.GetBool("send-email")
	if !sendEmail && os.Getenv("SMTP_HOST") != "" {
		sendEmail = true
	}
	language := viper.GetString("language")
	if language == "" {
		language = os.Getenv("LANGUAGE")
	}

	pushEngine := sync.NewPushEngine(logtoClient, &sync.PushOptions{
		DryRun:            viper.GetBool("dry-run"),
		Verbose:           viper.GetBool("verbose"),
		OrganizationsOnly: viper.GetBool("organizations-only"),
		UsersOnly:         viper.GetBool("users-only"),
		SendEmail:         sendEmail,
		SMTPHost:          smtpHost,
		SMTPPort:          smtpPort,
		SMTPUser:          smtpUser,
		SMTPPassword:      smtpPassword,
		SMTPFrom:          smtpFrom,
		SMTPFromName:      smtpFromName,
		SMTPUseTLS:        smtpTLS,
		FrontendURL:       frontendURL,
		APIBaseURL:        syncmd.GetAPIBaseURL(),
		Language:          language,
	})

	logger.Info("Starting push to Logto...")
	result, err := pushEngine.Push()
	if err != nil {
		return fmt.Errorf("push operation failed: %w", err)
	}

	if err := pushcmd.OutputResult(result); err != nil {
		return fmt.Errorf("failed to output results: %w", err)
	}

	if viper.GetBool("dry-run") {
		logger.Info("Dry run completed - no changes made")
	} else {
		logger.Info("Push operation completed successfully")
	}

	return nil
}
