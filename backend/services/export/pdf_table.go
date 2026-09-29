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
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/image"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/extension"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/orientation"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
	"github.com/jung-kurt/gofpdf"
)

//go:embed assets/logo.png
var pdfLogo []byte

const (
	dateLayout     = "2006-01-02"
	dateTimeLayout = "2006-01-02 15:04 MST"

	// A4 landscape, millimetres
	pdfPageWidth    = 297.0
	pdfMarginX      = 10.0
	pdfMarginTop    = 10.0
	pdfMarginBottom = 14.0
	pdfGridColumns  = 12

	pdfCellPadding  = 2.0
	pdfHeadHeight   = 7.5
	pdfRowHeight    = 6.5
	pdfHeadFontSize = 8.0
	pdfBodyFontSize = 7.5
)

var (
	pdfBrand = props.Color{Red: 2, Green: 132, Blue: 199} // sky-600, the frontend primary
	pdfWhite = props.Color{Red: 255, Green: 255, Blue: 255}
	pdfInk   = props.Color{Red: 31, Green: 41, Blue: 55}    // gray-800
	pdfMuted = props.Color{Red: 107, Green: 114, Blue: 128} // gray-500
	pdfZebra = props.Color{Red: 246, Green: 248, Blue: 250}
	pdfGreen = props.Color{Red: 21, Green: 128, Blue: 61} // green-700
	pdfAmber = props.Color{Red: 180, Green: 83, Blue: 9}  // amber-700
	pdfRed   = props.Color{Red: 185, Green: 28, Blue: 28} // red-700
)

// pdfColumn is one column of a tabular report; the widths of a report's
// columns add up to pdfGridColumns.
type pdfColumn struct {
	Title string
	Width int
	// Align is the horizontal alignment of both the title and the values;
	// counters read best right-aligned
	Align align.Type
	// Status columns colour their value by meaning (active, suspended, ...)
	Status bool
}

// pdfReport is the tabular content every export renders through renderTablePDF,
// so all the PDF exports share one layout.
type pdfReport struct {
	Title      string
	Filters    map[string]interface{}
	ExportedBy string
	Columns    []pdfColumn
	Rows       [][]string
	Empty      string
}

// renderTablePDF lays the report out as a landscape table: a branded header
// with the report metadata, a coloured column header and zebra rows, all
// repeated on every page. Cell values are cut to the column width, so each
// record stays on a single line.
func renderTablePDF(r pdfReport) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}

	cfg := config.NewBuilder().
		WithOrientation(orientation.Horizontal).
		WithLeftMargin(pdfMarginX).
		WithRightMargin(pdfMarginX).
		WithTopMargin(pdfMarginTop).
		WithBottomMargin(pdfMarginBottom).
		WithDefaultFont(&props.Font{Family: fontfamily.Helvetica, Size: pdfBodyFontSize, Color: &pdfInk}).
		WithPageNumber(props.PageNumber{
			Pattern: "Page {current} of {total}",
			Place:   props.RightBottom,
			Family:  fontfamily.Helvetica,
			Size:    7,
			Color:   &pdfMuted,
		}).
		// plain (non UTF-8) metadata: browsers show a UTF-16 title as garbled text
		WithTitle(fmt.Sprintf("%s - Export %s", r.Title, formatDate(time.Now())), false).
		WithCreator("my.nethesis.it", false).
		Build()

	m := maroto.New(cfg)

	measure := newTextMeasurer()

	if err := m.RegisterHeader(r.headerRows(measure)...); err != nil {
		return nil, fmt.Errorf("failed to register PDF header: %w", err)
	}

	if len(r.Rows) == 0 {
		m.AddRows(r.emptyRow())
	} else {
		m.AddRows(r.bodyRows(measure)...)
	}

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return doc.GetBytes(), nil
}

func (r pdfReport) validate() error {
	width := 0
	for _, c := range r.Columns {
		width += c.Width
	}
	if width != pdfGridColumns {
		return fmt.Errorf("pdf report %q: column widths add up to %d, want %d", r.Title, width, pdfGridColumns)
	}
	for i, cells := range r.Rows {
		if len(cells) != len(r.Columns) {
			return fmt.Errorf("pdf report %q: row %d has %d cells, want %d", r.Title, i, len(cells), len(r.Columns))
		}
	}
	return nil
}

