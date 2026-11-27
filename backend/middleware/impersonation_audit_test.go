/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */
package middleware

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeRequestDataRedactsJSON(t *testing.T) {
	got := sanitizeRequestData(`{"name":"acme","password":"hunter2","nested":{"api_key":"k1"}}`)
	assert.NotContains(t, got, "hunter2")
	assert.NotContains(t, got, "k1")
	assert.Contains(t, got, "acme")
}

func TestSanitizeRequestDataRedactsBeforeTruncating(t *testing.T) {
	body := `{"filler":"` + strings.Repeat("x", 20000) + `","password":"hunter2"}`
	got := sanitizeRequestData(body)
	assert.NotContains(t, got, "hunter2")
	assert.True(t, strings.HasSuffix(got, "... (truncated)"))
	assert.LessOrEqual(t, len(got), 10000+len("... (truncated)"))
}

func TestSanitizeRequestDataOmitsNonJSON(t *testing.T) {
	for name, body := range map[string]string{
		"malformed json": `{"password":"hunter2"`,
		"form":           "user=a&password=hunter2",
		"multipart":      "--b\r\nContent-Disposition: form-data; name=\"password\"\r\n\r\nhunter2\r\n--b--",
	} {
		t.Run(name, func(t *testing.T) {
			got := sanitizeRequestData(body)
			assert.NotContains(t, got, "hunter2")
			assert.Contains(t, got, "body omitted")
		})
	}
}

func TestSanitizeRequestDataTruncatesOnRuneBoundary(t *testing.T) {
	got := sanitizeRequestData(`{"note":"` + strings.Repeat("è", 8000) + `"}`)
	assert.True(t, utf8.ValidString(got))
}

func TestSanitizeRequestDataEmptyBody(t *testing.T) {
	assert.Equal(t, "", sanitizeRequestData(""))
}
