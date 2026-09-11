/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package export

import "strings"

// csvSafe neutralises a cell that a spreadsheet would evaluate as a formula.
// Names, notes and e-mail addresses in an export come from lower tiers of the
// hierarchy and are opened by higher ones: a leading =, +, -, @ or a control
// character is what turns a cell into =HYPERLINK(...) or a DDE call on the
// reader's workstation. A leading apostrophe makes the spreadsheet keep it
// as text.
func csvSafe(cell string) string {
	trimmed := strings.TrimLeft(cell, " \t\r\n")
	if trimmed == "" {
		return cell
	}
	switch trimmed[0] {
	case '=', '+', '-', '@', '\t', '\r', '\n':
		return "'" + cell
	}
	return cell
}

// csvSafeRow applies csvSafe to every cell of a row.
func csvSafeRow(row []string) []string {
	out := make([]string, len(row))
	for i, cell := range row {
		out[i] = csvSafe(cell)
	}
	return out
}
