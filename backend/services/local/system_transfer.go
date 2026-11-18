/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/alerting"
)

// ErrSystemTransferTarget: the caller cannot assign systems to the target.
var ErrSystemTransferTarget = errors.New("the target organization is not in the caller's hierarchy")

// PlanSystemTransfer validates assigning a system to targetOrgID and computes
// what the move does to its add-ons, without changing anything.
func (s *LocalSystemsService) PlanSystemTransfer(systemID, targetOrgID, userOrgRole, userOrgID string) (*models.SystemTransferPlan, error) {
	system, err := s.GetSystem(systemID, userOrgRole, userOrgID)
	if err != nil {
		return nil, err
	}
	if canUpdate, reason := s.CanUpdateSystem(system, userOrgRole, userOrgID); !canUpdate {
		return nil, fmt.Errorf("access denied: %s", reason)
	}
	if canCreate, _ := s.CanCreateSystemForOrganization(userOrgRole, userOrgID, targetOrgID); !canCreate {
		return nil, ErrSystemTransferTarget
	}
	return s.planSystemTransfer(system.ID, system.Organization, s.getOrganizationInfo(targetOrgID), userOrgRole, userOrgID)
}

// planSystemTransfer applies the rule of the customer transfer to one
// system: a renewal survives only when its buyer is the new organization or
// sits above it.
func (s *LocalSystemsService) planSystemTransfer(systemID string, from, to models.Organization, userOrgRole, userOrgID string) (*models.SystemTransferPlan, error) {
	plan := &models.SystemTransferPlan{
		From:   models.CustomerTransferOrganization{ID: from.LogtoID, Name: from.Name, Type: from.Type},
		To:     models.CustomerTransferOrganization{ID: to.LogtoID, Name: to.Name, Type: to.Type},
		Addons: []models.CustomerTransferAddon{},
	}
	if from.LogtoID == to.LogtoID {
		return plan, nil
	}

	keepOrgIDs, err := alerting.ResolveAncestorChain(to.LogtoID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the target hierarchy: %w", err)
	}

	// Buyer names are shown only within the caller's hierarchy, as on the
	// grants endpoints.
	role := strings.ToLower(userOrgRole)
	var callerScope []string
	if !models.IsGlobalOrgRole(role) {
		callerScope, err = NewUserService().GetHierarchicalOrganizationIDs(role, userOrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve caller hierarchy: %w", err)
		}
	}

	grants, err := entities.NewLocalSystemEntitlementRepository().ListLiveForSystem(systemID)
	if err != nil {
		return nil, err
	}
	stopping := []string{}
	for _, g := range grants {
		addon := models.CustomerTransferAddon{
			SystemID:    g.SystemID,
			SystemName:  g.SystemName,
			Entitlement: g.Entitlement,
			DisplayName: g.DisplayName,
			Scope:       g.Scope,
			ValidUntil:  g.ValidUntil,
			Renewal:     transferRenewalOutcome(g, keepOrgIDs),
		}
		if g.PurchasedByOrgID != "" && (callerScope == nil || slices.Contains(callerScope, g.PurchasedByOrgID)) {
			addon.PurchasedByOrganizationName = g.PurchasedByOrgName
		}
		if addon.Renewal == models.TransferRenewalStops {
			stopping = append(stopping, g.ID)
		}
		plan.Addons = append(plan.Addons, addon)
	}
	plan.RenewalsStopping = len(stopping)
	plan.SetStoppingEntitlements(stopping)
	return plan, nil
}
