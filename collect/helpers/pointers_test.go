/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrPtr(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"non-empty string", "hello"},
		{"empty string", ""},
		{"string with spaces", "hello world"},
		{"unicode string", "日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StrPtr(tt.input)
			assert.NotNil(t, result)
			assert.Equal(t, tt.input, *result)
		})
	}
}

func TestStrPtrReturnsDistinctPointers(t *testing.T) {
	a := StrPtr("same")
	b := StrPtr("same")
	assert.Equal(t, *a, *b)
	assert.NotSame(t, a, b, "each call returns a distinct pointer")
}

func TestNilIfEmpty(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantNil bool
		wantVal string
	}{
		{"empty string returns nil", "", true, ""},
		{"non-empty string returns pointer", "hello", false, "hello"},
		{"whitespace is not empty", " ", false, " "},
		{"unicode string", "日本語", false, "日本語"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NilIfEmpty(tt.input)
			if tt.wantNil {
				assert.Nil(t, result)
			} else {
				assert.NotNil(t, result)
				assert.Equal(t, tt.wantVal, *result)
			}
		})
	}
}

func TestDerefString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected string
	}{
		{"nil pointer returns empty string", nil, ""},
		{"non-empty pointer returns value", StrPtr("hello"), "hello"},
		{"empty string pointer returns empty", StrPtr(""), ""},
		{"whitespace pointer returns whitespace", StrPtr("  "), "  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DerefString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPointerRoundTrip(t *testing.T) {
	// StrPtr -> DerefString round-trip
	original := "round-trip-test"
	assert.Equal(t, original, DerefString(StrPtr(original)))

	// NilIfEmpty -> DerefString round-trip
	assert.Equal(t, "", DerefString(NilIfEmpty("")))
	assert.Equal(t, "value", DerefString(NilIfEmpty("value")))
}
