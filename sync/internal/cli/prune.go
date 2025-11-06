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
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "🧹 Clean up all organizations and users from Logto",
	Long: `🧹 Remove all organizations and users from Logto tenant

This command will delete all organizations and users, with special handling for:
  • Owner organization: prompts for confirmation before deletion
  • owner user: prompts for confirmation before deletion

⚠️  WARNING: This operation is IRREVERSIBLE!
    All deleted data cannot be recovered.

💡 USAGE:
  sync prune                    # Interactive mode with confirmations
  sync prune --force            # Skip all confirmations (DANGEROUS)
  sync prune --dry-run          # Preview what would be deleted

After pruning, if Owner/owner are deleted, run 'sync init' to reinitialize.`,
	RunE: runPrune,
}

var pruneForce bool

func init() {
	rootCmd.AddCommand(pruneCmd)
	pruneCmd.Flags().BoolVar(&pruneForce, "force", false, "skip all confirmations (DANGEROUS)")
}

func runPrune(cmd *cobra.Command, args []string) error {
	logger.Info("Starting prune operation...")

	// Get credentials from environment
	tenantID := os.Getenv("TENANT_ID")
	clientID := os.Getenv("BACKEND_APP_ID")
	clientSecret := os.Getenv("BACKEND_APP_SECRET")

	if tenantID == "" || clientID == "" || clientSecret == "" {
		return fmt.Errorf("missing required environment variables: TENANT_ID, BACKEND_APP_ID, BACKEND_APP_SECRET")
	}

	// Create Logto client
	baseURL := fmt.Sprintf("https://%s.logto.app", tenantID)
	logtoClient := client.NewLogtoClient(baseURL, clientID, clientSecret)

	// Test connection
	logger.Info("Testing connection to Logto...")
	if err := logtoClient.TestConnection(); err != nil {
		return fmt.Errorf("failed to connect to Logto: %w", err)
	}

	if !dryRun && !pruneForce {
		fmt.Println("\n⚠️  WARNING: This will delete ALL organizations and users!")
		fmt.Print("Type 'yes' to continue: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "yes" {
			logger.Info("Prune operation cancelled by user")
			return nil
		}
	}

	// Get all organizations (across all pages)
	logger.Info("Fetching all organizations...")
	orgs, err := logtoClient.GetAllOrganizations()
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}

	// Get all users (across all pages)
	logger.Info("Fetching all users...")
	users, err := logtoClient.GetAllUsers()
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	logger.Info("Found %d organizations and %d users", len(orgs), len(users))

	var deletedOrgs, skippedOrgs int
	var deletedUsers, skippedUsers int
	var ownerOrgDeleted, ownerUserDeleted bool

	// Process organizations
	logger.Info("\nProcessing organizations...")
	for _, org := range orgs {
		if org.Name == "Owner" {
			if dryRun {
				logger.Info("Would ask to delete Owner organization: %s (ID: %s)", org.Name, org.ID)
				skippedOrgs++
			} else if pruneForce {
				logger.Warn("Force mode: deleting Owner organization...")
				if err := logtoClient.DeleteOrganization(org.ID); err != nil {
					logger.Error("Failed to delete Owner organization: %v", err)
					skippedOrgs++
					continue
				}
				logger.Info("✓ Deleted Owner organization (ID: %s)", org.ID)
				deletedOrgs++
				ownerOrgDeleted = true
			} else {
				fmt.Printf("\n❓ Delete Owner organization (ID: %s)? [y/N]: ", org.ID)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response == "y" || response == "yes" {
					if err := logtoClient.DeleteOrganization(org.ID); err != nil {
						logger.Error("Failed to delete Owner organization: %v", err)
						skippedOrgs++
						continue
					}
					logger.Info("✓ Deleted Owner organization (ID: %s)", org.ID)
					deletedOrgs++
					ownerOrgDeleted = true
				} else {
					logger.Info("⊘ Skipped Owner organization")
					skippedOrgs++
				}
			}
		} else {
			if dryRun {
				logger.Info("Would delete organization: %s (ID: %s)", org.Name, org.ID)
			} else {
				if err := logtoClient.DeleteOrganization(org.ID); err != nil {
					logger.Error("Failed to delete organization %s: %v", org.Name, err)
					skippedOrgs++
					continue
				}
				logger.Info("✓ Deleted organization: %s (ID: %s)", org.Name, org.ID)
			}
			deletedOrgs++
		}
	}

	// Process users
	logger.Info("\nProcessing users...")
	for _, user := range users {
		username, _ := user["username"].(string)
		userID, _ := user["id"].(string)

		if username == "owner" {
			if dryRun {
				logger.Info("Would ask to delete owner user: %s (ID: %s)", username, userID)
				skippedUsers++
			} else if pruneForce {
				logger.Warn("Force mode: deleting owner user...")
				if err := logtoClient.DeleteUser(userID); err != nil {
					logger.Error("Failed to delete owner user: %v", err)
					skippedUsers++
					continue
				}
				logger.Info("✓ Deleted owner user (ID: %s)", userID)
				deletedUsers++
				ownerUserDeleted = true
			} else {
				fmt.Printf("\n❓ Delete owner user (ID: %s)? [y/N]: ", userID)
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.TrimSpace(strings.ToLower(response))
				if response == "y" || response == "yes" {
					if err := logtoClient.DeleteUser(userID); err != nil {
						logger.Error("Failed to delete owner user: %v", err)
						skippedUsers++
						continue
					}
					logger.Info("✓ Deleted owner user (ID: %s)", userID)
					deletedUsers++
					ownerUserDeleted = true
				} else {
					logger.Info("⊘ Skipped owner user")
					skippedUsers++
				}
			}
		} else {
			if dryRun {
				logger.Info("Would delete user: %s (ID: %s)", username, userID)
			} else {
				if err := logtoClient.DeleteUser(userID); err != nil {
					logger.Error("Failed to delete user %s: %v", username, err)
					skippedUsers++
					continue
				}
				logger.Info("✓ Deleted user: %s (ID: %s)", username, userID)
			}
			deletedUsers++
		}
	}

	// Summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🧹 PRUNE SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	if dryRun {
		fmt.Println("Mode: DRY RUN (no changes made)")
	}
	fmt.Printf("Organizations: %d deleted, %d skipped\n", deletedOrgs, skippedOrgs)
	fmt.Printf("Users:         %d deleted, %d skipped\n", deletedUsers, skippedUsers)
	fmt.Println(strings.Repeat("=", 60))

	// Suggest re-initialization if needed
	if ownerOrgDeleted || ownerUserDeleted {
		fmt.Println("\n💡 NEXT STEPS:")
		fmt.Println("The Owner organization or owner user was deleted.")
		fmt.Println("Run 'sync init' to reinitialize your Logto tenant.")
	}

	// Database cleanup
	if err := cleanupDatabase(dryRun, pruneForce); err != nil {
		logger.Error("Failed to cleanup database: %v", err)
		// Don't fail the whole command, just warn
		fmt.Println("\n⚠️  Database cleanup failed. You may need to clean it manually.")
	}

	logger.Info("Prune operation completed successfully")
	return nil
}

