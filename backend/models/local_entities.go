/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

import (
	"encoding/json"
	"time"
)

// LocalDistributor represents a distributor stored in local database
type LocalDistributor struct {
	ID             string                 `json:"id" db:"id"`
	LogtoID        *string                `json:"logto_id" db:"logto_id"`
	Name           string                 `json:"name" db:"name"`
	Description    string                 `json:"description" db:"description"`
	CustomData     map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt  *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LogtoSyncError *string                `json:"logto_sync_error" db:"logto_sync_error"`
	DeletedAt      *time.Time             `json:"deleted_at" db:"deleted_at"`
	SuspendedAt    *time.Time             `json:"suspended_at" db:"suspended_at"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`

	// Inline stats (populated by List queries only, omitted in other responses)
	SystemsCount      *int `json:"systems_count,omitempty"`
	ResellersCount    *int `json:"resellers_count,omitempty"`
	CustomersCount    *int `json:"customers_count,omitempty"`
	ApplicationsCount *int `json:"applications_count,omitempty"`

	// Creator snapshot (resolved from custom_data.createdByUser at read time)
	CreatedBy *OrgCreator `json:"created_by,omitempty"`
}

// OrgCreator is a point-in-time snapshot of the user who created an organization.
// It is stored in custom_data.createdByUser at creation and surfaced as the
// top-level created_by field on the organization, mirroring SystemCreator.
type OrgCreator struct {
	UserID           string `json:"user_id"`
	Username         string `json:"username"`
	Name             string `json:"name"`
	Email            string `json:"email"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	// OnBehalfOf is true when the entity was attributed to a different org via
	// created_by_organization_id: the user acted on behalf of organization_name
	// rather than belonging to it. Lets the UI render "created by <user> on
	// behalf of <org>". Omitted (false) on the default own-org path.
	OnBehalfOf bool `json:"on_behalf_of,omitempty"`
}

// NewOrgCreatorFromUser builds an OrgCreator snapshot from the authenticated user.
func NewOrgCreatorFromUser(u User) *OrgCreator {
	logtoID := ""
	if u.LogtoID != nil {
		logtoID = *u.LogtoID
	}
	return &OrgCreator{
		UserID:           logtoID,
		Username:         u.Username,
		Name:             u.Name,
		Email:            u.Email,
		OrganizationID:   u.OrganizationID,
		OrganizationName: u.OrganizationName,
	}
}

// AttributeToOrg re-points the creator snapshot's organization to an attributed
// owner org (the one resolved from created_by_organization_id) while preserving
// the user identity that actually performed the action. This mirrors the system
// creator behaviour: the top-level created_by shows the owning reseller/distributor
// rather than the API/distributor account that made the call. It is a no-op when
// orgID/orgName are empty or already match the creator's org, so the default
// "own org" path is unaffected.
func (c *OrgCreator) AttributeToOrg(orgID, orgName string) {
	if c == nil || orgID == "" || orgName == "" || orgID == c.OrganizationID {
		return
	}
	c.OrganizationID = orgID
	c.OrganizationName = orgName
	c.OnBehalfOf = true
}

// ExtractOrgCreator pulls the createdByUser snapshot out of an organization's
// custom_data (populated at creation time), returning it as a typed value and
// removing the raw key from the map so it is exposed only as the top-level
// created_by field. Returns nil when no creator snapshot is present.
func ExtractOrgCreator(customData map[string]interface{}) *OrgCreator {
	if customData == nil {
		return nil
	}
	raw, ok := customData["createdByUser"]
	if !ok || raw == nil {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var creator OrgCreator
	if err := json.Unmarshal(encoded, &creator); err != nil {
		return nil
	}
	delete(customData, "createdByUser")
	return &creator
}

// LocalReseller represents a reseller stored in local database
type LocalReseller struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	CustomData       map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LogtoSyncError   *string                `json:"logto_sync_error" db:"logto_sync_error"`
	DeletedAt        *time.Time             `json:"deleted_at" db:"deleted_at"`
	SuspendedAt      *time.Time             `json:"suspended_at" db:"suspended_at"`
	SuspendedByOrgID *string                `json:"suspended_by_org_id" db:"suspended_by_org_id"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`

	// Inline stats (populated by List queries only, omitted in other responses)
	SystemsCount      *int `json:"systems_count,omitempty"`
	CustomersCount    *int `json:"customers_count,omitempty"`
	ApplicationsCount *int `json:"applications_count,omitempty"`

	// Creator snapshot (resolved from custom_data.createdByUser at read time)
	CreatedBy *OrgCreator `json:"created_by,omitempty"`
}

// LocalCustomer represents a customer stored in local database
type LocalCustomer struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	CustomData       map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LogtoSyncError   *string                `json:"logto_sync_error" db:"logto_sync_error"`
	DeletedAt        *time.Time             `json:"deleted_at" db:"deleted_at"`
	SuspendedAt      *time.Time             `json:"suspended_at" db:"suspended_at"`
	SuspendedByOrgID *string                `json:"suspended_by_org_id" db:"suspended_by_org_id"`

	// Rebranding info (populated by handler)
	RebrandingEnabled bool    `json:"rebranding_enabled"`
	RebrandingOrgID   *string `json:"rebranding_org_id,omitempty"`

	// Inline stats (populated by List queries only, omitted in other responses)
	SystemsCount      *int `json:"systems_count,omitempty"`
	ApplicationsCount *int `json:"applications_count,omitempty"`

	// Creator snapshot (resolved from custom_data.createdByUser at read time)
	CreatedBy *OrgCreator `json:"created_by,omitempty"`
}

// CustomerFilters represents filters for customer queries
type CustomerFilters struct {
	Search string `json:"search,omitempty"` // general search term
	Status string `json:"status,omitempty"` // enabled, blocked, or empty for all
}

// DistributorFilters represents filters for distributor queries
type DistributorFilters struct {
	Search string `json:"search,omitempty"` // general search term
	Status string `json:"status,omitempty"` // enabled, blocked, or empty for all
}

// ResellerFilters represents filters for reseller queries
type ResellerFilters struct {
	Search string `json:"search,omitempty"` // general search term
	Status string `json:"status,omitempty"` // enabled, blocked, or empty for all
}

// UserOrganization represents organization info in user responses
type UserOrganization struct {
	ID      string `json:"id"`
	LogtoID string `json:"logto_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// UserRole represents role info in user responses
type UserRole struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LocalUser represents a user stored in local database
type LocalUser struct {
	ID               string                 `json:"id" db:"id"`
	LogtoID          *string                `json:"logto_id" db:"logto_id"`
	Username         string                 `json:"username" db:"username"`
	Email            string                 `json:"email" db:"email"`
	Name             string                 `json:"name" db:"name"`
	Phone            *string                `json:"phone" db:"phone"`
	Organization     *UserOrganization      `json:"organization,omitempty"`
	Roles            []UserRole             `json:"roles,omitempty"`
	CustomData       map[string]interface{} `json:"custom_data" db:"custom_data"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	LogtoSyncedAt    *time.Time             `json:"logto_synced_at" db:"logto_synced_at"`
	LatestLoginAt    *time.Time             `json:"latest_login_at" db:"latest_login_at"`
	DeletedAt        *time.Time             `json:"deleted_at" db:"deleted_at"`                   // Soft delete timestamp
	SuspendedAt      *time.Time             `json:"suspended_at" db:"suspended_at"`               // Suspension timestamp
	SuspendedByOrgID *string                `json:"suspended_by_org_id" db:"suspended_by_org_id"` // Organization that caused cascade suspension
	DeletedByOrgID   *string                `json:"deleted_by_org_id" db:"deleted_by_org_id"`     // Organization that caused cascade soft-deletion

	// Creator snapshot (set at creation, display/filter only - RBAC is by organization_id)
	CreatedBy *OrgCreator `json:"created_by,omitempty" db:"created_by"`

	// Internal fields for database operations (not serialized to JSON)
	UserRoleIDs         []string `json:"-" db:"user_role_ids"`
	OrganizationID      *string  `json:"-" db:"organization_id"`
	OrganizationName    *string  `json:"-"`
	OrganizationLocalID *string  `json:"-"`
	OrganizationType    *string  `json:"-"`
}

// SystemTotals represents total counts and status for systems
type SystemTotals struct {
	Total          int `json:"total"`
	Active         int `json:"active"`
	Inactive       int `json:"inactive"`
	Unknown        int `json:"unknown"`
	TimeoutMinutes int `json:"timeout_minutes"`
}

// UserTotals represents total counts and status breakdown for user accounts
type UserTotals struct {
	Total     int `json:"total"`
	Enabled   int `json:"enabled"`
	Suspended int `json:"suspended"`
}

// OrganizationStats represents statistics for an organization (users and systems count)
type OrganizationStats struct {
	UsersCount   int `json:"users_count"`
	SystemsCount int `json:"systems_count"`
}

// DistributorStats represents statistics for a distributor (includes resellers, customers, and applications)
type DistributorStats struct {
	UsersCount                 int `json:"users_count"`
	UsersHierarchyCount        int `json:"users_hierarchy_count"`
	SystemsCount               int `json:"systems_count"`
	SystemsHierarchyCount      int `json:"systems_hierarchy_count"`
	ResellersCount             int `json:"resellers_count"`
	CustomersCount             int `json:"customers_count"`
	ApplicationsCount          int `json:"applications_count"`           // direct applications
	ApplicationsHierarchyCount int `json:"applications_hierarchy_count"` // applications in hierarchy
}

// ResellerStats represents statistics for a reseller (includes customers and applications)
type ResellerStats struct {
	UsersCount                 int `json:"users_count"`
	UsersHierarchyCount        int `json:"users_hierarchy_count"`
	SystemsCount               int `json:"systems_count"`
	SystemsHierarchyCount      int `json:"systems_hierarchy_count"`
	CustomersCount             int `json:"customers_count"`
	ApplicationsCount          int `json:"applications_count"`           // direct applications
	ApplicationsHierarchyCount int `json:"applications_hierarchy_count"` // applications in hierarchy
}

// CustomerStats represents statistics for a customer (includes applications)
type CustomerStats struct {
	UsersCount        int `json:"users_count"`
	SystemsCount      int `json:"systems_count"`
	ApplicationsCount int `json:"applications_count"` // direct applications only (leaf node)
}

// Create requests
type CreateLocalDistributorRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
}

type CreateLocalResellerRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	// CreatedByOrganizationID, when set by an owner, attributes the new reseller
	// to that ancestor distributor (its custom_data.createdBy) instead of the
	// caller's own org. Empty = owned by the caller's org. See
	// LocalOrganizationService.ResolveCreatedByOrg.
	CreatedByOrganizationID string `json:"created_by_organization_id,omitempty"`
}

type CreateLocalCustomerRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=255"`
	Description string                 `json:"description,omitempty"`
	CustomData  map[string]interface{} `json:"custom_data,omitempty"`
	// CreatedByOrganizationID, when set by an owner or distributor, attributes
	// the new customer to that ancestor org (its custom_data.createdBy) instead
	// of the caller's own org — used to preserve hierarchical ownership when an
	// upper tier creates a customer on behalf of a reseller (e.g. the migration
	// import). Empty = owned by the caller's org. See
	// LocalOrganizationService.ResolveCreatedByOrg.
	CreatedByOrganizationID string `json:"created_by_organization_id,omitempty"`
}

// Update requests
type UpdateLocalDistributorRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	CustomData  *map[string]interface{} `json:"custom_data,omitempty"`
}

type UpdateLocalResellerRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	CustomData  *map[string]interface{} `json:"custom_data,omitempty"`
}

type UpdateLocalCustomerRequest struct {
	Name        *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string                 `json:"description,omitempty"`
	CustomData  *map[string]interface{} `json:"custom_data,omitempty"`
}

// User CRUD requests
type CreateLocalUserRequest struct {
	Username       string                 `json:"username,omitempty" validate:"omitempty,min=1,max=255"`
	Email          string                 `json:"email" validate:"required,email,max=255"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	Phone          *string                `json:"phone,omitempty"`
	UserRoleIDs    []string               `json:"user_role_ids,omitempty"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	CustomData     map[string]interface{} `json:"custom_data,omitempty"`

	// CreatedBy is the creator snapshot, set by the service from the
	// authenticated user - never bound from the request body.
	CreatedBy *OrgCreator `json:"-"`
}

type UpdateLocalUserRequest struct {
	Username       *string                 `json:"username,omitempty" validate:"omitempty,min=1,max=255"`
	Email          *string                 `json:"email,omitempty" validate:"omitempty,email,max=255"`
	Name           *string                 `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Phone          *string                 `json:"phone,omitempty"`
	UserRoleIDs    *[]string               `json:"user_role_ids,omitempty"`
	OrganizationID *string                 `json:"organization_id,omitempty"`
	CustomData     *map[string]interface{} `json:"custom_data,omitempty"`
}

// Suspension/reactivation requests
type SuspendUserRequest struct {
	Reason *string `json:"reason,omitempty"`
}

type ReactivateUserRequest struct {
	Reason *string `json:"reason,omitempty"`
}

// Active returns true if the user is not deleted and not suspended
func (u *LocalUser) Active() bool {
	return u.DeletedAt == nil && u.SuspendedAt == nil
}

// IsDeleted returns true if the user is soft-deleted
func (u *LocalUser) IsDeleted() bool {
	return u.DeletedAt != nil
}

// IsSuspended returns true if the user is suspended
func (u *LocalUser) IsSuspended() bool {
	return u.SuspendedAt != nil
}

// Active returns true if the distributor is not deleted and not suspended
func (d *LocalDistributor) Active() bool {
	return d.DeletedAt == nil && d.SuspendedAt == nil
}

// IsDeleted returns true if the distributor is soft-deleted
func (d *LocalDistributor) IsDeleted() bool {
	return d.DeletedAt != nil
}

// IsSuspended returns true if the distributor is suspended
func (d *LocalDistributor) IsSuspended() bool {
	return d.SuspendedAt != nil
}

// Active returns true if the reseller is not deleted and not suspended
func (r *LocalReseller) Active() bool {
	return r.DeletedAt == nil && r.SuspendedAt == nil
}

// IsDeleted returns true if the reseller is soft-deleted
func (r *LocalReseller) IsDeleted() bool {
	return r.DeletedAt != nil
}

// IsSuspended returns true if the reseller is suspended
func (r *LocalReseller) IsSuspended() bool {
	return r.SuspendedAt != nil
}

// Active returns true if the customer is not deleted and not suspended
func (c *LocalCustomer) Active() bool {
	return c.DeletedAt == nil && c.SuspendedAt == nil
}

// IsDeleted returns true if the customer is soft-deleted
func (c *LocalCustomer) IsDeleted() bool {
	return c.DeletedAt != nil
}

// IsSuspended returns true if the customer is suspended
func (c *LocalCustomer) IsSuspended() bool {
	return c.SuspendedAt != nil
}

// VATValidationResponse represents a VAT validation response
type VATValidationResponse struct {
	Exists bool `json:"exists"`
}
