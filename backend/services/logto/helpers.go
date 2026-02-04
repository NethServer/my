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
