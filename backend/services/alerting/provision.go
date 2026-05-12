/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"fmt"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// provisionRetryDelays controls the backoff between retry attempts when
// pushing the default config to Mimir fails with a transient error.
var provisionRetryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// ProvisionDefaultConfig is called when a new organization is created. It
// pushes the effective merged config for that org's tenant to Mimir so any
// layers already saved by ancestors (Owner/Distributor/Reseller) take
// effect immediately. The new org itself starts with no layer of its own;
// the admin opts in to notifications by saving a layer via POST /alerts/config.
//
// The built-in history webhook is always active so resolved alerts are
// persisted in alert_history regardless of admin choices.
func ProvisionDefaultConfig(orgID string) error {
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	// Compute the effective merged config from any ancestor layers that
	// exist. Fail closed: a misconfigured hierarchy (cycle, missing parent
	// row, transient DB error) must NOT silently provision a less-protected
	// config than the Owner intended. The org creation flow can retry; the
	// alternative — "fall back to local defaults" — risks losing Owner-set
	// recipients/severity rules during a window we cannot otherwise detect.
	effective, err := computeEffectiveLayer(orgID)
	if err != nil {
		return fmt.Errorf("compute effective config at provision: %w", err)
	}

	cfg := configuration.Config
	yamlConfig, err := RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		cfg.AlertingHistoryWebhookURL, cfg.AlertingHistoryWebhookSecret,
		&effective,
	)
	if err != nil {
		return fmt.Errorf("rendering default alerting config: %w", err)
	}

	templateFiles, err := BuildTemplateFiles(cfg.AppURL)
	if err != nil {
		return fmt.Errorf("building default alerting templates: %w", err)
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
				Int("attempt", attempt).
				Dur("delay", delay).
				Msg("retrying default alerting config push to mimir")
			time.Sleep(delay)
		}

		if err := PushConfig(orgID, yamlConfig, templateFiles); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	return fmt.Errorf("pushing default alerting config for %s after %d attempts: %w", orgID, len(provisionRetryDelays)+1, lastErr)
}
