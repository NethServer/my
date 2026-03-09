/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package response

import (
	"net/http"
	"testing"
)

func TestOK(t *testing.T) {
	r := OK("success", nil)
	if r.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", r.Code)
	}
	if r.Message != "success" {
		t.Fatalf("expected 'success', got %s", r.Message)
	}
}

func TestCreated(t *testing.T) {
	r := Created("created", map[string]string{"id": "123"})
	if r.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", r.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	r := Unauthorized("denied", nil)
	if r.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", r.Code)
	}
}

func TestNotFound(t *testing.T) {
	r := NotFound("not found", nil)
	if r.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", r.Code)
	}
}

func TestInternalServerError(t *testing.T) {
	r := InternalServerError("error", nil)
	if r.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", r.Code)
	}
}
