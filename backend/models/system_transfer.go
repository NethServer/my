/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

// SystemTransferPlan is what assigning a system to another organization does
// to its add-ons: returned by the preview and applied by the update.
type SystemTransferPlan struct {
	From                 CustomerTransferOrganization `json:"from"`
	To                   CustomerTransferOrganization `json:"to"`
	Addons               []CustomerTransferAddon      `json:"addons"`
	RenewalsStopping     int                          `json:"renewals_stopping"`
	stoppingEntitlements []string
}

// StoppingEntitlements never leaves the backend.
func (p *SystemTransferPlan) StoppingEntitlements() []string { return p.stoppingEntitlements }

// SetStoppingEntitlements records the grants the move stops renewing.
func (p *SystemTransferPlan) SetStoppingEntitlements(ids []string) { p.stoppingEntitlements = ids }
