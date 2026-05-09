/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func ptrBool(b bool) *bool { return &b }

func TestMergeLayers_Empty(t *testing.T) {
	out := MergeLayers(nil)
	assert.False(t, out.MailEnabled)
	assert.False(t, out.WebhookEnabled)
	assert.False(t, out.TelegramEnabled)
	assert.Empty(t, out.MailAddresses)
	assert.Empty(t, out.WebhookReceivers)
	assert.Empty(t, out.TelegramReceivers)
	assert.Empty(t, out.Severities)
	assert.Empty(t, out.Systems)
	assert.Empty(t, out.EmailTemplateLang)
}

func TestMergeLayers_BoolsAreOR(t *testing.T) {
	// Owner enables mail; Reseller has no opinion; Customer explicitly false.
	// In the additive model the customer's false is normalized to nil before
	// storage; even if it leaked through, MergeLayers OR should still keep
	// mail enabled because Owner set true.
	out := MergeLayers([]models.AlertingConfigLayer{
		{MailEnabled: ptrBool(true)},
		{},
		{MailEnabled: ptrBool(false)},
	})
	assert.True(t, out.MailEnabled, "any true must win, descendant false cannot disable")
}

func TestMergeLayers_BoolsNilNeverEnables(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{{}, {}})
	assert.False(t, out.MailEnabled)
	assert.False(t, out.WebhookEnabled)
	assert.False(t, out.TelegramEnabled)
}

func TestMergeLayers_ListsUnionDedup(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{MailAddresses: []string{"a@x.it", "b@x.it"}},
		{MailAddresses: []string{"b@x.it", "c@x.it"}}, // b is dup
		{MailAddresses: []string{"  ", "d@x.it"}},     // empty/whitespace ignored
	})
	assert.Equal(t, []string{"a@x.it", "b@x.it", "c@x.it", "d@x.it"}, out.MailAddresses)
}

func TestMergeLayers_WebhookDedupByNameAndURL(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{WebhookReceivers: []models.WebhookReceiver{
			{Name: "slack", URL: "https://slack.example/a"},
			{Name: "pd", URL: "https://pagerduty/x"},
		}},
		{WebhookReceivers: []models.WebhookReceiver{
			{Name: "slack", URL: "https://slack.example/a"}, // exact dup
			{Name: "slack", URL: "https://slack.example/b"}, // same name, different URL → kept
		}},
	})
	assert.Len(t, out.WebhookReceivers, 3)
}

func TestMergeLayers_TelegramDedupByBotAndChat(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{TelegramReceivers: []models.TelegramReceiver{{BotToken: "abc", ChatID: -1001}}},
		{TelegramReceivers: []models.TelegramReceiver{{BotToken: "abc", ChatID: -1001}}}, // dup
		{TelegramReceivers: []models.TelegramReceiver{{BotToken: "abc", ChatID: -1002}}}, // different chat → kept
	})
	assert.Len(t, out.TelegramReceivers, 2)
}

func TestMergeLayers_LangDeepestWins(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{EmailTemplateLang: "en"}, // Owner
		{},                        // Distributor: no opinion
		{EmailTemplateLang: "it"}, // Reseller: chooses italian for its subtree
	})
	assert.Equal(t, "it", out.EmailTemplateLang)
}

func TestMergeLayers_LangAncestorIfNobodyOverrides(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{EmailTemplateLang: "en"},
		{},
		{},
	})
	assert.Equal(t, "en", out.EmailTemplateLang)
}

func TestMergeLayers_LangTrimsAndIgnoresEmpty(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{EmailTemplateLang: "en"},
		{EmailTemplateLang: "  "}, // whitespace = no opinion
	})
	assert.Equal(t, "en", out.EmailTemplateLang)
}

