/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nethesis/my/sync/internal/config"
)

func TestBuildSignInExperienceConfigMFA(t *testing.T) {
	engine := NewEngine(nil, &Options{ConfigFile: filepath.Join(t.TempDir(), "config.yml")})

	t.Run("no mfa block leaves the tenant policy alone", func(t *testing.T) {
		built, err := engine.buildSignInExperienceConfig(&config.SignInExperience{
			Language: &config.SignInLanguage{AutoDetect: true, FallbackLanguage: "en"},
		}, "configs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if built.MFA != nil || built.TrustedDevice != nil {
			t.Errorf("expected no MFA payload, got mfa=%+v trustedDevice=%+v", built.MFA, built.TrustedDevice)
		}

		payload, err := json.Marshal(built)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		if strings.Contains(string(payload), `"mfa"`) || strings.Contains(string(payload), `"trustedDevice"`) {
			t.Errorf("PATCH body must not touch MFA fields, got %s", payload)
		}
	})

	t.Run("mfa block fills both Logto fields", func(t *testing.T) {
		built, err := engine.buildSignInExperienceConfig(&config.SignInExperience{
			MFA: &config.SignInMFA{
				Policy:  "Mandatory",
				Factors: []string{"Totp", "WebAuthn", "BackupCode"},
				TrustedDevice: &config.SignInTrustedDevice{
					Enabled:      true,
					DurationDays: 30,
				},
			},
		}, "configs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		payload, err := json.Marshal(built)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var body map[string]json.RawMessage
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if got := string(body["mfa"]); got != `{"policy":"Mandatory","factors":["Totp","WebAuthn","BackupCode"]}` {
			t.Errorf("unexpected mfa payload: %s", got)
		}
		if got := string(body["trustedDevice"]); got != `{"enabled":true,"durationDays":30}` {
			t.Errorf("unexpected trustedDevice payload: %s", got)
		}
	})

	t.Run("missing factors are sent as an empty list, not null", func(t *testing.T) {
		built, err := engine.buildSignInExperienceConfig(&config.SignInExperience{
			MFA: &config.SignInMFA{Policy: "NoPrompt"},
		}, "configs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		payload, err := json.Marshal(built.MFA)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		if got := string(payload); got != `{"policy":"NoPrompt","factors":[]}` {
			t.Errorf("unexpected mfa payload: %s", got)
		}
		if built.TrustedDevice != nil {
			t.Errorf("expected no trustedDevice payload without the block, got %+v", built.TrustedDevice)
		}
	})

	t.Run("disabled trusted device omits the duration", func(t *testing.T) {
		built, err := engine.buildSignInExperienceConfig(&config.SignInExperience{
			MFA: &config.SignInMFA{
				Policy:        "Mandatory",
				Factors:       []string{"Totp"},
				TrustedDevice: &config.SignInTrustedDevice{Enabled: false},
			},
		}, "configs")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		payload, err := json.Marshal(built.TrustedDevice)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		if got := string(payload); got != `{"enabled":false}` {
			t.Errorf("unexpected trustedDevice payload: %s", got)
		}
	})
}
