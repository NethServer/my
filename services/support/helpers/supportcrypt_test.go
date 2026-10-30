/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"bytes"
	"testing"
)

// Cross-check vector shared with backend/helpers/supportcrypt_test.go.
const supportUsersVector = `{"enc":"v1:AAAAAAAAAAAAAAAAKJAPINuxVrUfx05uRJBk7lGl/GbYQpU="}`

func TestSupportUsersFixedVector(t *testing.T) {
	got, err := encryptSupportUsersWithNonce("test-secret", []byte(`{"a":1}`), make([]byte, 12))
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	t.Logf("vector: %s", got)
	if string(got) != supportUsersVector {
		t.Fatalf("vector mismatch:\n got %s\nwant %s", got, supportUsersVector)
	}
	plain, err := DecryptSupportUsers("test-secret", got)
	if err != nil || string(plain) != `{"a":1}` {
		t.Fatalf("decrypt: %q %v", plain, err)
	}
}

func TestSupportUsersRoundTrip(t *testing.T) {
	plain := []byte(`{"users":{"session_id":"x"}}`)
	enc, err := EncryptSupportUsers("s3cret", plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if bytes.Contains(enc, []byte("session_id")) {
		t.Fatal("plaintext leaked into envelope")
	}
	got, err := DecryptSupportUsers("s3cret", enc)
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("round trip: %q %v", got, err)
	}
	if _, err := DecryptSupportUsers("other", enc); err == nil {
		t.Fatal("expected failure with wrong secret")
	}
	if _, err := DecryptSupportUsers("s3cret", plain); err == nil {
		t.Fatal("expected failure on plaintext row")
	}
}
