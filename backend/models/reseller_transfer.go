/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// TransferResellerRequest moves a reseller under another distributor.
type TransferResellerRequest struct {
	OrganizationID string `json:"organization_id" binding:"required"`
}

// ResellerTransferPlan is what moving a reseller under another distributor
// does: returned as is by the preview, and filled with the applied counters by
// the transfer itself. The counters span the reseller and its customers,
// which follow it.
type ResellerTransferPlan struct {
	Reseller             CustomerTransferOrganization `json:"reseller"`
	From                 CustomerTransferOrganization `json:"from"`
	To                   CustomerTransferOrganization `json:"to"`
	CustomersCount       int                          `json:"customers_count"`
	SystemsCount         int                          `json:"systems_count"`
	UsersCount           int                          `json:"users_count"`
	Addons               []CustomerTransferAddon      `json:"addons"`
	RenewalsStopping     int                          `json:"renewals_stopping"`
	AssignmentsCount     int                          `json:"assignments_count"`
	Warnings             []string                     `json:"warnings,omitempty"`
	subtreeOrgIDs        []string
	stoppingEntitlements []string
	keepOrgIDs           []string
}

// SubtreeOrgIDs, StoppingEntitlements and KeepOrgIDs carry the plan internals
// the transfer applies: they never leave the backend.
func (p *ResellerTransferPlan) SubtreeOrgIDs() []string        { return p.subtreeOrgIDs }
func (p *ResellerTransferPlan) StoppingEntitlements() []string { return p.stoppingEntitlements }
func (p *ResellerTransferPlan) KeepOrgIDs() []string           { return p.keepOrgIDs }

// SetInternals records the plan internals computed by the service.
func (p *ResellerTransferPlan) SetInternals(subtreeOrgIDs, stoppingEntitlements, keepOrgIDs []string) {
	p.subtreeOrgIDs = subtreeOrgIDs
	p.stoppingEntitlements = stoppingEntitlements
	p.keepOrgIDs = keepOrgIDs
}
