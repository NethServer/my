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

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"hello", false},
		{"", true},
		{"  ", true},
		{"  a  ", false},
	}
	for _, tt := range tests {
		err := ValidateRequired("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateRequired(%q): wantErr=%v, got=%v", tt.value, tt.wantErr, err)
		}
	}
}

func TestValidateMaxLength(t *testing.T) {
	if err := ValidateMaxLength("f", "abc", 5); err != nil {
		t.Error("expected no error for short string")
	}
	if err := ValidateMaxLength("f", "abcdef", 5); err == nil {
		t.Error("expected error for long string")
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"", false}, // optional
		{"user@example.com", false},
		{"invalid", true},
		{"@missing.com", true},
		{"user@", true},
	}
	for _, tt := range tests {
		err := ValidateEmail("email", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateEmail(%q): wantErr=%v, got=%v", tt.value, tt.wantErr, err)
		}
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"", false},
		{"+39 02 1234567", false},
		{"1234567", false},
		{"+1 (555) 123-4567", false},
		{"abc", true},
		{"123", true}, // too short (< 7)
	}
	for _, tt := range tests {
		err := ValidatePhone("phone", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidatePhone(%q): wantErr=%v, got=%v", tt.value, tt.wantErr, err)
		}
	}
}

func TestValidateLanguage(t *testing.T) {
	tests := []struct {
		value   string
		wantErr bool
	}{
		{"", false},
		{"it", false},
		{"en", false},
		{"IT", false},
		{"fr", true},
		{"de", true},
	}
	for _, tt := range tests {
		err := ValidateLanguage("language", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateLanguage(%q): wantErr=%v, got=%v", tt.value, tt.wantErr, err)
		}
	}
}

func TestCheckDuplicateInSet(t *testing.T) {
	seen := make(map[string]int)

	// First occurrence should be fine
	err := CheckDuplicateInSet("name", "Alice", seen, 2)
	if err != nil {
		t.Fatal("first occurrence should not be a duplicate")
	}

	// Second occurrence should be detected
	err = CheckDuplicateInSet("name", "alice", seen, 3)
	if err == nil {
		t.Fatal("expected duplicate error for case-insensitive match")
	}
	if err.Message != "duplicate_in_csv" {
		t.Fatalf("expected duplicate_in_csv message, got: %s", err.Message)
	}

	// Empty value should not be checked
	err = CheckDuplicateInSet("name", "", seen, 4)
	if err != nil {
		t.Fatal("empty value should not be checked for duplicates")
	}
}

func TestValidateInSet(t *testing.T) {
	allowed := []string{"Admin", "Support"}

	if err := ValidateInSet("role", "admin", allowed); err != nil {
		t.Error("expected case-insensitive match")
	}
	if err := ValidateInSet("role", "Unknown", allowed); err == nil {
		t.Error("expected error for unknown value")
	}
	if err := ValidateInSet("role", "", allowed); err != nil {
		t.Error("expected no error for empty value")
	}
}
