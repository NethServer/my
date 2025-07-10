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
	"fmt"
	"strings"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

// User represents a user structure
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// CreateOwnerUser creates the owner user in Logto
func CreateOwnerUser(client *client.LogtoClient, username, email, displayName string) (*User, error) {
	logger.Info("Creating Owner user...")

	password := GenerateSecurePassword()

	// Create user
	userData := map[string]interface{}{
		"username":     username,
		"primaryEmail": email,
		"name":         displayName,
	}

	createdUser, err := client.CreateUser(userData)
	if err != nil {
		// Check if user already exists
		errStr := err.Error()
		if strings.Contains(errStr, "username_already_in_use") || strings.Contains(errStr, "already in use") {
			logger.Warn("User 'owner' already exists")
			logger.Info("Using existing user for configuration (password not updated)")

			// Find existing user
			users, userErr := client.GetUsers()
			if userErr != nil {
				return nil, fmt.Errorf("failed to get existing users: %w", userErr)
			}

			var existingUserID string
			for _, user := range users {
				if userUsername, ok := user["username"].(string); ok && userUsername == "owner" {
					existingUserID = user["id"].(string)
					break
				}
			}

			if existingUserID == "" {
				return nil, fmt.Errorf("could not find existing owner user")
			}

			result := &User{
				ID:       existingUserID,
				Username: username,
				Email:    email,
				Password: "[NOT CHANGED]",
			}

			logger.Info("Using owner user: %s", result.ID)
			return result, nil
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID := createdUser["id"].(string)

	// Set password
	if err := client.SetUserPassword(userID, password); err != nil {
		return nil, fmt.Errorf("failed to set user password: %w", err)
	}

	result := &User{
		ID:       userID,
		Username: username,
		Email:    email,
		Password: password,
	}

	logger.Info("Created owner user: %s", result.ID)
	return result, nil
}
