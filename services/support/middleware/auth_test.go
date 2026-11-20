/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/nethesis/my/services/support/configuration"
)

const testSystemKey = "NETH-0123-4567-89AB-CDEF-0123-4567-89AB-CDEF"

func TestAuthInvalidationChannel(t *testing.T) {
	configuration.Config.AppEnv = "prod"
	defer func() { configuration.Config.AppEnv = "" }()
	if got := authInvalidationChannel(); got != "my:auth:invalidate:prod" {
		t.Fatalf("unexpected channel %q", got)
	}
	configuration.Config.AppEnv = ""
	if got := authInvalidationChannel(); got != "my:auth:invalidate:dev" {
		t.Fatalf("unexpected default channel %q", got)
	}
}

func TestVerifyInvalidationPayload(t *testing.T) {
	configuration.Config.InternalHMACSecret = ""
	if key, ok := verifyInvalidationPayload(testSystemKey); !ok || key != testSystemKey {
		t.Fatal("bare system_key must be accepted without a secret")
	}
	if _, ok := verifyInvalidationPayload("NETH-*"); ok {
		t.Fatal("glob payload must be rejected")
	}

	secret := "0123456789abcdef0123456789abcdef"
	configuration.Config.InternalHMACSecret = secret
	defer func() { configuration.Config.InternalHMACSecret = "" }()
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(testSystemKey))
	signed := testSystemKey + "|" + hex.EncodeToString(mac.Sum(nil))

	if key, ok := verifyInvalidationPayload(signed); !ok || key != testSystemKey {
		t.Fatal("signed payload must be accepted")
	}
	if _, ok := verifyInvalidationPayload(testSystemKey); ok {
		t.Fatal("unsigned payload must be rejected when a secret is set")
	}
	if _, ok := verifyInvalidationPayload(testSystemKey + "|00"); ok {
		t.Fatal("bad signature must be rejected")
	}
}

func TestInProcessCacheIgnoresEntriesBeforeInvalidation(t *testing.T) {
	configuration.Config.SystemAuthCacheTTL = time.Hour
	setInProcessCache(testSystemKey, "secret", "sys-1", true)
	if _, _, found := checkInProcessCache(testSystemKey, "secret"); !found {
		t.Fatal("expected cache hit")
	}
	lastInvalidatedAt.Store(testSystemKey, time.Now())
	if _, _, found := checkInProcessCache(testSystemKey, "secret"); found {
		t.Fatal("entry cached before the invalidation must be ignored")
	}
}

func TestAuthFailureLimiter(t *testing.T) {
	rl := newRateLimiter(2, time.Minute)
	if rl.exceeded("1.2.3.4") {
		t.Fatal("fresh key must not be blocked")
	}
	rl.allow("1.2.3.4")
	rl.allow("1.2.3.4")
	if !rl.exceeded("1.2.3.4") {
		t.Fatal("key must be blocked after reaching the limit")
	}
	if rl.exceeded("5.6.7.8") {
		t.Fatal("other keys must not be affected")
	}
}
