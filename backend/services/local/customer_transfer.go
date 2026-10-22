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
	"slices"
	"strings"
	"time"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/alerting"
)

// Guardrails for customer transfers, surfaced by the handler as 403/409s.
var (
	ErrTransferCustomerAccessDenied   = errors.New("access denied to transfer customer")
	ErrTransferCustomerSuspended      = errors.New("the customer is suspended: reactivate it before changing its manager")
	ErrTransferCustomerNotSynced      = errors.New("the customer is not synced with logto yet")
	ErrTransferCustomerSameManager    = errors.New("the customer is already managed by this organization")
	ErrTransferCustomerTargetInactive = errors.New("the target organization is suspended")
	ErrTransferCustomerConcurrent     = errors.New("the customer manager changed during the transfer")
)

// CustomerTransferActor is who asks for a transfer, and from where.
type CustomerTransferActor struct {
	UserID    string
	OrgID     string
	OrgRole   string
	ActorIP   string
	UserAgent string
}

// PlanCustomerTransfer validates moving a customer under targetOrgID and
// computes what the move does, without changing anything. Only the Owner
// organization and distributors move customers, within their hierarchy: the
// customer must be theirs and so must the target, a reseller or distributor.
func (s *LocalOrganizationService) PlanCustomerTransfer(customerID, targetOrgID string, actor CustomerTransferActor) (*models.CustomerTransferPlan, error) {
	role := strings.ToLower(actor.OrgRole)
	if !models.IsGlobalOrgRole(role) && role != "distributor" {
		return nil, ErrTransferCustomerAccessDenied
	}

	customer, err := s.customerRepo.GetByID(customerID)
	if err != nil {
		return nil, err
	}
	if customer.LogtoID == nil || *customer.LogtoID == "" {
		return nil, ErrTransferCustomerNotSynced
	}
	customerOrgID := *customer.LogtoID
	if !models.IsGlobalOrgRole(role) && !NewUserService().IsOrganizationInHierarchy(role, actor.OrgID, customerOrgID) {
		return nil, ErrTransferCustomerAccessDenied
	}
	if customer.SuspendedAt != nil {
		return nil, ErrTransferCustomerSuspended
	}

	fromOrgID := customDataString(customer.CustomData, "createdBy")
	if targetOrgID == fromOrgID {
		return nil, ErrTransferCustomerSameManager
	}

	resolvedID, _, allowed, reason := s.ResolveCreatedByOrg(role, actor.OrgID, targetOrgID, "customer")
	if !allowed {
		code := "not_a_manager"
		if strings.Contains(reason, "hierarchy") {
			code = "not_in_hierarchy"
		}
		return nil, transferTargetValidationError(targetOrgID, code)
	}
	to, active, found := s.managingOrganization(resolvedID)
	if !found {
		return nil, transferTargetValidationError(targetOrgID, "not_a_manager")
	}
	if !active {
		return nil, ErrTransferCustomerTargetInactive
	}

	from, _, found := s.managingOrganization(fromOrgID)
	if !found {
		from = models.CustomerTransferOrganization{ID: fromOrgID, Name: "Owner", Type: "owner"}
	}

	plan := &models.CustomerTransferPlan{
		Customer: models.CustomerTransferOrganization{ID: customerOrgID, Name: customer.Name, Type: "customer"},
		From:     from,
		To:       to,
		Addons:   []models.CustomerTransferAddon{},
	}

	if err := database.DB.QueryRow(`
		SELECT (SELECT COUNT(*) FROM systems WHERE organization_id = $1 AND deleted_at IS NULL),
		       (SELECT COUNT(*) FROM users WHERE organization_id = $1 AND deleted_at IS NULL)
	`, customerOrgID).Scan(&plan.SystemsCount, &plan.UsersCount); err != nil {
		return nil, fmt.Errorf("failed to count customer systems and users: %w", err)
	}

	systemKeys, err := liveSystemKeys(customerOrgID)
	if err != nil {
		return nil, err
	}

	chain, err := alerting.ResolveAncestorChain(to.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the target hierarchy: %w", err)
	}
	keepOrgIDs := append(chain, customerOrgID)

	// Buyer names are shown only within the caller's hierarchy, as on the
	// grants endpoints.
	var callerScope []string
	if !models.IsGlobalOrgRole(role) {
		callerScope, err = NewUserService().GetHierarchicalOrganizationIDs(role, actor.OrgID)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve caller hierarchy: %w", err)
		}
	}

	grants, err := entities.NewLocalSystemEntitlementRepository().ListLiveForOrganization(customerOrgID)
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
		}
		if g.PurchasedByOrgID != "" && (callerScope == nil || slices.Contains(callerScope, g.PurchasedByOrgID)) {
			addon.PurchasedByOrganizationName = g.PurchasedByOrgName
		}
		addon.Renewal = transferRenewalOutcome(g, keepOrgIDs)
		if addon.Renewal == models.TransferRenewalStops {
			stopping = append(stopping, g.ID)
		}
		plan.Addons = append(plan.Addons, addon)
	}
	plan.RenewalsStopping = len(stopping)

	if plan.AssignmentsCount, err = entities.NewLocalAlertAssignmentRepository().CountOutsideOrganizations(customerOrgID, keepOrgIDs); err != nil {
		return nil, err
	}

	silences, err := alerting.SilencesForSystems(from.ID, systemKeys)
	if err != nil {
		logger.Warn().Err(err).Str("customer_id", customerOrgID).Str("tenant", from.ID).Msg("Failed to count silences for customer transfer")
		plan.Warnings = append(plan.Warnings, "silences_unavailable")
	}
	plan.SilencesCount = len(silences)

	plan.SetInternals(systemKeys, stopping, keepOrgIDs)
	return plan, nil
}

