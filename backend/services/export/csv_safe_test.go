/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"plain name":                      "plain name",
		"":                                "",
		"  leading spaces":                "  leading spaces",
		"=HYPERLINK(\"https://x\";\"y\")": "'=HYPERLINK(\"https://x\";\"y\")",
		"+1-2":                            "'+1-2",
		"-cmd":                            "'-cmd",
		"@SUM(A1)":                        "'@SUM(A1)",
		"  =1+1":                          "'  =1+1",
		"\t=1":                            "'\t=1",
		"user@example.com":                "user@example.com",
		"note with = inside":              "note with = inside",
	}
	for in, want := range cases {
		assert.Equal(t, want, csvSafe(in), "input %q", in)
	}
	assert.Equal(t, []string{"a", "'=b", ""}, csvSafeRow([]string{"a", "=b", ""}))
}
