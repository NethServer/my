/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package csvimport

import (
	"testing"
)

func TestValidateUserRow_Valid(t *testing.T) {
	row := map[string]string{
		"email":        "user@example.com",
		"name":         "Test User",
		"phone":        "+39 333 1234567",
		"organization": "Acme Corp",
		"roles":        "Admin",
	}
	errs := ValidateUserRow(row)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateUserRow_MissingRequired(t *testing.T) {
	row := map[string]string{
		"email":        "",
		"name":         "",
		"phone":        "",
		"organization": "",
		"roles":        "",
	}
	errs := ValidateUserRow(row)

	fields := make(map[string]bool)
	for _, e := range errs {
		fields[e.Field] = true
	}

	requiredFields := []string{"email", "name", "organization", "roles"}
	for _, f := range requiredFields {
		if !fields[f] {
			t.Errorf("expected error for missing %s", f)
		}
	}
}

func TestValidateUserRow_InvalidEmail(t *testing.T) {
	row := map[string]string{
		"email":        "not-an-email",
		"name":         "Test",
		"phone":        "",
		"organization": "Org",
		"roles":        "Admin",
	}
	errs := ValidateUserRow(row)
	found := false
	for _, e := range errs {
		if e.Field == "email" && e.Message == "invalid_format" {
			found = true
		}
	}
	if !found {
		t.Error("expected invalid_format error for email field")
	}
}

func TestValidateUserRow_InvalidPhone(t *testing.T) {
	row := map[string]string{
		"email":        "user@test.com",
		"name":         "Test",
		"phone":        "abc",
		"organization": "Org",
		"roles":        "Admin",
	}
	errs := ValidateUserRow(row)
	found := false
	for _, e := range errs {
		if e.Field == "phone" {
			found = true
		}
	}
	if !found {
		t.Error("expected error for invalid phone")
	}
}

func TestUserRowToData(t *testing.T) {
	row := map[string]string{
		"email":        "test@test.com",
		"name":         "Test User",
		"phone":        "+39 123",
		"organization": "Acme Corp",
		"roles":        "Admin;Support",
	}
	data := UserRowToData(row, "org-123", []string{"role-1", "role-2"})

	if data["email"] != "test@test.com" {
		t.Fatalf("expected email, got %v", data["email"])
	}
	if data["organization_id"] != "org-123" {
		t.Fatalf("expected organization_id org-123, got %v", data["organization_id"])
	}
	roleIDs, ok := data["role_ids"].([]string)
	if !ok || len(roleIDs) != 2 {
		t.Fatalf("expected 2 role IDs, got %v", data["role_ids"])
	}
}

func TestUserDataToCreateRequest(t *testing.T) {
	data := map[string]interface{}{
		"email":           "test@test.com",
		"name":            "Test User",
		"phone":           "+39 333 1234567",
		"organization":    "Acme Corp",
		"roles":           "Admin",
		"organization_id": "org-123",
		"role_ids":        []string{"role-1"},
	}

	req := UserDataToCreateRequest(data)
	if req.Email != "test@test.com" {
		t.Fatalf("expected email, got %q", req.Email)
	}
	if req.Name != "Test User" {
		t.Fatalf("expected name, got %q", req.Name)
	}
	if req.Phone == nil || *req.Phone != "+39 333 1234567" {
		t.Fatalf("expected phone, got %v", req.Phone)
	}
	if *req.OrganizationID != "org-123" {
		t.Fatalf("expected org ID, got %v", req.OrganizationID)
	}
	if len(req.UserRoleIDs) != 1 || req.UserRoleIDs[0] != "role-1" {
		t.Fatalf("expected role IDs, got %v", req.UserRoleIDs)
	}
}

func TestUserDataToCreateRequest_NoPhone(t *testing.T) {
	data := map[string]interface{}{
		"email":           "test@test.com",
		"name":            "Test",
		"phone":           "",
		"organization":    "Org",
		"roles":           "Admin",
		"organization_id": "org-1",
		"role_ids":        []string{"r1"},
	}

	req := UserDataToCreateRequest(data)
	if req.Phone != nil {
		t.Fatalf("expected nil phone, got %v", req.Phone)
	}
}
