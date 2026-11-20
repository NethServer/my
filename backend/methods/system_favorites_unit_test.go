/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestFavoriteUserID(t *testing.T) {
	logtoID := "logto-1"
	empty := ""
	tests := []struct {
		name     string
		user     *models.User
		expected string
	}{
		{"logto id wins over the local id", &models.User{ID: "local-1", LogtoID: &logtoID}, "logto-1"},
		{"owner without local row", &models.User{ID: "logto-1"}, "logto-1"},
		{"empty logto id falls back to the local id", &models.User{ID: "local-1", LogtoID: &empty}, "local-1"},
		{"nil user", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, favoriteUserID(tt.user))
		})
	}
}

func TestApplySystemFavoriteFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logtoID := "logto-1"

	tests := []struct {
		name       string
		url        string
		user       *models.User
		wantOf     string
		wantFilter bool
	}{
		{"caller scopes the flag", "/api/systems", &models.User{ID: "local-1", LogtoID: &logtoID}, "logto-1", false},
		{"favorite=true filters", "/api/systems?favorite=true", &models.User{LogtoID: &logtoID}, "logto-1", true},
		{"any other value does not filter", "/api/systems?favorite=1", &models.User{LogtoID: &logtoID}, "logto-1", false},
		{"no user in context stars nothing", "/api/systems?favorite=true", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", tt.url, nil)
			if tt.user != nil {
				c.Set("user", tt.user)
			}

			var f models.SystemListFilters
			applySystemFavoriteFilters(c, &f)

			assert.Equal(t, tt.wantOf, f.FavoritesOf)
			assert.Equal(t, tt.wantFilter, f.FavoritesOnly)
		})
	}
}
