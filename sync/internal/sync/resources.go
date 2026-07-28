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
	"strings"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
)

// syncResources synchronizes the Logto API resources that host our scopes.
//
// Logto bills per API resource, not per scope, so config resources are grouped
// into containers (see config.Resource.Container) and only the distinct
// containers become Logto resources. Scope names remain "{action}:{resource}",
// so neither roles nor the backend can tell the difference.
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

	containers := cfg.GetResourceContainers()
	for _, container := range containers {
		configResourceMap[container.Name] = true
	}

	// Create or recreate containers, then reconcile the scopes they hold
	for _, container := range containers {
		resourceIndicator := fmt.Sprintf("%s/%s", e.options.APIBaseURL, container.Name)

		if existingResource, exists := existingResourceMap[container.Name]; exists {
			// Only recreate when the indicator actually drifted. Deleting a
			// resource cascades to its scopes, which strips them from every
			// role that references them, so this must not run on every sync.
			if existingResource.Indicator == resourceIndicator {
				logger.Debug("Resource %s already exists with correct indicator", container.Name)
			} else if e.options.DryRun {
				logger.Info("DRY RUN: Would recreate resource with new indicator: %s", container.Name)
				e.addOperation(result, "resource", "delete", container.Name,
					"Would delete resource for recreation", nil)
				e.addOperation(result, "resource", "create", container.Name,
					"Would create resource with new indicator", nil)
				result.Summary.ResourcesDeleted++
				result.Summary.ResourcesCreated++
			} else {
				logger.Info("Recreating resource with new indicator: %s", container.Name)

				// Delete existing resource
				err := e.client.DeleteResource(existingResource.ID)
				e.addOperation(result, "resource", "delete", container.Name,
					"Deleted resource for recreation", err)
				if err != nil {
					return fmt.Errorf("failed to delete resource %s: %w", container.Name, err)
				}
				result.Summary.ResourcesDeleted++

				// Create new resource with correct indicator
				logger.Info("Creating resource: %s", container.Name)
				err = e.client.CreateResource(client.LogtoResource{
					Name:      container.Name,
					Indicator: resourceIndicator,
				})
				e.addOperation(result, "resource", "create", container.Name,
					"Created resource with new indicator", err)
				if err != nil {
					return fmt.Errorf("failed to create resource %s: %w", container.Name, err)
				}
				result.Summary.ResourcesCreated++
			}
		} else {
			// Create new resource
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create resource: %s", container.Name)
				e.addOperation(result, "resource", "create", container.Name,
					"Would create new resource", nil)
				result.Summary.ResourcesCreated++
			} else {
				logger.Info("Creating resource: %s", container.Name)
				err := e.client.CreateResource(client.LogtoResource{
					Name:      container.Name,
					Indicator: resourceIndicator,
				})
				e.addOperation(result, "resource", "create", container.Name,
					"Created new resource", err)
				if err != nil {
					return fmt.Errorf("failed to create resource %s: %w", container.Name, err)
				}
				result.Summary.ResourcesCreated++
			}
		}

		// Sync scopes for this container
		if err := e.syncScopes(container, result); err != nil {
			return fmt.Errorf("failed to sync scopes for resource %s: %w", container.Name, err)
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
			// Never remove the tenant audience resource. Its indicator is the
			// API base URL itself and the backend validates the "aud" claim of
			// incoming Logto tokens against it (backend/middleware/logto.go),
			// which is what the third-party apps hitting /api/user/* rely on.
			// It is maintained by hand, so it never appears in the config.
			if strings.TrimSuffix(existingResource.Indicator, "/") == strings.TrimSuffix(e.options.APIBaseURL, "/") {
				logger.Debug("Skipping tenant audience resource: %s", existingResource.Name)
				continue
			}

			// Skip management API resource
			if existingResource.Name == "Logto Management API" {
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

// simulateScopeCreation records the scopes a container would gain, for dry-run
// paths where the container itself does not exist yet.
func (e *Engine) simulateScopeCreation(container config.ResourceContainer, result *Result) {
	for _, configScope := range container.Scopes {
		logger.Info("DRY RUN: Would create scope: %s", configScope.Name)
		e.addOperation(result, "scope", "create", configScope.Name,
			fmt.Sprintf("Would create scope for %s", container.Name), nil)
		result.Summary.ScopesCreated++
	}
}

// syncScopes synchronizes the scopes held by a single resource container
func (e *Engine) syncScopes(container config.ResourceContainer, result *Result) error {
	logger.Debug("Syncing scopes for resource: %s", container.Name)

	// First, get the resource ID
	resources, err := e.client.GetResources()
	if err != nil {
		if e.options.DryRun {
			logger.Warn("DRY RUN: Could not get resources for scope sync: %v", err)
			// In dry-run, simulate scope creation
			e.simulateScopeCreation(container, result)
			return nil
		}
		return fmt.Errorf("failed to get resources: %w", err)
	}

	var resourceID string
	for _, r := range resources {
		if r.Name == container.Name {
			resourceID = r.ID
			break
		}
	}

	if resourceID == "" {
		if e.options.DryRun {
			logger.Debug("DRY RUN: Resource %s not found, would be created", container.Name)
			// In dry-run, the resource would be created, so simulate scope creation
			e.simulateScopeCreation(container, result)
			return nil
		}
		return fmt.Errorf("resource %s not found", container.Name)
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

	// Create the scopes this container holds
	configScopeMap := make(map[string]bool)
	for _, configScope := range container.Scopes {
		configScopeMap[configScope.Name] = true

		if !existingScopeMap[configScope.Name] {
			if e.options.DryRun {
				logger.Info("DRY RUN: Would create scope: %s", configScope.Name)
				e.addOperation(result, "scope", "create", configScope.Name,
					fmt.Sprintf("Would create scope for %s", container.Name), nil)
				result.Summary.ScopesCreated++
			} else {
				logger.Info("Creating scope: %s", configScope.Name)
				err := e.client.CreateScope(resourceID, client.LogtoScope{
					Name:        configScope.Name,
					Description: configScope.Description,
				})
				e.addOperation(result, "scope", "create", configScope.Name,
					fmt.Sprintf("Created scope for %s", container.Name), err)
				if err != nil {
					return fmt.Errorf("failed to create scope %s: %w", configScope.Name, err)
				}
				result.Summary.ScopesCreated++
			}
		} else {
			logger.Debug("Scope %s already exists", configScope.Name)
		}
	}

	// Cleanup phase: remove scopes not in config (only if --cleanup flag is set)
	if e.options.Cleanup {
		for _, existingScope := range existingScopes {
			if !configScopeMap[existingScope.Name] {
				if e.options.DryRun {
					logger.Info("DRY RUN: Would remove scope not in config: %s", existingScope.Name)
					e.addOperation(result, "scope", "cleanup", existingScope.Name,
						fmt.Sprintf("Would remove scope not in config from %s", container.Name), nil)
					result.Summary.ScopesDeleted++
				} else {
					logger.Info("Removing scope not in config: %s", existingScope.Name)
					err := e.client.DeleteScope(resourceID, existingScope.ID)
					e.addOperation(result, "scope", "cleanup", existingScope.Name,
						fmt.Sprintf("Removed scope not in config from %s", container.Name), err)
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
