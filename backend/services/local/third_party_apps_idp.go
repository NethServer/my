/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

// Hiding a portal tile is not a gate: the application's login URL is public.
// For the applications the sync config flags idp_enforced, the backend mirrors
// the ORGANIZATION dimension of the portal filter into Logto app-level access
// control, so a user outside the admitted organizations is refused at sign-in.
//
// Only the organization dimension is enforceable. Logto combines its rule lists
// with OR, so a user-role rule next to an organization rule would admit that
// role in every organization: the technical-role filter stays on the portal
// (and in the applications that check the token). The organizations admitted
// for an application are: the Owner organization; every distributor, when the
// application admits the distributor role; the resellers and customers whose
// distributor grants the portal, when their role is admitted. An application
// pinned to organization_ids gets exactly those.
//
// The reconcile is idempotent and runs after organization changes (debounced),
// periodically, and on demand from the owner-only endpoint. It never enables
// Logto enforcement with an empty organization list: that would lock everybody
// out, so it reports the problem and leaves the application as it is.

// IdPAccessReport is the outcome of the reconcile for one managed application.
type IdPAccessReport struct {
	Application     string   `json:"application"`
	ApplicationID   string   `json:"application_id"`
	Enforced        bool     `json:"enforced"`
	OrganizationIDs []string `json:"organization_ids"`
	Enabled         bool     `json:"enabled"`
	RulesChanged    bool     `json:"rules_changed"`
	EnabledChanged  bool     `json:"enabled_changed"`
	Error           string   `json:"error,omitempty"`
}

var (
	// idpReconcileRunMu serializes reconciles: two concurrent runs would race on
	// the same PUT.
	idpReconcileRunMu sync.Mutex

	idpReconcileRequestMu sync.Mutex
	idpReconcilePending   bool
)

// ReconcileIdPAccess aligns Logto app-level access control with the portal
// lists for every third-party application whose access_control carries
// idp_enforced. With dryRun nothing is written: the report says what would
// change.
func (s *ThirdPartyAppsService) ReconcileIdPAccess(dryRun bool) ([]IdPAccessReport, error) {
	idpReconcileRunMu.Lock()
	defer idpReconcileRunMu.Unlock()

	client := logto.NewManagementClient()
	apps, err := client.GetThirdPartyApplications()
	if err != nil {
		return nil, fmt.Errorf("failed to list third-party applications: %w", err)
	}

	ownerOrgID, err := NewOrganizationService().resolveOwnerOrgID("")
	if err != nil {
		if !errors.Is(err, ErrPromoteOwnerOrgUnknown) {
			return nil, fmt.Errorf("failed to resolve the owner organization: %w", err)
		}
		logger.Warn().Msg("owner organization unknown (no distributor yet): IdP rules will not include it")
		ownerOrgID = ""
	}

	reports := make([]IdPAccessReport, 0)
	for _, app := range apps {
		accessControl := app.ExtractAccessControlFromCustomData()
		if accessControl == nil || accessControl.IDPEnforced == nil {
			continue
		}
		reports = append(reports, s.reconcileApplication(client, app, accessControl, ownerOrgID, dryRun))
	}

	changed := 0
	failed := 0
	for _, r := range reports {
		if r.RulesChanged || r.EnabledChanged {
			changed++
		}
		if r.Error != "" {
			failed++
		}
	}
	logger.Info().
		Bool("dry_run", dryRun).
		Int("managed", len(reports)).
		Int("changed", changed).
		Int("failed", failed).
		Msg("Reconciled Logto app-level access control for third-party applications")

	return reports, nil
}

func (s *ThirdPartyAppsService) reconcileApplication(client *logto.LogtoManagementClient, app models.LogtoThirdPartyApp, accessControl *models.AccessControl, ownerOrgID string, dryRun bool) IdPAccessReport {
	report := IdPAccessReport{
		Application:     app.Name,
		ApplicationID:   app.ID,
		Enforced:        *accessControl.IDPEnforced,
		OrganizationIDs: []string{},
		Enabled:         app.AppLevelAccessControlEnabled,
	}

	// idp_enforced: false is the kill switch: Logto must not enforce anything
	// for this application, whatever rules it still carries.
	if !report.Enforced {
		if app.AppLevelAccessControlEnabled {
			report.EnabledChanged = true
			report.Enabled = false
			if !dryRun {
				if err := client.SetApplicationAccessControlEnabled(app.ID, false); err != nil {
					report.Error = err.Error()
					report.Enabled = true
					report.EnabledChanged = false
				}
			}
		}
		return report
	}

	orgIDs, err := s.organizationsAdmittedAtIdP(app.Name, accessControl, ownerOrgID)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.OrganizationIDs = orgIDs
	if len(orgIDs) == 0 {
		report.Error = "no organization admitted: refusing to enforce an empty rule"
		return report
	}

	current, err := client.GetApplicationAccessControl(app.ID)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	desired := logto.ApplicationAccessControl{OrganizationIDs: orgIDs}
	if !sameOrganizationRules(*current, desired) {
		report.RulesChanged = true
		if !dryRun {
			if err := client.ReplaceApplicationAccessControl(app.ID, desired); err != nil {
				report.Error = err.Error()
				report.RulesChanged = false
				// Never turn enforcement on over a rule set we could not write.
				return report
			}
		}
	}

	if !app.AppLevelAccessControlEnabled {
		report.EnabledChanged = true
		report.Enabled = true
		if !dryRun {
			if err := client.SetApplicationAccessControlEnabled(app.ID, true); err != nil {
				report.Error = err.Error()
				report.Enabled = false
				report.EnabledChanged = false
			}
		}
	}

	return report
}

