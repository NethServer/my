/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/backend/models"
)

func set(ids ...string) map[string]struct{} {
	m := map[string]struct{}{}
	for _, id := range ids {
		m[id] = struct{}{}
	}
	return m
}

func staticSystems(ids ...string) systemsLookup {
	return func() (map[string]struct{}, error) { return set(ids...), nil }
}

func TestScopeLayerForOrg(t *testing.T) {
	layer := models.AlertingConfigLayer{
		EmailRecipients: []models.EmailRecipient{
			{Address: "all@x.it"},
			{Address: "reseller@x.it", Scope: &models.RecipientScope{OrganizationIDs: []string{"reseller-1"}}},
			{Address: "other-org@x.it", Scope: &models.RecipientScope{OrganizationIDs: []string{"reseller-2"}}},
			{Address: "sys@x.it", Scope: &models.RecipientScope{SystemIDs: []string{"sys-a", "sys-z"}}},
			{Address: "foreign-sys@x.it", Scope: &models.RecipientScope{SystemIDs: []string{"sys-z"}}},
		},
	}
	chain := set("owner", "dist-1", "reseller-1", "cust-1")
	out, err := scopeLayerForOrg(layer, chain, staticSystems("sys-a", "sys-b"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]*models.RecipientScope{}
	for _, r := range out.EmailRecipients {
		got[r.Address] = r.Scope
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 recipients, got %v", got)
	}
	if got["all@x.it"] != nil || got["reseller@x.it"] != nil {
		t.Errorf("unscoped and chain-matching org scopes must reach every alert, got %v", got)
	}
	if s := got["sys@x.it"]; s == nil || len(s.SystemIDs) != 1 || s.SystemIDs[0] != "sys-a" {
		t.Errorf("system scope must keep only the org's systems, got %+v", s)
	}
}

func TestScopeLayerForOrg_SkipsSystemLookupWithoutSystemScope(t *testing.T) {
	layer := models.AlertingConfigLayer{
		EmailRecipients: []models.EmailRecipient{
			{Address: "a@x.it", Scope: &models.RecipientScope{OrganizationIDs: []string{"cust-1"}}},
		},
	}
	lookup := func() (map[string]struct{}, error) {
		t.Fatal("systems must not be loaded")
		return nil, nil
	}
	if _, err := scopeLayerForOrg(layer, set("cust-1"), lookup); err != nil {
		t.Fatal(err)
	}
}

func TestMergeLayers_ScopedCopyStaysSeparate(t *testing.T) {
	merged := MergeLayers([]models.AlertingConfigLayer{
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.it", Severities: []string{"critical"}}}},
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.it", Scope: &models.RecipientScope{SystemIDs: []string{"s2", "s1"}}}}},
		{EmailRecipients: []models.EmailRecipient{{Address: "a@x.it", Severities: []string{"warning"}, Scope: &models.RecipientScope{SystemIDs: []string{"s1", "s2"}}}}},
	})
	if len(merged.EmailRecipients) != 2 {
		t.Fatalf("expected unscoped + one scoped copy, got %+v", merged.EmailRecipients)
	}
	scoped := merged.EmailRecipients[1]
	if len(scoped.Severities) != 0 {
		t.Errorf("identical scopes must union severities (all wins), got %v", scoped.Severities)
	}
	if strings.Join(scoped.Scope.SystemIDs, ",") != "s1,s2" {
		t.Errorf("scope must be normalized, got %v", scoped.Scope.SystemIDs)
	}
}

func TestRenderConfig_SystemScopedRoutes(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrTrue(), Webhook: ptrTrue()},
		EmailRecipients: []models.EmailRecipient{
			{Address: "all@x.it", Severities: []string{"critical"}},
			{Address: "ops@x.it", Scope: &models.RecipientScope{SystemIDs: []string{"sys-a", "sys-b"}}},
			// Already on the unscoped critical receiver: no duplicate mail.
			{Address: "all@x.it", Severities: []string{"critical"}, Scope: &models.RecipientScope{SystemIDs: []string{"sys-a"}}},
		},
		WebhookRecipients: []models.WebhookRecipient{
			{Name: "hook", URL: "https://hook.example/x", Severities: []string{"critical"}, Scope: &models.RecipientScope{SystemIDs: []string{"sys-b"}}},
		},
	}
	out := renderForTest(t, cfg)

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("rendered YAML does not parse: %v\n%s", err, out)
	}

	// critical: sys-a → ops; sys-b → ops + hook. Two groups.
	if !strings.Contains(out, `system_id=~"sys-a"`) || !strings.Contains(out, `system_id=~"sys-b"`) {
		t.Errorf("expected one critical route per destination set, got:\n%s", out)
	}
	// warning/info: both systems only reach ops → grouped in one route.
	if strings.Count(out, `system_id=~"sys-a|sys-b"`) != 2 {
		t.Errorf("expected grouped warning+info routes, got:\n%s", out)
	}
	if strings.Count(out, "to: 'all@x.it'") != 1 {
		t.Errorf("all@x.it must only sit on the unscoped receiver, got:\n%s", out)
	}
	// Scoped routes must precede the severity routes and continue.
	scopedAt := strings.Index(out, "system_id=~")
	severityAt := strings.Index(out, "- severity=\"critical\"\n          receiver: 'cust-test-severity-critical-receiver'")
	if scopedAt < 0 || severityAt < 0 || scopedAt > severityAt {
		t.Errorf("scoped routes must come first, got:\n%s", out)
	}
	if !strings.Contains(out, "receiver: 'cust-test-severity-critical-scoped-1-receiver'\n          continue: true") {
		t.Errorf("scoped routes must continue, got:\n%s", out)
	}
}

func TestRenderConfig_OnlyScopedRecipientsStillRendersCustomer(t *testing.T) {
	cfg := &models.AlertingConfigLayer{
		Enabled: models.ChannelToggles{Email: ptrTrue()},
		EmailRecipients: []models.EmailRecipient{
			{Address: "ops@x.it", Scope: &models.RecipientScope{SystemIDs: []string{"sys-a"}}},
		},
	}
	out := renderForTest(t, cfg)
	if !strings.Contains(out, `organization_id="cust-test"`) || !strings.Contains(out, "to: 'ops@x.it'") {
		t.Errorf("customer with only scoped recipients must be routed, got:\n%s", out)
	}
	if strings.Contains(out, "cust-test-severity-critical-receiver") {
		t.Errorf("no unscoped recipient: severity routes must fall to blackhole, got:\n%s", out)
	}
}

func TestValidate_RecipientScope(t *testing.T) {
	bad := models.AlertingConfigLayer{
		EmailRecipients: []models.EmailRecipient{
			{Address: "a@x.it", Scope: &models.RecipientScope{SystemIDs: []string{`sys"|.*`}}},
		},
	}
	if err := bad.Validate(); err == nil || !strings.Contains(err.Error(), "email_recipients.0.scope") {
		t.Errorf("expected scope validation error, got %v", err)
	}
	ok := models.AlertingConfigLayer{
		TelegramRecipients: []models.TelegramRecipient{
			{BotToken: "t", ChatID: 1, Scope: &models.RecipientScope{OrganizationIDs: []string{"57rrvkk0hhh3"}, SystemIDs: []string{"e2b55e57-ddb1-489d-91be-0da16bb08319"}}},
		},
	}
	if err := ok.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
