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
// pushes the EFFECTIVE merged config for that org's tenant to Mimir, so any
// layers already saved by ancestors (Owner/Distributor/Reseller) take effect
// immediately. The new org itself starts with no layer of its own; the
// admin can save one later via POST /alerts/config.
//
// defaultEmail (typically the org's contact email captured at creation time)
// and defaultLang are kept as a convenience for first-time provisioning when
// no ancestor has saved a layer yet. They render into the YAML directly,
// without being persisted as a layer — the user is expected to confirm and
// save them via the UI to make them part of the layered model.
//
// The built-in history webhook is always active so resolved alerts are
// persisted in alert_history regardless of the admin's choices.
func ProvisionDefaultConfig(orgID, defaultEmail, defaultLang string) error {
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	// Compute the effective merged config from any ancestor layers that exist.
	effective, _, err := ComputeEffectiveConfig(orgID)
	if err != nil {
		// Non-fatal: fall back to the legacy default-email/lang behavior so
		// the org is at least provisioned with a valid YAML.
		logger.Warn().Err(err).Str("org_id", orgID).Msg("could not compute effective config at provision; using local defaults")
	}

	// If no ancestor has populated anything, seed the org's first push with
	// the local defaults (email pre-filled, channel disabled, lang chosen).
	if len(effective.MailAddresses) == 0 && defaultEmail != "" {
		effective.MailAddresses = []string{defaultEmail}
	}
	if effective.EmailTemplateLang == "" {
		switch defaultLang {
		case "it", "en":
			effective.EmailTemplateLang = defaultLang
		}
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

	templateFiles, err := BuildTemplateFiles(effective.EmailTemplateLang, cfg.AppURL)
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
