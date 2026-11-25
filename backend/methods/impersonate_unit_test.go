/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

func TestIsImpersonationTarget(t *testing.T) {
	callerLogto := "caller-logto"
	otherLogto := "other-logto"
	caller := &models.User{ID: "caller", LogtoID: &callerLogto}
	now := time.Now()

	tests := []struct {
		name    string
		account models.LocalUser
		want    bool
	}{
		{"an active user", models.LocalUser{ID: "u1", LogtoID: &otherLogto, Roles: []models.UserRole{{Name: "Admin"}}}, true},
		{"the caller, by id", models.LocalUser{ID: "caller"}, false},
		{"the caller, by Logto id", models.LocalUser{ID: "u2", LogtoID: &callerLogto}, false},
		{"a suspended user", models.LocalUser{ID: "u3", SuspendedAt: &now}, false},
		{"a deleted user", models.LocalUser{ID: "u4", DeletedAt: &now}, false},
		{"an Owner-role account", models.LocalUser{ID: "u5", Roles: []models.UserRole{{Name: "Owner"}}}, false},
		{"an Owner-role account, any case", models.LocalUser{ID: "u6", Roles: []models.UserRole{{Name: "owner"}}}, false},
		{"a Staff user", models.LocalUser{ID: "u7", Roles: []models.UserRole{{Name: "Staff"}}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isImpersonationTarget(caller, &tt.account))
		})
	}
}

func TestExtendImpersonation_RefusesWithoutAnImpersonation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name  string
		setup func(c *gin.Context)
		want  int
	}{
		{"a regular token", func(c *gin.Context) {}, http.StatusBadRequest},
		{"an impersonation token without a session", func(c *gin.Context) {
			c.Set("is_impersonated", true)
			c.Set("impersonated_by", &models.User{ID: "staff"})
			c.Set("user", &models.User{ID: "target"})
		}, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/impersonate/extend", nil)
			tt.setup(c)

			ExtendImpersonation(c)

			assert.Equal(t, tt.want, w.Code)
		})
	}
}
