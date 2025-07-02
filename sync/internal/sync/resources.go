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
	"github.com/nethesis/my/sync/internal/constants"
	"github.com/nethesis/my/sync/internal/logger"
)

// syncResources synchronizes resources and their scopes
func (e *Engine) syncResources(cfg *config.Config, result *Result) error {
	logger.Info("Syncing resources...")

	// Get existing resources
	existingResources, err := e.client.GetResources()
	if err != nil {
		if e.options.DryRun {
			logger.Warn("DRY RUN: Could not get existing resources: %v", err)
			return nil
		}
		return fmt.Errorf("failed to get existing resources: %w", err)
	}

	existingResourceMap := make(map[string]client.LogtoResource)
	configResourceMap := make(map[string]bool)

	for _, resource := range existingResources {
		existingResourceMap[resource.Name] = resource
	}

	for _, resource := range cfg.Hierarchy.Resources {
		configResourceMap[resource.Name] = true
	}

	// Create or recreate resources
	for _, configResource := range cfg.Hierarchy.Resources {
		expectedIndicator := fmt.Sprintf("%s/api/%s", e.options.APIBaseURL, configResource.Name)

		if existingResource, exists := existingResourceMap[configResource.Name]; exists {
			// Check if indicator needs to be changed
			if existingResource.Indicator != expectedIndicator {
				if e.options.DryRun {
					logger.Info("DRY RUN: Would recreate resource with new indicator: %s", configResource.Name)
					e.addOperation(result, "resource", "delete", configResource.Name,
						"Would delete resource for recreation", nil)
					e.addOperation(result, "resource", "create", configResource.Name,
						"Would create resource with new indicator", nil)
					result.Summary.ResourcesDeleted++
					result.Summary.ResourcesCreated++
				} else {
					logger.Info("Recreating resource with new indicator: %s", configResource.Name)

					// Delete existing resource
					err := e.client.DeleteResource(existingResource.ID)
					e.addOperation(result, "resource", "delete", configResource.Name,
						"Deleted resource for recreation", err)
					if err != nil {
						return fmt.Errorf("failed to delete resource %s: %w", configResource.Name, err)
					}
					result.Summary.ResourcesDeleted++

					// Create new resource with correct indicator
					logger.Info("Creating resource: %s", configResource.Name)
					logtoResource := client.LogtoResource{
						Name:           configResource.Name,
						Indicator:      expectedIndicator,
						IsDefault:      false,
						AccessTokenTTL: constants.DefaultTokenTTL,
					}

					err = e.client.CreateResource(logtoResource)
					e.addOperation(result, "resource", "create", configResource.Name,
						"Created resource with new indicator", err)
					if err != nil {
						return fmt.Errorf("failed to create resource %s: %w", configResource.Name, err)
					}
					result.Summary.ResourcesCreated++
				}
			} else {
				logger.Debug("Resource %s already exists with correct indicator", configResource.Name)
			}
		} else {
			// Create new resource
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create resource: %s", configResource.Name)
				e.addOperation(result, "resource", "create", configResource.Name,
					"Would create new resource", nil)
				result.Summary.ResourcesCreated++
			} else {
				logger.Info("Creating resource: %s", configResource.Name)
				logtoResource := client.LogtoResource{
					Name:           configResource.Name,
					Indicator:      expectedIndicator,
					IsDefault:      false,
					AccessTokenTTL: constants.DefaultTokenTTL,
				}

				err := e.client.CreateResource(logtoResource)
				e.addOperation(result, "resource", "create", configResource.Name,
					"Created new resource", err)
				if err != nil {
					return fmt.Errorf("failed to create resource %s: %w", configResource.Name, err)
				}
				result.Summary.ResourcesCreated++
			}
		}

		// Sync scopes for this resource
		if err := e.syncScopes(configResource, result); err != nil {
			return fmt.Errorf("failed to sync scopes for resource %s: %w", configResource.Name, err)
		}
	}

	// Cleanup phase: remove resources not in config (only if --cleanup flag is set)
	if e.options.Cleanup {
		if e.options.DryRun {
			logger.Info("DRY RUN: Would check for resources to remove...")
		} else {
			logger.Info("Cleanup mode: checking for resources to remove...")
		}

		for _, existingResource := range existingResources {
			// Skip system/default resources
			if existingResource.IsDefault {
				logger.Debug("Skipping system resource: %s", existingResource.Name)
				continue
			}

			// Skip management API resource
			if existingResource.Name == "Logto Management API" ||
				existingResource.Indicator == "https://default.logto.app/api" {
				logger.Debug("Skipping Logto system resource: %s", existingResource.Name)
				continue
			}

			if !configResourceMap[existingResource.Name] {
				if e.options.DryRun {
					logger.Warn("DRY RUN: Would remove resource not in config: %s", existingResource.Name)
					e.addOperation(result, "resource", "cleanup", existingResource.Name,
						"Would remove resource not in config", nil)
					result.Summary.ResourcesDeleted++
				} else {
					logger.Warn("Removing resource not in config: %s", existingResource.Name)
					err := e.client.DeleteResource(existingResource.ID)
					e.addOperation(result, "resource", "cleanup", existingResource.Name,
						"Removed resource not in config", err)
					if err != nil {
						logger.Error("Failed to remove resource %s: %v", existingResource.Name, err)
					} else {
						result.Summary.ResourcesDeleted++
					}
				}
			}
		}
	}

	logger.Info("Resources sync completed")
	return nil
}

