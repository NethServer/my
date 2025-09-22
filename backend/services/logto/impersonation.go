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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// ImpersonationRequest represents the request for user impersonation
type ImpersonationRequest struct {
	Subject string            `json:"subject"`
	Actor   string            `json:"actor"`
	Context map[string]string `json:"context,omitempty"`
}

// ImpersonationResponse represents the response from Logto impersonation
type ImpersonationResponse struct {
	SubjectToken string `json:"subject_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// RequestImpersonationToken requests a subject token from Logto for user impersonation
func (c *LogtoManagementClient) RequestImpersonationToken(impersonatedUserID, impersonatorUserID string) (*ImpersonationResponse, error) {
	// Prepare request payload
	reqBody := ImpersonationRequest{
		Subject: impersonatedUserID,
		Actor:   impersonatorUserID,
		Context: map[string]string{
			"source": "nethesis-backend",
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "impersonation_request_marshal").
			Str("impersonated_user_id", impersonatedUserID).
			Str("impersonator_user_id", impersonatorUserID).
			Msg("Failed to marshal impersonation request")
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	logger.ComponentLogger("logto").Info().
		Str("operation", "impersonation_request_start").
		Str("impersonated_user_id", impersonatedUserID).
		Str("impersonator_user_id", impersonatorUserID).
		Msg("Requesting impersonation token from Logto")

	// Make request to Logto
	resp, err := c.makeRequest("POST", "/api/subject-tokens", bytes.NewReader(jsonData))
	if err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "impersonation_request_failed").
			Str("impersonated_user_id", impersonatedUserID).
			Str("impersonator_user_id", impersonatorUserID).
			Msg("Failed to request impersonation token")
		return nil, fmt.Errorf("failed to request impersonation token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logger.ComponentLogger("logto").Error().
			Str("operation", "impersonation_request_error_status").
			Int("status_code", resp.StatusCode).
			Str("response_body", string(body)).
			Str("impersonated_user_id", impersonatedUserID).
			Str("impersonator_user_id", impersonatorUserID).
			Msg("Impersonation request failed with error status")
		return nil, fmt.Errorf("impersonation request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var impResp ImpersonationResponse
	if err := json.NewDecoder(resp.Body).Decode(&impResp); err != nil {
		logger.ComponentLogger("logto").Error().
			Err(err).
			Str("operation", "impersonation_response_decode").
			Str("impersonated_user_id", impersonatedUserID).
			Str("impersonator_user_id", impersonatorUserID).
			Msg("Failed to decode impersonation response")
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.ComponentLogger("logto").Info().
		Str("operation", "impersonation_request_success").
		Str("impersonated_user_id", impersonatedUserID).
		Str("impersonator_user_id", impersonatorUserID).
		Int("expires_in", impResp.ExpiresIn).
		Msg("Successfully obtained impersonation token from Logto")

	return &impResp, nil
}

// GetUserForImpersonation fetches user information specifically for impersonation
func GetUserForImpersonation(userID string) (*models.User, error) {
	client := NewManagementClient()

	// Get user profile from Logto
	userProfile, err := client.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	// Create user model
	user := models.User{
		ID:       userID, // Use Logto ID as primary for impersonation
		LogtoID:  &userID,
		Username: userProfile.Username,
		Email:    userProfile.PrimaryEmail,
		Name:     userProfile.Name,
	}

	if userProfile.PrimaryPhone != "" {
		user.Phone = &userProfile.PrimaryPhone
	}

	// Enrich with roles and permissions
	enrichedUser, err := EnrichUserWithRolesAndPermissions(userID)
	if err != nil {
		logger.ComponentLogger("logto").Warn().
			Err(err).
			Str("operation", "enrich_user_impersonation").
			Str("user_id", userID).
			Msg("Failed to enrich user for impersonation")
		return &user, nil
	}

	user.UserRoles = enrichedUser.UserRoles
	user.UserRoleIDs = enrichedUser.UserRoleIDs
	user.UserPermissions = enrichedUser.UserPermissions
	user.OrgRole = enrichedUser.OrgRole
	user.OrgRoleID = enrichedUser.OrgRoleID
	user.OrgPermissions = enrichedUser.OrgPermissions
	user.OrganizationID = enrichedUser.OrganizationID
	user.OrganizationName = enrichedUser.OrganizationName

	logger.ComponentLogger("logto").Debug().
		Str("operation", "user_impersonation_prepared").
		Str("user_id", userID).
		Str("username", user.Username).
		Str("organization_id", user.OrganizationID).
		Str("org_role", user.OrgRole).
		Strs("user_roles", user.UserRoles).
		Msg("User prepared for impersonation")

	return &user, nil
}
