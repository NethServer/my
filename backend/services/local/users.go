/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/logto"
)

// ValidationError represents a validation error that should be returned as 400 instead of 500
type ValidationError struct {
	StatusCode int
	ErrorData  response.ErrorData
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error (status %d)", e.StatusCode)
}

// LocalUserService handles local-first user CRUD operations with Logto sync
type LocalUserService struct {
	userRepo    *entities.LocalUserRepository
	logtoClient *logto.LogtoManagementClient
}

// NewUserService creates a new local user service
func NewUserService() *LocalUserService {
	return &LocalUserService{
		userRepo:    entities.NewLocalUserRepository(),
		logtoClient: logto.NewManagementClient(),
	}
}

// =============================================================================
// PUBLIC METHODS
// =============================================================================

// CreateUser creates a user locally and syncs to Logto
func (s *LocalUserService) CreateUser(req *models.CreateLocalUserRequest, createdByUserID, createdByOrgID string) (*models.LocalUser, error) {
	// Generate username from email if not provided (clean for Logto format)
	if req.Username == "" {
		req.Username = s.generateUsernameFromEmail(req.Email)
	}

	// 1. Create in Logto FIRST for validation (before consuming local resources)
	// Start with user-provided custom data (allows custom fields)
	customData := make(map[string]interface{})
	if req.CustomData != nil {
		for k, v := range req.CustomData {
			customData[k] = v
		}
	}

	// System fields - these override any user-provided values and are always maintained
	customData["organizationId"] = req.OrganizationID
	customData["userRoleIds"] = req.UserRoleIDs
	customData["createdBy"] = createdByOrgID
	// Set initial creation timestamp
	customData["createdAt"] = time.Now().Format(time.RFC3339)

	logtoUserReq := models.CreateUserRequest{
		Username:     req.Username,
		Password:     req.Password,
		Name:         req.Name,
		PrimaryEmail: req.Email,
		CustomData:   customData,
	}

	// Add phone if provided
	if req.Phone != nil && *req.Phone != "" {
		logtoUserReq.PrimaryPhone = *req.Phone
	}

	logtoUser, err := s.logtoClient.CreateUser(logtoUserReq)
	if err != nil {
		logger.Error().
			Err(err).
			Str("username", req.Username).
			Str("email", req.Email).
			Msg("Failed to create user in Logto - validation failed")

		// Parse the error with context for better error handling
		context := map[string]interface{}{
			"email":    req.Email,
			"phone":    req.Phone,
			"username": req.Username,
		}
		parsedErr := s.parseLogtoError(err, context)

		// No rollback needed since we haven't created anything locally yet
		return nil, fmt.Errorf("failed to create user in Logto: %w", parsedErr)
	}

	// 2. Begin transaction for local operations (after Logto validation passes)
	tx, err := database.DB.Begin()
	if err != nil {
		// Cleanup the Logto user since we can't proceed with local creation
		if deleteErr := s.logtoClient.DeleteUser(logtoUser.ID); deleteErr != nil {
			logger.Warn().
				Err(deleteErr).
				Str("logto_user_id", logtoUser.ID).
				Msg("Failed to cleanup Logto user after local transaction failure")
		}
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 3. Create in local DB (after Logto validation passes)
	user, err := s.userRepo.Create(req)
	if err != nil {
		// Cleanup the Logto user since local creation failed
		if deleteErr := s.logtoClient.DeleteUser(logtoUser.ID); deleteErr != nil {
			logger.Warn().
				Err(deleteErr).
				Str("logto_user_id", logtoUser.ID).
				Msg("Failed to cleanup Logto user after local creation failure")
		}
		// Transaction will be rolled back by defer
		return nil, fmt.Errorf("failed to create user locally: %w", err)
	}

	// 4. Assign user roles if provided
	if len(req.UserRoleIDs) > 0 {
		err = s.logtoClient.AssignUserRoles(logtoUser.ID, req.UserRoleIDs)
		if err != nil {
			logger.Error().
				Err(err).
				Str("user_id", user.ID).
				Str("logto_user_id", logtoUser.ID).
				Strs("user_role_ids", req.UserRoleIDs).
				Msg("Failed to assign user roles to Logto user - rolling back local creation")

			// Delete the created Logto user to keep consistency
			if deleteErr := s.logtoClient.DeleteUser(logtoUser.ID); deleteErr != nil {
				logger.Warn().
					Err(deleteErr).
					Str("logto_user_id", logtoUser.ID).
					Msg("Failed to cleanup Logto user after role assignment failure")
			}

			// Transaction will be rolled back by defer
			return nil, fmt.Errorf("failed to assign user roles to Logto user: %w", err)
		}

		logger.Info().
			Str("user_id", user.ID).
			Str("logto_user_id", logtoUser.ID).
			Strs("user_role_ids", req.UserRoleIDs).
			Msg("User roles assigned successfully")
	}

	// 5. Assign user to organization if provided
	if req.OrganizationID != nil && *req.OrganizationID != "" {
		err = s.logtoClient.AssignUserToOrganization(*req.OrganizationID, logtoUser.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Str("user_id", user.ID).
				Str("logto_user_id", logtoUser.ID).
				Str("organization_id", *req.OrganizationID).
				Msg("Failed to assign user to organization in Logto - rolling back local creation")

			// Delete the created Logto user to keep consistency
			if deleteErr := s.logtoClient.DeleteUser(logtoUser.ID); deleteErr != nil {
				logger.Warn().
					Err(deleteErr).
					Str("logto_user_id", logtoUser.ID).
					Msg("Failed to cleanup Logto user after organization assignment failure")
			}

			// Transaction will be rolled back by defer
			return nil, fmt.Errorf("failed to assign user to organization in Logto: %w", err)
		}

		logger.Info().
			Str("user_id", user.ID).
			Str("logto_user_id", logtoUser.ID).
			Str("organization_id", *req.OrganizationID).
			Msg("User assigned to organization successfully")

		// 6. Determine and assign organization role
		orgRoleName := s.determineOrganizationRoleName(*req.OrganizationID)
		if orgRoleName != "" {
			// Get the organization role ID from Logto by name
			orgRole, err := s.logtoClient.GetOrganizationRoleByName(orgRoleName)
			if err != nil {
				logger.Error().
					Err(err).
					Str("user_id", user.ID).
					Str("logto_user_id", logtoUser.ID).
					Str("organization_id", *req.OrganizationID).
					Str("org_role_name", orgRoleName).
					Msg("Failed to get organization role from Logto - rolling back local creation")

				// Delete the created Logto user to keep consistency
				if deleteErr := s.logtoClient.DeleteUser(logtoUser.ID); deleteErr != nil {
					logger.Warn().
						Err(deleteErr).
						Str("logto_user_id", logtoUser.ID).
						Msg("Failed to cleanup Logto user after organization role lookup failure")
				}

				// Transaction will be rolled back by defer
				return nil, fmt.Errorf("failed to get organization role from Logto: %w", err)
			}

			// Assign organization role using the ID
			err = s.logtoClient.AssignOrganizationRolesToUser(*req.OrganizationID, logtoUser.ID, []string{orgRole.ID}, nil)
			if err != nil {
				logger.Error().
					Err(err).
					Str("user_id", user.ID).
					Str("logto_user_id", logtoUser.ID).
					Str("organization_id", *req.OrganizationID).
					Str("org_role_name", orgRoleName).
					Str("org_role_id", orgRole.ID).
					Msg("Failed to assign organization role to user in Logto - rolling back local creation")

				// Delete the created Logto user to keep consistency
				if deleteErr := s.logtoClient.DeleteUser(logtoUser.ID); deleteErr != nil {
					logger.Warn().
						Err(deleteErr).
						Str("logto_user_id", logtoUser.ID).
						Msg("Failed to cleanup Logto user after organization role assignment failure")
				}

				// Transaction will be rolled back by defer
				return nil, fmt.Errorf("failed to assign organization role to user in Logto: %w", err)
			}

			logger.Info().
				Str("user_id", user.ID).
				Str("logto_user_id", logtoUser.ID).
				Str("organization_id", *req.OrganizationID).
				Str("org_role_name", orgRoleName).
				Str("org_role_id", orgRole.ID).
				Msg("Organization role assigned successfully")
		}
	}

	// 7. Mark as synced
	err = s.markUserSynced(user.ID, logtoUser.ID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("user_id", user.ID).
			Msg("Failed to mark user as synced")
	}

	// 8. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("user_id", user.ID).
		Str("username", user.Username).
		Str("logto_user_id", logtoUser.ID).
		Str("created_by", createdByUserID).
		Msg("User created successfully with Logto sync")

	return user, nil
}

// GetUser retrieves a user by ID with RBAC validation
func (s *LocalUserService) GetUser(id, userOrgRole, userOrgID string) (*models.LocalUser, error) {
	// Get the user first
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Validate hierarchical access - user must be able to access the target user's organization
	var targetUserOrgID string
	if user.OrganizationID != nil {
		targetUserOrgID = *user.OrganizationID
	}

	if canAccess, reason := s.CanAccessUser(userOrgRole, userOrgID, targetUserOrgID); !canAccess {
		return nil, fmt.Errorf("access denied: %s", reason)
	}

	return user, nil
}

// GetUserByLogtoID retrieves a user by Logto ID (without RBAC validation, used for auth)
func (s *LocalUserService) GetUserByLogtoID(logtoID string) (*models.LocalUser, error) {
	return s.userRepo.GetByLogtoID(logtoID)
}

// ListUsers returns paginated users based on hierarchical RBAC (excluding specified user)
func (s *LocalUserService) ListUsers(userOrgRole, userOrgID, excludeUserID string, page, pageSize int) ([]*models.LocalUser, int, error) {
	return s.userRepo.List(userOrgRole, userOrgID, excludeUserID, page, pageSize)
}

// GetTotals returns total count of users based on hierarchical RBAC
func (s *LocalUserService) GetTotals(userOrgRole, userOrgID string) (int, error) {
	return s.userRepo.GetTotals(userOrgRole, userOrgID)
}

// UpdateUser updates a user locally and syncs to Logto
func (s *LocalUserService) UpdateUser(id string, req *models.UpdateLocalUserRequest, updatedByUserID, updatedByOrgID string) (*models.LocalUser, error) {
	// 1. Get current user before update to detect organization changes and for validation
	currentUser, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get current user for update: %w", err)
	}

	// Check if user is synced to Logto
	if currentUser.LogtoID == nil {
		return nil, fmt.Errorf("user not synced to Logto yet - missing logto_id")
	}

	// 2. Validate changes in Logto FIRST (before consuming local resources)

	updateReq := models.UpdateUserRequest{}
	if req.Username != nil {
		updateReq.Username = req.Username
	}
	if req.Name != nil {
		updateReq.Name = req.Name
	}
	if req.Email != nil {
		updateReq.PrimaryEmail = req.Email
	}
	if req.Phone != nil {
		if *req.Phone == "" {
			// Phone is being cleared, send empty string to clear it in Logto
			emptyPhone := ""
			updateReq.PrimaryPhone = &emptyPhone
		} else {
			updateReq.PrimaryPhone = req.Phone
		}
	}

	// Update custom data with user info
	// Start with existing custom data to preserve user-defined fields
	customData := make(map[string]interface{})
	if currentUser.CustomData != nil {
		for k, v := range currentUser.CustomData {
			customData[k] = v
		}
	}

	// Merge user-provided custom data (allows users to update their custom fields)
	if req.CustomData != nil {
		for k, v := range *req.CustomData {
			customData[k] = v
		}
	}

	// System fields - these override any user-provided values and are always maintained
	// Update organization if provided in request
	if req.OrganizationID != nil {
		customData["organizationId"] = req.OrganizationID
	} else {
		// Preserve existing organizationId
		customData["organizationId"] = currentUser.OrganizationID
	}

	// Update user roles if provided in request
	if req.UserRoleIDs != nil {
		customData["userRoleIds"] = req.UserRoleIDs
	} else {
		// Preserve existing userRoleIds
		customData["userRoleIds"] = currentUser.UserRoleIDs
	}

	// CRITICAL: Preserve original createdBy - never change it
	if existingCreatedBy, exists := customData["createdBy"]; exists {
		customData["createdBy"] = existingCreatedBy
	} else {
		// Fallback if somehow missing (should not happen in normal operation)
		customData["createdBy"] = updatedByOrgID
	}

	// Add update tracking (these are additional fields, not replacements)
	customData["updatedBy"] = updatedByOrgID
	customData["updatedAt"] = time.Now().Format(time.RFC3339)

	updateReq.CustomData = customData

	// Try the update in Logto first for validation
	_, err = s.logtoClient.UpdateUser(*currentUser.LogtoID, updateReq)
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", id).
			Str("username", currentUser.Username).
			Msg("Failed to validate user update in Logto")

		// Parse the error with context for better error handling
		context := map[string]interface{}{
			"email":    req.Email,
			"phone":    req.Phone,
			"username": req.Username,
		}
		parsedErr := s.parseLogtoError(err, context)

		// No rollback needed since we haven't changed anything locally yet
		return nil, fmt.Errorf("failed to validate user update in Logto: %w", parsedErr)
	}

	// 3. Begin transaction for local operations (after Logto validation passes)
	tx, err := database.DB.Begin()
	if err != nil {
		// Revert the Logto changes since we can't proceed with local update
		// We need to restore the original data in Logto
		originalReq := models.UpdateUserRequest{
			Username:     &currentUser.Username,
			Name:         &currentUser.Name,
			PrimaryEmail: &currentUser.Email,
		}
		if currentUser.Phone != nil {
			originalReq.PrimaryPhone = currentUser.Phone
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentUser.CustomData != nil {
			for k, v := range currentUser.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateUser(*currentUser.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("user_id", id).
				Msg("Failed to revert Logto changes after local transaction failure")
		}

		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 4. Update in local DB (after Logto validation passes)
	user, err := s.userRepo.Update(id, req)
	if err != nil {
		// Revert the Logto changes since local update failed
		originalReq := models.UpdateUserRequest{
			Username:     &currentUser.Username,
			Name:         &currentUser.Name,
			PrimaryEmail: &currentUser.Email,
		}
		if currentUser.Phone != nil {
			originalReq.PrimaryPhone = currentUser.Phone
		}
		// Restore original custom data
		originalCustomData := make(map[string]interface{})
		if currentUser.CustomData != nil {
			for k, v := range currentUser.CustomData {
				originalCustomData[k] = v
			}
		}
		originalReq.CustomData = originalCustomData

		if _, revertErr := s.logtoClient.UpdateUser(*currentUser.LogtoID, originalReq); revertErr != nil {
			logger.Warn().
				Err(revertErr).
				Str("user_id", id).
				Msg("Failed to revert Logto changes after local update failure")
		}

		// Transaction will be rolled back by defer
		return nil, fmt.Errorf("failed to update user locally: %w", err)
	}

	// 5. Update user roles if provided
	if req.UserRoleIDs != nil {
		// Get current user roles from Logto to know what to remove
		currentRoles, err := s.logtoClient.GetUserRoles(*user.LogtoID)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("user_id", id).
				Str("logto_user_id", *user.LogtoID).
				Msg("Failed to get current user roles from Logto for update")
		} else {
			// Remove all current roles
			if len(currentRoles) > 0 {
				currentRoleIDs := make([]string, len(currentRoles))
				for i, role := range currentRoles {
					currentRoleIDs[i] = role.ID
				}

				err = s.logtoClient.RemoveUserRoles(*user.LogtoID, currentRoleIDs)
				if err != nil {
					logger.Error().
						Err(err).
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Strs("current_role_ids", currentRoleIDs).
						Msg("Failed to remove current user roles from Logto")
					// Continue anyway to try assigning new roles
				} else {
					logger.Info().
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Strs("removed_role_ids", currentRoleIDs).
						Msg("Current user roles removed successfully")
				}
			}
		}

		// Assign new roles if any
		if len(*req.UserRoleIDs) > 0 {
			err = s.logtoClient.AssignUserRoles(*user.LogtoID, *req.UserRoleIDs)
			if err != nil {
				logger.Error().
					Err(err).
					Str("user_id", id).
					Str("logto_user_id", *user.LogtoID).
					Strs("new_user_role_ids", *req.UserRoleIDs).
					Msg("Failed to assign new user roles to Logto user")
				// Don't fail the entire operation for role assignment issues
			} else {
				logger.Info().
					Str("user_id", id).
					Str("logto_user_id", *user.LogtoID).
					Strs("new_user_role_ids", *req.UserRoleIDs).
					Msg("New user roles assigned successfully")
			}
		}
	}

	// 6. Update organization assignment if provided
	if req.OrganizationID != nil {
		// First need to check if organization changed
		oldOrgID := ""
		if currentUser.OrganizationID != nil {
			oldOrgID = *currentUser.OrganizationID
		}
		newOrgID := ""
		if req.OrganizationID != nil {
			newOrgID = *req.OrganizationID
		}

		if oldOrgID != newOrgID {
			logger.Info().
				Str("user_id", id).
				Str("logto_user_id", *user.LogtoID).
				Str("old_org_id", oldOrgID).
				Str("new_org_id", newOrgID).
				Msg("Organization change detected, updating Logto assignments")

			// Remove from old organization if it exists
			if oldOrgID != "" {
				// Get current organization roles to remove them
				oldOrgRoles, err := s.logtoClient.GetUserOrganizationRoles(oldOrgID, *user.LogtoID)
				if err != nil {
					logger.Warn().
						Err(err).
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Str("old_org_id", oldOrgID).
						Msg("Failed to get current organization roles for removal")
				} else if len(oldOrgRoles) > 0 {
					// Remove organization roles first (Logto requirement)
					for _, role := range oldOrgRoles {
						err = s.logtoClient.RemoveUserFromOrganizationRole(oldOrgID, *user.LogtoID, role.ID)
						if err != nil {
							logger.Warn().
								Err(err).
								Str("user_id", id).
								Str("logto_user_id", *user.LogtoID).
								Str("old_org_id", oldOrgID).
								Str("role_id", role.ID).
								Msg("Failed to remove user from old organization role")
						}
					}
				}

				// Remove from organization
				err = s.logtoClient.RemoveUserFromOrganization(oldOrgID, *user.LogtoID)
				if err != nil {
					logger.Warn().
						Err(err).
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Str("old_org_id", oldOrgID).
						Msg("Failed to remove user from old organization")
				} else {
					logger.Info().
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Str("old_org_id", oldOrgID).
						Msg("User removed from old organization successfully")
				}
			}

			// Add to new organization if specified
			if newOrgID != "" {
				err = s.logtoClient.AssignUserToOrganization(newOrgID, *user.LogtoID)
				if err != nil {
					logger.Error().
						Err(err).
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Str("new_org_id", newOrgID).
						Msg("Failed to assign user to new organization")
				} else {
					logger.Info().
						Str("user_id", id).
						Str("logto_user_id", *user.LogtoID).
						Str("new_org_id", newOrgID).
						Msg("User assigned to new organization successfully")

					// Determine and assign new organization role
					newOrgRoleName := s.determineOrganizationRoleName(newOrgID)
					if newOrgRoleName != "" {
						// Get the organization role ID from Logto by name
						newOrgRole, err := s.logtoClient.GetOrganizationRoleByName(newOrgRoleName)
						if err != nil {
							logger.Error().
								Err(err).
								Str("user_id", id).
								Str("logto_user_id", *user.LogtoID).
								Str("new_org_id", newOrgID).
								Str("new_org_role_name", newOrgRoleName).
								Msg("Failed to get new organization role from Logto")
						} else {
							// Assign new organization role
							err = s.logtoClient.AssignOrganizationRolesToUser(newOrgID, *user.LogtoID, []string{newOrgRole.ID}, nil)
							if err != nil {
								logger.Error().
									Err(err).
									Str("user_id", id).
									Str("logto_user_id", *user.LogtoID).
									Str("new_org_id", newOrgID).
									Str("new_org_role_name", newOrgRoleName).
									Str("new_org_role_id", newOrgRole.ID).
									Msg("Failed to assign new organization role to user")
							} else {
								logger.Info().
									Str("user_id", id).
									Str("logto_user_id", *user.LogtoID).
									Str("new_org_id", newOrgID).
									Str("new_org_role_name", newOrgRoleName).
									Str("new_org_role_id", newOrgRole.ID).
									Msg("New organization role assigned successfully")
							}
						}
					}
				}
			}
		}
	}

	// 7. Mark as synced
	err = s.markUserSynced(id, *user.LogtoID)
	if err != nil {
		logger.Warn().
			Err(err).
			Str("user_id", id).
			Msg("Failed to mark user as synced after update")
	}

	// 8. Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("user_id", id).
		Str("username", user.Username).
		Str("updated_by", updatedByUserID).
		Msg("User updated successfully with Logto sync")

	return user, nil
}

// DeleteUser soft-deletes a user locally and syncs to Logto
func (s *LocalUserService) DeleteUser(id, deletedByUserID, deletedByOrgID string) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Get user before deletion for logging and logto_id
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// 1. Soft delete in local DB first
	err = s.userRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("failed to delete user locally: %w", err)
	}

	// 2. Delete from Logto using logto_id
	if user.LogtoID != nil {
		err = s.logtoClient.DeleteUser(*user.LogtoID)
	} else {
		logger.Warn().Str("user_id", id).Msg("User has no logto_id, skipping Logto deletion")
		err = nil // Don't fail if not synced to Logto
	}
	if err != nil {
		logger.Error().
			Err(err).
			Str("user_id", id).
			Str("username", user.Username).
			Msg("Failed to sync user deletion to Logto")
		return fmt.Errorf("failed to sync user deletion to Logto: %w", err)
	}

	// 3. Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info().
		Str("user_id", id).
		Str("username", user.Username).
		Str("deleted_by", deletedByUserID).
		Msg("User deleted successfully with Logto sync")

	return nil
}

