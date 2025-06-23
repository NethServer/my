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
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/logto-sync/internal/client"
	"github.com/nethesis/my/logto-sync/internal/config"
	"github.com/nethesis/my/logto-sync/internal/logger"
)

// Engine handles the synchronization process
type Engine struct {
	client  *client.LogtoClient
	options *Options
}

// Options contains synchronization options
type Options struct {
	DryRun          bool
	Verbose         bool
	SkipResources   bool
	SkipRoles       bool
	SkipPermissions bool
	Cleanup         bool
	APIBaseURL      string
}

// Result contains the results of a synchronization
type Result struct {
	StartTime  time.Time     `json:"start_time" yaml:"start_time"`
	EndTime    time.Time     `json:"end_time" yaml:"end_time"`
	Duration   time.Duration `json:"duration" yaml:"duration"`
	DryRun     bool          `json:"dry_run" yaml:"dry_run"`
	Success    bool          `json:"success" yaml:"success"`
	Summary    *Summary      `json:"summary" yaml:"summary"`
	Operations []Operation   `json:"operations" yaml:"operations"`
	Errors     []string      `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// Summary contains a summary of changes
type Summary struct {
	ResourcesCreated   int `json:"resources_created" yaml:"resources_created"`
	ResourcesUpdated   int `json:"resources_updated" yaml:"resources_updated"`
	ResourcesDeleted   int `json:"resources_deleted" yaml:"resources_deleted"`
	RolesCreated       int `json:"roles_created" yaml:"roles_created"`
	RolesUpdated       int `json:"roles_updated" yaml:"roles_updated"`
	RolesDeleted       int `json:"roles_deleted" yaml:"roles_deleted"`
	PermissionsCreated int `json:"permissions_created" yaml:"permissions_created"`
	PermissionsUpdated int `json:"permissions_updated" yaml:"permissions_updated"`
	PermissionsDeleted int `json:"permissions_deleted" yaml:"permissions_deleted"`
	ScopesCreated      int `json:"scopes_created" yaml:"scopes_created"`
	ScopesUpdated      int `json:"scopes_updated" yaml:"scopes_updated"`
	ScopesDeleted      int `json:"scopes_deleted" yaml:"scopes_deleted"`
}

// Operation represents a single operation performed
type Operation struct {
	Type        string    `json:"type" yaml:"type"`
	Action      string    `json:"action" yaml:"action"`
	Resource    string    `json:"resource" yaml:"resource"`
	Description string    `json:"description" yaml:"description"`
	Success     bool      `json:"success" yaml:"success"`
	Error       string    `json:"error,omitempty" yaml:"error,omitempty"`
	Timestamp   time.Time `json:"timestamp" yaml:"timestamp"`
}

// NewEngine creates a new synchronization engine
func NewEngine(client *client.LogtoClient, options *Options) *Engine {
	if options == nil {
		options = &Options{}
	}

	return &Engine{
		client:  client,
		options: options,
	}
}

// Sync performs the synchronization
func (e *Engine) Sync(cfg *config.Config) (*Result, error) {
	result := &Result{
		StartTime:  time.Now(),
		DryRun:     e.options.DryRun,
		Summary:    &Summary{},
		Operations: []Operation{},
		Errors:     []string{},
	}

	logger.Info("Starting synchronization for %s v%s", cfg.Metadata.Name, cfg.Metadata.Version)

	if e.options.DryRun {
		logger.Info("Running in dry-run mode - no changes will be made")
	}

	// Sync resources first (needed for scopes)
	if !e.options.SkipResources {
		if err := e.syncResources(cfg, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Resource sync failed: %v", err))
		}
	}

	// Sync organization scopes (needed for organization roles)
	if !e.options.SkipPermissions {
		if err := e.syncOrganizationScopes(cfg, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Organization scopes sync failed: %v", err))
		}
	}

	// Sync organization roles
	if !e.options.SkipRoles {
		if err := e.syncOrganizationRoles(cfg, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Organization roles sync failed: %v", err))
		}

		// Sync organization role scopes
		if !e.options.SkipPermissions {
			if err := e.syncOrganizationRoleScopes(cfg, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Organization role scopes sync failed: %v", err))
			}
		}
	}

	// Sync user roles
	if !e.options.SkipRoles {
		if err := e.syncUserRoles(cfg, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("User roles sync failed: %v", err))
		}

		// Sync user role permissions
		if !e.options.SkipPermissions {
			if err := e.syncUserRolePermissions(cfg, result); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("User role permissions sync failed: %v", err))
			}
		}
	}

	// Sync customizations
	if err := e.syncCustomizations(cfg, result); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Customizations sync failed: %v", err))
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
	result.Success = len(result.Errors) == 0

	if result.Success {
		logger.Info("Synchronization completed successfully in %v", result.Duration)
	} else {
		logger.Error("Synchronization completed with %d errors in %v", len(result.Errors), result.Duration)
	}

	return result, nil
}

// addOperation adds an operation to the result
func (e *Engine) addOperation(result *Result, opType, action, resource, description string, err error) {
	op := Operation{
		Type:        opType,
		Action:      action,
		Resource:    resource,
		Description: description,
		Success:     err == nil,
		Timestamp:   time.Now(),
	}

	if err != nil {
		op.Error = err.Error()
		logger.Error("Operation failed: %s %s %s - %v", opType, action, resource, err)
	} else {
		logger.Info("Operation successful: %s %s %s - %s", opType, action, resource, description)
	}

	result.Operations = append(result.Operations, op)
}

// syncCustomizations synchronizes Logto customizations
func (e *Engine) syncCustomizations(cfg *config.Config, result *Result) error {
	return SyncCustomizations(e.client, cfg, e.options.DryRun)
}

// OutputText outputs the result in text format
func (r *Result) OutputText(w io.Writer) error {
	fmt.Fprintf(w, "Synchronization Results\n")
	fmt.Fprintf(w, "======================\n\n")
	fmt.Fprintf(w, "Status: %s\n", map[bool]string{true: "SUCCESS", false: "FAILED"}[r.Success])
	fmt.Fprintf(w, "Duration: %v\n", r.Duration)
	fmt.Fprintf(w, "Dry Run: %v\n\n", r.DryRun)

	fmt.Fprintf(w, "Summary:\n")
	fmt.Fprintf(w, "  Resources: %d created, %d updated, %d deleted\n",
		r.Summary.ResourcesCreated, r.Summary.ResourcesUpdated, r.Summary.ResourcesDeleted)
	fmt.Fprintf(w, "  Roles: %d created, %d updated, %d deleted\n",
		r.Summary.RolesCreated, r.Summary.RolesUpdated, r.Summary.RolesDeleted)
	fmt.Fprintf(w, "  Permissions: %d created, %d updated, %d deleted\n",
		r.Summary.PermissionsCreated, r.Summary.PermissionsUpdated, r.Summary.PermissionsDeleted)
	fmt.Fprintf(w, "  Scopes: %d created, %d updated, %d deleted\n\n",
		r.Summary.ScopesCreated, r.Summary.ScopesUpdated, r.Summary.ScopesDeleted)

	if len(r.Errors) > 0 {
		fmt.Fprintf(w, "Errors:\n")
		for _, err := range r.Errors {
			fmt.Fprintf(w, "  - %s\n", err)
		}
		fmt.Fprintf(w, "\n")
	}

	if len(r.Operations) > 0 {
		fmt.Fprintf(w, "Operations:\n")
		for _, op := range r.Operations {
			status := "✓"
			if !op.Success {
				status = "✗"
			}
			fmt.Fprintf(w, "  %s %s %s %s - %s\n", status, op.Type, op.Action, op.Resource, op.Description)
		}
	}

	return nil
}

// OutputJSON outputs the result in JSON format
func (r *Result) OutputJSON(w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(r)
}

// OutputYAML outputs the result in YAML format
func (r *Result) OutputYAML(w io.Writer) error {
	encoder := yaml.NewEncoder(w)
	defer encoder.Close()
	return encoder.Encode(r)
}
