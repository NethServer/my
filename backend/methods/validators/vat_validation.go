/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package validators

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/response"
)

// ValidateVAT validates if a VAT exists in the specified entity type
func ValidateVAT(c *gin.Context) {
	entityType := c.Param("entity_type")
	if entityType == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("entity_type parameter is required", nil))
		return
	}

	// Validate entity type
	validEntityTypes := map[string]bool{
		"distributors": true,
		"resellers":    true,
		"customers":    true,
	}
	if !validEntityTypes[entityType] {
		c.JSON(http.StatusBadRequest, response.BadRequest("entity_type must be one of: distributors, resellers, customers", nil))
		return
	}

	vat := c.Query("vat")
	if vat == "" {
		c.JSON(http.StatusBadRequest, response.BadRequest("vat query parameter is required", nil))
		return
	}

	excludeID := c.Query("exclude_id") // Optional parameter for updates

	exists, err := helpers.CheckVATExists(vat, entityType, excludeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.InternalServerError(err.Error(), nil))
		return
	}

	result := models.VATValidationResponse{
		Exists: exists,
	}

	c.JSON(http.StatusOK, response.Success(http.StatusOK, "VAT validation completed", result))
}
