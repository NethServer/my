/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestApplyUnifiedStatus(t *testing.T) {
	stamp := time.Date(2026, 2, 4, 9, 12, 0, 0, time.UTC)
	service := &LocalSystemsService{}

	tests := []struct {
		name   string
		system models.System
		want   string
	}{
		{
			name:   "heartbeat status survives when no lifecycle flag is set",
			system: models.System{Status: "active"},
			want:   "active",
		},
		{
			name:   "suspension hides the heartbeat status",
			system: models.System{Status: "active", SuspendedAt: &stamp},
			want:   "suspended",
		},
		{
			name:   "unregistration hides the heartbeat status",
			system: models.System{Status: "inactive", UnregisteredAt: &stamp},
			want:   "unregistered",
		},
		{
			// Reactivating the organization cannot bring an unregistered
			// machine back, so it must not read as merely suspended.
			name:   "unregistration outranks a cascade suspension",
			system: models.System{Status: "active", UnregisteredAt: &stamp, SuspendedAt: &stamp},
			want:   "unregistered",
		},
		{
			// The frontend offers restore on deleted and reactivate on
			// suspended; a deleted row must get the first.
			name:   "deletion outranks every other state",
			system: models.System{Status: "active", DeletedAt: &stamp, UnregisteredAt: &stamp, SuspendedAt: &stamp},
			want:   "deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			system := tc.system
			service.applyUnifiedStatus(&system)
			assert.Equal(t, tc.want, system.Status)
		})
	}
}
