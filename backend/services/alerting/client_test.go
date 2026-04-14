/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSilence(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	expected := models.AlertmanagerSilenceRequest{
		Matchers: []models.AlertmanagerMatcher{
			{Name: "alertname", Value: "HostDown", IsRegex: false},
			{Name: "system_key", Value: "system-1", IsRegex: false},
		},
		StartsAt:  "2026-04-14T10:00:00Z",
		EndsAt:    "2026-04-14T11:00:00Z",
		Comment:   "silenced from my",
		CreatedBy: "admin@example.com",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/alertmanager/api/v2/silences", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "org-1", r.Header.Get("X-Scope-OrgID"))

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var got models.AlertmanagerSilenceRequest
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, expected, got)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"silenceID":"silence-1"}`))
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	resp, err := CreateSilence("org-1", &expected)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "silence-1", resp.SilenceID)
}

func TestCreateSilenceReturnsMimirError(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid silence", http.StatusBadRequest)
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	_, err := CreateSilence("org-1", &models.AlertmanagerSilenceRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mimir returned 400")
}
