/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package config

import "testing"

func TestValidateTransport(t *testing.T) {
	cases := []struct {
		name     string
		url      string
		insecure bool
		dev      bool
		wantErr  bool
	}{
		{"wss ok", "wss://support.example.com/api/tunnel", false, false, false},
		{"ws refused", "ws://support.example.com/api/tunnel", false, false, true},
		{"ws allowed in dev", "ws://localhost:8082/api/tunnel", false, true, false},
		{"insecure refused", "wss://support.example.com/api/tunnel", true, false, true},
		{"insecure allowed in dev", "wss://localhost/api/tunnel", true, true, false},
		{"https refused", "https://support.example.com/api/tunnel", false, false, true},
		{"http refused in dev", "http://localhost/api/tunnel", false, true, true},
		{"missing host", "wss:///api/tunnel", false, false, true},
		{"garbage", "::not a url", false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTransport(tc.url, tc.insecure, tc.dev)
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateTransport(%q, %v, %v) err=%v, wantErr=%v", tc.url, tc.insecure, tc.dev, err, tc.wantErr)
			}
		})
	}
}

func TestIsTruthy(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", "yes", "on"} {
		if !IsTruthy(v) {
			t.Errorf("IsTruthy(%q) = false", v)
		}
	}
	for _, v := range []string{"", "0", "false", "no", "maybe"} {
		if IsTruthy(v) {
			t.Errorf("IsTruthy(%q) = true", v)
		}
	}
}
