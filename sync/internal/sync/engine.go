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
	"encoding/json"
	"fmt"
	"io"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/config"
	"github.com/nethesis/my/sync/internal/logger"
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
	ConfigFile      string
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
	ResourcesCreated    int `json:"resources_created" yaml:"resources_created"`
	ResourcesUpdated    int `json:"resources_updated" yaml:"resources_updated"`
	ResourcesDeleted    int `json:"resources_deleted" yaml:"resources_deleted"`
	RolesCreated        int `json:"roles_created" yaml:"roles_created"`
	RolesUpdated        int `json:"roles_updated" yaml:"roles_updated"`
	RolesDeleted        int `json:"roles_deleted" yaml:"roles_deleted"`
	PermissionsCreated  int `json:"permissions_created" yaml:"permissions_created"`
	PermissionsUpdated  int `json:"permissions_updated" yaml:"permissions_updated"`
	PermissionsDeleted  int `json:"permissions_deleted" yaml:"permissions_deleted"`
	ScopesCreated       int `json:"scopes_created" yaml:"scopes_created"`
	ScopesUpdated       int `json:"scopes_updated" yaml:"scopes_updated"`
	ScopesDeleted       int `json:"scopes_deleted" yaml:"scopes_deleted"`
	ApplicationsCreated int `json:"applications_created" yaml:"applications_created"`
	ApplicationsUpdated int `json:"applications_updated" yaml:"applications_updated"`
	ApplicationsDeleted int `json:"applications_deleted" yaml:"applications_deleted"`
	ConnectorsSynced    int `json:"connectors_synced" yaml:"connectors_synced"`
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

	// Sync third-party applications
	if len(cfg.ThirdPartyApps) > 0 {
		if err := e.syncThirdPartyApplications(cfg, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Third-party applications sync failed: %v", err))
		}
	}

	// Sync sign-in experience
	if cfg.SignInExperience != nil {
		if err := e.syncSignInExperience(cfg, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Sign-in experience sync failed: %v", err))
		}
	}

	// Sync SMTP connector from configuration
	if cfg.Connectors != nil && cfg.Connectors.SMTP != nil {
		if err := e.syncSMTPConnector(cfg.Connectors.SMTP, result); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("SMTP connector sync failed: %v", err))
		}
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
		logger.LogSyncOperation(opType, resource, action, false, err)
	} else {
		logger.LogSyncOperation(opType, resource, action, true, nil)
	}

	result.Operations = append(result.Operations, op)
}

// OutputText outputs the result in text format
func (r *Result) OutputText(w io.Writer) error {
	_, _ = fmt.Fprintf(w, "Synchronization Results\n")
	_, _ = fmt.Fprintf(w, "======================\n\n")
	_, _ = fmt.Fprintf(w, "Status: %s\n", map[bool]string{true: "SUCCESS", false: "FAILED"}[r.Success])
	_, _ = fmt.Fprintf(w, "Duration: %v\n", r.Duration)
	_, _ = fmt.Fprintf(w, "Dry Run: %v\n\n", r.DryRun)

	_, _ = fmt.Fprintf(w, "Summary:\n")
	_, _ = fmt.Fprintf(w, "  Resources: %d created, %d updated, %d deleted\n",
		r.Summary.ResourcesCreated, r.Summary.ResourcesUpdated, r.Summary.ResourcesDeleted)
	_, _ = fmt.Fprintf(w, "  Roles: %d created, %d updated, %d deleted\n",
		r.Summary.RolesCreated, r.Summary.RolesUpdated, r.Summary.RolesDeleted)
	_, _ = fmt.Fprintf(w, "  Permissions: %d created, %d updated, %d deleted\n",
		r.Summary.PermissionsCreated, r.Summary.PermissionsUpdated, r.Summary.PermissionsDeleted)
	_, _ = fmt.Fprintf(w, "  Scopes: %d created, %d updated, %d deleted\n",
		r.Summary.ScopesCreated, r.Summary.ScopesUpdated, r.Summary.ScopesDeleted)
	_, _ = fmt.Fprintf(w, "  Applications: %d created, %d updated, %d deleted\n",
		r.Summary.ApplicationsCreated, r.Summary.ApplicationsUpdated, r.Summary.ApplicationsDeleted)
	_, _ = fmt.Fprintf(w, "  Connectors: %d synced\n\n", r.Summary.ConnectorsSynced)

	if len(r.Errors) > 0 {
		_, _ = fmt.Fprintf(w, "Errors:\n")
		for _, err := range r.Errors {
			_, _ = fmt.Fprintf(w, "  - %s\n", err)
		}
		_, _ = fmt.Fprintf(w, "\n")
	}

	if len(r.Operations) > 0 {
		_, _ = fmt.Fprintf(w, "Operations:\n")
		for _, op := range r.Operations {
			status := "✓"
			if !op.Success {
				status = "✗"
			}
			_, _ = fmt.Fprintf(w, "  %s %s %s %s - %s\n", status, op.Type, op.Action, op.Resource, op.Description)
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
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(r)
}

// syncSMTPConnector synchronizes SMTP connector configuration
func (e *Engine) syncSMTPConnector(smtpConfig *config.SMTPConnector, result *Result) error {
	logger.Info("Synchronizing SMTP connector configuration")

	if e.options.DryRun {
		logger.Info("DRY RUN: Would sync SMTP connector with host: %s", smtpConfig.Host)
		e.addOperation(result, "connector", "sync", "smtp", "SMTP connector configuration", nil)
		result.Summary.ConnectorsSynced++
		return nil
	}

	err := e.client.SyncSMTPConnector(smtpConfig)
	e.addOperation(result, "connector", "sync", "smtp", "SMTP connector configuration", err)

	if err != nil {
		return fmt.Errorf("failed to sync SMTP connector: %w", err)
	}

	result.Summary.ConnectorsSynced++
	return nil
}
