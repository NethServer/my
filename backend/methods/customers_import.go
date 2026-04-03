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
	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/services/csvimport"
)

// GetCustomersImportTemplate handles GET /api/customers/import/template
func GetCustomersImportTemplate(c *gin.Context) {
	sendTemplateCSV(c, "customers_import_template.csv",
		csvimport.OrganizationCSVHeaders, csvimport.OrganizationTemplateExamples)
}

// ValidateCustomersImport handles POST /api/customers/import/validate
func ValidateCustomersImport(c *gin.Context) {
	validateOrganizationImport(c, "customers")
}

// ConfirmCustomersImport handles POST /api/customers/import/confirm
func ConfirmCustomersImport(c *gin.Context) {
	confirmOrganizationImport(c, "customers")
}