// headerRows are repeated at the top of every page: logo and title, the
// export metadata, the filters (when any) and the column header.
func (r pdfReport) headerRows(measure *textMeasurer) []core.Row {
	subtitle := fmt.Sprintf("%d records   |   Generated %s", len(r.Rows), time.Now().Format(dateTimeLayout))
	if r.ExportedBy != "" {
		subtitle += "   |   Exported by " + r.ExportedBy
	}

	rows := []core.Row{
		row.New(12).Add(
			col.New(4).Add(image.NewFromBytes(pdfLogo, extension.Png, props.Rect{Percent: 62, Top: 1})),
			col.New(8).Add(
				text.New(r.Title, props.Text{
					Size:  15,
					Style: fontstyle.Bold,
					Align: align.Right,
					Color: &pdfInk,
					Top:   0.5,
				}),
				text.New(subtitle, props.Text{
					Size:  7,
					Align: align.Right,
					Color: &pdfMuted,
					Top:   7.5,
				}),
			),
		),
	}

	if len(r.Filters) > 0 {
		filtersText := "Filters: " + formatFilters(r.Filters)
		rows = append(rows, row.New(5).Add(
			col.New(pdfGridColumns).Add(text.New(measure.fit(filtersText, 7, pdfUsableWidth()), props.Text{
				Size:  7,
				Align: align.Right,
				Color: &pdfMuted,
				Top:   0.5,
			})),
		))
	}

	rows = append(rows, row.New(3))

	head := row.New(pdfHeadHeight).WithStyle(&props.Cell{BackgroundColor: &pdfBrand})
	for _, c := range r.Columns {
		head.Add(text.NewCol(c.Width, c.Title, props.Text{
			Size:  pdfHeadFontSize,
			Style: fontstyle.Bold,
			Color: &pdfWhite,
			Align: c.Align,
			Left:  pdfCellPadding,
			Right: pdfCellPadding,
			Top:   2.3,
		}))
	}
	rows = append(rows, head)

	return rows
}

func (r pdfReport) bodyRows(measure *textMeasurer) []core.Row {
	rows := make([]core.Row, 0, len(r.Rows))
	for i, cells := range r.Rows {
		line := row.New(pdfRowHeight)
		if i%2 == 0 {
			line.WithStyle(&props.Cell{BackgroundColor: &pdfZebra})
		}
		for j, c := range r.Columns {
			available := pdfColumnWidth(c.Width) - 2*pdfCellPadding
			color := &pdfInk
			if c.Status {
				color = statusColor(cells[j])
			}
			line.Add(text.NewCol(c.Width, measure.fit(cells[j], pdfBodyFontSize, available), props.Text{
				Size:  pdfBodyFontSize,
				Color: color,
				Align: c.Align,
				Left:  pdfCellPadding,
				Right: pdfCellPadding,
				Top:   2.0,
			}))
		}
		rows = append(rows, line)
	}
	return rows
}

func (r pdfReport) emptyRow() core.Row {
	return row.New(14).Add(
		col.New(pdfGridColumns).Add(text.New(r.Empty, props.Text{
			Size:  9,
			Style: fontstyle.Italic,
			Align: align.Center,
			Color: &pdfMuted,
			Top:   5,
		})),
	)
}

// statusColor maps a status label to the colour the frontend badges use for it
func statusColor(label string) *props.Color {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "active", "assigned":
		return &pdfGreen
	case "suspended", "deleted":
		return &pdfRed
	case "inactive", "not active", "unassigned", "unknown":
		return &pdfAmber
	default:
		return &pdfInk
	}
}

func pdfUsableWidth() float64 {
	return pdfPageWidth - 2*pdfMarginX
}

func pdfColumnWidth(units int) float64 {
	return pdfUsableWidth() / pdfGridColumns * float64(units)
}

// textMeasurer measures strings with the same core font the PDF is rendered
// with, so a cell is cut exactly where it would stop fitting its column.
type textMeasurer struct {
	pdf       *gofpdf.Fpdf
	translate func(string) string
}

func newTextMeasurer() *textMeasurer {
	pdf := gofpdf.New("L", "mm", "A4", "")
	return &textMeasurer{pdf: pdf, translate: pdf.UnicodeTranslatorFromDescriptor("")}
}

func (t *textMeasurer) width(s string, size float64) float64 {
	t.pdf.SetFont("Helvetica", "", size)
	return t.pdf.GetStringWidth(t.translate(s))
}

// fit returns s when it fits in width millimetres at the given font size,
// otherwise the longest prefix that does, followed by an ellipsis.
func (t *textMeasurer) fit(s string, size, width float64) string {
	s = strings.TrimSpace(strings.Join(strings.Fields(s), " "))
	full := t.width(s, size)
	if full <= width {
		return s
	}

	const ellipsis = "..."
	runes := []rune(s)

	// start from the proportional estimate and shrink until it fits
	n := int(float64(len(runes))*width/full) + 2
	if n > len(runes) {
		n = len(runes)
	}
	for ; n > 0; n-- {
		candidate := strings.TrimRight(string(runes[:n]), " ") + ellipsis
		if t.width(candidate, size) <= width {
			return candidate
		}
	}
	return ellipsis
}

// formatFilters renders the applied filters in a stable order
func formatFilters(filters map[string]interface{}) string {
	if len(filters) == 0 {
		return "None"
	}

	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s = %v", k, filters[k]))
	}
	return strings.Join(parts, ", ")
}

// titleCase turns a snake_case status value into a label ("not_active" -> "Not active")
func titleCase(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// formatCount renders an inline counter, empty when the list did not compute it
func formatCount(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}

func formatDate(t time.Time) string {
	return t.Format(dateLayout)
}

func formatOptionalDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(dateLayout)
}
