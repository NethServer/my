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

// SaveState writes the provisioned users to a state file for crash recovery.
// If the tunnel-client crashes before cleanup, the next startup reads this
// file and runs Delete + Teardown to remove orphaned users.
func SaveState(stateFile string, users *SessionUsers) error {
	// Ensure parent directory exists
	dir := filepath.Dir(stateFile)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("cannot create state directory: %w", err)
	}

	data, err := json.Marshal(users)
	if err != nil {
		return fmt.Errorf("cannot marshal state: %w", err)
	}

	// Write atomically via temp file + rename
	tmp := stateFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
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
func RemoveState(stateFile string) {
	_ = os.Remove(stateFile)
}

// generateUsername creates a support username from a system key.
// Uses the first two and last segment for readability and uniqueness:
// "NETH-2239-DE49-87D6-44AA-B1CE-E07F-9B03-7D37" → "support-neth-2239-7d37"
func generateUsername(systemKey string) string {
	key := strings.ToLower(systemKey)
	parts := strings.Split(key, "-")
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
