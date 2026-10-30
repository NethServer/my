/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package users

import "testing"

func TestIsSupportUsername(t *testing.T) {
	valid := []string{"support-neth-2239-7d37", "support-abc", "support-a-b-c"}
	invalid := []string{"", "support-", "admin", "root", "support-ABC", "support-a b", "support-a;rm", "xsupport-a", "support-a\n", "support-a'--"}
	for _, u := range valid {
		if !IsSupportUsername(u) {
			t.Errorf("IsSupportUsername(%q) = false", u)
		}
	}
	for _, u := range invalid {
		if IsSupportUsername(u) {
			t.Errorf("IsSupportUsername(%q) = true", u)
		}
	}
}

func TestGenerateUsernameIsAlwaysValid(t *testing.T) {
	keys := []string{
		"NETH-2239-DE49-87D6-44AA-B1CE-E07F-9B03-7D37",
		"NETH-ABCD",
		"weird key'; DROP TABLE--",
		"",
		"---",
		"ÄÖÜ-ß",
	}
	for _, k := range keys {
		if u := generateUsername(k); !IsSupportUsername(u) {
			t.Errorf("generateUsername(%q) = %q, not a valid support username", k, u)
		}
	}
	if u := generateUsername("NETH-2239-DE49-87D6-44AA-B1CE-E07F-9B03-7D37"); u != "support-neth-2239-7d37" {
		t.Errorf("unexpected username %q", u)
	}
}

func TestGeneratedPasswordPassesValidation(t *testing.T) {
	for i := 0; i < 100; i++ {
		if p := generatePassword(); !passwordRegex.MatchString(p) {
			t.Fatalf("generated password %q fails validation", p)
		}
	}
}

func TestSanitizeFetched(t *testing.T) {
	good := "Abcdef1!Abcdef1!Abcd"
	fetched := &SessionUsers{
		Platform:     "nethsecurity",
		ClusterAdmin: &UserCredential{Username: "admin", Password: good},
		DomainUsers: []DomainUser{
			{Domain: "ad.example.com", Module: "samba1", Username: "support-neth-1", Password: good},
			{Domain: "ad.example.com", Module: "samba1", Username: "support-neth-1", Password: "x'; rm -rf / #"},
			{Domain: "ad.example.com", Module: "../etc", Username: "support-neth-1", Password: good},
			{Domain: "a b", Module: "openldap1", Username: "support-neth-1", Password: good},
			{Domain: "ldap.example.com", Module: "openldap1", Username: "root", Password: good},
		},
		LocalUsers: []UserCredential{{Username: "support-neth-1", Password: good}},
		Apps:       []AppConfig{{ID: "evil"}},
	}
	got := SanitizeFetched(fetched, "NETH-1")
	if got == nil {
		t.Fatal("expected the valid domain user to be kept")
	}
	if got.ClusterAdmin != nil {
		t.Error("cluster admin with a non-support username must be dropped")
	}
	if len(got.DomainUsers) != 1 || got.DomainUsers[0].Module != "samba1" {
		t.Errorf("unexpected domain users: %+v", got.DomainUsers)
	}
	if len(got.LocalUsers) != 0 || len(got.Apps) != 0 {
		t.Error("local users and apps from the server must be dropped")
	}
	if got.Platform != "ns8" || got.SessionID != "NETH-1" {
		t.Errorf("platform/session must come from the client: %+v", got)
	}

	if SanitizeFetched(&SessionUsers{ClusterAdmin: &UserCredential{Username: "root", Password: good}}, "x") != nil {
		t.Error("nothing valid must yield nil")
	}
}