// TransferCustomer moves a customer under another managing organization by
// rewriting its custom_data.createdBy, in Logto first and then locally. The
// systems, users, backups and alert history stay with the customer and
// follow it: visibility, rebranding and the alerting tenant all derive from
// createdBy. What belongs to the manager it leaves is dropped: add-ons bought
// by that side stop renewing (they stay valid until they expire), its
// silences on the customer's systems are removed and its users' alert
// assignments are released.
func (s *LocalOrganizationService) TransferCustomer(customerID, targetOrgID string, actor CustomerTransferActor) (*models.CustomerTransferPlan, error) {
	release, err := cache.AcquireCustomerTransferLock(context.Background(), customerID, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	defer release()

	plan, err := s.PlanCustomerTransfer(customerID, targetOrgID, actor)
	if err != nil {
		return nil, err
	}

	customer, err := s.customerRepo.GetByID(customerID)
	if err != nil {
		return nil, err
	}
	original := make(map[string]interface{}, len(customer.CustomData)+1)
	for k, v := range customer.CustomData {
		original[k] = v
	}
	if customer.CreatedBy != nil {
		original["createdByUser"] = customer.CreatedBy.ForStorage()
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

	if _, err := s.logtoClient.UpdateOrganization(plan.Customer.ID, models.UpdateOrganizationRequest{
		Name:        &customer.Name,
		Description: &customer.Description,
		CustomData:  moved,
	}); err != nil {
		return nil, fmt.Errorf("failed to sync the customer manager to Logto: %w", err)
	}

	result, err := database.DB.Exec(`
		UPDATE customers SET custom_data = $2::jsonb, updated_at = NOW()
		WHERE logto_id = $1 AND deleted_at IS NULL AND COALESCE(custom_data->>'createdBy', '') = $3
	`, plan.Customer.ID, movedJSON, customDataString(customer.CustomData, "createdBy"))
	affected := int64(0)
	if err == nil {
		affected, err = result.RowsAffected()
	}
	if err != nil || affected != 1 {
		if _, undoErr := s.logtoClient.UpdateOrganization(plan.Customer.ID, models.UpdateOrganizationRequest{
			Name:        &customer.Name,
			Description: &customer.Description,
			CustomData:  original,
		}); undoErr != nil {
			logger.Error().Err(undoErr).Str("customer_id", plan.Customer.ID).Msg("Failed to restore customer custom data in Logto after a failed transfer")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to update customer manager: %w", err)
		}
		return nil, ErrTransferCustomerConcurrent
	}

	cache.GetRBACCache().InvalidateAll()
	cache.GetAppsCache().InvalidateAll()
	refreshUnifiedOrganizationsAsync()
	RequestIdPAccessReconcile()

	// Best effort from here on: the customer already belongs to the new
	// manager, each step records what it managed to do.
	applied := *plan
	applied.Warnings = nil

	stopped, err := entities.NewLocalSystemEntitlementRepository().StopRenewals(plan.StoppingEntitlements(), models.EntitlementRenewalStoppedCustomerTransferred)
	if err != nil {
		logger.Error().Err(err).Str("customer_id", plan.Customer.ID).Msg("Failed to stop add-on renewals after customer transfer")
		applied.Warnings = append(applied.Warnings, "renewals_not_stopped")
	}
	applied.RenewalsStopping = stopped

	released, err := entities.NewLocalAlertAssignmentRepository().ReleaseOutsideOrganizations(plan.Customer.ID, plan.KeepOrgIDs())
	if err != nil {
		logger.Warn().Err(err).Str("customer_id", plan.Customer.ID).Msg("Failed to release alert assignments after customer transfer")
		applied.Warnings = append(applied.Warnings, "assignments_not_released")
	}
	applied.AssignmentsCount = released

	cleared, err := alerting.CleanupSilencesForSystems(plan.From.ID, plan.SystemKeys())
	if err != nil {
		logger.Warn().Err(err).Str("customer_id", plan.Customer.ID).Str("tenant", plan.From.ID).Msg("Failed to clear silences after customer transfer")
		applied.Warnings = append(applied.Warnings, "silences_not_cleared")
	}
	applied.SilencesCount = cleared

	audit := &entities.CustomerOrgTransfer{
		CustomerID:          plan.Customer.ID,
		FromOrgID:           plan.From.ID,
		ToOrgID:             plan.To.ID,
		ActorUserID:         sql.NullString{String: actor.UserID, Valid: actor.UserID != ""},
		ActorOrganizationID: sql.NullString{String: actor.OrgID, Valid: actor.OrgID != ""},
		ActorIP:             sql.NullString{String: actor.ActorIP, Valid: actor.ActorIP != ""},
		UserAgent:           sql.NullString{String: actor.UserAgent, Valid: actor.UserAgent != ""},
		SystemsCount:        plan.SystemsCount,
		RenewalsStopped:     stopped,
		SilencesCleared:     cleared,
		AssignmentsReleased: released,
	}
	if _, err := entities.NewLocalCustomerOrgTransfersRepository().Insert(audit); err != nil {
		logger.Warn().Err(err).Str("customer_id", plan.Customer.ID).Msg("Failed to record customer transfer audit row")
	}

	// The customer's route leaves the donor tenant and joins the recipient.
	alerting.ProvisionDefaultConfigAsync(plan.From.ID, "customer", plan.Customer.ID)
	alerting.ProvisionDefaultConfigAsync(plan.To.ID, "customer", plan.Customer.ID)

	logger.Info().
		Str("customer_id", plan.Customer.ID).
		Str("from_org_id", plan.From.ID).
		Str("to_org_id", plan.To.ID).
		Int("systems_count", plan.SystemsCount).
		Int("renewals_stopped", stopped).
		Int("silences_cleared", cleared).
		Int("assignments_released", released).
		Str("transferred_by_user_id", actor.UserID).
		Str("transferred_by_org_id", actor.OrgID).
		Msg("Customer transferred to another manager")

	return &applied, nil
}

// transferRenewalOutcome decides what a transfer does to a grant's shop
// renewal. A perpetual grant has nothing to renew. Otherwise the renewal
// survives only when the buyer is known and still manages the customer
// (keepOrgIDs: the target chain plus the customer); an unknown buyer cannot
// be told apart from the manager the customer leaves.
func transferRenewalOutcome(g *models.OrganizationEntitlementGrant, keepOrgIDs []string) string {
	switch {
	case g.ValidUntil == nil:
		return models.TransferRenewalPerpetual
	case g.RenewalStoppedAt != nil:
		return models.TransferRenewalAlreadyStopped
	case g.PurchasedByOrgID != "" && slices.Contains(keepOrgIDs, g.PurchasedByOrgID):
		return models.TransferRenewalContinues
	default:
		return models.TransferRenewalStops
	}
}

// managingOrganization resolves a reseller or distributor by logto_id.
func (s *LocalOrganizationService) managingOrganization(orgID string) (org models.CustomerTransferOrganization, active, found bool) {
	if orgID == "" {
		return org, false, false
	}
	if reseller, err := s.resellerRepo.GetByID(orgID); err == nil {
		return models.CustomerTransferOrganization{ID: orgID, Name: reseller.Name, Type: "reseller"}, reseller.Active(), true
	}
	if distributor, err := s.distributorRepo.GetByID(orgID); err == nil {
		return models.CustomerTransferOrganization{ID: orgID, Name: distributor.Name, Type: "distributor"}, distributor.Active(), true
	}
	return org, false, false
}

func liveSystemKeys(orgID string) ([]string, error) {
	rows, err := database.DB.Query(`
		SELECT system_key FROM systems
		WHERE organization_id = $1 AND deleted_at IS NULL AND COALESCE(system_key, '') <> ''
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list customer system keys: %w", err)
	}
	defer func() { _ = rows.Close() }()
	keys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("failed to scan system key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// transferTargetValidationError reports an unusable target with a stable
// code: not_in_hierarchy (outside the caller's hierarchy) or not_a_manager
// (not a reseller or distributor).
func transferTargetValidationError(targetOrgID, code string) error {
	return &ValidationError{
		StatusCode: 400,
		ErrorData: response.ErrorData{
			Type: "validation_error",
			Errors: []response.ValidationError{{
				Key:     "organization_id",
				Message: code,
				Value:   targetOrgID,
			}},
		},
	}
}
