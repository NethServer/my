/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package middleware

import (
	"testing"

	"github.com/nethesis/my/collect/configuration"
	"github.com/stretchr/testify/assert"
)

// systemKeyFormat must match the real key shape produced by the backend's
// generateSystemKey: "NETH-" + eight 4-char uppercase-hex groups (a 32-char
// UUID). A regression here silently disables the whole auth invalidation bus,
// because verifyInvalidationPayload drops anything that fails this regex.
func TestSystemKeyFormat_MatchesRealKeys(t *testing.T) {
	valid := []string{
		"NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0D",
		"NETH-FBB2-1A6E-7CAD-44A4-A772-B3EE-F0F6-F371",
		"NETH-0CE1-5AA2-7B25-4273-83E3-CDFB-BA35-A75A",
	}
	for _, k := range valid {
		assert.Truef(t, systemKeyFormat.MatchString(k), "expected %q to be accepted", k)
	}

	invalid := []string{
		"NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0D-EXTRA", // nine groups (the old, wrong shape)
		"NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C",            // seven groups
		"NETH-d49f-f40a-8650-4786-9ce1-e6de-6e6c-2b0d",       // lowercase
		"NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0G",       // non-hex char
		"*", // Redis SCAN glob must never be accepted
		"NETH-*",
		"",
	}
	for _, k := range invalid {
		assert.Falsef(t, systemKeyFormat.MatchString(k), "expected %q to be rejected", k)
	}
}

// With no shared HMAC secret configured (the dev default), a bare, well-formed
// system_key payload is accepted; globs and malformed keys are rejected.
func TestVerifyInvalidationPayload_BareKey(t *testing.T) {
	configuration.Config.InternalHMACSecret = ""

	key, ok := verifyInvalidationPayload("NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0D")
	assert.True(t, ok)
	assert.Equal(t, "NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0D", key)

	_, ok = verifyInvalidationPayload("*")
	assert.False(t, ok)

	_, ok = verifyInvalidationPayload("not-a-key")
	assert.False(t, ok)
}
