/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nethesis/my/logto-sync/internal/client"
	"github.com/nethesis/my/logto-sync/internal/config"
	"github.com/nethesis/my/logto-sync/internal/logger"
)

// SyncCustomizations synchronizes Logto customizations
func SyncCustomizations(logtoClient *client.LogtoClient, cfg *config.Config, dryRun bool) error {
	if cfg.Customizations.CustomJwtClaims == nil || !cfg.Customizations.CustomJwtClaims.Enabled {
		logger.Info("Custom JWT claims are disabled, skipping synchronization")
		return nil
	}

	claims := cfg.Customizations.CustomJwtClaims
	logger.Info("Synchronizing custom JWT claims: %s", claims.ScriptPath)

	// Validate script file exists
	absolutePath, err := filepath.Abs(claims.ScriptPath)
	if err != nil {
		return fmt.Errorf("failed to resolve script path: %w", err)
	}

	if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
		return fmt.Errorf("custom JWT claims script file does not exist: %s", absolutePath)
	}

	// Read local script content
	localScript, err := os.ReadFile(absolutePath)
	if err != nil {
		return fmt.Errorf("failed to read local script: %w", err)
	}

	// Get current script from Logto (if any)
	remoteScript, err := logtoClient.GetCustomJwtClaims()
	if err != nil {
		return fmt.Errorf("failed to get current custom JWT claims: %w", err)
	}

	// Compare scripts (normalize line endings and whitespace)
	localNormalized := normalizeScript(string(localScript))
	remoteNormalized := normalizeScript(remoteScript)

	if localNormalized == remoteNormalized {
		logger.Info("Custom JWT claims script is already up to date")
		return nil
	}

	if dryRun {
		logger.Info("DRY RUN: Would update custom JWT claims script")
		logger.Debug("Script differences detected - local: %d lines, remote: %d lines",
			strings.Count(localNormalized, "\n"),
			strings.Count(remoteNormalized, "\n"))
		return nil
	}

	// Update the script
	if err := logtoClient.UpdateCustomJwtClaims(absolutePath); err != nil {
		return fmt.Errorf("failed to update custom JWT claims: %w", err)
	}

	logger.Info("Successfully updated custom JWT claims script")
	return nil
}

// normalizeScript normalizes script content for comparison
func normalizeScript(script string) string {
	// Normalize line endings
	normalized := strings.ReplaceAll(script, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	// Trim leading/trailing whitespace
	normalized = strings.TrimSpace(normalized)

	return normalized
}
