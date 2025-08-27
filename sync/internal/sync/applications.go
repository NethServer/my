/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"fmt"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// syncThirdPartyApplications synchronizes third-party applications
func (e *Engine) syncThirdPartyApplications(cfg *config.Config, result *Result) error {
	if len(cfg.ThirdPartyApps) == 0 {
		logger.Info("No third-party applications to sync")
		return nil
	}

	logger.Info("Syncing %d third-party applications", len(cfg.ThirdPartyApps))

	// Get existing third-party applications
	existingAppsList, err := e.client.GetThirdPartyApplications()
	if err != nil {
		return fmt.Errorf("failed to get existing third-party applications: %w", err)
	}

	// Index existing applications by name for efficient lookup
	existingApps := make(map[string]client.ThirdPartyApplication)
	for _, app := range existingAppsList {
		existingApps[app.Name] = app
	}

	// Sync each configured application
	for _, appConfig := range cfg.ThirdPartyApps {
		if err := e.syncSingleApplication(appConfig, existingApps, cfg, result); err != nil {
			logger.Error("Failed to sync application %s: %v", appConfig.Name, err)
			result.Errors = append(result.Errors, fmt.Sprintf("Application %s sync failed: %v", appConfig.Name, err))
		}
	}

	// Cleanup unused applications if enabled
	if e.options.Cleanup {
		if err := e.cleanupThirdPartyApplications(cfg, existingApps, result); err != nil {
			logger.Error("Failed to cleanup applications: %v", err)
			result.Errors = append(result.Errors, fmt.Sprintf("Applications cleanup failed: %v", err))
		}
	}

	return nil
}

