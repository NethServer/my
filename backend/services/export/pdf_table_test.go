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
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFitText(t *testing.T) {
	d := newPDFDocument(pdfReport{Title: "Test"})

	assert.Equal(t, "short", d.fit("short", 40))
	assert.Equal(t, "two words", d.fit("two   words", 40), "internal whitespace is collapsed")

	long := strings.Repeat("Rimini Informatica ", 10)
	cut := d.fit(long, 30)
	assert.True(t, strings.HasSuffix(cut, "..."), "cut text ends with an ellipsis: %q", cut)
	assert.Less(t, len(cut), len(long))
	assert.LessOrEqual(t, d.textWidth(cut), 30.0, "cut text fits the requested width")

	// nothing but the ellipsis fits in a sliver
	assert.Equal(t, "...", d.fit("WWWWWWWWWW", 3))
}

func TestFormatFiltersIsSorted(t *testing.T) {
	assert.Equal(t, "None", formatFilters(nil))
	assert.Equal(t, "status = active, type = mail, webtop", formatFilters(map[string]interface{}{
		"type":   "mail, webtop",
		"status": "active",
	}))
}

func TestTitleCase(t *testing.T) {
	assert.Equal(t, "", titleCase(""))
	assert.Equal(t, "Active", titleCase("active"))
	assert.Equal(t, "Not active", titleCase("not_active"))
}

func TestRenderTablePDFValidatesLayout(t *testing.T) {
	_, err := renderTablePDF(pdfReport{
		Title:   "Broken",
		Columns: []pdfColumn{{Title: "A", Width: 5}},
	})
	require.Error(t, err, "column widths must fill the grid")

	_, err = renderTablePDF(pdfReport{
		Title:   "Broken",
		Columns: []pdfColumn{{Title: "A", Width: 6}, {Title: "B", Width: 6}},
		Rows:    [][]string{{"only one cell"}},
	})
	require.Error(t, err, "every row must have one cell per column")
}

func TestRenderTablePDF(t *testing.T) {
	report := pdfReport{
		Title:      "Things",
		Filters:    map[string]interface{}{"type": "demo"},
		ExportedBy: "Owner (owner@example.com)",
		Columns:    []pdfColumn{{Title: "Name", Width: 8}, {Title: "Status", Width: 4}},
		Empty:      "No things found.",
	}

	empty, err := renderTablePDF(report)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(empty, []byte("%PDF")))
	// the title is what a browser tab shows: it must be plain text, not UTF-16
	assert.Contains(t, string(empty), "/Title (Things - Export "+formatDate(time.Now())+")")

	for i := 0; i < 120; i++ {
		report.Rows = append(report.Rows, []string{strings.Repeat("Very long thing name ", 6), "Active"})
	}
	full, err := renderTablePDF(report)
	require.NoError(t, err)
	assert.True(t, bytes.HasPrefix(full, []byte("%PDF")))
	assert.Greater(t, len(full), len(empty), "rows add pages to the document")
}
