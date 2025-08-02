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

// CreateRoleNameToIDMapping creates a mapping from role names to IDs
func CreateRoleNameToIDMapping(roles []client.LogtoRole) map[string]string {
	mapping := make(map[string]string)
	for _, role := range roles {
		mapping[strings.ToLower(role.Name)] = role.ID
	}
	return mapping
}

// CreateResourceNameToIDMapping creates a mapping from resource names to IDs
func CreateResourceNameToIDMapping(resources []client.LogtoResource) map[string]string {
	mapping := make(map[string]string)
	for _, resource := range resources {
		mapping[resource.Name] = resource.ID
	}
	return mapping
}

// CreateScopeNameToIDMapping creates a mapping from scope names to IDs
func CreateScopeNameToIDMapping(scopes []client.LogtoScope) map[string]string {
	mapping := make(map[string]string)
	for _, scope := range scopes {
		mapping[scope.Name] = scope.ID
	}
	return mapping
}

// SystemEntityDetector checks if an entity is system-managed
type SystemEntityDetector struct {
	systemPatterns []string
}

// NewSystemEntityDetector creates a new system entity detector
func NewSystemEntityDetector() *SystemEntityDetector {
	return &SystemEntityDetector{
		systemPatterns: []string{
			"logto:",
			"urn:logto:",
			"management api",
			"machine to machine",
		},
	}
}

// IsSystemEntity checks if an entity is system-managed based on name and description
func (d *SystemEntityDetector) IsSystemEntity(name, description string) bool {
	nameUpper := strings.ToUpper(name)
	descUpper := strings.ToUpper(description)

	for _, pattern := range d.systemPatterns {
		patternUpper := strings.ToUpper(pattern)
		if strings.Contains(nameUpper, patternUpper) || strings.Contains(descUpper, patternUpper) {
			return true
		}
	}
	return false
}

// ScopeMapping represents a mapping between scope names and IDs
type ScopeMapping struct {
	NameToID map[string]string
	IDToName map[string]string
}

// BuildGlobalScopeMapping builds a comprehensive scope mapping from all resources
func BuildGlobalScopeMapping(client *client.LogtoClient, cfg *config.Config, resourceNameToID map[string]string) (*ScopeMapping, error) {
	allScopeNameToID := make(map[string]string)
	allScopeIDToName := make(map[string]string)

	for _, configResource := range cfg.Resources {
		resourceID, exists := resourceNameToID[configResource.Name]
		if !exists {
			syncLogger := logger.ComponentLogger("sync")
			syncLogger.Warn().
				Str("resource", configResource.Name).
				Msg("Resource not found, skipping scope mappings")
			continue
		}

		scopes, err := client.GetScopes(resourceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get scopes for resource %s: %w", configResource.Name, err)
		}

		for _, scope := range scopes {
			allScopeNameToID[scope.Name] = scope.ID
			allScopeIDToName[scope.ID] = scope.Name
		}
	}

	return &ScopeMapping{
		NameToID: allScopeNameToID,
		IDToName: allScopeIDToName,
	}, nil
}

// PermissionDiff represents the difference between current and desired permissions
type PermissionDiff struct {
	ToAdd    []string
	ToRemove []string
}

// CalculatePermissionDiff calculates what permissions need to be added or removed
func CalculatePermissionDiff(current []string, desired []string) *PermissionDiff {
	currentMap := make(map[string]bool)
	for _, perm := range current {
		currentMap[perm] = true
	}

	desiredMap := make(map[string]bool)
	for _, perm := range desired {
		desiredMap[perm] = true
	}

	diff := &PermissionDiff{
		ToAdd:    make([]string, 0),
		ToRemove: make([]string, 0),
	}

	// Find permissions to add
	for perm := range desiredMap {
		if !currentMap[perm] {
			diff.ToAdd = append(diff.ToAdd, perm)
		}
	}

	// Find permissions to remove (excluding system permissions)
	detector := NewSystemEntityDetector()
	for perm := range currentMap {
		if !desiredMap[perm] && !detector.IsSystemEntity(perm, "") {
			diff.ToRemove = append(diff.ToRemove, perm)
		}
	}

	return diff
}
