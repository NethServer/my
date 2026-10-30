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
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	supportUsersKeyInfo   = "my-support-users-v1"
	supportUsersEncPrefix = "v1:"
)

// ErrSupportUsersSecretMissing is returned when an encrypted users column is
// read without SUPPORT_INTERNAL_SECRET configured.
var ErrSupportUsersSecretMissing = errors.New("support users are encrypted but no secret is configured")

type supportUsersEnvelope struct {
	Enc string `json:"enc"`
}

func supportUsersAEAD(secret string) (cipher.AEAD, error) {
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, supportUsersKeyInfo, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to derive support users key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptSupportUsers wraps the plaintext users JSON in the
// {"enc":"v1:<base64(nonce||ciphertext)>"} envelope written by the support service.
func EncryptSupportUsers(secret string, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return encryptSupportUsersWithNonce(secret, plaintext, nonce)
}

func encryptSupportUsersWithNonce(secret string, plaintext, nonce []byte) ([]byte, error) {
	if secret == "" {
		return nil, ErrSupportUsersSecretMissing
	}
	aead, err := supportUsersAEAD(secret)
	if err != nil {
		return nil, err
	}
	sealed := aead.Seal(append([]byte{}, nonce...), nonce, plaintext, nil)
	return json.Marshal(supportUsersEnvelope{Enc: supportUsersEncPrefix + base64.StdEncoding.EncodeToString(sealed)})
}

// DecryptSupportUsers returns the plaintext users JSON. Rows written before
// encryption (anything but the envelope) are returned unchanged.
func DecryptSupportUsers(secret string, raw []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || len(fields) != 1 || fields["enc"] == nil {
		return raw, nil
	}
	var env supportUsersEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || !strings.HasPrefix(env.Enc, supportUsersEncPrefix) {
		return nil, fmt.Errorf("unsupported support users envelope")
	}
	if secret == "" {
		return nil, ErrSupportUsersSecretMissing
	}
	sealed, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(env.Enc, supportUsersEncPrefix))
	if err != nil {
		return nil, fmt.Errorf("invalid support users ciphertext encoding: %w", err)
	}
	aead, err := supportUsersAEAD(secret)
	if err != nil {
		return nil, err
	}
	if len(sealed) < aead.NonceSize()+aead.Overhead() {
		return nil, fmt.Errorf("support users ciphertext too short")
	}
	plaintext, err := aead.Open(nil, sealed[:aead.NonceSize()], sealed[aead.NonceSize():], nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt support users: %w", err)
	}
	return plaintext, nil
}