// ResetUserPassword resets a user's password with Logto validation
func (s *LocalUserService) ResetUserPassword(userID, password string) error {
	err := s.logtoClient.ResetUserPassword(userID, password)
	if err != nil {
		// Parse the error with context for better error handling
		context := map[string]interface{}{
			"password": password,
		}
		parsedErr := s.parseLogtoError(err, context)
		return parsedErr
	}
	return nil
}

// CanCreateUser validates if a user can create another user based on hierarchical permissions
func (s *LocalUserService) CanCreateUser(userOrgRole, userOrgID string, req *models.CreateLocalUserRequest) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can create users in organizations they manage hierarchically
		if req.OrganizationID != nil {
			if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, *req.OrganizationID) {
				return true, ""
			}
		}
		return false, "distributors can only create users in organizations they manage"
	case "reseller":
		// Reseller can create users in organizations they manage hierarchically
		if req.OrganizationID != nil {
			if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, *req.OrganizationID) {
				return true, ""
			}
		}
		return false, "resellers can only create users in their own organization or customers they manage"
	case "customer":
		// Customer can only create users in their own organization
		if req.OrganizationID != nil && *req.OrganizationID == userOrgID {
			return true, ""
		}
		return false, "customers can only create users in their own organization"
	default:
		return false, "insufficient permissions to create users"
	}
}