func TestMergeLayers_SeverityMergeByKey(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{Severities: []models.SeverityOverride{
			{Severity: "critical", MailEnabled: ptrBool(true), MailAddresses: []string{"oncall@msp.it"}},
		}},
		{Severities: []models.SeverityOverride{
			{Severity: "critical", MailAddresses: []string{"backup@cust.it"}},
			{Severity: "warning", MailEnabled: ptrBool(true), MailAddresses: []string{"low@cust.it"}},
		}},
	})

	// 3 standard severities ordered: critical, warning, (info dropped — not present)
	assert.Len(t, out.Severities, 2)
	// critical: union of two address lists, mail enabled (Owner)
	crit := findSeverity(out.Severities, "critical")
	assert.NotNil(t, crit)
	assert.NotNil(t, crit.MailEnabled)
	assert.True(t, *crit.MailEnabled)
	assert.ElementsMatch(t, []string{"oncall@msp.it", "backup@cust.it"}, crit.MailAddresses)
	// warning: only customer entry
	warn := findSeverity(out.Severities, "warning")
	assert.NotNil(t, warn)
	assert.True(t, *warn.MailEnabled)
}

func TestMergeLayers_SystemMergeByKey(t *testing.T) {
	out := MergeLayers([]models.AlertingConfigLayer{
		{Systems: []models.SystemOverride{
			{SystemKey: "NETH-A", MailAddresses: []string{"sysA@msp.it"}, WebhookEnabled: ptrBool(true)},
		}},
		{Systems: []models.SystemOverride{
			{SystemKey: "NETH-A", MailAddresses: []string{"sysA-cust@x.it"}},
			{SystemKey: "NETH-B", MailAddresses: []string{"sysB@x.it"}},
		}},
	})
	assert.Len(t, out.Systems, 2)
	a := findSystem(out.Systems, "NETH-A")
	assert.NotNil(t, a)
	assert.NotNil(t, a.WebhookEnabled)
	assert.True(t, *a.WebhookEnabled)
	assert.ElementsMatch(t, []string{"sysA@msp.it", "sysA-cust@x.it"}, a.MailAddresses)
}

func TestMergeLayers_OrderIndependentForBoolsAndLists(t *testing.T) {
	// Same inputs in different order produce the same output for bools+lists
	// (only EmailTemplateLang depends on order).
	a := []models.AlertingConfigLayer{
		{MailEnabled: ptrBool(true), MailAddresses: []string{"a@x"}},
		{MailAddresses: []string{"b@x"}},
	}
	b := []models.AlertingConfigLayer{
		{MailAddresses: []string{"b@x"}},
		{MailEnabled: ptrBool(true), MailAddresses: []string{"a@x"}},
	}
	outA := MergeLayers(a)
	outB := MergeLayers(b)
	assert.Equal(t, outA.MailEnabled, outB.MailEnabled)
	assert.ElementsMatch(t, outA.MailAddresses, outB.MailAddresses)
}

func TestNormalizeLayerForRole_ReplacesFalseWithNilForNonOwner(t *testing.T) {
	layer := &models.AlertingConfigLayer{
		MailEnabled:     ptrBool(false),
		WebhookEnabled:  ptrBool(true), // explicit enable preserved
		TelegramEnabled: ptrBool(false),
		Severities: []models.SeverityOverride{
			{Severity: "critical", MailEnabled: ptrBool(false)},
		},
	}
	NormalizeLayerForRole(layer, "customer")
	assert.Nil(t, layer.MailEnabled)
	assert.NotNil(t, layer.WebhookEnabled)
	assert.True(t, *layer.WebhookEnabled)
	assert.Nil(t, layer.TelegramEnabled)
	assert.Nil(t, layer.Severities[0].MailEnabled)
}

func TestNormalizeLayerForRole_OwnerKeepsFalse(t *testing.T) {
	layer := &models.AlertingConfigLayer{MailEnabled: ptrBool(false)}
	NormalizeLayerForRole(layer, "owner")
	assert.NotNil(t, layer.MailEnabled)
	assert.False(t, *layer.MailEnabled)
}

func findSeverity(list []models.SeverityOverride, sev string) *models.SeverityOverride {
	for i := range list {
		if list[i].Severity == sev {
			return &list[i]
		}
	}
	return nil
}

func findSystem(list []models.SystemOverride, key string) *models.SystemOverride {
	for i := range list {
		if list[i].SystemKey == key {
			return &list[i]
		}
	}
	return nil
}
