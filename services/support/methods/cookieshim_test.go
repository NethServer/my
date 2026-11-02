/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"net/http"
	"strings"
	"testing"
)

func TestInjectCookieShimRunsFirst(t *testing.T) {
	shim := "<script>" + cookieShimFor("sp12345678_") + "</script>"
	tests := []struct{ in, want string }{
		{`<html><head lang="en"><script src="/app.js"></script></head></html>`,
			`<html><head lang="en">` + shim + `<script src="/app.js"></script></head></html>`},
		{`<!DOCTYPE html><html><script>window.location = '/api'</script></html>`,
			`<!DOCTYPE html><html>` + shim + `<script>window.location = '/api'</script></html>`},
		// not a document: a JSON answer labelled text/html stays as it is
		{`{"ret":"ok","data":{"token":"x"}}`, `{"ret":"ok","data":{"token":"x"}}`},
		{`<script>x()</script>`, `<script>x()</script>`},
		{`<header>menu</header><script>x()</script>`, `<header>menu</header><script>x()</script>`},
	}
	for _, tt := range tests {
		if got := string(injectCookieShim([]byte(tt.in), "sp12345678_")); got != tt.want {
			t.Errorf("injectCookieShim(%q) = %q", tt.in, got)
		}
	}
}

func TestCookieShimAllowedByCSP(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Security-Policy", "default-src 'self'")
	adjustCSP(h, "", cookieShimHash("sp12345678_"))
	if !strings.Contains(h.Get("Content-Security-Policy"), cookieShimHash("sp12345678_")) {
		t.Fatalf("the shim hash must be allowed: %q", h.Get("Content-Security-Policy"))
	}
}
