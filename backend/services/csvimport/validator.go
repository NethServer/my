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
	"net/mail"
	"regexp"
	"strings"

	"github.com/nethesis/my/backend/models"
)

// Phone numbers in CSV imports must include the international "+CC" prefix.
// The single-user create/edit UI handles country selection in a dropdown and
// always submits a "+CC ..." value, so it stays compatible. CSV authors must
// write "+39 333 1234567" (or any other country code) — a leading bare local
// number like "333 1234567" is rejected with `invalid_phone` at validate time.
var phoneRegex = regexp.MustCompile(`^\+[\d\s\-\(\)]{7,20}$`)

// ValidateRequired checks that the field value is not empty.
func ValidateRequired(field, value string) *models.ImportFieldError {
	if strings.TrimSpace(value) == "" {
		return &models.ImportFieldError{
			Field:   field,
			Message: "required",
		}
	}
	return nil
}

// ValidateMaxLength checks that the value does not exceed maxLen characters.
func ValidateMaxLength(field, value string, maxLen int) *models.ImportFieldError {
	if len(value) > maxLen {
		return &models.ImportFieldError{
			Field:   field,
			Message: "too_long",
			Value:   value,
		}
	}
	return nil
}

// ValidateEmail checks that the value is a valid email address.
func ValidateEmail(field, value string) *models.ImportFieldError {
	if value == "" {
		return nil // optional field, skip if empty
	}
	_, err := mail.ParseAddress(value)
	if err != nil {
		return &models.ImportFieldError{
			Field:   field,
			Message: "invalid_email",
			Value:   value,
		}
	}
	return nil
}

// ValidatePhone checks that the value matches the phone regex.
func ValidatePhone(field, value string) *models.ImportFieldError {
	if value == "" {
		return nil // optional field, skip if empty
	}
	if !phoneRegex.MatchString(value) {
		return &models.ImportFieldError{
			Field:   field,
			Message: "invalid_phone",
			Value:   value,
		}
	}
	return nil
}

// ValidateLanguage checks that the value is a supported language code.
func ValidateLanguage(field, value string) *models.ImportFieldError {
	if value == "" {
		return nil // optional, defaults to "it"
	}
	switch strings.ToLower(value) {
	case "it", "en":
		return nil
	default:
		return &models.ImportFieldError{
			Field:   field,
			Message: "invalid_language",
			Value:   value,
		}
	}
}

// ValidateInSet checks that the value is one of the allowed values (case-insensitive).
func ValidateInSet(field, value string, allowed []string) *models.ImportFieldError {
	if value == "" {
		return nil
	}
	lower := strings.ToLower(value)
	for _, a := range allowed {
		if strings.ToLower(a) == lower {
			return nil
		}
	}
	return &models.ImportFieldError{
		Field:   field,
		Message: "invalid_value",
		Value:   value,
	}
}

// CheckDuplicateInSet checks if the value already exists in the seen set.
// If it does, returns a duplicate error. Otherwise, adds it to the set.
func CheckDuplicateInSet(field, value string, seen map[string]int, currentRow int) *models.ImportFieldError {
	key := strings.ToLower(strings.TrimSpace(value))
	if key == "" {
		return nil
	}
	if firstRow, exists := seen[key]; exists {
		return &models.ImportFieldError{
			Field:   field,
			Message: "duplicate_in_csv",
			Value:   value + " (same as row " + strings.TrimSpace(intToStr(firstRow)) + ")",
		}
	}
	seen[key] = currentRow
	return nil
}

// intToStr is a simple helper to avoid importing strconv for one usage
func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
