/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

// Package freshdesk talks to the Freshdesk REST API v2 for the helpdesk
// contact sync.
package freshdesk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	pageSize            = 100
	maxRateLimitRetries = 5
)

// sleep is stubbed by the unit tests
var sleep = time.Sleep

// Contact is the part of a Freshdesk contact the sync reads and writes
type Contact struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Client calls the Freshdesk API with an agent API key
type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewClient builds a client for the account at domain (e.g. acme.freshdesk.com)
func NewClient(domain, apiKey string) *Client {
	return &Client{
		baseURL: "https://" + strings.TrimSuffix(domain, "/") + "/api/v2",
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// ListContacts pages through every contact the account lists (Freshdesk
// leaves out the deleted and blocked ones)
func (c *Client) ListContacts() ([]Contact, error) {
	var all []Contact
	for page := 1; ; page++ {
		var items []Contact
		path := fmt.Sprintf("/contacts?per_page=%d&page=%d", pageSize, page)
		if err := c.do(http.MethodGet, path, nil, &items); err != nil {
			return nil, fmt.Errorf("list contacts page %d: %w", page, err)
		}
		all = append(all, items...)
		if len(items) < pageSize {
			return all, nil
		}
	}
}

// FindContactsByEmail returns the contacts with the given primary email
func (c *Client) FindContactsByEmail(email string) ([]Contact, error) {
	var items []Contact
	if err := c.do(http.MethodGet, "/contacts?email="+url.QueryEscape(email), nil, &items); err != nil {
		return nil, fmt.Errorf("find contact %s: %w", email, err)
	}
	return items, nil
}

// UpdateContactName sets the display name of a contact
func (c *Client) UpdateContactName(id int64, name string) error {
	if err := c.do(http.MethodPut, fmt.Sprintf("/contacts/%d", id), map[string]string{"name": name}, nil); err != nil {
		return fmt.Errorf("update contact %d: %w", id, err)
	}
	return nil
}

// do sends one request, waiting out 429s as Retry-After says
func (c *Client) do(method, path string, body, out any) error {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return err
		}
	}

	for attempt := 0; ; attempt++ {
		var reader io.Reader
		if payload != nil {
			reader = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(context.Background(), method, c.baseURL+path, reader)
		if err != nil {
			return err
		}
		req.SetBasicAuth(c.apiKey, "X")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxRateLimitRetries {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			sleep(retryAfter(resp.Header.Get("Retry-After")))
			continue
		}

		return readResponse(resp, out)
	}
}

func readResponse(resp *http.Response, out any) error {
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func retryAfter(header string) time.Duration {
	if seconds, err := strconv.Atoi(header); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return 5 * time.Second
}
