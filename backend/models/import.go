/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

// ImportRowStatus represents the validation status of a single CSV row
type ImportRowStatus string

const (
	ImportRowValid     ImportRowStatus = "valid"
	ImportRowInvalid   ImportRowStatus = "error"
	ImportRowDuplicate ImportRowStatus = "duplicate"
)

// ImportFieldError represents a validation error for a specific field in a CSV row
type ImportFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ImportRow represents a single validated CSV row with its status and any errors
type ImportRow struct {
	RowNumber int                    `json:"row_number"`
	Status    ImportRowStatus        `json:"status"`
	Data      map[string]interface{} `json:"data"`
	Errors    []ImportFieldError     `json:"errors,omitempty"`
}

// ImportValidationResult represents the result of validating a CSV file
type ImportValidationResult struct {
	ImportID      string      `json:"import_id"`
	TotalRows     int         `json:"total_rows"`
	ValidRows     int         `json:"valid_rows"`
	ErrorRows     int         `json:"error_rows"`
	DuplicateRows int         `json:"duplicate_rows"`
	Rows          []ImportRow `json:"rows"`
}

// ImportConfirmRequest represents the request to confirm an import
type ImportConfirmRequest struct {
	ImportID string `json:"import_id" validate:"required"`
	SkipRows []int  `json:"skip_rows,omitempty"`
}

// ImportResultStatus represents the status of a single row after import execution
type ImportResultStatus string

const (
	ImportResultCreated ImportResultStatus = "created"
	ImportResultSkipped ImportResultStatus = "skipped"
	ImportResultFailed  ImportResultStatus = "failed"
)

// ImportResultRow represents the result of importing a single row
type ImportResultRow struct {
	RowNumber int                `json:"row_number"`
	Status    ImportResultStatus `json:"status"`
	ID        string             `json:"id,omitempty"`
	Error     string             `json:"error,omitempty"`
}

// ImportConfirmResult represents the result of confirming an import
type ImportConfirmResult struct {
	Created int               `json:"created"`
	Skipped int               `json:"skipped"`
	Failed  int               `json:"failed"`
	Results []ImportResultRow `json:"results"`
}

// ImportSessionData holds validated import data stored in Redis between validate and confirm
type ImportSessionData struct {
	EntityType string      `json:"entity_type"`
	Rows       []ImportRow `json:"rows"`
	UserID     string      `json:"user_id"`
	UserOrgID  string      `json:"user_org_id"`
	OrgRole    string      `json:"org_role"`
}
