/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package freshdesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := NewClient("acme.freshdesk.com", "secret-key")
	client.baseURL = server.URL + "/api/v2"
	return client
}

func TestListContactsPaginates(t *testing.T) {
	var pages []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		require.True(t, ok)
		assert.Equal(t, "secret-key", user)
		assert.Equal(t, "X", pass)
		assert.Equal(t, "/api/v2/contacts", r.URL.Path)
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))
		page := r.URL.Query().Get("page")
		pages = append(pages, page)

		count := pageSize
		if page == "2" {
			count = 3
		}
		contacts := make([]Contact, 0, count)
		for i := 0; i < count; i++ {
			contacts = append(contacts, Contact{ID: int64(i), Name: "n", Email: fmt.Sprintf("u%s-%d@example.com", page, i)})
		}
		_ = json.NewEncoder(w).Encode(contacts)
	})

	contacts, err := client.ListContacts()
	require.NoError(t, err)
	assert.Len(t, contacts, pageSize+3)
	assert.Equal(t, []string{"1", "2"}, pages)
}

func TestFindContactsByEmail(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/contacts", r.URL.Path)
		assert.Equal(t, "mario+test@example.com", r.URL.Query().Get("email"))
		_, _ = w.Write([]byte(`[{"id":7,"name":"mario_test","email":"mario+test@example.com"}]`))
	})

	contacts, err := client.FindContactsByEmail("mario+test@example.com")
	require.NoError(t, err)
	assert.Equal(t, []Contact{{ID: 7, Name: "mario_test", Email: "mario+test@example.com"}}, contacts)
}

func TestUpdateContactName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		assert.Equal(t, "/api/v2/contacts/7", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var body map[string]string
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, map[string]string{"name": "Mario Rossi"}, body)
		_, _ = w.Write([]byte(`{"id":7,"name":"Mario Rossi"}`))
	})

	require.NoError(t, client.UpdateContactName(7, "Mario Rossi"))
}

func TestUpdateContactNameError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"message":"deleted contact"}`))
	})

	err := client.UpdateContactName(7, "Mario Rossi")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 405")
	assert.Contains(t, err.Error(), "deleted contact")
}

func TestRateLimitRetry(t *testing.T) {
	var waited []time.Duration
	sleep = func(d time.Duration) { waited = append(waited, d) }
	t.Cleanup(func() { sleep = time.Sleep })

	calls := 0
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`[]`))
	})

	contacts, err := client.ListContacts()
	require.NoError(t, err)
	assert.Empty(t, contacts)
	assert.Equal(t, 2, calls)
	assert.Equal(t, []time.Duration{7 * time.Second}, waited)
}

func TestRateLimitGivesUp(t *testing.T) {
	sleep = func(time.Duration) {}
	t.Cleanup(func() { sleep = time.Sleep })

	calls := 0
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusTooManyRequests)
	})

	_, err := client.ListContacts()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 429")
	assert.Equal(t, maxRateLimitRetries+1, calls)
}
