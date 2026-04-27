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
	"fmt"
	"strings"

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// UserCSVHeaders defines the expected CSV columns for user import
var UserCSVHeaders = []string{
	"email", "name", "phone", "organization", "roles",
}

// UserTemplateExamples provides example rows for the CSV template
var UserTemplateExamples = [][]string{
	{"mario.rossi@acme.it", "Mario Rossi", "+39 333 1234567", "Acme Corp", "Admin"},
	{"support@beta.it", "Support Team", "", "Beta Inc", "Support"},
	{"multi@example.com", "Multi Role User", "", "Acme Corp", "Admin;Support"},
}

// ValidateUserRow validates a single CSV row for user import.
func ValidateUserRow(row map[string]string) []models.ImportFieldError {
	var errs []models.ImportFieldError

	addErr := func(e *models.ImportFieldError) {
		if e != nil {
			errs = append(errs, *e)
		}
	}

	// Required fields
	addErr(ValidateRequired("email", row["email"]))
	addErr(ValidateEmail("email", row["email"]))
	addErr(ValidateMaxLength("email", row["email"], 255))
	addErr(ValidateRequired("name", row["name"]))
	addErr(ValidateMaxLength("name", row["name"], 255))
	addErr(ValidateRequired("organization", row["organization"]))
	addErr(ValidateRequired("roles", row["roles"]))

	// Optional fields
	addErr(ValidatePhone("phone", row["phone"]))

	return errs
}

// OrgMatch represents a matched organization from the unified_organizations view.
type OrgMatch struct {
	LogtoID string
	Name    string
	OrgType string
}

// ResolveOrganizationByName looks up an organization by name, scoped to the allowed org IDs.
// This ensures that users only resolve organizations within their own hierarchy.
// Returns (logto_id, org_type, error). Returns empty strings if not found.
// When multiple matches are found, returns an AmbiguousOrgError containing the candidates.
func ResolveOrganizationByName(name string, allowedOrgIDs []string) (string, string, error) {
	if len(allowedOrgIDs) == 0 {
		return "", "", nil
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(allowedOrgIDs))
	args := make([]interface{}, len(allowedOrgIDs)+1)
	args[0] = strings.TrimSpace(name)
	for i, id := range allowedOrgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = id
	}

	query := fmt.Sprintf(`
		SELECT logto_id, name, org_type FROM unified_organizations
		WHERE LOWER(name) = LOWER($1) AND logto_id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := database.DB.Query(query, args...)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = rows.Close() }()

	var matches []OrgMatch
	for rows.Next() {
		var m OrgMatch
		if err := rows.Scan(&m.LogtoID, &m.Name, &m.OrgType); err != nil {
			return "", "", err
		}
		matches = append(matches, m)
	}

	if len(matches) == 0 {
		return "", "", nil
	}
	if len(matches) > 1 {
		return "", "", &AmbiguousOrgError{Candidates: matches}
	}
	return matches[0].LogtoID, matches[0].OrgType, nil
}

// AmbiguousOrgError is returned when an organization name matches multiple organizations.
type AmbiguousOrgError struct {
	Candidates []OrgMatch
}

func (e *AmbiguousOrgError) Error() string {
	return "ambiguous_name"
}

// ResolveRolesByNames converts semicolon-separated role names to role IDs.
// Returns (roleIDs, invalidNames). Uses the in-memory role cache.
func ResolveRolesByNames(rolesStr string) ([]string, []string) {
	roleCache := cache.GetRoleNames()
	if !roleCache.IsLoaded() {
		return nil, []string{"role cache not available"}
	}

	names := strings.Split(rolesStr, ";")
	var roleIDs []string
	var invalid []string

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		id := roleCache.GetIDByName(name)
		if id == "" {
			invalid = append(invalid, name)
		} else {
			roleIDs = append(roleIDs, id)
		}
	}

	return roleIDs, invalid
}

// CheckUserExistsByEmail checks if a user with the given email exists in the database.
func CheckUserExistsByEmail(email string) (bool, error) {
	id, err := GetUserIDByEmail(email)
	if err != nil {
		return false, err
	}
	return id != "", nil
}

// GetUserIDByEmail returns the user's Logto ID for the given email (case-insensitive match
// against non-deleted users). Returns "" if no user matches. The repository methods that
// consume this value (GetByID, UpdateUser) all key off `logto_id`, despite the misleading
// parameter name.
func GetUserIDByEmail(email string) (string, error) {
	query := `SELECT logto_id FROM users WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL AND logto_id IS NOT NULL LIMIT 1`
	var logtoID string
	err := database.DB.QueryRow(query, strings.TrimSpace(email)).Scan(&logtoID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return logtoID, nil
}

// UserRowToData converts a validated CSV row into the data map stored in ImportRow,
// including resolved organization_id and role_ids.
func UserRowToData(row map[string]string, orgLogtoID string, roleIDs []string) map[string]interface{} {
	data := map[string]interface{}{
		"email":           row["email"],
		"name":            row["name"],
		"phone":           row["phone"],
		"organization":    row["organization"],
		"roles":           row["roles"],
		"organization_id": orgLogtoID,
		"role_ids":        roleIDs,
	}
	return data
}

// UserDataToCreateRequest converts import data to a CreateLocalUserRequest.
func UserDataToCreateRequest(data map[string]interface{}) *models.CreateLocalUserRequest {
	email, _ := data["email"].(string)
	name, _ := data["name"].(string)
	orgID, _ := data["organization_id"].(string)

	var phone *string
	if p, ok := data["phone"].(string); ok && p != "" {
		phone = &p
	}

	var roleIDs []string
	if ids, ok := data["role_ids"].([]interface{}); ok {
		for _, id := range ids {
			if s, ok := id.(string); ok {
				roleIDs = append(roleIDs, s)
			}
		}
	} else if ids, ok := data["role_ids"].([]string); ok {
		roleIDs = ids
	}

	return &models.CreateLocalUserRequest{
		Email:          strings.TrimSpace(email),
		Name:           strings.TrimSpace(name),
		Phone:          phone,
		UserRoleIDs:    roleIDs,
		OrganizationID: &orgID,
	}
}
