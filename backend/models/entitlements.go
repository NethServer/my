/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// Entitlement grant sources.
const (
	EntitlementSourceManual       = "manual"
	EntitlementSourceShop         = "shop"
	EntitlementSourceLegacyImport = "legacy-import"
)

// Entitlement grant lifecycle statuses (server-computed, the single source
// of truth for UI badges). Revoked and expired are facts of the grant
// itself; suspended means the grant is fine but the system — directly or
// through its organization's suspension/deletion cascade — cannot use it
// (collect rejects its credentials before even looking at grants).
const (
	EntitlementStatusActive    = "active"
	EntitlementStatusExpired   = "expired"
	EntitlementStatusRevoked   = "revoked"
	EntitlementStatusSuspended = "suspended"
	EntitlementStatusPending   = "pending"
)

// EntitlementStatus derives the lifecycle status of a grant. `active` is the
// SQL-computed grant validity (not revoked, not expired), `systemBlocked`
// whether the owning system is suspended or deleted, `pendingRef` a shop
// order awaiting payment. Pending masks revoked/expired (an order is already
// on its way — the UI must not offer another purchase) but never an active
// grant (a pending renewal doesn't change what the customer has today);
// suspension masks otherwise-active grants only.
func EntitlementStatus(active bool, revokedAt *time.Time, systemBlocked bool, pendingRef string) string {
	switch {
	case active && systemBlocked:
		return EntitlementStatusSuspended
	case active:
		return EntitlementStatusActive
	case pendingRef != "":
		return EntitlementStatusPending
	case revokedAt != nil:
		return EntitlementStatusRevoked
	default:
		return EntitlementStatusExpired
	}
}

// EntitlementCatalogItem is one grantable add-on type. The 3 legacy ng-* ids
// are the wire ids the appliance feeds call on /auth/service/<id> — never
// rename them. New ids follow the convention: nsec-<service> (firewall
// services), ns8-<app> (application enablement on a cluster), <app>-<module>
// (per-application-instance modules, Scoped=true).
// Catalog item kinds — both sellable on the shop: services are firewall
// add-ons granted system-wide, modules are add-ons for a single application
// instance of an NS8 cluster.
const (
	EntitlementKindService = "service"
	EntitlementKindModule  = "module"
)

type EntitlementCatalogItem struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Scoped      bool      `json:"scoped"`
	Kind        string    `json:"kind"`
	SystemType  string    `json:"system_type,omitempty"`
	LegacyAlias string    `json:"legacy_alias,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateEntitlementCatalogRequest adds a new add-on type to the catalog.
// LegacyAlias is the old wire id consumers still call on /auth/service/<id>
// (only needed for types migrated from the legacy my).
type CreateEntitlementCatalogRequest struct {
	ID          string `json:"id" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	Description string `json:"description,omitempty"`
	Scoped      bool   `json:"scoped,omitempty"`
	Kind        string `json:"kind,omitempty"`
	SystemType  string `json:"system_type,omitempty"`
	LegacyAlias string `json:"legacy_alias,omitempty"`
}

// UpdateEntitlementCatalogRequest updates the display fields of a catalog
// item. The id and scoped flag are immutable (grants may already reference
// them with the current semantics).
type UpdateEntitlementCatalogRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// EntitlementAvailability is one commercial unlock: the catalog item can be
// bought/self-activated by a whole hierarchy role OR by one specific
// organization (exactly one of the two is set). It does not affect /auth
// enforcement.
type EntitlementAvailability struct {
	ID             string                 `json:"id"`
	Entitlement    string                 `json:"entitlement"`
	OrgRole        string                 `json:"org_role,omitempty"`
	OrganizationID string                 `json:"organization_id,omitempty"`
	CreatedBy      map[string]interface{} `json:"created_by,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CreateEntitlementAvailabilityRequest unlocks a catalog item for a role or
// a specific organization.
type CreateEntitlementAvailabilityRequest struct {
	OrgRole        string `json:"org_role,omitempty"`
	OrganizationID string `json:"organization_id,omitempty"`
}

// EntitlementGrantReportRow is one row of the fleet-wide grants report
// (owner/Super Admin): the grant plus the system identity it belongs to.
type EntitlementGrantReportRow struct {
	SystemEntitlement
	SystemName       string `json:"system_name"`
	SystemKey        string `json:"system_key"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
}

// EntitlementStatsRow aggregates active grants per entitlement per org.
type EntitlementStatsRow struct {
	Entitlement      string `json:"entitlement"`
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	ActiveGrants     int    `json:"active_grants"`
}