// syncSingleApplication synchronizes a single third-party application
func (e *Engine) syncSingleApplication(appConfig config.Application, existingApps map[string]client.ThirdPartyApplication, cfg *config.Config, result *Result) error {
	logger.Info("Processing application: %s", appConfig.Name)

	// Determine scopes to assign
	scopes := cfg.GetDefaultScopes()
	if len(appConfig.Scopes) > 0 {
		scopes = appConfig.Scopes
	}

	// Check if application exists
	if existingApp, exists := existingApps[appConfig.Name]; exists {
		// Update existing application
		logger.Info("Updating existing third-party application: %s", appConfig.Name)

		updatedApp := client.ThirdPartyApplication{
			ID:           existingApp.ID,
			Name:         appConfig.Name,
			Description:  appConfig.Description,
			Type:         existingApp.Type,
			IsThirdParty: true,
		}

		// Add custom data (access control and login_url)
		customData := make(map[string]interface{})

		// Add access control if configured
		if appConfig.AccessControl != nil {
			customData["access_control"] = map[string]interface{}{
				"organization_roles": appConfig.AccessControl.OrganizationRoles,
				"user_roles":         appConfig.AccessControl.UserRoles,
			}
		}

		// Add login_url if configured
		if appConfig.LoginURL != "" {
			customData["login_url"] = appConfig.LoginURL
		}

		// Set custom data if we have any
		if len(customData) > 0 {
			updatedApp.CustomData = customData
		}

		// Add OIDC metadata only if URIs are provided
		if len(appConfig.RedirectUris) > 0 || len(appConfig.PostLogoutRedirectUris) > 0 {
			updatedApp.OidcClientMetadata = &client.OidcClientMetadata{
				RedirectUris:           appConfig.RedirectUris,
				PostLogoutRedirectUris: appConfig.PostLogoutRedirectUris,
			}
		}

		if !e.options.DryRun {
			if err := e.client.UpdateThirdPartyApplication(existingApp.ID, updatedApp); err != nil {
				e.addOperation(result, "application", "update", appConfig.Name, fmt.Sprintf("Update application %s", appConfig.Name), err)
				return fmt.Errorf("failed to update application %s: %w", appConfig.Name, err)
			}

			if err := e.client.UpdateThirdPartyApplicationBranding(existingApp.ID, appConfig.DisplayName); err != nil {
				e.addOperation(result, "application", "update_branding", appConfig.Name, fmt.Sprintf("Update branding for %s", appConfig.Name), err)
				return fmt.Errorf("failed to update branding for %s: %w", appConfig.Name, err)
			}

			if err := e.client.UpdateThirdPartyApplicationScopes(existingApp.ID, scopes); err != nil {
				e.addOperation(result, "application", "update_scopes", appConfig.Name, fmt.Sprintf("Update scopes for %s", appConfig.Name), err)
				return fmt.Errorf("failed to update scopes for %s: %w", appConfig.Name, err)
			}
		}

		e.addOperation(result, "application", "update", appConfig.Name, fmt.Sprintf("Update application %s", appConfig.Name), nil)
		result.Summary.ApplicationsUpdated++
		logger.Info("Updated application: %s", appConfig.Name)
	} else {
		// Create new application
		logger.Info("Creating new application: %s", appConfig.Name)

		newApp := client.ThirdPartyApplication{
			Name:         appConfig.Name,
			Description:  appConfig.Description,
			Type:         "Traditional",
			IsThirdParty: true,
		}

		// Add custom data (access control and login_url)
		customData := make(map[string]interface{})

		// Add access control if configured
		if appConfig.AccessControl != nil {
			customData["access_control"] = map[string]interface{}{
				"organization_roles": appConfig.AccessControl.OrganizationRoles,
				"user_roles":         appConfig.AccessControl.UserRoles,
			}
		}

		// Add login_url if configured
		if appConfig.LoginURL != "" {
			customData["login_url"] = appConfig.LoginURL
		}

		// Set custom data if we have any
		if len(customData) > 0 {
			newApp.CustomData = customData
		}

		// Add OIDC metadata only if URIs are provided
		if len(appConfig.RedirectUris) > 0 || len(appConfig.PostLogoutRedirectUris) > 0 {
			newApp.OidcClientMetadata = &client.OidcClientMetadata{
				RedirectUris:           appConfig.RedirectUris,
				PostLogoutRedirectUris: appConfig.PostLogoutRedirectUris,
			}
		}

		if !e.options.DryRun {
			createdApp, err := e.client.CreateThirdPartyApplication(newApp)
			if err != nil {
				e.addOperation(result, "application", "create", appConfig.Name, fmt.Sprintf("Create application %s", appConfig.Name), err)
				return fmt.Errorf("failed to create application %s: %w", appConfig.Name, err)
			}

			if err := e.client.UpdateThirdPartyApplicationBranding(createdApp.ID, appConfig.DisplayName); err != nil {
				e.addOperation(result, "application", "create_branding", appConfig.Name, fmt.Sprintf("Set branding for %s", appConfig.Name), err)
				return fmt.Errorf("failed to set branding for %s: %w", appConfig.Name, err)
			}

			if err := e.client.UpdateThirdPartyApplicationScopes(createdApp.ID, scopes); err != nil {
				e.addOperation(result, "application", "create_scopes", appConfig.Name, fmt.Sprintf("Set scopes for %s", appConfig.Name), err)
				return fmt.Errorf("failed to set scopes for %s: %w", appConfig.Name, err)
			}
		}

		e.addOperation(result, "application", "create", appConfig.Name, fmt.Sprintf("Create application %s", appConfig.Name), nil)
		result.Summary.ApplicationsCreated++
		logger.Info("Created application: %s", appConfig.Name)
	}

	return nil
}

// cleanupThirdPartyApplications removes applications not defined in config
func (e *Engine) cleanupThirdPartyApplications(cfg *config.Config, existingApps map[string]client.ThirdPartyApplication, result *Result) error {
	// Build set of configured application names
	configuredNames := make(map[string]bool)
	for _, app := range cfg.ThirdPartyApps {
		configuredNames[app.Name] = true
	}

	// Find applications to delete
	var toDelete []client.ThirdPartyApplication
	for name, app := range existingApps {
		if !configuredNames[name] {
			toDelete = append(toDelete, app)
		}
	}

	if len(toDelete) == 0 {
		logger.Info("No applications to cleanup")
		return nil
	}

	logger.Info("Cleaning up %d applications", len(toDelete))

	// Delete each application
	for _, app := range toDelete {
		logger.Info("Deleting application: %s", app.Name)

		if !e.options.DryRun {
			if err := e.client.DeleteThirdPartyApplication(app.ID); err != nil {
				e.addOperation(result, "application", "delete", app.Name, fmt.Sprintf("Delete application %s", app.Name), err)
				return fmt.Errorf("failed to delete application %s: %w", app.Name, err)
			}
		}

		e.addOperation(result, "application", "delete", app.Name, fmt.Sprintf("Delete application %s", app.Name), nil)
		result.Summary.ApplicationsDeleted++
		logger.Info("Deleted application: %s", app.Name)
	}

	return nil
}
