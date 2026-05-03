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
	"database/sql"
	"strings"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// OrganizationCSVHeaders defines the expected CSV columns for organization import
var OrganizationCSVHeaders = []string{
	"company_name", "description", "vat_number", "address", "city",
	"main_contact", "email", "phone", "language", "notes",
}

// OrganizationTemplateExamples provides example rows for the CSV template
var OrganizationTemplateExamples = [][]string{
	{"Acme Corp", "Main distributor", "IT12345678901", "Via Roma 1", "Milano", "Mario Rossi", "info@acme.it", "+39 02 1234567", "it", "VIP client"},
	{"Beta Inc", "", "IT98765432100", "Via Verdi 5", "Roma", "", "contact@beta.it", "", "en", ""},
}

// ValidateOrganizationRow validates a single CSV row for organization import.
// It checks field formats and returns a list of validation errors.
func ValidateOrganizationRow(row map[string]string) []models.ImportFieldError {
	var errs []models.ImportFieldError

	addErr := func(e *models.ImportFieldError) {
		if e != nil {
			errs = append(errs, *e)
		}
	}

	// Required fields
	addErr(ValidateRequired("company_name", row["company_name"]))
	addErr(ValidateMaxLength("company_name", row["company_name"], 255))
	addErr(ValidateRequired("vat_number", row["vat_number"]))

	// Optional fields with format validation
	addErr(ValidateMaxLength("description", row["description"], 500))
	addErr(ValidateMaxLength("address", row["address"], 255))
	addErr(ValidateMaxLength("city", row["city"], 255))
	addErr(ValidateMaxLength("main_contact", row["main_contact"], 255))
	addErr(ValidateEmail("email", row["email"]))
	addErr(ValidatePhone("phone", row["phone"]))
	addErr(ValidateLanguage("language", row["language"]))
	addErr(ValidateMaxLength("notes", row["notes"], 1000))

	return errs
}

// OrgExistenceState tells whether an organization is missing, active, or
// soft-deleted in the specified table.
type OrgExistenceState int

const (
	OrgNotExisting OrgExistenceState = iota
	OrgExistsActive
	OrgSoftDeleted
)

// CheckOrganizationExistenceStateByVAT returns the org-existence state for the
// given VAT in the specified table.
//
// Uniqueness is keyed off `vat` because the DB triggers
// (check_unique_vat_distributors / check_unique_vat_resellers) only enforce
// uniqueness on the VAT, not on the name — two distributors can share a name
// as long as their VAT differs. Customers carry no DB-level uniqueness at all,
// so callers are expected not to invoke this for the "customers" table.
func CheckOrganizationExistenceStateByVAT(vat, entityType string) (OrgExistenceState, error) {
	trimmed := strings.TrimSpace(vat)
	if trimmed == "" {
		return OrgNotExisting, nil
	}
	// Prefer the active row when both an active and a soft-deleted row share
	// the same VAT, so the caller sees `already_exists` (override-able) rather
	// than `archived` (blocking).
	query := `SELECT deleted_at IS NULL FROM ` + entityType + ` WHERE TRIM(custom_data->>'vat') = $1 ORDER BY deleted_at IS NULL DESC LIMIT 1`
	var isActive bool
	err := database.DB.QueryRow(query, trimmed).Scan(&isActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return OrgNotExisting, nil
		}
		return OrgNotExisting, err
	}
	if isActive {
		return OrgExistsActive, nil
	}
	return OrgSoftDeleted, nil
}

// GetOrganizationIDByVAT returns the Logto ID of the active organization with
// the given VAT in the specified table. Returns "" if no row matches. Mirrors
// the by-VAT semantics of the validate-time check so the override-driven UPDATE
// path looks up the same row that was flagged as a duplicate.
func GetOrganizationIDByVAT(vat, entityType string) (string, error) {
	trimmed := strings.TrimSpace(vat)
	if trimmed == "" {
		return "", nil
	}
	query := `SELECT logto_id FROM ` + entityType + ` WHERE TRIM(custom_data->>'vat') = $1 AND deleted_at IS NULL AND logto_id IS NOT NULL LIMIT 1`
	var logtoID string
	err := database.DB.QueryRow(query, trimmed).Scan(&logtoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return logtoID, nil
}

// OrganizationRowToData converts a validated CSV row map into the data map stored
// in ImportRow. Keys mirror the CSV column names so the validate response (which
// echoes `data` back to the frontend) lines up 1:1 with what the user typed.
func OrganizationRowToData(row map[string]string) map[string]interface{} {
	data := map[string]interface{}{
		"company_name": row["company_name"],
		"description":  row["description"],
		"vat_number":   row["vat_number"],
		"address":      row["address"],
		"city":         row["city"],
		"main_contact": row["main_contact"],
		"email":        row["email"],
		"phone":        row["phone"],
		"language":     row["language"],
		"notes":        row["notes"],
	}
	return data
}

// OrganizationDataToCreateRequest converts the CSV-side data map (CSV column
// names) into a CreateLocalDistributorRequest (which uses the internal Logto
// schema: `Name` for company_name and `custom_data.vat` for vat_number).
// Same struct shape works for resellers and customers.
func OrganizationDataToCreateRequest(data map[string]interface{}) *models.CreateLocalDistributorRequest {
	name, _ := data["company_name"].(string)
	description, _ := data["description"].(string)

	language, _ := data["language"].(string)
	if language == "" {
		language = "it"
	}

	customData := map[string]interface{}{
		"vat":          data["vat_number"],
		"address":      data["address"],
		"city":         data["city"],
		"main_contact": data["main_contact"],
		"email":        data["email"],
		"phone":        data["phone"],
		"language":     language,
		"notes":        data["notes"],
	}

	return &models.CreateLocalDistributorRequest{
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
		CustomData:  customData,
	}
}
