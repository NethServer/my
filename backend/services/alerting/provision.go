/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"fmt"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// provisionRetryDelays controls the backoff between retry attempts when
// pushing the default config to Mimir fails with a transient error.
var provisionRetryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// ProvisionDefaultConfig is called when a new organization is created. It
// resolves the org's Mimir tenant (its managing reseller, or itself for a
// non-customer org — see TenantForOrg) and re-renders+pushes that whole tenant
// config so the new org's route (and any ancestor layers already saved) take
// effect immediately. The new org itself starts with no layer of its own; the
// admin opts in to notifications by saving a layer via POST /alerts/config.
//
// The built-in history webhook is always active so resolved alerts are
// persisted in alert_history regardless of admin choices.
func ProvisionDefaultConfig(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	// The tenant that actually holds this org's alerting (reseller for a
	// customer, else the org itself). Fail closed on resolution error: a
	// misconfigured hierarchy must NOT silently provision the wrong tenant.
	tenant, err := TenantForOrg(orgID)
	if err != nil {
		return fmt.Errorf("resolve tenant at provision: %w", err)
	}

	// Render the full nested tenant config (reseller + all its customers). Fail
	// closed: a misconfigured hierarchy (cycle, missing parent row, transient
	// DB error) must NOT silently provision a less-protected config than the
	// Owner intended. The org creation flow can retry.
	yamlConfig, templateFiles, err := renderTenantConfig(tenant)
	if err != nil {
		return fmt.Errorf("render default alerting config at provision: %w", err)
	}

	// Retry with backoff to tolerate transient Mimir errors (startup delays,
	// brief network hiccups). Total max wait: ~9 seconds across 4 attempts.
	var lastErr error
	for attempt := 0; attempt <= len(provisionRetryDelays); attempt++ {
		if attempt > 0 {
			delay := provisionRetryDelays[attempt-1]
			logger.Warn().
				Err(lastErr).
				Str("org_id", orgID).
				Str("tenant", tenant).
				Int("attempt", attempt).
				Dur("delay", delay).
				Msg("retrying default alerting config push to mimir")
			time.Sleep(delay)
		}

		if err := PushConfig(tenant, yamlConfig, templateFiles); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	return fmt.Errorf("pushing default alerting config for tenant %s after %d attempts: %w", tenant, len(provisionRetryDelays)+1, lastErr)
}

// ProvisionDefaultConfigAsync runs ProvisionDefaultConfig in the background so
// a slow or struggling Mimir cannot delay the caller's HTTP response: creating
// a customer/reseller/distributor must not block on this. entityType/entityID
// are only used to correlate the failure log with the org that triggered it.
func ProvisionDefaultConfigAsync(orgID, entityType, entityID string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().
					Interface("panic", r).
					Str("org_id", orgID).
					Str("entity_type", entityType).
					Str("entity_id", entityID).
					Msg("panic while provisioning default alerting config")
			}
		}()

		if err := ProvisionDefaultConfig(orgID); err != nil {
			logger.Warn().
				Err(err).
				Str("org_id", orgID).
				Str("entity_type", entityType).
				Str("entity_id", entityID).
				Msg("failed to provision default alerting config")
		}
	}()
}
