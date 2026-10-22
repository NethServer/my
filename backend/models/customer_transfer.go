/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "time"

// TransferCustomerRequest moves a customer under another managing
// organization (reseller or distributor).
type TransferCustomerRequest struct {
	OrganizationID string `json:"organization_id" binding:"required"`
}

// Renewal outcomes of an add-on in a customer transfer.
const (
	TransferRenewalContinues      = "continues"
	TransferRenewalStops          = "stops"
	TransferRenewalAlreadyStopped = "already_stopped"
	TransferRenewalPerpetual      = "perpetual"
)

// CustomerTransferOrganization identifies one organization of a transfer.
type CustomerTransferOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// CustomerTransferAddon is a live add-on on a system of the customer and what
// the transfer does to its shop renewal. PurchasedByOrganizationName is empty
// when the buyer is unknown or outside the caller's hierarchy.
type CustomerTransferAddon struct {
	SystemID                    string     `json:"system_id"`
	SystemName                  string     `json:"system_name"`
	Entitlement                 string     `json:"entitlement"`
	DisplayName                 string     `json:"display_name"`
	Scope                       string     `json:"scope,omitempty"`
	ValidUntil                  *time.Time `json:"valid_until,omitempty"`
	PurchasedByOrganizationName string     `json:"purchased_by_organization_name,omitempty"`
	Renewal                     string     `json:"renewal"`
}

// CustomerTransferPlan is what moving a customer to another manager does:
// returned as is by the preview, and filled with the applied counters by the
// transfer itself.
type CustomerTransferPlan struct {
	Customer             CustomerTransferOrganization `json:"customer"`
	From                 CustomerTransferOrganization `json:"from"`
	To                   CustomerTransferOrganization `json:"to"`
	SystemsCount         int                          `json:"systems_count"`
	UsersCount           int                          `json:"users_count"`
	Addons               []CustomerTransferAddon      `json:"addons"`
	RenewalsStopping     int                          `json:"renewals_stopping"`
	SilencesCount        int                          `json:"silences_count"`
	AssignmentsCount     int                          `json:"assignments_count"`
	Warnings             []string                     `json:"warnings,omitempty"`
	systemKeys           []string
	stoppingEntitlements []string
	keepOrgIDs           []string
}

// SystemKeys, StoppingEntitlements and KeepOrgIDs carry the plan internals
// the transfer applies: they never leave the backend.
func (p *CustomerTransferPlan) SystemKeys() []string           { return p.systemKeys }
func (p *CustomerTransferPlan) StoppingEntitlements() []string { return p.stoppingEntitlements }
func (p *CustomerTransferPlan) KeepOrgIDs() []string           { return p.keepOrgIDs }

// SetInternals records the plan internals computed by the service.
func (p *CustomerTransferPlan) SetInternals(systemKeys, stoppingEntitlements, keepOrgIDs []string) {
	p.systemKeys = systemKeys
	p.stoppingEntitlements = stoppingEntitlements
	p.keepOrgIDs = keepOrgIDs
}
