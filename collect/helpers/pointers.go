/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

// StrPtr returns a pointer to a string
func StrPtr(s string) *string {
	return &s
}

// NilIfEmpty returns nil if the string is empty, otherwise returns a pointer to the string
func NilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// DerefString returns the value of a string pointer or empty string if nil
func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
