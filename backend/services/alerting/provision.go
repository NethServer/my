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
	"github.com/nethesis/my/backend/models"
)

// provisionRetryDelays controls the backoff between retry attempts when
// pushing the default config to Mimir fails with a transient error.
var provisionRetryDelays = []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

// ProvisionDefaultConfig pushes a minimal default alerting configuration to Mimir
// for the given organization. The built-in history webhook is always active so
// resolved alerts are persisted in the alert_history table.
//
// If defaultEmail is non-empty, email notifications are enabled with the given
// address as the sole recipient. Otherwise email notifications are disabled.
// Webhook notifications are always disabled by default.
//
// defaultLang sets the email template language: "it" or "en". Invalid or empty
// values default to English.
//
// This is typically called when a new organization is created, to ensure that
// alerts received before the user configures alerting manually are still
// captured in the history and that the empty-receiver fallback is never used.
func ProvisionDefaultConfig(orgID, defaultEmail, defaultLang string) error {
	if orgID == "" {
		return fmt.Errorf("orgID is required")
	}

	// Normalize language: accept only "it" or "en", fall back to "en" otherwise.
	lang := ""
	switch defaultLang {
	case "it", "en":
		lang = defaultLang
	}

	cfg := configuration.Config
	defaultAlerting := &models.AlertingConfig{
		MailEnabled:       defaultEmail != "",
		WebhookEnabled:    false,
		MailAddresses:     []string{},
		WebhookReceivers:  []models.WebhookReceiver{},
		EmailTemplateLang: lang,
	}
	if defaultEmail != "" {
		defaultAlerting.MailAddresses = []string{defaultEmail}
	}

	yamlConfig, err := RenderConfig(
		cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom, cfg.SMTPTLS,
		cfg.AlertingHistoryWebhookURL, cfg.AlertingHistoryWebhookToken,
		defaultAlerting,
	)
	if err != nil {
		return fmt.Errorf("rendering default alerting config: %w", err)
	}

	templateFiles, err := BuildTemplateFiles(lang, cfg.AppURL)
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
