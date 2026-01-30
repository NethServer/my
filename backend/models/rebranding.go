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

// RebrandingProductStatus represents a product's rebranding status for an organization
type RebrandingProductStatus struct {
	ProductID          string   `json:"product_id"`
	ProductDisplayName string   `json:"product_display_name"`
	ProductType        string   `json:"product_type"`
	ProductName        *string  `json:"product_name"`
	Assets             []string `json:"assets"` // list of uploaded asset names
}

// RebrandingOrgStatus represents the full rebranding status for an organization
type RebrandingOrgStatus struct {
	Enabled  bool                      `json:"enabled"`
	Products []RebrandingProductStatus `json:"products"`
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