// CanUpdateUser validates if a user can update another user based on hierarchical permissions
func (s *LocalUserService) CanUpdateUser(userOrgRole, userOrgID, targetUserOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can update users in organizations they manage hierarchically
		if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetUserOrgID) {
			return true, ""
		}
		return false, "distributors can only update users in organizations they manage"
	case "reseller":
		// Reseller can update users in organizations they manage hierarchically
		if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetUserOrgID) {
			return true, ""
		}
		return false, "resellers can only update users in their own organization or customers they manage"
	case "customer":
		// Customer can only update users in their own organization
		if targetUserOrgID == userOrgID {
			return true, ""
		}
		return false, "customers can only update users in their own organization"
	default:
		return false, "insufficient permissions to update users"
	}
}

// CanDeleteUser validates if a user can delete another user based on hierarchical permissions
func (s *LocalUserService) CanDeleteUser(userOrgRole, userOrgID, targetUserOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can delete users in organizations they manage hierarchically
		if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetUserOrgID) {
			return true, ""
		}
		return false, "distributors can only delete users in organizations they manage"
	case "reseller":
		// Reseller can delete users in organizations they manage hierarchically
		if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetUserOrgID) {
			return true, ""
		}
		return false, "resellers can only delete users in their own organization or customers they manage"
	case "customer":
		// Customer can only delete users in their own organization
		if targetUserOrgID == userOrgID {
			return true, ""
		}
		return false, "customers can only delete users in their own organization"
	default:
		return false, "insufficient permissions to delete users"
	}
}