// EntitlementReport is the owner/Super Admin analytics snapshot of the whole
// add-on fleet: lifecycle totals, the per-type breakdown, the renewal
// distribution and an activation trend. The per-organization and per-tier
// breakdowns live on their own paginated+searchable endpoints
// (/report/organizations, /report/tiers) — orgs can be hundreds. Deleted
// systems are excluded everywhere.
type EntitlementReport struct {
	Totals        EntitlementReportTotals     `json:"totals"`
	ByEntitlement []EntitlementReportByType   `json:"by_entitlement"`
	Renewals      EntitlementReportRenewals   `json:"renewals"`
	Trend         []EntitlementReportTrendRow `json:"trend"`
}

// EntitlementReportTotals is the fleet-wide lifecycle breakdown. Statuses
// follow the same precedence as EntitlementStatus; Perpetual counts active
// grants without an expiry (legacy imports); the Expiring* buckets count
// active grants whose expiry falls within the window (cumulative).
type EntitlementReportTotals struct {
	Total         int `json:"total"`
	Active        int `json:"active"`
	Expired       int `json:"expired"`
	Revoked       int `json:"revoked"`
	Pending       int `json:"pending"`
	Suspended     int `json:"suspended"`
	Perpetual     int `json:"perpetual"`
	ExpiringIn30d int `json:"expiring_in_30d"`
	ExpiringIn60d int `json:"expiring_in_60d"`
	ExpiringIn90d int `json:"expiring_in_90d"`
	Systems       int `json:"systems"`       // distinct systems with at least one grant
	Organizations int `json:"organizations"` // distinct orgs owning those systems
	// Breakdown of Systems by the hierarchy role of the owning org (the
	// four sum up to Systems).
	DistributorSystems int `json:"distributor_systems"`
	ResellerSystems    int `json:"reseller_systems"`
	CustomerSystems    int `json:"customer_systems"`
	OwnerSystems       int `json:"owner_systems"`
	TotalRenewals      int `json:"total_renewals"` // sum of renewal_count over all grants
}

// EntitlementReportByType is the lifecycle breakdown of one add-on type.
type EntitlementReportByType struct {
	Entitlement string `json:"entitlement"`
	DisplayName string `json:"display_name"`
	Active      int    `json:"active"`
	Expired     int    `json:"expired"`
	Revoked     int    `json:"revoked"`
	Pending     int    `json:"pending"`
	Suspended   int    `json:"suspended"`
	Total       int    `json:"total"`
}

// EntitlementReportByOrg aggregates grants per organization owning the
// systems, with the reseller/distributor hierarchy role for grouping.
type EntitlementReportByOrg struct {
	OrganizationID   string `json:"organization_id"`
	OrganizationName string `json:"organization_name"`
	OrgType          string `json:"org_type"`
	Systems          int    `json:"systems"`
	Active           int    `json:"active"`
	Total            int    `json:"total"`
}

// EntitlementReportByVariant counts grants per shop tier of one add-on.
type EntitlementReportByVariant struct {
	Entitlement string `json:"entitlement"`
	Label       string `json:"label"`
	Count       int    `json:"count"`
}

// EntitlementReportRenewals is the renewal distribution across grants.
type EntitlementReportRenewals struct {
	Never     int `json:"never"`
	Once      int `json:"once"`
	Twice     int `json:"twice"`
	ThreePlus int `json:"three_plus"`
}

// EntitlementReportTrendRow is one month of the activation trend (grants
// created in that month, by created_at).
type EntitlementReportTrendRow struct {
	Month       string `json:"month"` // YYYY-MM
	Activations int    `json:"activations"`
}

