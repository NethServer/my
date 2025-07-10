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
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// InitResult represents the result of the init command
type InitResult struct {
	BackendApp      Application `json:"backend_app"`
	FrontendApp     Application `json:"frontend_app"`
	OwnerUser       User        `json:"owner_user"`
	CustomDomain    string      `json:"custom_domain"`
	GeneratedSecret string      `json:"generated_jwt_secret"`
	AlreadyInit     bool        `json:"already_initialized"`
	TenantInfo      TenantInfo  `json:"tenant_info"`
	NextSteps       []string    `json:"next_steps"`
}

// TenantInfo represents tenant information
type TenantInfo struct {
	TenantID string `json:"tenant_id"`
	BaseURL  string `json:"base_url"`
	Mode     string `json:"mode"` // "env" or "cli"
}

// OutputSetupInstructions outputs setup instructions in the requested format
func OutputSetupInstructions(result *InitResult) error {
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
		fmt.Println("⚠️  Note: Some components are initialized (use --force to recreate)")
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
	fmt.Printf("\n# Custom JWT Configuration\n")
	fmt.Printf("JWT_SECRET=%v\n", backendEnv["JWT_SECRET"])
	fmt.Printf("JWT_ISSUER=%v\n", backendEnv["JWT_ISSUER"])
	fmt.Printf("JWT_EXPIRATION=%v\n", backendEnv["JWT_EXPIRATION"])
	fmt.Printf("JWT_REFRESH_EXPIRATION=%v\n", backendEnv["JWT_REFRESH_EXPIRATION"])
	fmt.Printf("\n# Logto Management API (from your M2M app)\n")
	fmt.Printf("BACKEND_APP_ID=%v\n", backendEnv["BACKEND_APP_ID"])
	fmt.Printf("BACKEND_APP_SECRET=%v\n", backendEnv["BACKEND_APP_SECRET"])
	fmt.Printf("LOGTO_MANAGEMENT_BASE_URL=%v\n", backendEnv["LOGTO_MANAGEMENT_BASE_URL"])
	fmt.Printf("\n# Redis Configuration (Required)\n")
	fmt.Printf("REDIS_URL=%v\n", "redis://localhost:6379")
	fmt.Printf("REDIS_DB=%v\n", "0")
	fmt.Printf("REDIS_PASSWORD=%v\n", "")
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
	fmt.Printf("Username: %s\n", result.OwnerUser.Username)
	fmt.Printf("Email:    %s\n", result.OwnerUser.Email)
	fmt.Printf("Password: %s\n", result.OwnerUser.Password)
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
