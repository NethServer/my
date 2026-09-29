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
	"time"

	"github.com/johnfercher/maroto/v2/pkg/consts/align"

	"github.com/nethesis/my/backend/models"
)

// The three organization reports share one row shape - identity, the inline
// counters the list shows on screen, then creator, status and creation date -
// and differ only in which counters their level carries.
var (
	distributorPDFColumns = organizationPDFColumns(2, 2,
		pdfColumn{Title: "Resellers", Width: 1, Align: align.Right},
		pdfColumn{Title: "Customers", Width: 1, Align: align.Right},
		pdfColumn{Title: "Systems", Width: 1, Align: align.Right},
	)
	resellerPDFColumns = organizationPDFColumns(3, 2,
		pdfColumn{Title: "Customers", Width: 1, Align: align.Right},
		pdfColumn{Title: "Systems", Width: 1, Align: align.Right},
	)
	customerPDFColumns = organizationPDFColumns(3, 3,
		pdfColumn{Title: "Systems", Width: 1, Align: align.Right},
	)
)

func organizationPDFColumns(nameWidth, descriptionWidth int, counters ...pdfColumn) []pdfColumn {
	columns := []pdfColumn{
		{Title: "Name", Width: nameWidth},
		{Title: "Description", Width: descriptionWidth},
		{Title: "VAT", Width: 1},
	}
	columns = append(columns, counters...)
	return append(columns,
		pdfColumn{Title: "Created by", Width: 2},
		pdfColumn{Title: "Status", Width: 1, Status: true},
		pdfColumn{Title: "Created", Width: 1},
	)
}

// organizationPDFRow lays out one organization in the column order above; the
// counters come in the same order as the level's counter columns.
func organizationPDFRow(name, description string, customData map[string]interface{}, createdBy *models.OrgCreator, suspendedAt *time.Time, createdAt time.Time, counters ...*int) []string {
	vat := ""
	if v, ok := customData["vat"].(string); ok {
		vat = v
	}

	creator := ""
	if createdBy != nil {
		creator = createdBy.Name
		if creator == "" {
			creator = createdBy.Username
		}
	}

	status := "Active"
	if suspendedAt != nil {
		status = "Suspended"
	}

	row := []string{name, description, vat}
	for _, n := range counters {
		row = append(row, formatCount(n))
	}
	return append(row, creator, status, formatDate(createdAt))
}
