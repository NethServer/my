/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"strings"
	"testing"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
)

// Access control scopes are declared under a role's access_control block, not
// its permissions, so syncUserRolePermissions always sees them as unexpected.
// They belong to syncUserRoleAccessControlScopes: if the permissions step also
// removed them, the two steps would fight and every sync would strip and
// re-add the scope.
func TestApplyUserRolePermissionChangesSkipsAccessControlScopes(t *testing.T) {
	logger.SetLevel("fatal")

	mapping := &ScopeMapping{
		NameToID: map[string]string{
			"read:systems":                 "scope-read-systems",
			"manage:systems":               "scope-manage-systems",
			"owner:role-access-control":    "scope-ac-owner",
			"reseller:role-access-control": "scope-ac-reseller",
		},
		IDToName: map[string]string{
			"scope-read-systems":   "read:systems",
			"scope-manage-systems": "manage:systems",
			"scope-ac-owner":       "owner:role-access-control",
			"scope-ac-reseller":    "reseller:role-access-control",
		},
	}

	newEngine := func() (*Engine, *Result) {
		engine := NewEngine(&client.LogtoClient{}, &Options{DryRun: true})
		return engine, &Result{Operations: []Operation{}, Summary: &Summary{}}
	}

	t.Run("access control scope alone is never removed", func(t *testing.T) {
		engine, result := newEngine()
		diff := &PermissionDiff{ToRemove: []string{"owner:role-access-control"}}

		if err := engine.applyUserRolePermissionChanges("role-1", "Super Admin", diff, mapping, result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Summary.PermissionsDeleted != 0 {
			t.Errorf("expected 0 permissions deleted, got %d", result.Summary.PermissionsDeleted)
		}
		if len(result.Operations) != 0 {
			t.Errorf("expected no operations recorded, got %d: %+v", len(result.Operations), result.Operations)
		}
	})

	t.Run("genuine permissions are still removed alongside", func(t *testing.T) {
		engine, result := newEngine()
		diff := &PermissionDiff{ToRemove: []string{
			"owner:role-access-control",
			"read:systems",
			"reseller:role-access-control",
			"manage:systems",
		}}

		if err := engine.applyUserRolePermissionChanges("role-1", "Super Admin", diff, mapping, result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Only the two real permissions must be counted for removal.
		if result.Summary.PermissionsDeleted != 2 {
			t.Errorf("expected 2 permissions deleted, got %d", result.Summary.PermissionsDeleted)
		}
		for _, op := range result.Operations {
			if strings.Contains(op.Resource, accessControlScopeSuffix) {
				t.Errorf("access control scope leaked into a removal operation: %+v", op)
			}
		}
	})

	t.Run("additions are unaffected", func(t *testing.T) {
		engine, result := newEngine()
		diff := &PermissionDiff{ToAdd: []string{"read:systems", "manage:systems"}}

		if err := engine.applyUserRolePermissionChanges("role-1", "Admin", diff, mapping, result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Summary.PermissionsCreated != 2 {
			t.Errorf("expected 2 permissions created, got %d", result.Summary.PermissionsCreated)
		}
	})
}

// The suffix must stay in sync with what syncUserRoleAccessControlScopes builds
// and with the backend, which parses it to derive the required org role
// (backend/cache/roles.go).
func TestAccessControlScopeSuffix(t *testing.T) {
	if accessControlScopeSuffix != ":role-access-control" {
		t.Errorf("unexpected suffix %q: the backend parses this exact string", accessControlScopeSuffix)
	}
	if got := "owner" + accessControlScopeSuffix; got != "owner:role-access-control" {
		t.Errorf("expected %q, got %q", "owner:role-access-control", got)
	}
}
