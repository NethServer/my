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
	_ "embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jung-kurt/gofpdf"
)

//go:embed assets/logo.png
var pdfLogo []byte

const (
	dateLayout     = "2006-01-02"
	dateTimeLayout = "2006-01-02 15:04 MST"

	// A4 landscape, millimetres
	pdfMarginX      = 10.0
	pdfMarginTop    = 10.0
	pdfMarginBottom = 14.0
	pdfGridColumns  = 12

	pdfFont         = "Helvetica"
	pdfCellPadding  = 2.0
	pdfHeadHeight   = 7.5
	pdfRowHeight    = 6.5
	pdfHeadFontSize = 8.0
	pdfBodyFontSize = 7.5
	pdfLogoHeight   = 8.0
	pdfLogoRatio    = 640.0 / 113.0 // width/height of assets/logo.png
)

type rgb struct{ r, g, b int }

var (
	pdfBrand = rgb{2, 132, 199} // sky-600, the frontend primary
	pdfWhite = rgb{255, 255, 255}
	pdfInk   = rgb{31, 41, 55}    // gray-800
	pdfMuted = rgb{107, 114, 128} // gray-500
	pdfZebra = rgb{246, 248, 250}
	pdfGreen = rgb{21, 128, 61} // green-700
	pdfAmber = rgb{180, 83, 9}  // amber-700
	pdfRed   = rgb{185, 28, 28} // red-700
)

// pdfAlign is the horizontal alignment of a column, in gofpdf notation
type pdfAlign string

const (
	alignLeft  pdfAlign = "L"
	alignRight pdfAlign = "R"
)

// pdfColumn is one column of a tabular report; the widths of a report's
// columns add up to pdfGridColumns.
type pdfColumn struct {
	Title string
	Width int
	// Align applies to both the title and the values; counters read best
	// right-aligned. Empty means left.
	Align pdfAlign
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
//
// The table is drawn with gofpdf directly: one CellFormat per cell keeps a
// 10,000-row export in the order of a second.
func renderTablePDF(r pdfReport) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}

	doc := newPDFDocument(r)
	doc.pdf.AddPage()

	if len(r.Rows) == 0 {
		doc.empty(r.Empty)
	} else {
		doc.rows(r)
	}

	var buf bytes.Buffer
	if err := doc.pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}
	return buf.Bytes(), nil
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

// pdfDocument wraps the gofpdf document with the pieces every drawing step
// needs: the core-font translator (built once) and the usable page width.
type pdfDocument struct {
	pdf   *gofpdf.Fpdf
	tr    func(string) string
	width float64
}

func newPDFDocument(r pdfReport) *pdfDocument {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetMargins(pdfMarginX, pdfMarginTop, pdfMarginX)
	pdf.SetAutoPageBreak(true, pdfMarginBottom)
	pdf.SetCellMargin(pdfCellPadding)
	// plain (non UTF-8) metadata: browsers show a UTF-16 title as garbled text
	pdf.SetTitle(fmt.Sprintf("%s - Export %s", r.Title, formatDate(time.Now())), false)
	pdf.SetCreator("my.nethesis.it", false)
	pdf.AliasNbPages("")
	pdf.RegisterImageOptionsReader("logo", gofpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(pdfLogo))

	pageWidth, _ := pdf.GetPageSize()
	d := &pdfDocument{
		pdf:   pdf,
		tr:    pdf.UnicodeTranslatorFromDescriptor(""),
		width: pageWidth - 2*pdfMarginX,
	}

	subtitle := fmt.Sprintf("%d records   |   Generated %s", len(r.Rows), time.Now().Format(dateTimeLayout))
	if r.ExportedBy != "" {
		subtitle += "   |   Exported by " + r.ExportedBy
	}
	filters := ""
	if len(r.Filters) > 0 {
		filters = "Filters: " + formatFilters(r.Filters)
	}

	pdf.SetHeaderFunc(func() { d.header(r, subtitle, filters) })
	pdf.SetFooterFunc(d.footer)

	// the body font is the state gofpdf restores after every page header
	d.setFont("", pdfBodyFontSize, pdfInk)
	return d
}

