/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"fmt"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// systemsLookup returns the live systems owned by one org, loaded lazily so
// orgs whose chain carries no system scope never hit the database.
type systemsLookup func() (map[string]struct{}, error)

// scopeForOrg narrows one recipient scope to the alerts of a single org.
// keep=false drops the recipient for this org. A nil result means the
// recipient gets every alert of the org; otherwise only the listed systems.
func scopeForOrg(s *models.RecipientScope, chain map[string]struct{}, systems systemsLookup) (*models.RecipientScope, bool, error) {
	if s.IsEmpty() {
		return nil, true, nil
	}
	for _, id := range s.OrganizationIDs {
		if _, ok := chain[id]; ok {
			return nil, true, nil
		}
	}
	if len(s.SystemIDs) == 0 {
		return nil, false, nil
	}
	owned, err := systems()
	if err != nil {
		return nil, false, err
	}
	kept := []string{}
	for _, id := range s.SystemIDs {
		if _, ok := owned[id]; ok {
			kept = append(kept, id)
		}
	}
	if len(kept) == 0 {
		return nil, false, nil
	}
	return &models.RecipientScope{SystemIDs: kept}, true, nil
}

// scopeLayerForOrg resolves every recipient scope of layer against orgID,
// whose ancestor chain (Owner→org, inclusive) is chain. Organization scopes
// become "all alerts" when they hit the chain, system scopes keep only the
// systems orgID owns, and recipients left with nothing are dropped.
func scopeLayerForOrg(layer models.AlertingConfigLayer, chain map[string]struct{}, systems systemsLookup) (models.AlertingConfigLayer, error) {
	out := layer
	out.EmailRecipients = make([]models.EmailRecipient, 0, len(layer.EmailRecipients))
	for _, r := range layer.EmailRecipients {
		scope, keep, err := scopeForOrg(r.Scope, chain, systems)
		if err != nil {
			return out, err
		}
		if keep {
			r.Scope = scope
			out.EmailRecipients = append(out.EmailRecipients, r)
		}
	}
	out.WebhookRecipients = make([]models.WebhookRecipient, 0, len(layer.WebhookRecipients))
	for _, r := range layer.WebhookRecipients {
		scope, keep, err := scopeForOrg(r.Scope, chain, systems)
		if err != nil {
			return out, err
		}
		if keep {
			r.Scope = scope
			out.WebhookRecipients = append(out.WebhookRecipients, r)
		}
	}
	out.TelegramRecipients = make([]models.TelegramRecipient, 0, len(layer.TelegramRecipients))
	for _, r := range layer.TelegramRecipients {
		scope, keep, err := scopeForOrg(r.Scope, chain, systems)
		if err != nil {
			return out, err
		}
		if keep {
			r.Scope = scope
			out.TelegramRecipients = append(out.TelegramRecipients, r)
		}
	}
	return out, nil
}

// lazyOrgSystems returns a systemsLookup that loads orgID's live systems on
// first use and caches them.
func lazyOrgSystems(orgID string) systemsLookup {
	var cached map[string]struct{}
	return func() (map[string]struct{}, error) {
		if cached != nil {
			return cached, nil
		}
		rows, err := database.DB.Query(
			`SELECT id FROM systems WHERE organization_id = $1 AND deleted_at IS NULL`,
			orgID,
		)
		if err != nil {
			return nil, fmt.Errorf("load systems of %s: %w", orgID, err)
		}
		defer func() { _ = rows.Close() }()
		owned := map[string]struct{}{}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			owned[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		cached = owned
		return cached, nil
	}
}

// effectiveLayerFromChain scopes every stored layer of chain to the org at
// its tail and merges them Owner→org.
func effectiveLayerFromChain(chain []string, layers []models.AlertingConfigLayer) (models.AlertingConfigLayer, error) {
	chainSet := make(map[string]struct{}, len(chain))
	for _, id := range chain {
		chainSet[id] = struct{}{}
	}
	systems := lazyOrgSystems(chain[len(chain)-1])
	scoped := make([]models.AlertingConfigLayer, 0, len(layers))
	for _, l := range layers {
		sl, err := scopeLayerForOrg(l, chainSet, systems)
		if err != nil {
			return models.AlertingConfigLayer{}, err
		}
		scoped = append(scoped, sl)
	}
	return MergeLayers(scoped), nil
}
