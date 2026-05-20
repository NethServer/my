/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"strings"
	"testing"

	"github.com/nethesis/my/backend/models"
)

func TestRedactEffectiveConfigReport(t *testing.T) {
	secretLayer := models.AlertingConfigLayer{
		WebhookRecipients: []models.WebhookRecipient{
			{Name: "slack", URL: "https://hooks.slack.com/services/T00/B00/XXXSECRET"},
		},
		TelegramRecipients: []models.TelegramRecipient{
			{BotToken: "123456:super-secret-token", ChatID: 42},
		},
	}
	report := EffectiveConfigReport{
		OrganizationID: "org-tenant",
		Chain: []EffectiveLayerContribution{
			{OrganizationID: "org-owner", OrganizationRole: "owner", HasLayer: true, Layer: secretLayer},
		},
		Effective: secretLayer,
		YAML:      "global:\n  smtp_auth_password: 'hunter2'\nreceivers:\n  - name: wh\n    webhook_configs:\n      - url: 'https://hooks.slack.com/services/T00/B00/XXXSECRET'\n  - name: tg\n    telegram_configs:\n      - bot_token: 123456:super-secret-token\n",
	}

	out := RedactEffectiveConfigReport(report)

	if got := out.Chain[0].Layer.TelegramRecipients[0].BotToken; got != RedactedSecretPlaceholder {
		t.Errorf("chain telegram token not redacted: %q", got)
	}
	if got := out.Chain[0].Layer.WebhookRecipients[0].URL; strings.Contains(got, "XXXSECRET") {
		t.Errorf("chain webhook url leaked secret: %q", got)
	}
	if got := out.Effective.TelegramRecipients[0].BotToken; got != RedactedSecretPlaceholder {
		t.Errorf("effective telegram token not redacted: %q", got)
	}
	if strings.Contains(out.YAML, "hunter2") || strings.Contains(out.YAML, "super-secret-token") || strings.Contains(out.YAML, "XXXSECRET") {
		t.Errorf("yaml leaked secrets: %q", out.YAML)
	}
	if !strings.Contains(out.YAML, "https://hooks.slack.com/"+RedactedSecretPlaceholder) {
		t.Errorf("yaml webhook url not masked to host: %q", out.YAML)
	}

	// Original report must be left untouched (defensive copy).
	if report.Effective.TelegramRecipients[0].BotToken != "123456:super-secret-token" {
		t.Errorf("source report mutated: %q", report.Effective.TelegramRecipients[0].BotToken)
	}
}
