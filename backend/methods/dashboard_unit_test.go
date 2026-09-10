/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package methods

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/models"
)

func dashboardContext(t *testing.T, body string, user *models.User) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/dashboard/generate", bytes.NewBufferString(body))
	c.Request.Header.Set("Content-Type", "application/json")
	if user != nil {
		c.Set("user", user)
	}
	return c, rec
}

func dashboardUser() *models.User {
	return &models.User{
		ID:              "u1",
		OrganizationID:  "org1",
		OrgRole:         "customer",
		UserRoles:       []string{"Admin"},
		UserPermissions: []string{"read:systems", "read:applications"},
	}
}

// stubGenerate swaps the service out so the handler test never builds a model
// client, and restores it afterwards.
func stubGenerate(t *testing.T, resp models.DashboardGenerateResponse) *models.DashboardGenerateRequest {
	t.Helper()
	var captured models.DashboardGenerateRequest

	original := generateDashboard
	generateDashboard = func(_ context.Context, _ *models.User, req models.DashboardGenerateRequest) models.DashboardGenerateResponse {
		captured = req
		return resp
	}
	t.Cleanup(func() { generateDashboard = original })

	return &captured
}

func TestGenerateDashboardReturnsTheSelection(t *testing.T) {
	captured := stubGenerate(t, models.DashboardGenerateResponse{
		Title:       "Operations board",
		GeneratedBy: "ai",
		Model:       "gemini-test",
		Widgets:     []models.DashboardWidget{{ID: "alerts_counter", Size: "sm"}},
	})

	c, rec := dashboardContext(t, `{"topics":["alerts","systems"],"hint":"noisy sites","locale":"it"}`, dashboardUser())
	GenerateDashboard(c)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, []string{"alerts", "systems"}, captured.Topics)
	assert.Equal(t, "noisy sites", captured.Hint)
	assert.Equal(t, "it", captured.Locale)

	var body struct {
		Code    int                              `json:"code"`
		Message string                           `json:"message"`
		Data    models.DashboardGenerateResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 200, body.Code)
	assert.Equal(t, "dashboard generated successfully", body.Message)
	assert.Equal(t, "ai", body.Data.GeneratedBy)
	require.Len(t, body.Data.Widgets, 1)
	assert.Equal(t, "alerts_counter", body.Data.Widgets[0].ID)
}

func TestGenerateDashboardRejectsInvalidRequests(t *testing.T) {
	stubGenerate(t, models.DashboardGenerateResponse{})

	tests := []struct {
		name string
		body string
	}{
		{"no topics", `{"topics":[]}`},
		{"missing topics", `{}`},
		{"unknown topic", `{"topics":["finances"]}`},
		{"unknown locale", `{"topics":["systems"],"locale":"de"}`},
		{"malformed json", `{"topics":`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := dashboardContext(t, tt.body, dashboardUser())
			GenerateDashboard(c)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestGenerateDashboardRejectsAnOversizeHint(t *testing.T) {
	stubGenerate(t, models.DashboardGenerateResponse{})

	hint := make([]byte, 400)
	for i := range hint {
		hint[i] = 'a'
	}
	body, err := json.Marshal(map[string]any{"topics": []string{"systems"}, "hint": string(hint)})
	require.NoError(t, err)

	c, rec := dashboardContext(t, string(body), dashboardUser())
	GenerateDashboard(c)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGenerateDashboardWithoutAUserInContext(t *testing.T) {
	stubGenerate(t, models.DashboardGenerateResponse{})

	c, rec := dashboardContext(t, `{"topics":["systems"]}`, nil)
	GenerateDashboard(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestGetDashboardCatalogFiltersByPermission(t *testing.T) {
	c, rec := dashboardContext(t, "", dashboardUser())
	GetDashboardCatalog(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Data models.DashboardCatalogResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))

	assert.NotEmpty(t, body.Data.Widgets)
	assert.Contains(t, body.Data.Topics, "alerts")
	assert.NotContains(t, body.Data.Topics, "organizations",
		"a persona without read:distributors|resellers|customers is never offered the topic")
	for _, widget := range body.Data.Widgets {
		assert.NotEqual(t, "users_counter", widget.ID)
	}
}

func TestGetDashboardCatalogWithoutAUserInContext(t *testing.T) {
	c, rec := dashboardContext(t, "", nil)
	GetDashboardCatalog(c)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
