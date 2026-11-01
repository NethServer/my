/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package gateway

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"path/filepath"
	"testing"

	gossh "golang.org/x/crypto/ssh"
)

func TestParseTarget(t *testing.T) {
	tok := "0123456789abcdef0123456789abcdef"
	tests := []struct {
		user    string
		want    Target
		wantErr bool
	}{
		{"NETH-AAAA-BBBB", Target{SystemKey: "NETH-AAAA-BBBB"}, false},
		{"NETH-AAAA-BBBB.2", Target{SystemKey: "NETH-AAAA-BBBB", NodeID: "2"}, false},
		{"NETH-AAAA-BBBB.2+" + tok, Target{SystemKey: "NETH-AAAA-BBBB", NodeID: "2", Token: tok}, false},
		{"NETH-AAAA-BBBB+" + tok, Target{SystemKey: "NETH-AAAA-BBBB", Token: tok}, false},
		{"NETH-AAAA-BBBB.2+short", Target{}, true},
		{"NETH-AAAA-BBBB.x", Target{}, true},
		{"NETH AAAA", Target{}, true},
		{"", Target{}, true},
	}
	for _, tt := range tests {
		got, err := ParseTarget(tt.user)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("ParseTarget(%q) = %+v, %v", tt.user, got, err)
		}
	}
	if (Target{SystemKey: "K", NodeID: "2", Token: tok}).Name() != "K.2" {
		t.Error("the name must leave the token out")
	}
}

func TestHostKeyStable(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(block))

	fromEnv, err := LoadOrGenerateHostKey(pemKey, filepath.Join(t.TempDir(), "unused"))
	if err != nil {
		t.Fatalf("PEM from the environment: %v", err)
	}
	signer, _ := gossh.NewSignerFromKey(priv)
	if gossh.FingerprintSHA256(fromEnv.PublicKey()) != gossh.FingerprintSHA256(signer.PublicKey()) {
		t.Fatal("the key from the environment must be used as is")
	}

	path := filepath.Join(t.TempDir(), "host_key")
	first, err := LoadOrGenerateHostKey("", path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrGenerateHostKey("", path)
	if err != nil {
		t.Fatal(err)
	}
	if gossh.FingerprintSHA256(first.PublicKey()) != gossh.FingerprintSHA256(second.PublicKey()) {
		t.Fatal("a generated key must be reused at the next start")
	}
}
