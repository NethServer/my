/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"errors"
	"strings"
	"testing"

	"github.com/nethesis/my/services/support/models"
)

func TestDecideTunnelSession(t *testing.T) {
	token := strings.Repeat("a", 64)
	live := &models.SupportSession{ID: "s1", ReconnectToken: token}
	ended := func(v bool) func() (bool, error) { return func() (bool, error) { return v, nil } }
	mustNotCall := func() (bool, error) {
		t.Fatal("token lookup must not run")
		return false, nil
	}

	tests := []struct {
		name     string
		existing *models.SupportSession
		token    string
		lookup   func() (bool, error)
		want     sessionDecision
	}{
		{"resume with matching token", live, token, mustNotCall, sessionResume},
		{"replace without token", live, "", mustNotCall, sessionReplace},
		{"replace with unknown token", live, strings.Repeat("b", 64), ended(false), sessionReplace},
		{"gone with token of ended session", live, strings.Repeat("b", 64), ended(true), sessionGone},
		{"gone with no live session", nil, token, ended(true), sessionGone},
		{"new without session", nil, "", mustNotCall, sessionNew},
		{"new with unknown token", nil, token, ended(false), sessionNew},
		{"stored token empty never matches", &models.SupportSession{ID: "s2"}, "", mustNotCall, sessionReplace},
	}
	for _, tt := range tests {
		got, err := decideTunnelSession(tt.existing, tt.token, tt.lookup)
		if err != nil || got != tt.want {
			t.Errorf("%s: got %v (%v), want %v", tt.name, got, err, tt.want)
		}
	}

	if _, err := decideTunnelSession(nil, token, func() (bool, error) { return false, errors.New("db") }); err == nil {
		t.Error("lookup errors must be returned")
	}
}

func TestReconnectTokenPattern(t *testing.T) {
	if !reconnectTokenPattern.MatchString(strings.Repeat("0f", 32)) {
		t.Fatal("generated tokens must be accepted")
	}
	for _, bad := range []string{"", "abc", strings.Repeat("G", 64), strings.Repeat("a", 65)} {
		if reconnectTokenPattern.MatchString(bad) {
			t.Errorf("token %q must be rejected", bad)
		}
	}
}
