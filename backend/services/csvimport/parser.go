/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package csvimport

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

const (
	MaxFileSize = 10 * 1024 * 1024 // 10MB
	MaxRows     = 1000
)

// ParseCSV reads a CSV file from raw bytes, handles BOM and encoding, and returns headers + rows.
// Returns an error if the file structure is invalid (bad CSV, empty, too many rows).
func ParseCSV(data []byte) ([]string, [][]string, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("file is empty")
	}

	// Strip UTF-8 BOM if present
	data = stripBOM(data)

	// Try UTF-8 first, fall back to Latin-1
	reader := tryCSVReader(bytes.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		// Try Latin-1 encoding
		latin1Reader := transform.NewReader(bytes.NewReader(data), charmap.ISO8859_1.NewDecoder())
		reader = tryCSVReader(latin1Reader)
		records, err = reader.ReadAll()
		if err != nil {
			return nil, nil, fmt.Errorf("invalid CSV format: %w", err)
		}
	}

	if len(records) == 0 {
		return nil, nil, fmt.Errorf("file is empty")
	}

	// First row is the header
	headers := records[0]
	for i, h := range headers {
		headers[i] = strings.TrimSpace(strings.ToLower(h))
	}

	rows := records[1:]

	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("file contains only headers, no data rows")
	}

	if len(rows) > MaxRows {
		return nil, nil, fmt.Errorf("too many rows (%d). maximum allowed: %d", len(rows), MaxRows)
	}

	return headers, rows, nil
}

// ValidateHeaders checks that the CSV headers match the expected columns.
// Returns an error listing missing/unexpected columns.
func ValidateHeaders(headers, expected []string) error {
	expectedSet := make(map[string]bool, len(expected))
	for _, e := range expected {
		expectedSet[e] = true
	}

	headerSet := make(map[string]bool, len(headers))
	for _, h := range headers {
		headerSet[h] = true
	}

	var missing []string
	for _, e := range expected {
		if !headerSet[e] {
			missing = append(missing, e)
		}
	}

	var unexpected []string
	for _, h := range headers {
		if !expectedSet[h] {
			unexpected = append(unexpected, h)
		}
	}

	if len(missing) > 0 || len(unexpected) > 0 {
		parts := []string{}
		if len(missing) > 0 {
			parts = append(parts, fmt.Sprintf("missing columns: %s", strings.Join(missing, ", ")))
		}
		if len(unexpected) > 0 {
			parts = append(parts, fmt.Sprintf("unexpected columns: %s", strings.Join(unexpected, ", ")))
		}
		return fmt.Errorf("invalid CSV headers: %s. expected: %s", strings.Join(parts, "; "), strings.Join(expected, ", "))
	}

	return nil
}

// RowToMap converts a CSV row (string slice) into a map using the headers as keys.
// Empty values are preserved as empty strings.
func RowToMap(headers []string, row []string) map[string]string {
	result := make(map[string]string, len(headers))
	for i, h := range headers {
		if i < len(row) {
			result[h] = strings.TrimSpace(row[i])
		} else {
			result[h] = ""
		}
	}
	return result
}

// GenerateTemplate creates a CSV template with headers and example rows.
func GenerateTemplate(headers []string, examples [][]string) []byte {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	_ = writer.Write(headers)
	for _, example := range examples {
		_ = writer.Write(example)
	}

	writer.Flush()
	return buf.Bytes()
}

// stripBOM removes UTF-8 BOM (byte order mark) from the beginning of data
func stripBOM(data []byte) []byte {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:]
	}
	return data
}

// tryCSVReader creates a csv.Reader with flexible settings
func tryCSVReader(r io.Reader) *csv.Reader {
	reader := csv.NewReader(r)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	return reader
}