// organizationsAdmittedAtIdP lists the organizations Logto must admit for an
// application: exactly organization_ids when the application pins them,
// otherwise the Owner organization plus the partner organizations admitted by
// organization_roles, with resellers and customers filtered by the portal
// list of the distributor at the top of their branch.
func (s *ThirdPartyAppsService) organizationsAdmittedAtIdP(appName string, accessControl *models.AccessControl, ownerOrgID string) ([]string, error) {
	if len(accessControl.OrganizationIDs) > 0 {
		return models.NormalizeThirdPartyAppNames(accessControl.OrganizationIDs), nil
	}

	roles := make(map[string]bool, len(accessControl.OrganizationRoles))
	for _, role := range accessControl.OrganizationRoles {
		roles[strings.ToLower(role)] = true
	}

	ids := []string{}
	if ownerOrgID != "" {
		ids = append(ids, ownerOrgID)
	}

	queries := []struct {
		role  string
		query string
	}{
		// Distributors are never bound by a portal list: the role decides.
		{"distributor", `SELECT logto_id FROM distributors WHERE deleted_at IS NULL AND logto_id IS NOT NULL`},
		{"reseller", `
			SELECT r.logto_id
			FROM resellers r
			JOIN distributors d ON d.logto_id = r.custom_data->>'createdBy' AND d.deleted_at IS NULL
			WHERE r.deleted_at IS NULL AND r.logto_id IS NOT NULL
			  AND $1 = ANY(d.third_party_apps)`},
		{"customer", `
			SELECT c.logto_id
			FROM customers c
			LEFT JOIN distributors d_direct
			       ON d_direct.logto_id = c.custom_data->>'createdBy' AND d_direct.deleted_at IS NULL
			LEFT JOIN resellers r
			       ON r.logto_id = c.custom_data->>'createdBy' AND r.deleted_at IS NULL
			LEFT JOIN distributors d_via_reseller
			       ON d_via_reseller.logto_id = r.custom_data->>'createdBy' AND d_via_reseller.deleted_at IS NULL
			WHERE c.deleted_at IS NULL AND c.logto_id IS NOT NULL
			  AND $1 = ANY(COALESCE(d_direct.third_party_apps, d_via_reseller.third_party_apps))`},
	}

	for _, q := range queries {
		if !roles[q.role] {
			continue
		}
		var rows interface {
			Next() bool
			Scan(dest ...interface{}) error
			Close() error
			Err() error
		}
		var err error
		if strings.Contains(q.query, "$1") {
			rows, err = database.DB.Query(q.query, appName)
		} else {
			rows, err = database.DB.Query(q.query)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list %s organizations admitted to %s: %w", q.role, appName, err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("failed to scan %s organization: %w", q.role, err)
			}
			ids = append(ids, id)
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, fmt.Errorf("failed to iterate %s organizations: %w", q.role, err)
		}
		_ = rows.Close()
	}

	ids = models.NormalizeThirdPartyAppNames(ids)
	sort.Strings(ids)
	return ids, nil
}

// sameOrganizationRules reports whether the current Logto rule set already is
// the desired one: the same organizations, and nothing else admitted.
func sameOrganizationRules(current, desired logto.ApplicationAccessControl) bool {
	if len(current.UserIDs) > 0 || len(current.UserRoleIDs) > 0 || len(current.OrganizationRoleRules) > 0 {
		return false
	}
	if len(current.OrganizationIDs) != len(desired.OrganizationIDs) {
		return false
	}
	have := make(map[string]bool, len(current.OrganizationIDs))
	for _, id := range current.OrganizationIDs {
		have[id] = true
	}
	for _, id := range desired.OrganizationIDs {
		if !have[id] {
			return false
		}
	}
	return true
}

// RequestIdPAccessReconcile schedules a reconcile shortly after an organization
// change. A burst of changes (an import, a cascade) settles into one run.
func RequestIdPAccessReconcile() {
	idpReconcileRequestMu.Lock()
	if idpReconcilePending {
		idpReconcileRequestMu.Unlock()
		return
	}
	idpReconcilePending = true
	idpReconcileRequestMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("panic", r).Msg("panic while reconciling Logto app-level access control")
			}
		}()

		time.Sleep(5 * time.Second)
		idpReconcileRequestMu.Lock()
		idpReconcilePending = false
		idpReconcileRequestMu.Unlock()

		if _, err := NewThirdPartyAppsService().ReconcileIdPAccess(false); err != nil {
			logger.Warn().Err(err).Msg("failed to reconcile Logto app-level access control after an organization change")
		}
	}()
}

// StartIdPAccessReconciler runs a reconcile shortly after startup and then
// every interval, so a flag changed by sync or a push that failed is picked
// up without anybody asking. interval <= 0 disables the loop.
func StartIdPAccessReconciler(interval time.Duration) {
	if interval <= 0 {
		logger.Info().Msg("Periodic Logto app-level access control reconcile disabled")
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("panic", r).Msg("panic in the Logto app-level access control reconcile loop")
			}
		}()
		time.Sleep(30 * time.Second)
		for {
			if _, err := NewThirdPartyAppsService().ReconcileIdPAccess(false); err != nil {
				logger.Warn().Err(err).Msg("periodic reconcile of Logto app-level access control failed")
			}
			time.Sleep(interval)
		}
	}()
}
