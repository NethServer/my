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
	"fmt"
	"strings"
)

// Envelope for support_sessions.users: {"enc":"v1:<base64std(nonce||ciphertext)>"},
// AES-256-GCM with key = HKDF-SHA256(SUPPORT_INTERNAL_SECRET, salt=nil, info).
// backend/helpers/supportcrypt.go mirrors this file.
const (
	supportUsersInfo    = "my-support-users-v1"
	supportUsersVersion = "v1:"
	supportUsersNonce   = 12
)

type supportUsersEnvelope struct {
	Enc string `json:"enc"`
}

func supportUsersAEAD(secret string) (cipher.AEAD, error) {
	if secret == "" {
		return nil, fmt.Errorf("empty secret")
	}
	key, err := hkdf.Key(sha256.New, []byte(secret), nil, supportUsersInfo, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptSupportUsers seals the users JSON into the JSONB envelope stored in the database.
func EncryptSupportUsers(secret string, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, supportUsersNonce)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return encryptSupportUsersWithNonce(secret, plaintext, nonce)
}

func encryptSupportUsersWithNonce(secret string, plaintext, nonce []byte) ([]byte, error) {
	aead, err := supportUsersAEAD(secret)
	if err != nil {
		return nil, err
	}
	sealed := aead.Seal(append([]byte{}, nonce...), nonce, plaintext, nil)
	return json.Marshal(supportUsersEnvelope{Enc: supportUsersVersion + base64.StdEncoding.EncodeToString(sealed)})
}

// DecryptSupportUsers opens an envelope written by EncryptSupportUsers.
func DecryptSupportUsers(secret string, stored []byte) ([]byte, error) {
	var env supportUsersEnvelope
	if err := json.Unmarshal(stored, &env); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(env.Enc, supportUsersVersion) {
		return nil, fmt.Errorf("unsupported users envelope")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(env.Enc, supportUsersVersion))
	if err != nil {
		return nil, err
	}
	if len(raw) < supportUsersNonce {
		return nil, fmt.Errorf("users envelope too short")
	}
	aead, err := supportUsersAEAD(secret)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, raw[:supportUsersNonce], raw[supportUsersNonce:], nil)
}
