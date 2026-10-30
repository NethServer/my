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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeProvisioner creates the given identities and fails to delete the
// usernames listed in failDelete.
type fakeProvisioner struct {
	toCreate   *SessionUsers
	failDelete map[string]bool
	deleted    []string
}

func (f *fakeProvisioner) Create(sessionID string, record RecordFunc) (*SessionUsers, error) {
	result := &SessionUsers{SessionID: sessionID, Platform: f.toCreate.Platform}
	if f.toCreate.ClusterAdmin != nil {
		result.ClusterAdmin = f.toCreate.ClusterAdmin
		if err := record(result); err != nil {
			result.ClusterAdmin = nil
			return result, err
		}
	}
	for _, du := range f.toCreate.DomainUsers {
		result.DomainUsers = append(result.DomainUsers, du)
		if err := record(result); err != nil {
			result.DomainUsers = result.DomainUsers[:len(result.DomainUsers)-1]
			return result, err
		}
	}
	return result, nil
}

func (f *fakeProvisioner) Delete(users *SessionUsers) error {
	var errs []error
	var remaining []DomainUser
	for _, du := range users.DomainUsers {
		if f.failDelete[du.Module] {
			errs = append(errs, errors.New("delete failed"))
			remaining = append(remaining, du)
			continue
		}
		f.deleted = append(f.deleted, du.Module+"/"+du.Username)
	}
	users.DomainUsers = remaining
	if users.ClusterAdmin != nil {
		if f.failDelete["cluster"] {
			errs = append(errs, errors.New("delete failed"))
		} else {
			f.deleted = append(f.deleted, "cluster/"+users.ClusterAdmin.Username)
			users.ClusterAdmin = nil
		}
	}
	return errors.Join(errs...)
}

func sampleUsers() *SessionUsers {
	return &SessionUsers{
		SessionID:    "NETH-AAAA-BBBB",
		Platform:     "ns8",
		ClusterAdmin: &UserCredential{Username: "support-neth-aaaa-bbbb", Password: "Secret1!Secret1!Secret"},
		DomainUsers: []DomainUser{
			{Domain: "ad.example.com", Module: "samba1", Username: "support-neth-aaaa-bbbb", Password: "Secret2!Secret2!Secret"},
			{Domain: "ldap.example.com", Module: "openldap1", Username: "support-neth-aaaa-bbbb", Password: "Secret3!Secret3!Secret"},
		},
	}
}

func TestSaveStatePermissionsAndNoSecrets(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my-support")
	path := filepath.Join(dir, "users-state.json")

	if err := SaveState(path, sampleUsers()); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("state dir mode = %04o, want 0700", dirInfo.Mode().Perm())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("state file mode = %04o, want 0600", info.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "Secret") {
		t.Errorf("state file contains a password: %s", data)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("leftover temp files in state dir: %v", entries)
	}

	loaded, err := LoadState(path)
	if err != nil || loaded == nil {
		t.Fatalf("LoadState: %v %v", loaded, err)
	}
	if loaded.ClusterAdmin.Username != "support-neth-aaaa-bbbb" || len(loaded.DomainUsers) != 2 || loaded.DomainUsers[1].Module != "openldap1" {
		t.Errorf("unexpected loaded state: %+v", loaded)
	}
}

func TestStateRecordedAfterEachCreate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users-state.json")
	want := sampleUsers()
	var snapshots []int
	record := func(created *SessionUsers) error {
		if err := PersistState(path, created); err != nil {
			return err
		}
		loaded, _ := LoadState(path)
		n := len(loaded.DomainUsers)
		if loaded.ClusterAdmin != nil {
			n++
		}
		snapshots = append(snapshots, n)
		return nil
	}
	if _, err := (&fakeProvisioner{toCreate: want}).Create("NETH-AAAA-BBBB", record); err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 3 || snapshots[0] != 1 || snapshots[2] != 3 {
		t.Errorf("state not recorded after each creation: %v", snapshots)
	}
}

func TestCleanupStateNoState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users-state.json")
	called := false
	err := CleanupState(path, func(string) Provisioner { called = true; return &fakeProvisioner{} }, nil)
	if err != nil || called {
		t.Fatalf("CleanupState without state: err=%v provisionerCalled=%v", err, called)
	}
}

func TestCleanupStateAllDeleted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users-state.json")
	if err := SaveState(path, sampleUsers()); err != nil {
		t.Fatal(err)
	}
	fake := &fakeProvisioner{}
	var platform string
	tornDown := false
	err := CleanupState(path, func(p string) Provisioner { platform = p; return fake }, func(*SessionUsers) { tornDown = true })
	if err != nil {
		t.Fatalf("CleanupState: %v", err)
	}
	if platform != "ns8" || !tornDown || len(fake.deleted) != 3 {
		t.Errorf("platform=%q tornDown=%v deleted=%v", platform, tornDown, fake.deleted)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("state file still present after full cleanup: %v", err)
	}
}

func TestCleanupStateKeepsFailedDeletes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users-state.json")
	if err := SaveState(path, sampleUsers()); err != nil {
		t.Fatal(err)
	}
	fake := &fakeProvisioner{failDelete: map[string]bool{"samba1": true}}
	if err := CleanupState(path, func(string) Provisioner { return fake }, nil); err == nil {
		t.Fatal("expected error when a delete fails")
	}
	remaining, err := LoadState(path)
	if err != nil || remaining == nil {
		t.Fatalf("state file must be kept: %v %v", remaining, err)
	}
	if remaining.ClusterAdmin != nil || len(remaining.DomainUsers) != 1 || remaining.DomainUsers[0].Module != "samba1" {
		t.Errorf("unexpected remaining state: %+v", remaining)
	}

	// Next start retries only what is left
	fake.failDelete = nil
	if err := CleanupState(path, func(string) Provisioner { return fake }, nil); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("state file still present after retry: %v", err)
	}
}

func TestMergeStateDedups(t *testing.T) {
	leftover := &SessionUsers{
		Platform:    "ns8",
		DomainUsers: []DomainUser{{Module: "samba1", Username: "support-x"}},
	}
	current := &SessionUsers{
		SessionID:    "NETH-X",
		Platform:     "ns8",
		ClusterAdmin: &UserCredential{Username: "support-x", Password: "pw"},
		DomainUsers:  []DomainUser{{Module: "samba1", Username: "support-x", Password: "pw"}, {Module: "openldap1", Username: "support-x", Password: "pw"}},
	}
	merged := MergeState(leftover, current)
	if merged.SessionID != "NETH-X" || merged.ClusterAdmin == nil || len(merged.DomainUsers) != 2 {
		t.Fatalf("unexpected merge: %+v", merged)
	}
	if merged.ClusterAdmin.Password != "" || merged.DomainUsers[1].Password != "" {
		t.Error("merge must strip passwords")
	}
	if HasIdentities(MergeState(nil, nil)) {
		t.Error("empty merge must have no identities")
	}
}

func TestPersistStateRemovesWhenEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "users-state.json")
	if err := SaveState(path, sampleUsers()); err != nil {
		t.Fatal(err)
	}
	if err := PersistState(path, &SessionUsers{Platform: "ns8"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("state file must be removed when no identity is pending")
	}
}