// SystemEntitlement is one add-on grant for one system, optionally narrowed
// to one application instance via Scope ("" = whole system). Active is
// derived: not revoked and not expired (valid_until NULL = perpetual).
type SystemEntitlement struct {
	ID          string     `json:"id"`
	SystemID    string     `json:"system_id"`
	Entitlement string     `json:"entitlement"`
	Scope       string     `json:"scope,omitempty"`
	Source      string     `json:"source"`
	SourceRef   string     `json:"source_ref,omitempty"`
	ValidFrom   time.Time  `json:"valid_from"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty"`
	// RevokedSource records WHO revoked: "manual" (admin, deliberate — not
	// re-buyable from the shop) or "shop" (deactivate webhook: subscription
	// cancelled / payment failed — re-buyable). Empty when not revoked.
	RevokedSource string `json:"revoked_source,omitempty"`
	// PendingRef is the shop order placed at checkout and not yet paid
	// (bank transfer/RiBa can take days): display-only, so the UI shows
	// "pending" instead of offering another purchase. Enforcement ignores
	// it. Cleared by activate (payment confirmed) or cancel.
	PendingRef   string                 `json:"pending_ref,omitempty"`
	PendingSince *time.Time             `json:"pending_since,omitempty"`
	Active       bool                   `json:"active"`
	Status       string                 `json:"status"` // active | expired | revoked | suspended | pending (see EntitlementStatus)
	CreatedBy    map[string]interface{} `json:"created_by,omitempty"`
	// PurchasedBy is the audit snapshot of the my user that BOUGHT the grant
	// on the shop, resolved from the order's customer email at activation:
	// {logto_id, name, email, organization_id, organization_name, org_role,
	// user_roles} — {email} only when the address matches no my user, nil for
	// manual grants, legacy imports and stamped legacy orders. CreatedBy stays
	// the webhook actor (owner key); this is the real buyer. Read endpoints
	// redact it to {out_of_scope: true} when the buyer's organization is
	// outside the viewer's hierarchy.
	PurchasedBy map[string]interface{} `json:"purchased_by,omitempty"`
	// Variant is the shop variation (tier) of the purchased product line,
	// {id, sku, label} (e.g. label "16-30 device"). Display metadata only:
	// the add-on mapping stays on the parent product and /auth enforcement
	// ignores it. Refreshed by activate, so upgrades/downgrades follow the
	// renewals; nil for manual grants and simple products.
	Variant map[string]interface{} `json:"variant,omitempty"`
	// RenewalCount is the number of paid shop orders on this grant beyond
	// the first: activate increments it when source_ref CHANGES (webhook
	// retries on the same order never double-count). 0 = first period.
	RenewalCount int       `json:"renewal_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateSystemEntitlementRequest grants an add-on to a system. Scope narrows
// the grant to one application instance and is only accepted for catalog
// items with Scoped=true.
type CreateSystemEntitlementRequest struct {
	Entitlement string     `json:"entitlement" binding:"required"`
	Scope       string     `json:"scope,omitempty"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"` // override (legacy import: the original order date); defaults to now
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	Source      string     `json:"source,omitempty"`
	SourceRef   string     `json:"source_ref,omitempty"`
}

// ActivateEntitlementRequest is the shop-facing activation/renewal call
// (webhook after purchase or subscription renewal). The system is addressed
// by its key (the shop never sees internal ids); the entitlement accepts the
// canonical id or the legacy alias. Idempotent: an existing grant is renewed
// in place (new expiry, revocation cleared).
type ActivateEntitlementRequest struct {
	SystemKey   string     `json:"system_key" binding:"required"`
	Entitlement string     `json:"entitlement" binding:"required"`
	Scope       string     `json:"scope,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	SourceRef   string     `json:"source_ref,omitempty"`
	// BuyerEmail is the email of the WordPress customer that owns the order
	// (server-to-server, trusted): the backend resolves it to a my user and
	// stores the purchased_by audit snapshot. Empty on stamped legacy orders.
	BuyerEmail string `json:"buyer_email,omitempty"`
	// Variant is the shop variation (tier) of the order line, {id, sku,
	// label}; stored as display metadata on the grant.
	Variant map[string]interface{} `json:"variant,omitempty"`
}

// DeactivateEntitlementRequest revokes a shop-managed grant (subscription
// cancelled/expired). When SourceRef is set it is matched against the grant:
// a pending activation with the same ref is cleared instead of revoked, and
// a grant owned by a DIFFERENT ref is left untouched (cancelling an old
// order must not kill an entitlement another order paid for).
type DeactivateEntitlementRequest struct {
	SystemKey   string `json:"system_key" binding:"required"`
	Entitlement string `json:"entitlement" binding:"required"`
	Scope       string `json:"scope,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
}

// PendingEntitlementRequest marks a shop order placed at checkout and not
// yet paid. SourceRef is required: it correlates the pending marker with the
// activate/cancel that will follow.
type PendingEntitlementRequest struct {
	SystemKey   string `json:"system_key" binding:"required"`
	Entitlement string `json:"entitlement" binding:"required"`
	Scope       string `json:"scope,omitempty"`
	SourceRef   string `json:"source_ref" binding:"required"`
	// BuyerEmail: see ActivateEntitlementRequest. Stamped on fresh pending
	// stubs only — a pending renewal must not overwrite who bought the grant
	// the customer currently has.
	BuyerEmail string `json:"buyer_email,omitempty"`
	// Variant: see ActivateEntitlementRequest. Fresh pending stubs only,
	// like BuyerEmail — the not-yet-paid tier must not overwrite the one the
	// customer currently has.
	Variant map[string]interface{} `json:"variant,omitempty"`
}

// UpdateSystemEntitlementRequest extends/reduces the expiry or toggles the
// revoked state. ClearValidUntil makes the grant perpetual (valid_until NULL);
// it wins over ValidUntil when both are set. The target grant is addressed by
// path (:entitlement) + optional ?scope= query.
type UpdateSystemEntitlementRequest struct {
	ValidUntil      *time.Time `json:"valid_until,omitempty"`
	ClearValidUntil bool       `json:"clear_valid_until,omitempty"`
	Revoked         *bool      `json:"revoked,omitempty"`
}
