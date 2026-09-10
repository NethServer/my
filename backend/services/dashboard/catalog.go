/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package dashboard

import (
	"slices"

	"github.com/nethesis/my/backend/models"
)

// Size is the footprint a widget asks for on the frontend's 12-column grid.
type Size string

const (
	SizeSmall  Size = "sm"
	SizeMedium Size = "md"
	SizeLarge  Size = "lg"
)

// Topic is a kind of information the user can ask for in the wizard. It is the
// vocabulary of the request, and the axis the rule-based fallback orders by.
type Topic string

const (
	TopicOrganizations Topic = "organizations"
	TopicAlerts        Topic = "alerts"
	TopicSystems       Topic = "systems"
	TopicUsers         Topic = "users"
	TopicApplications  Topic = "applications"
	TopicEntitlements  Topic = "entitlements"
	TopicRebranding    Topic = "rebranding"
)

// Widget is one entry of the catalog the model chooses from. Title and
// Description are what the model reads; the frontend renders its own
// translated strings and only consumes ID and DefaultSize.
type Widget struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Topic       Topic  `json:"topic"`
	// RequiredPermission is the permission the widget's own data endpoint is
	// gated on. Empty means any authenticated user may see it.
	RequiredPermission string `json:"-"`
	DefaultSize        Size   `json:"default_size"`
}

// catalog is the single source of truth for what a dashboard may contain. Each
// RequiredPermission is the gate of the endpoint the widget actually calls, not
// a guess from the widget's name: every alert list and aggregate endpoint is on
// read:systems (only /alerts/config* uses read:alerts), and the per-system
// backup endpoint is on read:systems too — there is no backups resource.
var catalog = []Widget{
	{
		ID:                 "alerts_counter",
		Title:              "Active alerts",
		Description:        "How many alerts are firing right now, split by critical, warning and muted.",
		Topic:              TopicAlerts,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "alerts_trend",
		Title:              "Alerts trend",
		Description:        "Alert volume over the last 30 days, with the change against the previous period.",
		Topic:              TopicAlerts,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "alerts_severity",
		Title:              "Alerts by severity",
		Description:        "Breakdown of alerts by severity, with mean time to resolution and mean time between failures.",
		Topic:              TopicAlerts,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "alerts_top_names",
		Title:              "Noisiest alerts",
		Description:        "The alert rules that fired most often, ranked by count.",
		Topic:              TopicAlerts,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "alerts_firing_now",
		Title:              "Firing now",
		Description:        "The most severe alerts currently firing, newest first, as an action list.",
		Topic:              TopicAlerts,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "systems_counter",
		Title:              "Systems",
		Description:        "Total registered systems, split by active, inactive and unknown heartbeat state.",
		Topic:              TopicSystems,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "systems_trend",
		Title:              "Systems trend",
		Description:        "How the number of registered systems moved over the last 30 days.",
		Topic:              TopicSystems,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "systems_recent",
		Title:              "Recently registered systems",
		Description:        "The systems registered most recently, with their product and registration date.",
		Topic:              TopicSystems,
		RequiredPermission: "read:systems",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "applications_counter",
		Title:              "Applications",
		Description:        "Total installed applications, including how many are unassigned or in error.",
		Topic:              TopicApplications,
		RequiredPermission: "read:applications",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "applications_trend",
		Title:              "Applications trend",
		Description:        "How the number of installed applications moved over the last 30 days.",
		Topic:              TopicApplications,
		RequiredPermission: "read:applications",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "applications_top",
		Title:              "Most installed applications",
		Description:        "The applications installed on most systems, ranked by install count.",
		Topic:              TopicApplications,
		RequiredPermission: "read:applications",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "users_counter",
		Title:              "Users",
		Description:        "Total user accounts, split by enabled and suspended.",
		Topic:              TopicUsers,
		RequiredPermission: "read:users",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "users_trend",
		Title:              "Users trend",
		Description:        "How the number of user accounts moved over the last 30 days.",
		Topic:              TopicUsers,
		RequiredPermission: "read:users",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "distributors_counter",
		Title:              "Distributors",
		Description:        "How many distributor organizations are in reach.",
		Topic:              TopicOrganizations,
		RequiredPermission: "read:distributors",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "resellers_counter",
		Title:              "Resellers",
		Description:        "How many reseller organizations are in reach.",
		Topic:              TopicOrganizations,
		RequiredPermission: "read:resellers",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "customers_counter",
		Title:              "Customers",
		Description:        "How many customer organizations are in reach.",
		Topic:              TopicOrganizations,
		RequiredPermission: "read:customers",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "customers_trend",
		Title:              "Customers trend",
		Description:        "How the number of customer organizations moved over the last 30 days.",
		Topic:              TopicOrganizations,
		RequiredPermission: "read:customers",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "addons_expiring",
		Title:              "Add-ons expiring soon",
		Description:        "Add-on grants that expire within 30 days, as an action list ordered by urgency.",
		Topic:              TopicEntitlements,
		RequiredPermission: "read:entitlements",
		DefaultSize:        SizeMedium,
	},
	{
		ID:                 "addons_expiry_windows",
		Title:              "Renewals due",
		Description:        "How many add-on grants come up for renewal within 30, 60 and 90 days.",
		Topic:              TopicEntitlements,
		RequiredPermission: "read:entitlements",
		DefaultSize:        SizeSmall,
	},
	{
		ID:                 "rebranding_summary",
		Title:              "Rebranding",
		Description:        "How many organizations have rebranding configured, split by organization type.",
		Topic:              TopicRebranding,
		RequiredPermission: "read:rebranding",
		DefaultSize:        SizeSmall,
	},
	{
		ID:          "third_party_apps",
		Title:       "Connected applications",
		Description: "Shortcuts into the third-party applications the user can open, with their account summary.",
		Topic:       TopicApplications,
		// No permission: /third-party-applications is authenticated-only.
		RequiredPermission: "",
		DefaultSize:        SizeLarge,
	},
}

// Catalog returns the whole catalog. The slice is a copy: callers rank, filter
// and reorder these entries, and the package-level catalog must survive that.
func Catalog() []Widget {
	return slices.Clone(catalog)
}

// CatalogFor returns the entries the user is allowed to see. It answers only
// "may this persona call that endpoint at all" — never which rows come back,
// which the endpoints scope by hierarchy themselves.
func CatalogFor(user *models.User) []Widget {
	allowed := make([]Widget, 0, len(catalog))
	for _, widget := range catalog {
		if hasPermission(user, widget.RequiredPermission) {
			allowed = append(allowed, widget)
		}
	}
	return allowed
}

// ByID looks a widget up in the full catalog.
func ByID(id string) (Widget, bool) {
	for _, widget := range catalog {
		if widget.ID == id {
			return widget, true
		}
	}
	return Widget{}, false
}

// Topics returns the distinct topics present in widgets, in catalog order.
func Topics(widgets []Widget) []Topic {
	topics := make([]Topic, 0, len(widgets))
	for _, widget := range widgets {
		if !slices.Contains(topics, widget.Topic) {
			topics = append(topics, widget.Topic)
		}
	}
	return topics
}

// hasPermission mirrors middleware.RequirePermission: the effective set is the
// union of the user's role permissions and its organization's.
func hasPermission(user *models.User, permission string) bool {
	if permission == "" {
		return true
	}
	if user == nil {
		return false
	}
	return slices.Contains(user.UserPermissions, permission) ||
		slices.Contains(user.OrgPermissions, permission)
}
