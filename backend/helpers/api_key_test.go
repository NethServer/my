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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateAPIKey_RoundTrip(t *testing.T) {
	full, public, secret, err := GenerateAPIKey()
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(full, APIKeyPrefix))
	assert.Equal(t, APIKeyPrefix+public+"."+secret, full)

	gotPublic, gotSecret, owner, err := ParseAPIKey(full)
	assert.NoError(t, err)
	assert.False(t, owner)
	assert.Equal(t, public, gotPublic)
	assert.Equal(t, secret, gotSecret)

	// The secret must survive the salted-hash verification round trip.
	hash, err := HashSystemSecretSHA256(secret)
	assert.NoError(t, err)
	ok, err := VerifySystemSecretSHA256(gotSecret, hash)
	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestGenerateOwnerAPIKey_RoundTrip(t *testing.T) {
	full, public, secret, err := GenerateOwnerAPIKey()
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(full, APIKeyOwnerPrefix))
	assert.Equal(t, APIKeyOwnerPrefix+public+"."+secret, full)

	gotPublic, gotSecret, owner, err := ParseAPIKey(full)
	assert.NoError(t, err)
	assert.True(t, owner)
	assert.Equal(t, public, gotPublic)
	assert.Equal(t, secret, gotSecret)
}

func TestGenerateAPIKey_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		_, public, secret, err := GenerateAPIKey()
		assert.NoError(t, err)
		assert.False(t, seen[public], "public part collision")
		assert.False(t, seen[secret], "secret part collision")
		seen[public] = true
		seen[secret] = true
	}
}

func TestParseAPIKey_Invalid(t *testing.T) {
	cases := []string{
		"",                   // empty
		"abc.def",            // missing prefix
		"myk_",               // missing body
		"myk_onlypublic",     // missing separator
		"myk_.secret",        // empty public
		"myk_public.",        // empty secret
		"myo_",               // owner prefix, missing body
		"myo_onlypublic",     // owner prefix, missing separator
		"myo_.secret",        // owner prefix, empty public
		"myo_public.",        // owner prefix, empty secret
		"Bearer myk_pub.sec", // prefix not at start
		"my_public.secret",   // systems prefix, not api key
	}
	for _, tc := range cases {
		_, _, _, err := ParseAPIKey(tc)
		assert.Error(t, err, "expected error for %q", tc)
	}
}
