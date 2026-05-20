/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"context"
	"fmt"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// ErrChainTooDeep is returned by ResolveAncestorChain when an org's parent
// chain exceeds the maximum hop count without reaching the Owner. Callers
// treat this as a hard error rather than continuing with a possibly
// truncated chain — silently dropping upstream layers can underspecify the
// effective config (recipients missing, channels accidentally disabled).
var ErrChainTooDeep = fmt.Errorf("ancestor chain exceeds max depth")

// MaxChainDepth caps how deep ResolveAncestorChain will walk. The
// application's hierarchy is fixed at 4 levels (Owner -> Distributor ->
// Reseller -> Customer), so 8 leaves comfortable margin while still
// catching pathological data (cycles, runaway parent chains).
const MaxChainDepth = 8

// ResolveAncestorChain returns the list of org IDs from the Owner (top) down
// to the given tenant (inclusive). The chain is built by walking the
// `custom_data->>'createdBy'` field of distributors/resellers/customers up
// to the org that has no parent recorded (the Owner). The Owner's org id is
// included as the first element so the merge picks up its layer when present.
//
// Cycle protection: capped at MaxChainDepth hops. If a chain would exceed
// the cap (cycle or genuinely too deep), returns ErrChainTooDeep so the
// caller can fail closed rather than render an incomplete config.
func ResolveAncestorChain(tenantOrgID string) ([]string, error) {
	if tenantOrgID == "" {
		return nil, fmt.Errorf("tenantOrgID is required")
	}
	chain := []string{tenantOrgID}
	current := tenantOrgID
	for i := 0; i < MaxChainDepth; i++ {
		parent, err := lookupCreatedBy(current)
		if err != nil {
			return nil, fmt.Errorf("walk hierarchy at %s: %w", current, err)
		}
		if parent == "" {
			return chain, nil
		}
		// Detect a cycle defensively: same id seen twice would loop forever.
		for _, seen := range chain {
			if seen == parent {
				return chain, nil
			}
		}
		chain = append([]string{parent}, chain...)
		current = parent
	}
	// MaxChainDepth exhausted without hitting an Owner (no-parent) row.
	// Either a cycle slipped through the in-loop check or the hierarchy is
	// deeper than the model supports. Either way, fail closed.
	return nil, fmt.Errorf("%w (tenant=%s, depth=%d)", ErrChainTooDeep, tenantOrgID, MaxChainDepth)
}

// lookupCreatedBy returns the createdBy field of the org matching orgID
// across the three org tables, or "" when not found (which is how we
// recognize the Owner — its org id is referenced in createdBy of others
// but doesn't have a row in any of the three tables).
func lookupCreatedBy(orgID string) (string, error) {
	row := database.DB.QueryRow(
		`SELECT custom_data->>'createdBy' FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL
		 UNION ALL
		 SELECT custom_data->>'createdBy' FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL
		 UNION ALL
		 SELECT custom_data->>'createdBy' FROM customers WHERE logto_id = $1 AND deleted_at IS NULL
		 LIMIT 1`,
		orgID,
	)
	var parent *string
	err := row.Scan(&parent)
	if err != nil {
		// sql.ErrNoRows means: orgID is not in any of the three tables.
		// Treat this as "no parent" (likely the Owner). Other errors bubble.
		if err.Error() == "sql: no rows in result set" {
			return "", nil
		}
		return "", err
	}
	if parent == nil {
		return "", nil
	}
	return *parent, nil
}

// computeEffectiveLayer walks the tenant's ancestor chain and merges every layer Owner→tenant (orgs with no row contribute nothing); used by RenderAndPushEffective for the Mimir push.
func computeEffectiveLayer(tenantOrgID string) (models.AlertingConfigLayer, error) {
	chain, err := ResolveAncestorChain(tenantOrgID)
	if err != nil {
		return models.AlertingConfigLayer{}, err
	}
	repo := entities.NewLocalAlertConfigLayersRepository()
	layersByOrg, err := repo.GetByOrgIDs(chain)
	if err != nil {
		return models.AlertingConfigLayer{}, err
	}

	ordered := make([]models.AlertingConfigLayer, 0, len(chain))
	for _, oid := range chain {
		rec, ok := layersByOrg[oid]
		if !ok {
			continue
		}
		ordered = append(ordered, rec.Config)
	}

	return MergeLayers(ordered), nil
}

