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

func TestValidateOrganizationRow_Valid(t *testing.T) {
	row := map[string]string{
		"company_name": "Acme Corp",
		"description":  "Main distributor",
		"vat_number":   "IT12345678901",
		"address":      "Via Roma 1",
		"city":         "Milano",
		"main_contact": "Mario Rossi",
		"email":        "info@acme.it",
		"phone":        "+39 02 1234567",
		"language":     "it",
		"notes":        "VIP client",
	}
	errs := ValidateOrganizationRow(row)
	if len(errs) != 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
}

func TestValidateOrganizationRow_MissingRequired(t *testing.T) {
	row := map[string]string{
		"company_name": "",
		"description":  "",
		"vat_number":   "",
		"address":      "",
		"city":         "",
		"main_contact": "",
		"email":        "",
		"phone":        "",
		"language":     "",
		"notes":        "",
	}
	errs := ValidateOrganizationRow(row)
	if len(errs) < 2 {
		t.Fatalf("expected at least 2 errors (company_name, vat_number), got: %d", len(errs))
	}

	fields := make(map[string]bool)
	for _, e := range errs {
		fields[e.Field] = true
	}
	if !fields["company_name"] {
		t.Error("expected error for missing company_name")
	}
	if !fields["vat_number"] {
		t.Error("expected error for missing vat_number")
	}
}

func TestValidateOrganizationRow_InvalidOptionalFields(t *testing.T) {
	row := map[string]string{
		"company_name": "Test",
		"description":  "",
		"vat_number":   "IT123",
		"address":      "",
		"city":         "",
		"main_contact": "",
		"email":        "not-an-email",
		"phone":        "abc",
		"language":     "fr",
		"notes":        "",
	}
	errs := ValidateOrganizationRow(row)

	fields := make(map[string]bool)
	for _, e := range errs {
		fields[e.Field] = true
	}
	if !fields["email"] {
		t.Error("expected error for invalid email")
	}
	if !fields["phone"] {
		t.Error("expected error for invalid phone")
	}
	if !fields["language"] {
		t.Error("expected error for invalid language")
	}
}

func TestOrganizationRowToData(t *testing.T) {
	row := map[string]string{
		"company_name": "Test Corp",
		"description":  "desc",
		"vat_number":   "IT123",
		"address":      "addr",
		"city":         "city",
		"main_contact": "contact",
		"email":        "test@test.com",
		"phone":        "+39 123",
		"language":     "en",
		"notes":        "notes",
	}
	data := OrganizationRowToData(row)
	if data["company_name"] != "Test Corp" {
		t.Fatalf("expected company_name 'Test Corp', got %v", data["company_name"])
	}
	if data["vat_number"] != "IT123" {
		t.Fatalf("expected vat_number 'IT123', got %v", data["vat_number"])
	}
}

func TestOrganizationDataToCreateRequest(t *testing.T) {
	data := map[string]interface{}{
		"company_name": "Test Corp",
		"description":  "desc",
		"vat_number":   "IT123",
		"address":      "addr",
		"city":         "city",
		"main_contact": "contact",
		"email":        "test@test.com",
		"phone":        "+39 123",
		"language":     "",
		"notes":        "",
	}
	req := OrganizationDataToCreateRequest(data)
	if req.Name != "Test Corp" {
		t.Fatalf("expected Name 'Test Corp', got %q", req.Name)
	}
	if req.CustomData["vat"] != "IT123" {
		t.Fatalf("expected custom_data.vat 'IT123' (Logto schema unchanged), got %v", req.CustomData["vat"])
	}
	if req.CustomData["language"] != "it" {
		t.Fatalf("expected default language 'it', got %v", req.CustomData["language"])
	}
}
