/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/local"
)

func TestAPIKeyAuditReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"revoked", local.ErrAPIKeyRevoked, models.APIKeyReasonRevoked},
		{"expired", local.ErrAPIKeyExpired, models.APIKeyReasonExpired},
		{"user inactive", local.ErrAPIKeyUserInactive, models.APIKeyReasonUserInactive},
		{"invalid secret", local.ErrAPIKeyInvalid, models.APIKeyReasonInvalidSecret},
		{"unknown maps to invalid secret", errors.New("boom"), models.APIKeyReasonInvalidSecret},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, apiKeyAuditReason(tc.err))
		})
	}
}