// RenderAndPushEffective re-computes and pushes the effective Mimir
// alertmanager config for one tenant. Used by the propagation path: when
// any layer in a tenant's chain is saved, RenderAndPushEffective is invoked
// for every affected tenant to keep Mimir in sync.
func RenderAndPushEffective(ctx context.Context, tenantOrgID string) error {
	effective, err := computeEffectiveLayer(tenantOrgID)
	if err != nil {
		return fmt.Errorf("compute effective for %s: %w", tenantOrgID, err)
	}
	cfg := configuration.Config
	yamlConfig, err := RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		cfg.AlertingHistoryWebhookURL, cfg.AlertingHistoryWebhookSecret,
		&effective,
	)
	if err != nil {
		return fmt.Errorf("render YAML for %s: %w", tenantOrgID, err)
	}
	templateFiles, err := BuildTemplateFiles(cfg.AppURL)
	if err != nil {
		return fmt.Errorf("build templates for %s: %w", tenantOrgID, err)
	}
	if err := PushConfig(tenantOrgID, yamlConfig, templateFiles); err != nil {
		return fmt.Errorf("push config for %s: %w", tenantOrgID, err)
	}
	logger.Debug().Str("tenant", tenantOrgID).Msg("effective alerting config pushed to mimir")
	return nil
}

// EffectiveLayerContribution is one org's stored layer in a tenant's chain; HasLayer false = no row (no contribution), Layer is unredacted until RedactEffectiveConfigReport.
type EffectiveLayerContribution struct {
	OrganizationID   string                     `json:"organization_id"`
	OrganizationName string                     `json:"organization_name"`
	OrganizationRole string                     `json:"organization_role"`
	HasLayer         bool                       `json:"has_layer"`
	Layer            models.AlertingConfigLayer `json:"layer"`
	UpdatedByName    *string                    `json:"updated_by_name"`
	UpdatedAt        *time.Time                 `json:"updated_at"`
}

// EffectiveConfigReport is a tenant's per-layer chain + merged layer + rendered Mimir YAML; unredacted until RedactEffectiveConfigReport.
type EffectiveConfigReport struct {
	OrganizationID string                       `json:"organization_id"`
	Chain          []EffectiveLayerContribution `json:"chain"`
	Effective      models.AlertingConfigLayer   `json:"effective"`
	YAML           string                       `json:"yaml"`
}

// BuildEffectiveConfigReport resolves the chain, merges layers Owner→tenant, and renders the YAML; read-only, no Mimir push.
func BuildEffectiveConfigReport(tenantOrgID string) (EffectiveConfigReport, error) {
	chain, err := ResolveAncestorChain(tenantOrgID)
	if err != nil {
		return EffectiveConfigReport{}, err
	}
	repo := entities.NewLocalAlertConfigLayersRepository()
	layersByOrg, err := repo.GetByOrgIDs(chain)
	if err != nil {
		return EffectiveConfigReport{}, err
	}

	contributions := make([]EffectiveLayerContribution, 0, len(chain))
	ordered := make([]models.AlertingConfigLayer, 0, len(chain))
	for _, oid := range chain {
		ident, err := lookupOrgIdentity(oid)
		if err != nil {
			return EffectiveConfigReport{}, fmt.Errorf("resolve org identity for %s: %w", oid, err)
		}
		contribution := EffectiveLayerContribution{
			OrganizationID:   oid,
			OrganizationName: ident.Name,
			OrganizationRole: ident.Role,
		}
		if rec, ok := layersByOrg[oid]; ok {
			updatedAt := rec.UpdatedAt
			contribution.HasLayer = true
			contribution.Layer = rec.Config
			contribution.UpdatedByName = rec.UpdatedByName
			contribution.UpdatedAt = &updatedAt
			ordered = append(ordered, rec.Config)
		}
		contributions = append(contributions, contribution)
	}

	effective := MergeLayers(ordered)

	cfg := configuration.Config
	yamlConfig, err := RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		cfg.AlertingHistoryWebhookURL, cfg.AlertingHistoryWebhookSecret,
		&effective,
	)
	if err != nil {
		return EffectiveConfigReport{}, fmt.Errorf("render YAML for %s: %w", tenantOrgID, err)
	}

	return EffectiveConfigReport{
		OrganizationID: tenantOrgID,
		Chain:          contributions,
		Effective:      effective,
		YAML:           yamlConfig,
	}, nil
}

// orgIdentity is best-effort display metadata for an org in a chain.
type orgIdentity struct {
	Name string
	Role string
}

// lookupOrgIdentity returns orgID's name and role from the org tables; absent from all three = Owner.
func lookupOrgIdentity(orgID string) (orgIdentity, error) {
	row := database.DB.QueryRow(
		`SELECT name, 'distributor' AS role FROM distributors WHERE logto_id = $1 AND deleted_at IS NULL
		 UNION ALL
		 SELECT name, 'reseller' AS role FROM resellers WHERE logto_id = $1 AND deleted_at IS NULL
		 UNION ALL
		 SELECT name, 'customer' AS role FROM customers WHERE logto_id = $1 AND deleted_at IS NULL
		 LIMIT 1`,
		orgID,
	)
	var ident orgIdentity
	if err := row.Scan(&ident.Name, &ident.Role); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return orgIdentity{Role: "owner"}, nil
		}
		return orgIdentity{}, err
	}
	return ident, nil
}