func cleanupDatabase(dryRun, force bool) error {
	logger.Info("\n🗄️  Checking local database...")

	// Get database URL from environment
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Warn("DATABASE_URL not set, skipping database cleanup")
		return nil
	}

	// Connect to database
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection: %v", err)
		}
	}()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	// Query counts for each table
	var distributorsCount, resellersCount, customersCount, usersCount int

	err = db.QueryRow("SELECT COUNT(*) FROM distributors WHERE deleted_at IS NULL").Scan(&distributorsCount)
	if err != nil {
		return fmt.Errorf("failed to count distributors: %w", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM resellers WHERE deleted_at IS NULL").Scan(&resellersCount)
	if err != nil {
		return fmt.Errorf("failed to count resellers: %w", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM customers WHERE deleted_at IS NULL").Scan(&customersCount)
	if err != nil {
		return fmt.Errorf("failed to count customers: %w", err)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&usersCount)
	if err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	totalRecords := distributorsCount + resellersCount + customersCount + usersCount

	// Display counts
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🗄️  DATABASE RECORDS")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Distributors: %d\n", distributorsCount)
	fmt.Printf("Resellers:    %d\n", resellersCount)
	fmt.Printf("Customers:    %d\n", customersCount)
	fmt.Printf("Users:        %d\n", usersCount)
	fmt.Printf("Total:        %d\n", totalRecords)
	fmt.Println(strings.Repeat("=", 60))

	if totalRecords == 0 {
		logger.Info("No database records to clean")
		return nil
	}

	// Ask for confirmation if not in dry-run or force mode
	if !dryRun && !force {
		fmt.Println("\n⚠️  WARNING: This will PERMANENTLY DELETE ALL records in the database!")
		fmt.Print("Delete all database records? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			logger.Info("Database cleanup cancelled by user")
			return nil
		}
	}

	if dryRun {
		logger.Info("DRY RUN: Would delete all database records")
		return nil
	}

	// Execute deletes
	logger.Info("Deleting database records...")

	var deletedDistributors, deletedResellers, deletedCustomers, deletedUsers int64

	if distributorsCount > 0 {
		result, err := db.Exec("DELETE FROM distributors WHERE deleted_at IS NULL")
		if err != nil {
			logger.Error("Failed to delete distributors: %v", err)
		} else {
			deletedDistributors, _ = result.RowsAffected()
			logger.Info("✓ Deleted %d distributors", deletedDistributors)
		}
	}

	if resellersCount > 0 {
		result, err := db.Exec("DELETE FROM resellers WHERE deleted_at IS NULL")
		if err != nil {
			logger.Error("Failed to delete resellers: %v", err)
		} else {
			deletedResellers, _ = result.RowsAffected()
			logger.Info("✓ Deleted %d resellers", deletedResellers)
		}
	}

	if customersCount > 0 {
		result, err := db.Exec("DELETE FROM customers WHERE deleted_at IS NULL")
		if err != nil {
			logger.Error("Failed to delete customers: %v", err)
		} else {
			deletedCustomers, _ = result.RowsAffected()
			logger.Info("✓ Deleted %d customers", deletedCustomers)
		}
	}

	if usersCount > 0 {
		result, err := db.Exec("DELETE FROM users WHERE deleted_at IS NULL")
		if err != nil {
			logger.Error("Failed to delete users: %v", err)
		} else {
			deletedUsers, _ = result.RowsAffected()
			logger.Info("✓ Deleted %d users", deletedUsers)
		}
	}

	// Database cleanup summary
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("🗄️  DATABASE CLEANUP SUMMARY")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("Distributors: %d deleted\n", deletedDistributors)
	fmt.Printf("Resellers:    %d deleted\n", deletedResellers)
	fmt.Printf("Customers:    %d deleted\n", deletedCustomers)
	fmt.Printf("Users:        %d deleted\n", deletedUsers)
	fmt.Printf("Total:        %d deleted\n", deletedDistributors+deletedResellers+deletedCustomers+deletedUsers)
	fmt.Println(strings.Repeat("=", 60))

	return nil
}
