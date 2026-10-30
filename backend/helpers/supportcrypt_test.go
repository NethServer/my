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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportUsersFixedVector(t *testing.T) {
	nonce := make([]byte, 12)
	plaintext := []byte(`{"a":1}`)

	expected := `{"enc":"v1:AAAAAAAAAAAAAAAAKJAPINuxVrUfx05uRJBk7lGl/GbYQpU="}`

	got, err := encryptSupportUsersWithNonce("test-secret", plaintext, nonce)
	require.NoError(t, err)
	assert.Equal(t, expected, string(got))

	dec, err := DecryptSupportUsers("test-secret", []byte(expected))
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec)
}

func TestSupportUsersRoundTrip(t *testing.T) {
	plaintext := []byte(`{"ssh":{"username":"support","password":"s3cret"}}`)
	enc, err := EncryptSupportUsers("another-secret", plaintext)
	require.NoError(t, err)
	assert.NotContains(t, string(enc), "s3cret")

	dec, err := DecryptSupportUsers("another-secret", enc)
	require.NoError(t, err)
	assert.Equal(t, plaintext, dec)

	_, err = DecryptSupportUsers("wrong-secret", enc)
	assert.Error(t, err)

	_, err = DecryptSupportUsers("", enc)
	assert.ErrorIs(t, err, ErrSupportUsersSecretMissing)
}

func TestSupportUsersLegacyPlaintext(t *testing.T) {
	legacy := []byte(`{"ssh":{"username":"support"},"enc_note":"x"}`)
	dec, err := DecryptSupportUsers("test-secret", legacy)
	require.NoError(t, err)
	assert.Equal(t, legacy, dec)
}
