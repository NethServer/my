/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package logto

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// decodeResponse reads the response body, checks the status code against expected values,
// and decodes the JSON body into the target type T.
// Returns a pointer to the decoded value, or an error if the status code is unexpected or decoding fails.
func decodeResponse[T any](resp *http.Response, expectedStatuses []int, operation string) (*T, error) {
	defer func() { _ = resp.Body.Close() }()

	if !isExpectedStatus(resp.StatusCode, expectedStatuses) {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to %s, status %d: %s", operation, resp.StatusCode, string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", operation, err)
	}

	return &result, nil
}

// fetchAllPages retrieves every item from a paginated Management API endpoint.
// Logto returns 20 items per page by default, so any list that can outgrow that
// must page explicitly or it gets silently truncated. A role holding more than
// 20 scopes is the case that matters here: a short read would drop permissions
// from the JWT and surface as intermittent 403s.
func fetchAllPages[T any](c *LogtoManagementClient, endpoint, operation string) ([]T, error) {
	const pageSize = 100

	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}

	var all []T
	for page := 1; ; page++ {
		resp, err := c.makeRequest("GET", fmt.Sprintf("%s%spage=%d&page_size=%d", endpoint, separator, page, pageSize), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to %s: %w", operation, err)
		}

		items, err := decodeSliceResponse[T](resp, []int{http.StatusOK}, operation)
		if err != nil {
			return nil, err
		}

		all = append(all, items...)

		if len(items) < pageSize {
			return all, nil
		}
	}
}

// decodeSliceResponse reads the response body, checks the status code, and decodes
// the JSON body into a slice of type T.
func decodeSliceResponse[T any](resp *http.Response, expectedStatuses []int, operation string) ([]T, error) {
	defer func() { _ = resp.Body.Close() }()

	if !isExpectedStatus(resp.StatusCode, expectedStatuses) {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to %s, status %d: %s", operation, resp.StatusCode, string(body))
	}

	var result []T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", operation, err)
	}

	return result, nil
}

// checkStatus checks the response status code against expected values.
// Used for DELETE and other operations that return no body.
func checkStatus(resp *http.Response, expectedStatuses []int, operation string) error {
	defer func() { _ = resp.Body.Close() }()

	if !isExpectedStatus(resp.StatusCode, expectedStatuses) {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to %s, status %d: %s", operation, resp.StatusCode, string(body))
	}

	return nil
}

func isExpectedStatus(status int, expected []int) bool {
	for _, s := range expected {
		if status == s {
			return true
		}
	}
	return false
}
