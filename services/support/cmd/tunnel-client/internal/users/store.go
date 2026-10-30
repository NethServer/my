/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package users

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

const passwordLength = 20
const passwordLower = "abcdefghijklmnopqrstuvwxyz"
const passwordUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const passwordDigits = "0123456789"
const passwordSpecial = "!@#%^&*"
const passwordAll = passwordLower + passwordUpper + passwordDigits + passwordSpecial

// SaveState atomically writes the identities pending deletion to stateFile.
// Passwords are never persisted: deletion and plugin teardown only need
// usernames.
func SaveState(stateFile string, users *SessionUsers) error {
	dir := filepath.Dir(stateFile)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cannot create state directory: %w", err)
	}

	data, err := json.Marshal(StripSecrets(users))
	if err != nil {
		return fmt.Errorf("cannot marshal state: %w", err)
	}

	f, err := os.CreateTemp(dir, filepath.Base(stateFile)+".*.tmp")
	if err != nil {
		return fmt.Errorf("cannot create state file: %w", err)
	}
	tmp := f.Name()
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("cannot write state file: %w", err)
	}
	if err := os.Rename(tmp, stateFile); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("cannot rename state file: %w", err)
	}
	return nil
}

// LoadState reads the state file. Returns nil if the file does not exist.
func LoadState(stateFile string) (*SessionUsers, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var users SessionUsers
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, fmt.Errorf("cannot parse state file: %w", err)
	}
	return &users, nil
}

// RemoveState deletes the state file.
func RemoveState(stateFile string) error {
	if err := os.Remove(stateFile); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// PersistState writes the pending identities, or removes the state file when
// none are left.
func PersistState(stateFile string, users *SessionUsers) error {
	if !HasIdentities(users) {
		return RemoveState(stateFile)
	}
	return SaveState(stateFile, users)
}

// HasIdentities reports whether users holds any created identity.
func HasIdentities(users *SessionUsers) bool {
	return users != nil && (users.ClusterAdmin != nil || len(users.DomainUsers) > 0 || len(users.LocalUsers) > 0)
}

// StripSecrets returns a copy of users holding only the identity fields.
func StripSecrets(users *SessionUsers) *SessionUsers {
	if users == nil {
		return nil
	}
	out := &SessionUsers{
		SessionID: users.SessionID,
		Platform:  users.Platform,
		CreatedAt: users.CreatedAt,
	}
	if users.ClusterAdmin != nil {
		out.ClusterAdmin = &UserCredential{Username: users.ClusterAdmin.Username}
	}
	for _, du := range users.DomainUsers {
		out.DomainUsers = append(out.DomainUsers, DomainUser{Domain: du.Domain, Module: du.Module, Username: du.Username})
	}
	for _, lu := range users.LocalUsers {
		out.LocalUsers = append(out.LocalUsers, UserCredential{Username: lu.Username})
	}
	return out
}

// MergeState returns the union of the identities in a and b, without
// secrets. Session metadata comes from b when set.
func MergeState(a, b *SessionUsers) *SessionUsers {
	if a == nil {
		return StripSecrets(b)
	}
	out := StripSecrets(a)
	if b == nil {
		return out
	}
	b = StripSecrets(b)
	if b.SessionID != "" {
		out.SessionID = b.SessionID
		out.Platform = b.Platform
		out.CreatedAt = b.CreatedAt
	}
	if out.ClusterAdmin == nil {
		out.ClusterAdmin = b.ClusterAdmin
	}
	for _, du := range b.DomainUsers {
		dup := false
		for _, existing := range out.DomainUsers {
			if existing.Module == du.Module && existing.Username == du.Username {
				dup = true
				break
			}
		}
		if !dup {
			out.DomainUsers = append(out.DomainUsers, du)
		}
	}
	for _, lu := range b.LocalUsers {
		dup := false
		for _, existing := range out.LocalUsers {
			if existing.Username == lu.Username {
				dup = true
				break
			}
		}
		if !dup {
			out.LocalUsers = append(out.LocalUsers, lu)
		}
	}
	return out
}

// CleanupState deletes the identities recorded in stateFile. Plugin teardown
// runs first (when teardown is not nil). Identities whose deletion fails stay
// recorded for the next attempt; the file is removed once all are gone.
// Returns nil when there is no state.
func CleanupState(stateFile string, newProvisioner func(platform string) Provisioner, teardown func(*SessionUsers)) error {
	state, err := LoadState(stateFile)
	if err != nil {
		return fmt.Errorf("cannot read users state file: %w", err)
	}
	if state == nil {
		return nil
	}
	if !HasIdentities(state) {
		return RemoveState(stateFile)
	}
	if teardown != nil {
		teardown(state)
	}
	deleteErr := newProvisioner(state.Platform).Delete(state)
	if err := PersistState(stateFile, state); err != nil {
		return errors.Join(deleteErr, fmt.Errorf("cannot update users state file: %w", err))
	}
	return deleteErr
}

// generateUsername creates a support username from a system key.
// Uses the first two and last segment for readability and uniqueness:
// "NETH-2239-DE49-87D6-44AA-B1CE-E07F-9B03-7D37" → "support-neth-2239-7d37"
func generateUsername(systemKey string) string {
	key := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, strings.ToLower(systemKey))
	var parts []string
	for _, p := range strings.Split(key, "-") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	var suffix string
	if len(parts) >= 3 {
		suffix = parts[0] + "-" + parts[1] + "-" + parts[len(parts)-1]
	} else if len(parts) == 2 {
		suffix = parts[0] + "-" + parts[1]
	} else if len(parts) == 1 && len(parts[0]) >= 4 {
		suffix = parts[0]
	} else {
		b := make([]byte, 8)
		for i := range b {
			n, _ := rand.Int(rand.Reader, big.NewInt(26))
			b[i] = byte('a') + byte(n.Int64())
		}
		suffix = string(b)
	}
	return "support-" + suffix
}

// randChar picks a random character from the given charset.
func randChar(charset string) byte {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
	return charset[n.Int64()]
}

// generatePassword creates a cryptographically random password that satisfies
// common LDAP password quality policies (uppercase, lowercase, digit, special).
func generatePassword() string {
	b := make([]byte, passwordLength)
	// Guarantee at least one character from each class
	b[0] = randChar(passwordLower)
	b[1] = randChar(passwordUpper)
	b[2] = randChar(passwordDigits)
	b[3] = randChar(passwordSpecial)
	// Fill the rest from the full charset
	for i := 4; i < passwordLength; i++ {
		b[i] = randChar(passwordAll)
	}
	// Shuffle to avoid predictable positions
	for i := len(b) - 1; i > 0; i-- {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		b[i], b[j.Int64()] = b[j.Int64()], b[i]
	}
	return string(b)
}

// NewProvisioner returns the appropriate provisioner based on platform detection.
// If redisAddr is set, NS8 is assumed; otherwise NethSecurity.
func NewProvisioner(redisAddr string) Provisioner {
	if redisAddr != "" {
		return &NethServerProvisioner{RedisAddr: redisAddr}
	}
	return &NethSecurityProvisioner{}
}

// ProvisionerForPlatform returns the provisioner that created a recorded state.
func ProvisionerForPlatform(platform, redisAddr string) Provisioner {
	if platform == "ns8" {
		return &NethServerProvisioner{RedisAddr: redisAddr}
	}
	return &NethSecurityProvisioner{}
}
