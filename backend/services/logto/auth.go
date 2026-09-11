/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logto

import (
	"fmt"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// GetUserProfileFromLogto fetches complete user profile from Logto Management API
func GetUserProfileFromLogto(userID string) (*models.LogtoUser, error) {
	client := NewManagementClient()

	// Use the GetUserByID method we already have
	user, err := client.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	logger.ComponentLogger("logto").Debug().
		Str("operation", "profile_response").
		Str("username", user.Username).
		Str("email", logger.SanitizeString(user.PrimaryEmail)).
		Str("name", user.Name).
		Msg("Logto profile API response")

	return user, nil
}
