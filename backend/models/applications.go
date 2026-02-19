/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import (
	"encoding/json"
	"time"
)

// Application represents an application instance extracted from system inventory
type Application struct {
	ID               string          `json:"id" db:"id"`
	SystemID         string          `json:"system_id" db:"system_id"`
	ModuleID         string          `json:"module_id" db:"module_id"`
	InstanceOf       string          `json:"instance_of" db:"instance_of"`
	Name             *string         `json:"name" db:"name"`
	Source           *string         `json:"source" db:"source"`
	DisplayName      *string         `json:"display_name" db:"display_name"`
	NodeID           *int            `json:"node_id" db:"node_id"`
	NodeLabel        *string         `json:"node_label" db:"node_label"`
	Version          *string         `json:"version" db:"version"`
	OrganizationID   *string         `json:"organization_id" db:"organization_id"`
	OrganizationType *string         `json:"organization_type" db:"organization_type"`
	Status           string          `json:"status" db:"status"`
	InventoryData    json.RawMessage `json:"inventory_data" db:"inventory_data"`
	BackupData       json.RawMessage `json:"backup_data" db:"backup_data"`
	ServicesData     json.RawMessage `json:"services_data" db:"services_data"`
	URL              *string         `json:"url" db:"url"`
	Notes            *string         `json:"notes" db:"notes"`
	IsUserFacing     bool            `json:"is_user_facing" db:"is_user_facing"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
	FirstSeenAt      time.Time       `json:"first_seen_at" db:"first_seen_at"`
	LastInventoryAt  *time.Time      `json:"last_inventory_at" db:"last_inventory_at"`
	DeletedAt        *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`

	// Joined data for responses
	System       *SystemSummary       `json:"system,omitempty"`
	Organization *OrganizationSummary `json:"organization,omitempty"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`
}

// SystemSummary represents a minimal system info for application responses
type SystemSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Note: OrganizationSummary is defined in organizations.go

// BackupInfo represents backup status information from inventory
type BackupInfo struct {
	Status          string     `json:"status"` // success, failed, not_run_yet, disabled
	Destination     *string    `json:"destination,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	TotalSizeBytes  *int64     `json:"total_size_bytes,omitempty"`
	TotalFiles      *int       `json:"total_files,omitempty"`
}

// ModuleInfo represents additional module status
type ModuleInfo struct {
	Enabled bool `json:"enabled"`
}

// ServicesInfo represents services health status from inventory
type ServicesInfo struct {
	Services   []ServiceStatus `json:"services"`
	HasErrors  bool            `json:"has_errors"`
	ErrorCount int             `json:"error_count"`
}

// ServiceStatus represents individual service status
type ServiceStatus struct {
	Name   string     `json:"name"`
	Status string     `json:"status"` // running, error, stopped
	Error  *string    `json:"error,omitempty"`
	Since  *time.Time `json:"since,omitempty"`
}

// ApplicationListItem represents a simplified application for list views
type ApplicationListItem struct {
	ID              string               `json:"id"`
	ModuleID        string               `json:"module_id"`
	InstanceOf      string               `json:"instance_of"`
	Name            *string              `json:"name"`
	Source          *string              `json:"source"`
	DisplayName     *string              `json:"display_name"`
	Version         *string              `json:"version"`
	Status          string               `json:"status"`
	NodeID          *int                 `json:"node_id"`
	NodeLabel       *string              `json:"node_label"`
	URL             *string              `json:"url"`
	Notes           *string              `json:"notes"`
	HasErrors       bool                 `json:"has_errors"`
	InventoryData   json.RawMessage      `json:"inventory_data"`
	BackupData      json.RawMessage      `json:"backup_data"`
	ServicesData    json.RawMessage      `json:"services_data"`
	System          *SystemSummary       `json:"system,omitempty"`
	Organization    *OrganizationSummary `json:"organization,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
	LastInventoryAt *time.Time           `json:"last_inventory_at"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`
}

// AssignApplicationRequest represents the request to assign an organization to an application
type AssignApplicationRequest struct {
	OrganizationID string `json:"organization_id" binding:"required"`
}

// UpdateApplicationRequest represents the request to update an application (only notes is editable)
type UpdateApplicationRequest struct {
	Notes *string `json:"notes"`
}

// ApplicationTotals represents statistics for applications
type ApplicationTotals struct {
	Total      int64            `json:"total"`
	Unassigned int64            `json:"unassigned"`
	Assigned   int64            `json:"assigned"`
	WithErrors int64            `json:"with_errors"`
	ByType     map[string]int64 `json:"by_type"`
	ByStatus   map[string]int64 `json:"by_status"`
}

// ApplicationFilters represents available filter options
type ApplicationFilters struct {
	Types     []string `json:"types"`
	Versions  []string `json:"versions"`
	Statuses  []string `json:"statuses"`
	SystemIDs []string `json:"system_ids"`
}

// ApplicationTypeSummary represents applications grouped by type with total count
type ApplicationTypeSummary struct {
	Total      int64             `json:"total"`
	TotalTypes int               `json:"total_types"`
	ByType     []ApplicationType `json:"by_type"`
}

// ApplicationType represents application type metadata for filter dropdowns
type ApplicationType struct {
	InstanceOf string `json:"instance_of"`
	Name       string `json:"name"`
	Count      int64  `json:"count"`
}

// GetEffectiveDisplayName returns the display name or falls back to module_id
func (a *Application) GetEffectiveDisplayName() string {
	if a.DisplayName != nil && *a.DisplayName != "" {
		return *a.DisplayName
	}
	return a.ModuleID
}

// HasServiceErrors checks if the application has service errors from services_data
func (a *Application) HasServiceErrors() bool {
	if a.ServicesData == nil {
		return false
	}
	var info ServicesInfo
	if err := json.Unmarshal(a.ServicesData, &info); err != nil {
		return false
	}
	return info.HasErrors
}

// GetBackupInfo parses and returns backup information
func (a *Application) GetBackupInfo() *BackupInfo {
	if a.BackupData == nil {
		return nil
	}
	var info BackupInfo
	if err := json.Unmarshal(a.BackupData, &info); err != nil {
		return nil
	}
	return &info
}

// GetServicesInfo parses and returns services information
func (a *Application) GetServicesInfo() *ServicesInfo {
	if a.ServicesData == nil {
		return nil
	}
	var info ServicesInfo
	if err := json.Unmarshal(a.ServicesData, &info); err != nil {
		return nil
	}
	return &info
}

// ToListItem converts a full application to a list item
func (a *Application) ToListItem() *ApplicationListItem {
	return &ApplicationListItem{
		ID:                a.ID,
		ModuleID:          a.ModuleID,
		InstanceOf:        a.InstanceOf,
		Name:              a.Name,
		Source:            a.Source,
		DisplayName:       a.DisplayName,
		Version:           a.Version,
		Status:            a.Status,
		NodeID:            a.NodeID,
		NodeLabel:         a.NodeLabel,
		URL:               a.URL,
		Notes:             a.Notes,
		HasErrors:         a.HasServiceErrors(),
		InventoryData:     a.InventoryData,
		BackupData:        a.BackupData,
		ServicesData:      a.ServicesData,
		System:            a.System,
		Organization:      a.Organization,
		CreatedAt:         a.CreatedAt,
		LastInventoryAt:   a.LastInventoryAt,
		RebrandingEnabled: a.RebrandingEnabled,
		RebrandingOrgID:   a.RebrandingOrgID,
	}
}
