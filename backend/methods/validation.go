/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package methods

import (
	"errors"

	"github.com/nethesis/my/backend/services/local"
)

// getValidationError checks if the error chain contains a ValidationError and returns it.
// Used across all entity handlers (users, distributors, resellers, customers) to convert
// service-layer validation errors into proper HTTP responses.
func getValidationError(err error) *local.ValidationError {
	var validationErr *local.ValidationError
	if errors.As(err, &validationErr) {
		return validationErr
	}
	return nil
}
