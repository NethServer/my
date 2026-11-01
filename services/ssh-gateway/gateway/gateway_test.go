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
	tests := []struct {
		user, key, node string
		wantErr         bool
	}{
		{"NETH-AAAA-BBBB", "NETH-AAAA-BBBB", "", false},
		{"NETH-AAAA-BBBB.2", "NETH-AAAA-BBBB", "2", false},
		{"NETH-AAAA-BBBB.x", "", "", true},
		{"NETH AAAA", "", "", true},
		{"", "", "", true},
	}
	for _, tt := range tests {
		key, node, err := ParseTarget(tt.user)
		if (err != nil) != tt.wantErr || key != tt.key || node != tt.node {
			t.Errorf("ParseTarget(%q) = %q, %q, %v", tt.user, key, node, err)
		}
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
