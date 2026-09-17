/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logto

import (
	"testing"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func appWithAccessControl(name string, accessControl map[string]interface{}) models.LogtoThirdPartyApp {
	app := models.LogtoThirdPartyApp{ID: "id-" + name, Name: name, IsThirdParty: true}
	if accessControl != nil {
		app.CustomData = map[string]interface{}{"access_control": accessControl}
	}
	return app
}

func TestIsPartnerAccessible(t *testing.T) {
	tests := []struct {
		name string
		app  models.LogtoThirdPartyApp
		want bool
	}{
		{
			name: "partner org role admitted",
			app: appWithAccessControl("nethshop.nethesis.it", map[string]interface{}{
				"organization_roles": []interface{}{"owner", "distributor", "reseller"},
			}),
			want: true,
		},
		{
			name: "case-insensitive role match",
			app: appWithAccessControl("helpdesk.nethesis.it", map[string]interface{}{
				"organization_roles": []interface{}{"Reseller"},
			}),
			want: true,
		},
		{
			name: "owner only",
			app: appWithAccessControl("stock.nethesis.it", map[string]interface{}{
				"organization_roles": []interface{}{"owner"},
			}),
			want: false,
		},
		{
			name: "pinned to organization ids",
			app: appWithAccessControl("stock.nethesis.it", map[string]interface{}{
				"organization_roles": []interface{}{"owner", "distributor"},
				"organization_ids":   []interface{}{"guwy6opsll17"},
			}),
			want: false,
		},
		{
			name: "no access control at all",
			app:  appWithAccessControl("unknown", nil),
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsPartnerAccessible(tt.app))
		})
	}
}

func TestFilterApplicationsByNames(t *testing.T) {
	apps := []models.LogtoThirdPartyApp{
		appWithAccessControl("nethshop.nethesis.it", nil),
		appWithAccessControl("helpdesk.nethesis.it", nil),
		appWithAccessControl("my.nethspot.com", nil),
	}

	filtered := FilterApplicationsByNames(apps, []string{"my.nethspot.com", "nethshop.nethesis.it", "not-registered"})
	names := make([]string, 0, len(filtered))
	for _, app := range filtered {
		names = append(names, app.Name)
	}
	// Order follows the catalogue, not the allow list; unknown names are ignored.
	assert.Equal(t, []string{"nethshop.nethesis.it", "my.nethspot.com"}, names)

	assert.Empty(t, FilterApplicationsByNames(apps, nil))
	assert.Empty(t, FilterApplicationsByNames(apps, []string{}))
	assert.NotNil(t, FilterApplicationsByNames(apps, []string{}), "an empty result is an empty slice, never nil")
}
