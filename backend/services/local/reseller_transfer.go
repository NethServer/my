/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package local

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/alerting"
)

// Guardrails for reseller transfers, surfaced by the handler as 403/409s.
var (
	ErrTransferResellerAccessDenied   = errors.New("access denied to transfer reseller")
	ErrTransferResellerSuspended      = errors.New("the reseller is suspended: reactivate it before changing its manager")
	ErrTransferResellerNotSynced      = errors.New("the reseller is not synced with logto yet")
	ErrTransferResellerSameManager    = errors.New("the reseller is already managed by this organization")
	ErrTransferResellerTargetInactive = errors.New("the target organization is suspended")
	ErrTransferResellerConcurrent     = errors.New("the reseller manager changed during the transfer")
)

// PlanResellerTransfer validates moving a reseller under the distributor
// targetOrgID and computes what the move does, without changing anything.
// Reshaping the distributor level is the Owner organization's call only.
func (s *LocalOrganizationService) PlanResellerTransfer(resellerID, targetOrgID string, actor CustomerTransferActor) (*models.ResellerTransferPlan, error) {
	if !models.IsGlobalOrgRole(strings.ToLower(actor.OrgRole)) {
		return nil, ErrTransferResellerAccessDenied
	}

	reseller, err := s.resellerRepo.GetByID(resellerID)
	if err != nil {
		return nil, err
	}
	if reseller.LogtoID == nil || *reseller.LogtoID == "" {
		return nil, ErrTransferResellerNotSynced
	}
	resellerOrgID := *reseller.LogtoID
	if reseller.SuspendedAt != nil {
		return nil, ErrTransferResellerSuspended
	}

	fromOrgID := customDataString(reseller.CustomData, "createdBy")
	if targetOrgID == fromOrgID {
		return nil, ErrTransferResellerSameManager
	}

	distributor, err := s.distributorRepo.GetByID(targetOrgID)
	if err != nil || distributor.LogtoID == nil {
		return nil, transferTargetValidationError(targetOrgID, "not_a_distributor")
	}
	if !distributor.Active() {
		return nil, ErrTransferResellerTargetInactive
	}
	to := models.CustomerTransferOrganization{ID: *distributor.LogtoID, Name: distributor.Name, Type: "distributor"}

	from, _, found := s.managingOrganization(fromOrgID)
	if !found {
		from = models.CustomerTransferOrganization{ID: fromOrgID, Name: "Owner", Type: "owner"}
	}

	customerIDs, err := s.customerRepo.GetLogtoIDsByCreatedByMultiple([]string{resellerOrgID})
	if err != nil {
		return nil, fmt.Errorf("failed to list the reseller's customers: %w", err)
	}
	subtree := append([]string{resellerOrgID}, customerIDs...)

	plan := &models.ResellerTransferPlan{
		Reseller:       models.CustomerTransferOrganization{ID: resellerOrgID, Name: reseller.Name, Type: "reseller"},
		From:           from,
		To:             to,
		CustomersCount: len(customerIDs),
		Addons:         []models.CustomerTransferAddon{},
	}

	if err := database.DB.QueryRow(`
		SELECT (SELECT COUNT(*) FROM systems WHERE organization_id = ANY($1) AND deleted_at IS NULL),
		       (SELECT COUNT(*) FROM users WHERE organization_id = ANY($1) AND deleted_at IS NULL)
	`, pq.Array(subtree)).Scan(&plan.SystemsCount, &plan.UsersCount); err != nil {
		return nil, fmt.Errorf("failed to count the reseller's systems and users: %w", err)
	}

	chain, err := alerting.ResolveAncestorChain(to.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the target hierarchy: %w", err)
	}
	keepOrgIDs := append(chain, subtree...)

	grantRepo := entities.NewLocalSystemEntitlementRepository()
	assignmentRepo := entities.NewLocalAlertAssignmentRepository()
	stopping := []string{}
	for _, orgID := range subtree {
		grants, err := grantRepo.ListLiveForOrganization(orgID)
		if err != nil {
			return nil, err
		}
		for _, g := range grants {
			addon := models.CustomerTransferAddon{
				SystemID:                    g.SystemID,
				SystemName:                  g.SystemName,
				Entitlement:                 g.Entitlement,
				DisplayName:                 g.DisplayName,
				Scope:                       g.Scope,
				ValidUntil:                  g.ValidUntil,
				PurchasedByOrganizationName: g.PurchasedByOrgName,
				Renewal:                     transferRenewalOutcome(g, keepOrgIDs),
			}
			if addon.Renewal == models.TransferRenewalStops {
				stopping = append(stopping, g.ID)
			}
			plan.Addons = append(plan.Addons, addon)
		}

		assigned, err := assignmentRepo.CountOutsideOrganizations(orgID, keepOrgIDs)
		if err != nil {
			return nil, err
		}
		plan.AssignmentsCount += assigned
	}
	plan.RenewalsStopping = len(stopping)

	plan.SetInternals(subtree, stopping, keepOrgIDs)
	return plan, nil
}

