/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"reflect"
	"testing"

	"github.com/nethesis/my/backend/models"
)

func ptrTrue() *bool {
	v := true
	return &v
}

func ptrFalse() *bool {
	v := false
	return &v
}

func TestMergeLayers_Empty(t *testing.T) {
	out := MergeLayers(nil)
	if out.Enabled.Email == nil || *out.Enabled.Email {
		t.Errorf("expected email disabled, got %#v", out.Enabled.Email)
	}
	if len(out.EmailRecipients) != 0 || len(out.WebhookRecipients) != 0 || len(out.TelegramRecipients) != 0 {
		t.Errorf("expected empty lists, got %#v", out)
	}
}

func TestMergeLayers_BoolsOR(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{Enabled: models.ChannelToggles{Email: ptrFalse()}},
		{Enabled: models.ChannelToggles{Email: ptrTrue()}},
	})
	if !*out.Enabled.Email {
		t.Errorf("OR semantics: expected email=true (one layer turned it on)")
	}
}

func TestMergeLayers_EmailDedupByAddress(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.com", Severities: []string{"critical"}, Language: "en"}}},
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.com", Severities: []string{"warning"}, Language: "it"}}},
	})
	if len(out.EmailRecipients) != 1 {
		t.Fatalf("expected 1 deduped recipient, got %d", len(out.EmailRecipients))
	}
	got := out.EmailRecipients[0]
	if !reflect.DeepEqual(got.Severities, []string{"critical", "warning"}) {
		t.Errorf("expected merged severities critical+warning, got %#v", got.Severities)
	}
	if got.Language != "en" {
		t.Errorf("expected first-occurrence language en, got %q", got.Language)
	}
}

func TestMergeLayers_AllSeveritiesWidens(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.com", Severities: []string{"critical"}}}},
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.com", Severities: []string{}}}},
	})
	if len(out.EmailRecipients) != 1 {
		t.Fatalf("expected 1 deduped recipient, got %d", len(out.EmailRecipients))
	}
	if len(out.EmailRecipients[0].Severities) != 0 {
		t.Errorf("[] widening: expected empty (all severities), got %#v", out.EmailRecipients[0].Severities)
	}
}

func TestMergeLayers_WebhookDedupByURL(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{WebhookRecipients: []models.WebhookRecipient{{Name: "owner-slack", URL: "https://hooks.example/x"}}},
		{WebhookRecipients: []models.WebhookRecipient{{Name: "reseller-slack", URL: "https://hooks.example/x"}}},
	})
	if len(out.WebhookRecipients) != 1 {
		t.Fatalf("expected 1 deduped webhook, got %d", len(out.WebhookRecipients))
	}
	if out.WebhookRecipients[0].Name != "owner-slack" {
		t.Errorf("first-occurrence name expected, got %q", out.WebhookRecipients[0].Name)
	}
}

func TestMergeLayers_TelegramDedupByBotAndChat(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{TelegramRecipients: []models.TelegramRecipient{{BotToken: "tok1", ChatID: -100}}},
		{TelegramRecipients: []models.TelegramRecipient{{BotToken: "tok1", ChatID: -100}}},
		{TelegramRecipients: []models.TelegramRecipient{{BotToken: "tok2", ChatID: -100}}},
	})
	if len(out.TelegramRecipients) != 2 {
		t.Fatalf("expected 2 telegram recipients (different bots), got %d", len(out.TelegramRecipients))
	}
}

func TestNormalizeLayerForRole_StripsFalseForNonOwner(t *testing.T) {
	layer := models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrFalse(), Webhook: ptrTrue()},
	}
	NormalizeLayerForRole(&layer, "Reseller")
	if layer.Enabled.Email != nil {
		t.Errorf("non-Owner explicit false must be normalised to nil, got %#v", layer.Enabled.Email)
	}
	if layer.Enabled.Webhook == nil || !*layer.Enabled.Webhook {
		t.Errorf("non-Owner explicit true must be preserved, got %#v", layer.Enabled.Webhook)
	}
}

func TestNormalizeLayerForRole_OwnerKeepsFalse(t *testing.T) {
	layer := models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Telegram: ptrFalse()},
	}
	NormalizeLayerForRole(&layer, "Owner")
	if layer.Enabled.Telegram == nil || *layer.Enabled.Telegram {
		t.Errorf("Owner explicit false must be preserved, got %#v", layer.Enabled.Telegram)
	}
}

func TestNormalizeSeverities_CanonicalOrderAndDrop(t *testing.T) {
	got := normalizeSeverities([]string{"info", "critical", "bogus", "critical"})
	want := []string{"critical", "info"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalizeSeverities: got %v, want %v", got, want)
	}
}

func TestUnionSeverities_EmptyWidens(t *testing.T) {
	got := unionSeverities([]string{"critical"}, []string{})
	if len(got) != 0 {
		t.Errorf("empty side widens: got %v, want []", got)
	}
}
