/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package response

// LogtoErrorMappings contains all mappings for Logto error codes
type LogtoErrorMappings struct {
	// Maps Logto error codes to our field names
	CodeToField map[string]string
	// Maps Logto error codes to our standard message codes
	CodeToMessage map[string]string
	// Maps Logto field names to our field names
	FieldMapping map[string]string
}

// GetLogtoErrorMappings returns the configured mappings for Logto errors
func GetLogtoErrorMappings() LogtoErrorMappings {
	return LogtoErrorMappings{
		CodeToField: map[string]string{
			// User error codes (from official Logto source)
			"user.username_already_in_use":          "username",
			"user.email_already_in_use":             "email",
			"user.phone_already_in_use":             "phone",
			"user.invalid_email":                    "email",
			"user.invalid_phone":                    "phone",
			"user.email_not_exist":                  "email",
			"user.phone_not_exist":                  "phone",
			"user.identity_not_exist":               "identity",
			"user.identity_already_in_use":          "identity",
			"user.social_account_exists_in_profile": "identity",

			// Organization error codes (from official Logto source)
			"organization.require_membership":   "organizationId",
			"organization.role_names_not_found": "userRoleIds",
		},

		CodeToMessage: map[string]string{
			// User error codes to standard message codes
			"user.username_already_in_use":          "already_exists",
			"user.email_already_in_use":             "already_exists",
			"user.phone_already_in_use":             "already_exists",
			"user.invalid_email":                    "invalid_format",
			"user.invalid_phone":                    "invalid_format",
			"user.email_not_exist":                  "not_found",
			"user.phone_not_exist":                  "not_found",
			"user.identity_not_exist":               "not_found",
			"user.identity_already_in_use":          "already_exists",
			"user.social_account_exists_in_profile": "already_exists",

			// Organization error codes to standard message codes
			"organization.require_membership":   "access_denied",
			"organization.role_names_not_found": "not_found",
		},

		FieldMapping: map[string]string{
			"primaryEmail": "email",
			"primaryPhone": "phone",
			"username":     "username",
			"password":     "password",
			"name":         "name",
		},
	}
}

// MapLogtoCodeToField maps Logto error codes to field names
func MapLogtoCodeToField(code string) string {
	mappings := GetLogtoErrorMappings()

	if field, exists := mappings.CodeToField[code]; exists {
		return field
	}

	// Fallback for unknown codes
	return "general"
}

// MapLogtoCodeToMessage maps Logto error codes to user-friendly message codes
func MapLogtoCodeToMessage(code string) string {
	mappings := GetLogtoErrorMappings()

	if message, exists := mappings.CodeToMessage[code]; exists {
		return message
	}

	// Fallback for unknown codes
	return "error"
}

// MapLogtoFieldToOurs maps Logto field names to our field names
func MapLogtoFieldToOurs(logtoField string) string {
	mappings := GetLogtoErrorMappings()

	if field, exists := mappings.FieldMapping[logtoField]; exists {
		return field
	}

	return logtoField
}