// CanAccessUser validates if a user can access another user based on hierarchical permissions
func (s *LocalUserService) CanAccessUser(userOrgRole, userOrgID, targetUserOrgID string) (bool, string) {
	switch userOrgRole {
	case "owner":
		return true, ""
	case "distributor":
		// Distributor can access users in organizations they manage hierarchically
		if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetUserOrgID) {
			return true, ""
		}
		return false, "distributors can only access users in organizations they manage"
	case "reseller":
		// Reseller can access users in organizations they manage hierarchically
		if s.IsOrganizationInHierarchy(userOrgRole, userOrgID, targetUserOrgID) {
			return true, ""
		}
		return false, "resellers can only access users in their own organization or customers they manage"
	case "customer":
		// Customer can only access users in their own organization
		if targetUserOrgID == userOrgID {
			return true, ""
		}
		return false, "customers can only access users in their own organization"
	default:
		return false, "insufficient permissions to access users"
	}
}

// IsOrganizationInHierarchy checks if targetOrgID is in the hierarchy under userOrgID
func (s *LocalUserService) IsOrganizationInHierarchy(userOrgRole, userOrgID, targetOrgID string) bool {
	if userOrgID == targetOrgID {
		return true // Direct match
	}

	switch userOrgRole {
	case "owner":
		// Owner can manage everything
		return true

	case "distributor":
		// Distributor can manage:
		// 1. Their own organization
		// 2. Resellers created by them
		// 3. Customers created by them or their resellers

		// Check if target is a reseller created by this distributor
		var count int
		query := `SELECT COUNT(*) FROM resellers WHERE logto_id = $1 AND custom_data->>'createdBy' = $2 AND active = TRUE`
		err := database.DB.QueryRow(query, targetOrgID, userOrgID).Scan(&count)
		if err == nil && count > 0 {
			return true
		}

		// Check if target is a customer created by this distributor
		query = `SELECT COUNT(*) FROM customers WHERE logto_id = $1 AND custom_data->>'createdBy' = $2 AND active = TRUE`
		err = database.DB.QueryRow(query, targetOrgID, userOrgID).Scan(&count)
		if err == nil && count > 0 {
			return true
		}

		// Check if target is a customer created by a reseller created by this distributor
		query = `
			SELECT COUNT(*) FROM customers c
			JOIN resellers r ON c.custom_data->>'createdBy' = r.logto_id
			WHERE c.logto_id = $1 AND r.custom_data->>'createdBy' = $2 AND c.active = TRUE AND r.active = TRUE
		`
		err = database.DB.QueryRow(query, targetOrgID, userOrgID).Scan(&count)
		if err == nil && count > 0 {
			return true
		}

	case "reseller":
		// Reseller can manage:
		// 1. Their own organization
		// 2. Customers created by them

		// Check if target is a customer created by this reseller
		var count int
		query := `SELECT COUNT(*) FROM customers WHERE logto_id = $1 AND custom_data->>'createdBy' = $2 AND active = TRUE`
		err := database.DB.QueryRow(query, targetOrgID, userOrgID).Scan(&count)
		if err == nil && count > 0 {
			return true
		}
	}

	return false
}

