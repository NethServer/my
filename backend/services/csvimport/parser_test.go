/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package csvimport

import (
	"strings"
	"testing"
)

func TestParseCSV_ValidFile(t *testing.T) {
	csv := "name,email\nAlice,alice@example.com\nBob,bob@example.com\n"
	headers, rows, err := ParseCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(headers) != 2 {
		t.Fatalf("expected 2 headers, got %d", len(headers))
	}
	if headers[0] != "name" || headers[1] != "email" {
		t.Fatalf("unexpected headers: %v", headers)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
}

func TestParseCSV_Empty(t *testing.T) {
	_, _, err := ParseCSV([]byte(""))
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseCSV_HeadersOnly(t *testing.T) {
	_, _, err := ParseCSV([]byte("name,email\n"))
	if err == nil {
		t.Fatal("expected error for headers-only file")
	}
	if !strings.Contains(err.Error(), "no data rows") {
		t.Fatalf("expected 'no data rows' error, got: %v", err)
	}
}

func TestParseCSV_TooManyRows(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("name\n")
	for i := 0; i < MaxRows+1; i++ {
		sb.WriteString("row\n")
	}
	_, _, err := ParseCSV([]byte(sb.String()))
	if err == nil {
		t.Fatal("expected error for too many rows")
	}
	if !strings.Contains(err.Error(), "too many rows") {
		t.Fatalf("expected 'too many rows' error, got: %v", err)
	}
}

func TestParseCSV_UTF8BOM(t *testing.T) {
	// UTF-8 BOM + valid CSV
	bom := []byte{0xEF, 0xBB, 0xBF}
	csv := append(bom, []byte("name\nAlice\n")...)
	headers, rows, err := ParseCSV(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if headers[0] != "name" {
		t.Fatalf("BOM not stripped, header: %q", headers[0])
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
}

func TestParseCSV_HeaderNormalization(t *testing.T) {
	csv := "  Name , EMAIL ,Phone\nAlice,a@b.com,123\n"
	headers, _, err := ParseCSV([]byte(csv))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"name", "email", "phone"}
	for i, h := range headers {
		if h != expected[i] {
			t.Fatalf("header %d: expected %q, got %q", i, expected[i], h)
		}
	}
}

func TestValidateHeaders_Match(t *testing.T) {
	err := ValidateHeaders([]string{"name", "email"}, []string{"name", "email"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateHeaders_Missing(t *testing.T) {
	err := ValidateHeaders([]string{"name"}, []string{"name", "email"})
	if err == nil {
		t.Fatal("expected error for missing column")
	}
	if !strings.Contains(err.Error(), "missing columns: email") {
		t.Fatalf("expected missing column error, got: %v", err)
	}
}

func TestValidateHeaders_Unexpected(t *testing.T) {
	err := ValidateHeaders([]string{"name", "email", "extra"}, []string{"name", "email"})
	if err == nil {
		t.Fatal("expected error for unexpected column")
	}
	if !strings.Contains(err.Error(), "unexpected columns: extra") {
		t.Fatalf("expected unexpected column error, got: %v", err)
	}
}

func TestRowToMap(t *testing.T) {
	headers := []string{"name", "email", "phone"}
	row := []string{"Alice", "alice@example.com"}

	result := RowToMap(headers, row)
	if result["name"] != "Alice" {
		t.Fatalf("expected Alice, got %q", result["name"])
	}
	if result["phone"] != "" {
		t.Fatalf("expected empty phone, got %q", result["phone"])
	}
}

func TestGenerateTemplate(t *testing.T) {
	headers := []string{"name", "email"}
	examples := [][]string{{"Alice", "a@b.com"}}
	data := GenerateTemplate(headers, examples)
	content := string(data)
	if !strings.Contains(content, "name,email") {
		t.Fatalf("expected headers in template, got: %q", content)
	}
	if !strings.Contains(content, "Alice,a@b.com") {
		t.Fatalf("expected example row in template, got: %q", content)
	}
}
