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

// GetResellersImportTemplate handles GET /api/resellers/import/template
func GetResellersImportTemplate(c *gin.Context) {
	sendTemplateCSV(c, "resellers_import_template.csv",
		csvimport.OrganizationCSVHeaders, csvimport.OrganizationTemplateExamples)
}

// ValidateResellersImport handles POST /api/resellers/import/validate
func ValidateResellersImport(c *gin.Context) {
	validateOrganizationImport(c, "resellers")
}

// ConfirmResellersImport handles POST /api/resellers/import/confirm
func ConfirmResellersImport(c *gin.Context) {
	confirmOrganizationImport(c, "resellers")
}