// GetHierarchicalOrganizationIDs returns all organization IDs that the user can manage
func (s *LocalUserService) GetHierarchicalOrganizationIDs(userOrgRole, userOrgID string) ([]string, error) {
	return s.userRepo.GetHierarchicalOrganizationIDs(userOrgRole, userOrgID)
}

// =============================================================================
// PRIVATE METHODS
// =============================================================================

// markUserSynced marks a user as synced with Logto
func (s *LocalUserService) markUserSynced(id, logtoID string) error {
	query := `UPDATE users SET logto_id = $1, logto_synced_at = $2 WHERE id = $3`
	_, err := database.DB.Exec(query, logtoID, time.Now(), id)
	return err
}

// generateUsernameFromEmail converts email to valid Logto username format
func (s *LocalUserService) generateUsernameFromEmail(email string) string {
	// Take the local part of email (before @)
	localPart := strings.Split(email, "@")[0]

	// Replace invalid characters with underscores
	reg := regexp.MustCompile(`[^A-Za-z0-9_]`)
	username := reg.ReplaceAllString(localPart, "_")

	// Ensure it starts with letter or underscore
	if len(username) > 0 && !regexp.MustCompile(`^[A-Za-z_]`).MatchString(username) {
		username = "_" + username
	}

	// Ensure it's not empty (fallback)
	if username == "" {
		username = "user_" + strings.ReplaceAll(email, "@", "_at_")
		username = reg.ReplaceAllString(username, "_")
	}

	return username
}