// TransferReseller moves a reseller under another distributor by rewriting
// its custom_data.createdBy, in Logto first and then locally. Its customers,
// systems and users follow it: the reseller stays their manager and its own
// alerting tenant, so nothing changes tenant. What belongs to the distributor
// it leaves is dropped: add-ons bought by that side stop renewing (they stay
// valid until they expire), its users' alert assignments are released and its
// notification layer no longer reaches the reseller.
func (s *LocalOrganizationService) TransferReseller(resellerID, targetOrgID string, actor CustomerTransferActor) (*models.ResellerTransferPlan, error) {
	release, err := cache.AcquireResellerTransferLock(context.Background(), resellerID, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	defer release()

	plan, err := s.PlanResellerTransfer(resellerID, targetOrgID, actor)
	if err != nil {
		return nil, err
	}

	reseller, err := s.resellerRepo.GetByID(resellerID)
	if err != nil {
		return nil, err
	}
	original := make(map[string]interface{}, len(reseller.CustomData)+1)
	for k, v := range reseller.CustomData {
		original[k] = v
	}
	if reseller.CreatedBy != nil {
		original["createdByUser"] = reseller.CreatedBy.ForStorage()
	}
	moved := make(map[string]interface{}, len(original)+2)
	for k, v := range original {
		moved[k] = v
	}
	moved["createdBy"] = plan.To.ID
	moved["updatedBy"] = actor.OrgID
	moved["updatedAt"] = time.Now().Format(time.RFC3339)

	movedJSON, err := json.Marshal(moved)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom_data: %w", err)
	}

	if _, err := s.logtoClient.UpdateOrganization(plan.Reseller.ID, models.UpdateOrganizationRequest{
		Name:        &reseller.Name,
		Description: &reseller.Description,
		CustomData:  moved,
	}); err != nil {
		return nil, fmt.Errorf("failed to sync the reseller manager to Logto: %w", err)
	}

	result, err := database.DB.Exec(`
		UPDATE resellers SET custom_data = $2::jsonb, updated_at = NOW()
		WHERE logto_id = $1 AND deleted_at IS NULL AND COALESCE(custom_data->>'createdBy', '') = $3
	`, plan.Reseller.ID, movedJSON, customDataString(reseller.CustomData, "createdBy"))
	affected := int64(0)
	if err == nil {
		affected, err = result.RowsAffected()
	}
	if err != nil || affected != 1 {
		if _, undoErr := s.logtoClient.UpdateOrganization(plan.Reseller.ID, models.UpdateOrganizationRequest{
			Name:        &reseller.Name,
			Description: &reseller.Description,
			CustomData:  original,
		}); undoErr != nil {
			logger.Error().Err(undoErr).Str("reseller_id", plan.Reseller.ID).Msg("Failed to restore reseller custom data in Logto after a failed transfer")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to update reseller manager: %w", err)
		}
		return nil, ErrTransferResellerConcurrent
	}

	cache.GetRBACCache().InvalidateAll()
	cache.GetAppsCache().InvalidateAll()
	refreshUnifiedOrganizationsAsync()
	RequestIdPAccessReconcile()

	// Best effort from here on: the reseller already belongs to the new
	// distributor, each step records what it managed to do.
	applied := *plan
	applied.Warnings = nil

	stopped, err := entities.NewLocalSystemEntitlementRepository().StopRenewals(plan.StoppingEntitlements(), models.EntitlementRenewalStoppedResellerTransferred)
	if err != nil {
		logger.Error().Err(err).Str("reseller_id", plan.Reseller.ID).Msg("Failed to stop add-on renewals after reseller transfer")
		applied.Warnings = append(applied.Warnings, "renewals_not_stopped")
	}
	applied.RenewalsStopping = stopped

	released := 0
	assignmentRepo := entities.NewLocalAlertAssignmentRepository()
	for _, orgID := range plan.SubtreeOrgIDs() {
		n, err := assignmentRepo.ReleaseOutsideOrganizations(orgID, plan.KeepOrgIDs())
		if err != nil {
			logger.Warn().Err(err).Str("reseller_id", plan.Reseller.ID).Str("organization_id", orgID).Msg("Failed to release alert assignments after reseller transfer")
			if len(applied.Warnings) == 0 || applied.Warnings[len(applied.Warnings)-1] != "assignments_not_released" {
				applied.Warnings = append(applied.Warnings, "assignments_not_released")
			}
			continue
		}
		released += n
	}
	applied.AssignmentsCount = released

	audit := &entities.ResellerOrgTransfer{
		ResellerID:          plan.Reseller.ID,
		FromOrgID:           plan.From.ID,
		ToOrgID:             plan.To.ID,
		ActorUserID:         sql.NullString{String: actor.UserID, Valid: actor.UserID != ""},
		ActorOrganizationID: sql.NullString{String: actor.OrgID, Valid: actor.OrgID != ""},
		ActorIP:             sql.NullString{String: actor.ActorIP, Valid: actor.ActorIP != ""},
		UserAgent:           sql.NullString{String: actor.UserAgent, Valid: actor.UserAgent != ""},
		CustomersCount:      plan.CustomersCount,
		SystemsCount:        plan.SystemsCount,
		RenewalsStopped:     stopped,
		AssignmentsReleased: released,
	}
	if _, err := entities.NewLocalResellerOrgTransfersRepository().Insert(audit); err != nil {
		logger.Warn().Err(err).Str("reseller_id", plan.Reseller.ID).Msg("Failed to record reseller transfer audit row")
	}

	// The reseller's tenant is rendered from its ancestors' layers: the old
	// distributor's leave it, the new one's join.
	alerting.ProvisionDefaultConfigAsync(plan.Reseller.ID, "reseller", plan.Reseller.ID)

	logger.Info().
		Str("reseller_id", plan.Reseller.ID).
		Str("from_org_id", plan.From.ID).
		Str("to_org_id", plan.To.ID).
		Int("customers_count", plan.CustomersCount).
		Int("systems_count", plan.SystemsCount).
		Int("renewals_stopped", stopped).
		Int("assignments_released", released).
		Str("transferred_by_user_id", actor.UserID).
		Str("transferred_by_org_id", actor.OrgID).
		Msg("Reseller transferred to another distributor")

	return &applied, nil
}
