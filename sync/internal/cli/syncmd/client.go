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

// CreateLogtoClient creates and tests a Logto client connection
func CreateLogtoClient() (*client.LogtoClient, error) {
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
		return nil, fmt.Errorf("failed to connect to Logto: %w", err)
	}

	return logtoClient, nil
}