// parseLogtoError parses Logto API errors and returns a ValidationError for client errors
func (s *LocalUserService) parseLogtoError(err error, context map[string]interface{}) error {
	if err == nil {
		return nil
	}

	errorStr := err.Error()

	// Check if it's a Logto API error with status code
	if strings.Contains(errorStr, "status ") {
		// Extract status code and JSON body
		parts := strings.Split(errorStr, "status ")
		if len(parts) >= 2 {
			statusAndBody := parts[len(parts)-1]
			colonIndex := strings.Index(statusAndBody, ": ")
			if colonIndex > 0 {
				statusStr := statusAndBody[:colonIndex]
				jsonBody := statusAndBody[colonIndex+2:]

				// Parse status code
				var statusCode int
				if _, parseErr := fmt.Sscanf(statusStr, "%d", &statusCode); parseErr == nil {
					// Check if it's a client error (4xx)
					if statusCode >= 400 && statusCode < 500 {
						// Parse JSON error body
						var logtoError interface{}
						if jsonErr := json.Unmarshal([]byte(jsonBody), &logtoError); jsonErr == nil {
							// Use existing response package to normalize Logto error
							errorData := response.NormalizeLogtoErrorWithContext(logtoError, context)

							return &ValidationError{
								StatusCode: statusCode,
								ErrorData:  errorData,
							}
						}
					}
				}
			}
		}
	}

	// Return original error if not parseable or not a client error
	return err
}

// determineOrganizationRoleName determines the organization role name based on organization type in database
func (s *LocalUserService) determineOrganizationRoleName(organizationID string) string {
	// Query each organization table to determine the type
	// organizationID is the logto_id, not the local database id

	// Check distributors table
	var count int
	query := `SELECT COUNT(*) FROM distributors WHERE logto_id = $1 AND active = TRUE`
	err := database.DB.QueryRow(query, organizationID).Scan(&count)
	if err == nil && count > 0 {
		return "Distributor"
	}

	// Check resellers table
	query = `SELECT COUNT(*) FROM resellers WHERE logto_id = $1 AND active = TRUE`
	err = database.DB.QueryRow(query, organizationID).Scan(&count)
	if err == nil && count > 0 {
		return "Reseller"
	}

	// Check customers table
	query = `SELECT COUNT(*) FROM customers WHERE logto_id = $1 AND active = TRUE`
	err = database.DB.QueryRow(query, organizationID).Scan(&count)
	if err == nil && count > 0 {
		return "Customer"
	}

	// If not found in any table, it might be the owner organization
	return "Owner"
}
