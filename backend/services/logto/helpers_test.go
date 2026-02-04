/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package logto

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// newMockResponse creates an http.Response with the given status code and body string.
func newMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// --- isExpectedStatus ---

func TestIsExpectedStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		expected []int
		want     bool
	}{
		{"match single", 200, []int{200}, true},
		{"match first of many", 200, []int{200, 201}, true},
		{"match second of many", 201, []int{200, 201}, true},
		{"no match", 404, []int{200, 201}, false},
		{"empty expected list", 200, []int{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExpectedStatus(tt.status, tt.expected)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- checkStatus ---

func TestCheckStatus_Success(t *testing.T) {
	resp := newMockResponse(http.StatusNoContent, "")
	err := checkStatus(resp, []int{http.StatusNoContent, http.StatusOK}, "delete user")
	assert.NoError(t, err)
}

func TestCheckStatus_UnexpectedStatus(t *testing.T) {
	resp := newMockResponse(http.StatusNotFound, `{"message":"not found"}`)
	err := checkStatus(resp, []int{http.StatusNoContent, http.StatusOK}, "delete user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete user, status 404")
	assert.Contains(t, err.Error(), "not found")
}

func TestCheckStatus_ServerError(t *testing.T) {
	resp := newMockResponse(http.StatusInternalServerError, "internal error")
	err := checkStatus(resp, []int{http.StatusOK}, "update user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

// --- decodeResponse ---

type testUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func TestDecodeResponse_Success(t *testing.T) {
	resp := newMockResponse(http.StatusOK, `{"id":"user-1","name":"Test User"}`)

	result, err := decodeResponse[testUser](resp, []int{http.StatusOK}, "fetch user")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user-1", result.ID)
	assert.Equal(t, "Test User", result.Name)
}

func TestDecodeResponse_UnexpectedStatus(t *testing.T) {
	resp := newMockResponse(http.StatusBadRequest, `{"error":"bad request"}`)

	result, err := decodeResponse[testUser](resp, []int{http.StatusOK}, "fetch user")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch user, status 400")
}

func TestDecodeResponse_InvalidJSON(t *testing.T) {
	resp := newMockResponse(http.StatusOK, `{invalid json`)

	result, err := decodeResponse[testUser](resp, []int{http.StatusOK}, "fetch user")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode fetch user response")
}

func TestDecodeResponse_EmptyBody(t *testing.T) {
	resp := newMockResponse(http.StatusOK, "")

	result, err := decodeResponse[testUser](resp, []int{http.StatusOK}, "fetch user")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestDecodeResponse_MultipleExpectedStatuses(t *testing.T) {
	resp := newMockResponse(http.StatusCreated, `{"id":"user-2","name":"New User"}`)

	result, err := decodeResponse[testUser](resp, []int{http.StatusOK, http.StatusCreated}, "create user")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "user-2", result.ID)
}

// --- decodeSliceResponse ---

func TestDecodeSliceResponse_Success(t *testing.T) {
	resp := newMockResponse(http.StatusOK, `[{"id":"u1","name":"User 1"},{"id":"u2","name":"User 2"}]`)

	result, err := decodeSliceResponse[testUser](resp, []int{http.StatusOK}, "fetch users")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "u1", result[0].ID)
	assert.Equal(t, "u2", result[1].ID)
}

func TestDecodeSliceResponse_EmptyArray(t *testing.T) {
	resp := newMockResponse(http.StatusOK, `[]`)

	result, err := decodeSliceResponse[testUser](resp, []int{http.StatusOK}, "fetch users")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestDecodeSliceResponse_UnexpectedStatus(t *testing.T) {
	resp := newMockResponse(http.StatusForbidden, `{"error":"forbidden"}`)

	result, err := decodeSliceResponse[testUser](resp, []int{http.StatusOK}, "fetch users")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to fetch users, status 403")
}

func TestDecodeSliceResponse_InvalidJSON(t *testing.T) {
	resp := newMockResponse(http.StatusOK, `not json`)

	result, err := decodeSliceResponse[testUser](resp, []int{http.StatusOK}, "fetch users")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode fetch users response")
}
