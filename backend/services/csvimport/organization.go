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
	"strings"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// OrganizationCSVHeaders defines the expected CSV columns for organization import
var OrganizationCSVHeaders = []string{
	"name", "description", "vat", "address", "city",
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
	addErr(ValidateRequired("name", row["name"]))
	addErr(ValidateMaxLength("name", row["name"], 255))
	addErr(ValidateRequired("vat", row["vat"]))

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

// CheckOrganizationExistsByName checks if an organization with the given name exists in the specified table.
// entityType must be one of: "distributors", "resellers", "customers".
func CheckOrganizationExistsByName(name, entityType string) (bool, error) {
	query := `SELECT COUNT(*) FROM ` + entityType + ` WHERE LOWER(name) = LOWER($1) AND deleted_at IS NULL`
	var count int
	err := database.DB.QueryRow(query, strings.TrimSpace(name)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// OrganizationRowToData converts a validated CSV row map into the data map stored in ImportRow.
func OrganizationRowToData(row map[string]string) map[string]interface{} {
	data := map[string]interface{}{
		"name":         row["name"],
		"description":  row["description"],
		"vat":          row["vat"],
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

// OrganizationDataToCreateRequest converts import data to a CreateLocalDistributorRequest
// (same struct shape for resellers and customers).
func OrganizationDataToCreateRequest(data map[string]interface{}) *models.CreateLocalDistributorRequest {
	name, _ := data["name"].(string)
	description, _ := data["description"].(string)

	language, _ := data["language"].(string)
	if language == "" {
		language = "it"
	}

	customData := map[string]interface{}{
		"vat":          data["vat"],
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
