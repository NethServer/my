/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package syncmd

import (
	"fmt"
	"os"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

// CheckLogtoInitialization checks if Logto is properly initialized for Operation Center
func CheckLogtoInitialization(client *client.LogtoClient) (bool, error) {
	// Check if backend and frontend applications exist
	backendAppID := os.Getenv("BACKEND_APP_ID")

	apps, err := client.GetApplications()
	if err != nil {
		return false, err
	}

	backendExists := false
	frontendExists := false

	for _, app := range apps {
		if appID, ok := app["id"].(string); ok && appID == backendAppID {
			backendExists = true
		}
		if name, ok := app["name"].(string); ok && name == "frontend" {
			frontendExists = true
		}
	}

	// Check if owner user exists using search
	ownerExists := false
	if _, err := client.GetUserByUsername("owner"); err == nil {
		ownerExists = true
	}

	return backendExists && frontendExists && ownerExists, nil
}

// ValidateInitialization validates that Logto is properly initialized
func ValidateInitialization(logtoClient *client.LogtoClient) error {
	if initialized, err := CheckLogtoInitialization(logtoClient); err != nil {
		logger.Warn("Could not check initialization status: %v", err)
	} else if !initialized {
		logger.Warn("Logto does not appear to be initialized for Operation Center")
		logger.Info("Run 'sync init' first to set up applications and users")
		return fmt.Errorf("initialization required - run 'sync init' first")
	}
	return nil
}
