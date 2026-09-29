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
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestOrganizationPDFColumnsFillTheGrid(t *testing.T) {
	for name, columns := range map[string][]pdfColumn{
		"distributor": distributorPDFColumns,
		"reseller":    resellerPDFColumns,
		"customer":    customerPDFColumns,
	} {
		width := 0
		for _, c := range columns {
			width += c.Width
		}
		assert.Equal(t, pdfGridColumns, width, name)
	}
}

func TestOrganizationPDFRow(t *testing.T) {
	created := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	resellers, customers, systems := 3, 12, 40

	row := organizationPDFRow("Dist S.p.A.", "Northern Italy", map[string]interface{}{"vat": "02734760412"},
		&models.OrgCreator{Name: "Owner"}, nil, created, &resellers, &customers, &systems)
	assert.Equal(t, []string{"Dist S.p.A.", "Northern Italy", "02734760412", "3", "12", "40", "Owner", "Active", "2026-09-01"}, row)
	assert.Len(t, row, len(distributorPDFColumns))

	// counters the list did not compute stay empty, the creator falls back to
	// the username and a suspension shows as status
	row = organizationPDFRow("Cust", "", nil, &models.OrgCreator{Username: "r1admin"}, &created, created, nil)
	assert.Equal(t, []string{"Cust", "", "", "", "r1admin", "Suspended", "2026-09-01"}, row)
	assert.Len(t, row, len(customerPDFColumns))
}
