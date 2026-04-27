/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package models

// ImportRowStatus represents the validation verdict for a single CSV row.
type ImportRowStatus string

const (
	// ImportRowValid — row passes every check and will be CREATEd at confirm time.
	ImportRowValid ImportRowStatus = "valid"

	// ImportRowInvalid — row has at least one blocking error and is always skipped.
	ImportRowInvalid ImportRowStatus = "error"

	// ImportRowWarning — row is structurally OK but the entity already exists in the database.
	// At confirm time it is UPDATED if the request has `override: true`, otherwise skipped.
	ImportRowWarning ImportRowStatus = "warning"

	// ImportRowAmbiguous — row could not be resolved unambiguously (e.g. organization name
	// matches multiple orgs). Imported only when an explicit resolution is provided at confirm.
	ImportRowAmbiguous ImportRowStatus = "ambiguous"
)

// ImportFieldError carries a single field-level validation issue. The same struct is used
// for both blocking errors (in `errors[]`) and non-blocking warnings (in `warnings[]`).
type ImportFieldError struct {
	Field      string               `json:"field"`
	Message    string               `json:"message"`
	Value      string               `json:"value,omitempty"`
	Candidates []ImportOrgCandidate `json:"candidates,omitempty"`
}

// ImportOrgCandidate represents a candidate organization for ambiguous-name disambiguation.
type ImportOrgCandidate struct {
	LogtoID string `json:"logto_id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
}

// ImportRow is a single CSV row enriched with its validation verdict and field-level diagnostics.
// `errors` are blocking; `warnings` can be turned into UPDATEs by setting `override: true` at confirm.
type ImportRow struct {
	RowNumber int                    `json:"row_number"`
	Status    ImportRowStatus        `json:"status"`
	Data      map[string]interface{} `json:"data"`
	Errors    []ImportFieldError     `json:"errors,omitempty"`
	Warnings  []ImportFieldError     `json:"warnings,omitempty"`
}

// ImportValidationResult is the response of /import/validate.
type ImportValidationResult struct {
	ImportID      string      `json:"import_id"`
	TotalRows     int         `json:"total_rows"`
	ValidRows     int         `json:"valid_rows"`
	ErrorRows     int         `json:"error_rows"`
	WarningRows   int         `json:"warning_rows"`
	AmbiguousRows int         `json:"ambiguous_rows"`
	Rows          []ImportRow `json:"rows"`
}

// ImportConfirmRequest is the body of /import/confirm.
//
// `override` is a global toggle: when true, every row with status `warning` is UPDATEd
// using the CSV values; when false (or omitted) those rows are skipped. Rows with status
// `error` are always skipped. `resolutions` map row_number → chosen organization for
// rows with status `ambiguous`; ambiguous rows without a resolution are skipped.
type ImportConfirmRequest struct {
	ImportID    string                      `json:"import_id" validate:"required"`
	Override    bool                        `json:"override,omitempty"`
	Resolutions map[string]ImportResolution `json:"resolutions,omitempty"`
}

// ImportResolution is a user-supplied choice for one ambiguous row.
type ImportResolution struct {
	OrganizationID string `json:"organization_id"`
}

// ImportResultStatus represents the per-row outcome of /import/confirm.
type ImportResultStatus string

const (
	ImportResultCreated ImportResultStatus = "created"
	ImportResultUpdated ImportResultStatus = "updated"
	ImportResultSkipped ImportResultStatus = "skipped"
	ImportResultFailed  ImportResultStatus = "failed"
)

// ImportSkipReason explains why a row was skipped at confirm time.
type ImportSkipReason string

const (
	ImportSkipError              ImportSkipReason = "error"
	ImportSkipWarningNotOverride ImportSkipReason = "warning_not_overridden"
	ImportSkipAmbiguousUnresolve ImportSkipReason = "ambiguous_unresolved"
)

// ImportResultRow is the per-row outcome of /import/confirm.
type ImportResultRow struct {
	RowNumber int                `json:"row_number"`
	Status    ImportResultStatus `json:"status"`
	ID        string             `json:"id,omitempty"`     // populated when created or updated
	Reason    ImportSkipReason   `json:"reason,omitempty"` // populated when skipped
	Error     string             `json:"error,omitempty"`  // populated when failed
}

// ImportConfirmResult is the response of /import/confirm.
type ImportConfirmResult struct {
	Created int               `json:"created"`
	Updated int               `json:"updated"`
	Skipped int               `json:"skipped"`
	Failed  int               `json:"failed"`
	Results []ImportResultRow `json:"results"`
}

// ImportSessionData holds validated import data stored in Redis between validate and confirm.
type ImportSessionData struct {
	EntityType string      `json:"entity_type"`
	Rows       []ImportRow `json:"rows"`
	UserID     string      `json:"user_id"`
	UserOrgID  string      `json:"user_org_id"`
	OrgRole    string      `json:"org_role"`
}