// syncScopes synchronizes scopes for a specific resource
func (e *Engine) syncScopes(resource config.Resource, result *Result) error {
	logger.Debug("Syncing scopes for resource: %s", resource.Name)

	// First, get the resource ID
	resources, err := e.client.GetResources()
	if err != nil {
		if e.options.DryRun {
			logger.Warn("DRY RUN: Could not get resources for scope sync: %v", err)
			// In dry-run, simulate scope creation
			for _, action := range resource.Actions {
				scopeName := fmt.Sprintf("%s:%s", action, resource.Name)
				logger.Info("DRY RUN: Would create scope: %s", scopeName)
				e.addOperation(result, "scope", "create", scopeName,
					fmt.Sprintf("Would create scope for %s", resource.Name), nil)
				result.Summary.ScopesCreated++
			}
			return nil
		}
		return fmt.Errorf("failed to get resources: %w", err)
	}

	var resourceID string
	for _, r := range resources {
		if r.Name == resource.Name {
			resourceID = r.ID
			break
		}
	}

	if resourceID == "" {
		if e.options.DryRun {
			logger.Debug("DRY RUN: Resource %s not found, would be created", resource.Name)
			// In dry-run, the resource would be created, so simulate scope creation
			for _, action := range resource.Actions {
				scopeName := fmt.Sprintf("%s:%s", action, resource.Name)
				logger.Info("DRY RUN: Would create scope: %s", scopeName)
				e.addOperation(result, "scope", "create", scopeName,
					fmt.Sprintf("Would create scope for %s", resource.Name), nil)
				result.Summary.ScopesCreated++
			}
			return nil
		}
		return fmt.Errorf("resource %s not found", resource.Name)
	}

	// Get existing scopes
	existingScopes, err := e.client.GetScopes(resourceID)
	if err != nil {
		if e.options.DryRun {
			logger.Warn("DRY RUN: Could not get existing scopes: %v", err)
			return nil
		}
		return fmt.Errorf("failed to get existing scopes: %w", err)
	}

	existingScopeMap := make(map[string]bool)
	for _, scope := range existingScopes {
		existingScopeMap[scope.Name] = true
	}

	// Create scopes for each action
	configScopeMap := make(map[string]bool)
	for _, action := range resource.Actions {
		scopeName := fmt.Sprintf("%s:%s", action, resource.Name)
		configScopeMap[scopeName] = true

		if !existingScopeMap[scopeName] {
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create scope: %s", scopeName)
				e.addOperation(result, "scope", "create", scopeName,
					fmt.Sprintf("Would create scope for %s", resource.Name), nil)
				result.Summary.ScopesCreated++
			} else {
				logger.Info("Creating scope: %s", scopeName)
				scope := client.LogtoScope{
					Name:        scopeName,
					Description: fmt.Sprintf("Permission to %s %s", action, resource.Name),
					ResourceID:  resourceID,
				}

				err := e.client.CreateScope(resourceID, scope)
				e.addOperation(result, "scope", "create", scopeName,
					fmt.Sprintf("Created scope for %s", resource.Name), err)
				if err != nil {
					return fmt.Errorf("failed to create scope %s: %w", scopeName, err)
				}
				result.Summary.ScopesCreated++
			}
		} else {
			logger.Debug("Scope %s already exists", scopeName)
		}
	}

	// Cleanup phase: remove scopes not in config (only if --cleanup flag is set)
	if e.options.Cleanup {
		for _, existingScope := range existingScopes {
			if !configScopeMap[existingScope.Name] {
				if e.options.DryRun {
					logger.Info("DRY RUN: Would remove scope not in config: %s", existingScope.Name)
					e.addOperation(result, "scope", "cleanup", existingScope.Name,
						fmt.Sprintf("Would remove scope not in config from %s", resource.Name), nil)
					result.Summary.ScopesDeleted++
				} else {
					logger.Info("Removing scope not in config: %s", existingScope.Name)
					err := e.client.DeleteScope(resourceID, existingScope.ID)
					e.addOperation(result, "scope", "cleanup", existingScope.Name,
						fmt.Sprintf("Removed scope not in config from %s", resource.Name), err)
					if err != nil {
						logger.Warn("Failed to remove scope %s: %v", existingScope.Name, err)
					} else {
						result.Summary.ScopesDeleted++
					}
				}
			}
		}
	}

	return nil
}
