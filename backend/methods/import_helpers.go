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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/backend/response"
	"github.com/nethesis/my/backend/services/csvimport"
	"github.com/nethesis/my/backend/services/local"
)

// readCSVFromRequest reads the uploaded CSV file from a multipart form request.
// Returns the raw bytes or writes an error response and returns nil.
func readCSVFromRequest(c *gin.Context) []byte {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("file parameter required", nil))
		return nil
	}
	defer func() { _ = file.Close() }()

	if header.Size > int64(csvimport.MaxFileSize) {
		c.JSON(http.StatusBadRequest, response.BadRequest(
			fmt.Sprintf("file too large (%d bytes). maximum allowed: %d bytes", header.Size, csvimport.MaxFileSize), nil))
		return nil
	}

	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.BadRequest("failed to read file", nil))
		return nil
	}

	return data
}

// formatImportError extracts detailed validation error messages from a creation error.
// If the error contains a ValidationError with field-level details, it returns a
// machine-readable string with field:message pairs for frontend i18n translation.
func formatImportError(err error) string {
	var validationErr *local.ValidationError
	if errors.As(err, &validationErr) && len(validationErr.ErrorData.Errors) > 0 {
		details := make([]string, 0, len(validationErr.ErrorData.Errors))
		for _, e := range validationErr.ErrorData.Errors {
			if e.Key != "" && e.Message != "" {
				details = append(details, fmt.Sprintf("%s: %s", e.Key, e.Message))
			} else if e.Message != "" {
				details = append(details, e.Message)
			}
		}
		if len(details) > 0 {
			return strings.Join(details, "; ")
		}
	}
	return err.Error()
}

// sendTemplateCSV sends a CSV template file as a download response.
func sendTemplateCSV(c *gin.Context, filename string, headers []string, examples [][]string) {
	data := csvimport.GenerateTemplate(headers, examples)
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, "text/csv", data)
}
