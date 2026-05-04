/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"encoding/json"
	"errors"
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
			{Name: "alertname", Value: "LinkFailed", IsRegex: false},
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

func TestGetSilence(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/alertmanager/api/v2/silence/silence-1", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "org-1", r.Header.Get("X-Scope-OrgID"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"silence-1","matchers":[{"name":"system_key","value":"system-1","isRegex":false}]}`))
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	resp, err := GetSilence("org-1", "silence-1")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "silence-1", resp.ID)
	assert.Equal(t, []models.AlertmanagerMatcher{
		{Name: "system_key", Value: "system-1", IsRegex: false},
	}, resp.Matchers)
}

func TestGetSilenceReturnsNotFound(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	_, err := GetSilence("org-1", "missing")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSilenceNotFound)
}

func TestDeleteSilence(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/alertmanager/api/v2/silence/silence-1", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "org-1", r.Header.Get("X-Scope-OrgID"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	require.NoError(t, DeleteSilence("org-1", "silence-1"))
}

func TestDeleteSilenceReturnsNotFound(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	err := DeleteSilence("org-1", "missing")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSilenceNotFound))
}

func TestCleanupSystemSilencesDeletesOnlyExactMatches(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	silences := []models.AlertmanagerSilence{
		{
			ID: "match-active",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-1", IsRegex: false},
			},
			Status: &models.AlertmanagerSilenceStatus{State: "active"},
		},
		{
			ID: "match-expired",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-1", IsRegex: false},
			},
			Status: &models.AlertmanagerSilenceStatus{State: "expired"},
		},
		{
			ID: "different-system",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-2", IsRegex: false},
			},
			Status: &models.AlertmanagerSilenceStatus{State: "active"},
		},
		{
			ID: "regex-broader",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-.*", IsRegex: true},
			},
			Status: &models.AlertmanagerSilenceStatus{State: "active"},
		},
	}

	deleted := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "org-1", r.Header.Get("X-Scope-OrgID"))
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/alertmanager/api/v2/silences":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(silences)
		case r.Method == http.MethodDelete:
			id := r.URL.Path[len("/alertmanager/api/v2/silence/"):]
			deleted[id] = true
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	count, err := CleanupSystemSilences("org-1", "system-1")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, map[string]bool{"match-active": true}, deleted)
}

func TestCleanupSystemSilencesContinuesOnPartialFailure(t *testing.T) {
	oldClient := httpClient
	oldMimirURL := configuration.Config.MimirURL
	defer func() {
		httpClient = oldClient
		configuration.Config.MimirURL = oldMimirURL
	}()

	silences := []models.AlertmanagerSilence{
		{
			ID: "ok-1",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-1", IsRegex: false},
			},
		},
		{
			ID: "fail",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-1", IsRegex: false},
			},
		},
		{
			ID: "ok-2",
			Matchers: []models.AlertmanagerMatcher{
				{Name: "system_key", Value: "system-1", IsRegex: false},
			},
		},
	}

	deleted := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(silences)
		case http.MethodDelete:
			id := r.URL.Path[len("/alertmanager/api/v2/silence/"):]
			if id == "fail" {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			deleted[id] = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	httpClient = server.Client()
	configuration.Config.MimirURL = server.URL

	count, err := CleanupSystemSilences("org-1", "system-1")
	require.Error(t, err)
	assert.Equal(t, 2, count)
	assert.True(t, deleted["ok-1"])
	assert.True(t, deleted["ok-2"])
}