// header is drawn at the top of every page: logo and title, the export
// metadata, the filters (when any) and the column header.
func (d *pdfDocument) header(r pdfReport, subtitle, filters string) {
	pdf := d.pdf

	pdf.ImageOptions("logo", pdfMarginX, pdfMarginTop+1, pdfLogoHeight*pdfLogoRatio, pdfLogoHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	pdf.SetXY(pdfMarginX, pdfMarginTop)
	d.setFont("B", 15, pdfInk)
	pdf.CellFormat(d.width, 8, d.tr(r.Title), "", 1, "R", false, 0, "")

	d.setFont("", 7, pdfMuted)
	pdf.CellFormat(d.width, 4, d.tr(subtitle), "", 1, "R", false, 0, "")
	if filters != "" {
		pdf.CellFormat(d.width, 4, d.tr(d.fit(filters, d.width-2*pdfCellPadding)), "", 1, "R", false, 0, "")
	}
	pdf.Ln(3)

	d.setFont("B", pdfHeadFontSize, pdfWhite)
	d.setFillColor(pdfBrand)
	for _, c := range r.Columns {
		pdf.CellFormat(d.columnWidth(c.Width), pdfHeadHeight, d.tr(c.Title), "", 0, c.align(), true, 0, "")
	}
	pdf.Ln(-1)
}

func (d *pdfDocument) footer() {
	d.pdf.SetY(-10)
	d.setFont("", 7, pdfMuted)
	d.pdf.CellFormat(0, 5, fmt.Sprintf("Page %d of {nb}", d.pdf.PageNo()), "", 0, "R", false, 0, "")
}

func (d *pdfDocument) rows(r pdfReport) {
	pdf := d.pdf
	widths := make([]float64, len(r.Columns))
	for j, c := range r.Columns {
		widths[j] = d.columnWidth(c.Width)
	}

	d.setFont("", pdfBodyFontSize, pdfInk)
	d.setFillColor(pdfZebra)
	for i, cells := range r.Rows {
		zebra := i%2 == 0
		for j, c := range r.Columns {
			value := d.fit(cells[j], widths[j]-2*pdfCellPadding)
			if c.Status {
				d.setTextColor(statusColor(cells[j]))
			}
			pdf.CellFormat(widths[j], pdfRowHeight, d.tr(value), "", 0, c.align(), zebra, 0, "")
			if c.Status {
				d.setTextColor(pdfInk)
			}
		}
		pdf.Ln(-1)
	}
}

func (d *pdfDocument) empty(message string) {
	d.pdf.Ln(4)
	d.setFont("I", 9, pdfMuted)
	d.pdf.CellFormat(d.width, 8, d.tr(message), "", 1, "C", false, 0, "")
}

func (d *pdfDocument) columnWidth(units int) float64 {
	return d.width / pdfGridColumns * float64(units)
}

func (d *pdfDocument) setFont(style string, size float64, color rgb) {
	d.pdf.SetFont(pdfFont, style, size)
	d.setTextColor(color)
}

func (d *pdfDocument) setTextColor(c rgb) { d.pdf.SetTextColor(c.r, c.g, c.b) }
func (d *pdfDocument) setFillColor(c rgb) { d.pdf.SetFillColor(c.r, c.g, c.b) }

// textWidth measures s with the current font, in millimetres
func (d *pdfDocument) textWidth(s string) float64 {
	return d.pdf.GetStringWidth(d.tr(s))
}

// fit returns s when it fits in width millimetres with the current font,
// otherwise the longest prefix that does, followed by an ellipsis.
func (d *pdfDocument) fit(s string, width float64) string {
	s = strings.Join(strings.Fields(s), " ")
	full := d.textWidth(s)
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
		if d.textWidth(candidate) <= width {
			return candidate
		}
	}
	return ellipsis
}

func (c pdfColumn) align() string {
	if c.Align == "" {
		return string(alignLeft)
	}
	return string(c.Align)
}

// statusColor maps a status label to the colour the frontend badges use for it
func statusColor(label string) rgb {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "active", "assigned":
		return pdfGreen
	case "suspended", "deleted":
		return pdfRed
	case "inactive", "not active", "unassigned", "unknown":
		return pdfAmber
	default:
		return pdfInk
	}
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
