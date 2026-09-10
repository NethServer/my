/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// DashboardGenerateRequest is what the frontend wizard sends: the kinds of
// information the user asked for, plus an optional free-text hint. The hint is
// untrusted user input and is treated as such by the prompt builder.
type DashboardGenerateRequest struct {
	Topics []string `json:"topics" binding:"required,min=1,max=12,dive,oneof=organizations alerts systems users applications entitlements rebranding"`
	Hint   string   `json:"hint" binding:"omitempty,max=280"`
	Locale string   `json:"locale" binding:"omitempty,oneof=en it"`
}

// DashboardWidget is one chosen widget. Size is always populated — the server
// fills the catalog default when the model omits or garbles it — so the
// frontend never has to branch on its absence.
type DashboardWidget struct {
	ID     string `json:"id"`
	Size   string `json:"size"`
	Reason string `json:"reason,omitempty"`
}

// DashboardGenerateResponse is the selection returned to the frontend.
//
// GeneratedBy is "ai" when the model produced the selection and "fallback"
// when the deterministic rule-based path did, which happens whenever the model
// is disabled, unreachable, or answered with nothing usable. There is no error
// shape for that case: the fallback is the answer, so this endpoint never
// returns 5xx.
//
// Deliberately not paginated: this is a bounded 3-7 element selection, not a
// queryable collection, so a pagination object would only assert things that
// are not true of it.
type DashboardGenerateResponse struct {
	Title       string            `json:"title"`
	GeneratedBy string            `json:"generated_by"`
	Model       string            `json:"model,omitempty"`
	Widgets     []DashboardWidget `json:"widgets"`
}

// DashboardCatalogWidget is one catalog entry as exposed to the frontend, so
// the wizard offers only the topics the caller can actually see.
type DashboardCatalogWidget struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Topic       string `json:"topic"`
	DefaultSize string `json:"default_size"`
}

// DashboardCatalogResponse is the caller's filtered catalog.
type DashboardCatalogResponse struct {
	Widgets []DashboardCatalogWidget `json:"widgets"`
	Topics  []string                 `json:"topics"`
}
