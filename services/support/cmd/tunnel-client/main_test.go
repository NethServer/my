/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/users"
)

func TestCleanupOnlyWithoutState(t *testing.T) {
	dir := t.TempDir()
	if code := runCleanupOnly(filepath.Join(dir, "users-state.json"), dir, "", time.Second); code != 0 {
		t.Errorf("exit code = %d, want 0 when there is no state", code)
	}
}

func TestCleanupOnlyEmptyStateIsRemoved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users-state.json")
	if err := os.WriteFile(path, []byte(`{"platform":"nethsecurity"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := runCleanupOnly(path, dir, "", time.Second); code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("empty state file must be removed")
	}
}

// Outside a NethSecurity system the nethsec delete cannot succeed, which is
// exactly the failure path: non-zero exit and the identity stays recorded.
func TestCleanupOnlyFailureKeepsState(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "users-state.json")
	t.Setenv("PATH", dir)
	state := &users.SessionUsers{
		SessionID:  "NETH-TEST",
		Platform:   "nethsecurity",
		LocalUsers: []users.UserCredential{{Username: "support-neth-test"}},
	}
	if err := users.SaveState(path, state); err != nil {
		t.Fatal(err)
	}
	if code := runCleanupOnly(path, dir, "", time.Second); code == 0 {
		t.Error("exit code = 0, want non-zero when a delete fails")
	}
	left, err := users.LoadState(path)
	if err != nil || left == nil || len(left.LocalUsers) != 1 {
		t.Errorf("failed identity must stay recorded: %+v %v", left, err)
	}
}
