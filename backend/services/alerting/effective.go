/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"context"
	"fmt"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/entities"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// ResolveAncestorChain returns the list of org IDs from the Owner (top) down
// to the given tenant (inclusive). The chain is built by walking the
// `custom_data->>'createdBy'` field of distributors/resellers/customers up
// to the org that has no parent recorded (the Owner). The Owner's org id is
// included as the first element so the merge picks up its layer when present.
//
// Cycle protection: capped at 8 hops, which is far more than the 4-level
// hierarchy the application supports (Owner -> Distributor -> Reseller ->
// Customer).
func ResolveAncestorChain(tenantOrgID string) ([]string, error) {
	if tenantOrgID == "" {
		return nil, fmt.Errorf("tenantOrgID is required")
	}
	chain := []string{tenantOrgID}
	current := tenantOrgID
	const maxHops = 8
	for i := 0; i < maxHops; i++ {
		parent, err := lookupCreatedBy(current)
		if err != nil {
			return nil, fmt.Errorf("walk hierarchy at %s: %w", current, err)
		}
		if parent == "" {
			break
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
	return chain, nil
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

// ComputeEffectiveConfig returns the merged AlertingConfig for the given
// tenant by walking its ancestor chain, fetching every layer in a single
// round-trip, and merging in order from Owner to tenant. Empty layers
// (orgs with no row in alert_config_layers) contribute nothing but don't
// break the chain.
func ComputeEffectiveConfig(tenantOrgID string) (models.AlertingConfig, []entities.AlertConfigLayerRecord, error) {
	chain, err := ResolveAncestorChain(tenantOrgID)
	if err != nil {
		return models.AlertingConfig{}, nil, err
	}
	repo := entities.NewLocalAlertConfigLayersRepository()
	layersByOrg, err := repo.GetByOrgIDs(chain)
	if err != nil {
		return models.AlertingConfig{}, nil, err
	}

	ordered := make([]models.AlertingConfigLayer, 0, len(chain))
	provenance := make([]entities.AlertConfigLayerRecord, 0, len(chain))
	for _, oid := range chain {
		rec, ok := layersByOrg[oid]
		if !ok {
			continue
		}
		ordered = append(ordered, rec.Config)
		provenance = append(provenance, *rec)
	}

	return MergeLayers(ordered), provenance, nil
}

// RenderAndPushEffective re-computes and pushes the effective Mimir
// alertmanager config for one tenant. Used by the propagation path: when
// any layer in a tenant's chain is saved, RenderAndPushEffective is invoked
// for every affected tenant to keep Mimir in sync.
//
// Renders the same YAML we've always pushed (using the existing RenderConfig
// path); this is what makes the layered model invisible to Alertmanager.
func RenderAndPushEffective(ctx context.Context, tenantOrgID string) error {
	effective, _, err := ComputeEffectiveConfig(tenantOrgID)
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
	templateFiles, err := BuildTemplateFiles(effective.EmailTemplateLang, cfg.AppURL)
	if err != nil {
		return fmt.Errorf("build templates for %s: %w", tenantOrgID, err)
	}
	if err := PushConfig(tenantOrgID, yamlConfig, templateFiles); err != nil {
		return fmt.Errorf("push config for %s: %w", tenantOrgID, err)
	}
	logger.Debug().Str("tenant", tenantOrgID).Msg("effective alerting config pushed to mimir")
	return nil
}
