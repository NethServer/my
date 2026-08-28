/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// RebrandableProduct represents a product that supports rebranding
type RebrandableProduct struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Type        string    `json:"type"` // "application" or "system"
	CreatedAt   time.Time `json:"created_at"`
}

// RebrandingEnabled represents the rebranding enablement status for an organization
type RebrandingEnabled struct {
	OrganizationID   string    `json:"organization_id"`
	OrganizationType string    `json:"organization_type"`
	EnabledAt        time.Time `json:"enabled_at"`
}

// RebrandingAsset represents rebranding configuration and assets for an organization+product
type RebrandingAsset struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ProductID      string    `json:"product_id"`
	ProductName    *string   `json:"product_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// Assets are not included in JSON responses; served via dedicated endpoints
}

// RebrandingAssetInfo describes one uploaded asset. The upload form renders the
// asset it already holds from these fields, and UpdatedAt doubles as the
// cache-busting key for the binary endpoint.
type RebrandingAssetInfo struct {
	Name      string    `json:"name"`
	Filename  *string   `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RebrandingProductStatus represents a product's rebranding status for an organization
type RebrandingProductStatus struct {
	ProductID          string                `json:"product_id"`
	ProductDisplayName string                `json:"product_display_name"`
	ProductType        string                `json:"product_type"`
	ProductName        *string               `json:"product_name"`
	Assets             []RebrandingAssetInfo `json:"assets"`
	UpdatedAt          *time.Time            `json:"updated_at"`
}

// RebrandingOrgStatus represents the full rebranding status for an organization
type RebrandingOrgStatus struct {
	Enabled  bool                      `json:"enabled"`
	Products []RebrandingProductStatus `json:"products"`
}

// RebrandingOrganizationProduct is a product an organization has branded, as
// shown in the "Branded products" column.
type RebrandingOrganizationProduct struct {
	ProductID          string  `json:"product_id"`
	ProductDisplayName string  `json:"product_display_name"`
	ProductName        *string `json:"product_name"`
}

// RebrandingOrganization is one row of the organizations-with-rebranding list.
type RebrandingOrganization struct {
	ID               string                          `json:"id"`
	LogtoID          string                          `json:"logto_id"`
	Name             string                          `json:"name"`
	OrganizationType string                          `json:"organization_type"`
	Products         []RebrandingOrganizationProduct `json:"products"`
	EnabledAt        time.Time                       `json:"enabled_at"`
	UpdatedAt        *time.Time                      `json:"updated_at"`
}

// RebrandingAvailableOrganization is an organization that can still be added to
// rebranding: it exists, and it is not enabled yet.
type RebrandingAvailableOrganization struct {
	ID               string `json:"id"`
	LogtoID          string `json:"logto_id"`
	Name             string `json:"name"`
	OrganizationType string `json:"organization_type"`
}

// RebrandingSummary feeds the counters above the list.
type RebrandingSummary struct {
	Total        int `json:"total"`
	Distributors int `json:"distributors"`
	Resellers    int `json:"resellers"`
	Customers    int `json:"customers"`
}

// EnableRebrandingBulkRequest adds several organizations to rebranding at once.
type EnableRebrandingBulkRequest struct {
	OrganizationIDs []string `json:"organization_ids" binding:"required,min=1,max=200,dive,required"`
}

// RebrandingUpload is one asset arriving from the configuration form.
type RebrandingUpload struct {
	Data     []byte
	MimeType string
	Filename string
}

// RebrandingConfig is a whole configuration form: the products the branding
// applies to, the brand name, the assets being written and the ones being
// emptied. Saving it is a single transaction, so a form that replaces one logo
// and clears another cannot land half-applied.
type RebrandingConfig struct {
	Products  []string
	BrandName *string
	Uploads   map[string]RebrandingUpload
	Clear     []string
}

// EnableRebrandingRequest represents the request to enable rebranding for an org
type EnableRebrandingRequest struct {
	OrganizationType string `json:"organization_type" binding:"required"`
}

// SystemRebrandingProduct represents a rebranded product for system consumption
type SystemRebrandingProduct struct {
	ProductID   string            `json:"product_id"`
	ProductName *string           `json:"product_name"`
	Assets      map[string]string `json:"assets"` // asset_name -> URL path
}

// SystemRebrandingResponse represents the full rebranding response for a system
type SystemRebrandingResponse struct {
	Enabled       bool                      `json:"enabled"`
	InheritedFrom *string                   `json:"inherited_from"` // null if own config, "distributor:org_id" if inherited
	System        []SystemRebrandingProduct `json:"system"`
	Applications  []SystemRebrandingProduct `json:"applications"`
}
