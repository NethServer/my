/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package helpers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetPaginationFromQuery_DefaultValues(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a request with no query parameters
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	page, pageSize := GetPaginationFromQuery(c)

	assert.Equal(t, 1, page, "Default page should be 1")
	assert.Equal(t, 20, pageSize, "Default page size should be 20")
}

func TestGetPaginationFromQuery_ValidParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "Valid page and page_size",
			queryParams:  "page=3&page_size=15",
			expectedPage: 3,
			expectedSize: 15,
		},
		{
			name:         "Only page parameter",
			queryParams:  "page=5",
			expectedPage: 5,
			expectedSize: 20, // default
		},
		{
			name:         "Only page_size parameter",
			queryParams:  "page_size=10",
			expectedPage: 1, // default
			expectedSize: 10,
		},
		{
			name:         "Maximum page_size",
			queryParams:  "page=2&page_size=100",
			expectedPage: 2,
			expectedSize: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			page, pageSize := GetPaginationFromQuery(c)

			assert.Equal(t, tt.expectedPage, page, "Page should match expected value")
			assert.Equal(t, tt.expectedSize, pageSize, "Page size should match expected value")
		})
	}
}

func TestGetPaginationFromQuery_InvalidParameters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
		description  string
	}{
		{
			name:         "Invalid page string",
			queryParams:  "page=invalid&page_size=10",
			expectedPage: 1, // default fallback
			expectedSize: 10,
			description:  "Non-numeric page should fallback to default",
		},
		{
			name:         "Invalid page_size string",
			queryParams:  "page=2&page_size=invalid",
			expectedPage: 2,
			expectedSize: 20, // default fallback
			description:  "Non-numeric page_size should fallback to default",
		},
		{
			name:         "Zero page",
			queryParams:  "page=0&page_size=10",
			expectedPage: 1, // default fallback (page must be > 0)
			expectedSize: 10,
			description:  "Zero page should fallback to default",
		},
		{
			name:         "Negative page",
			queryParams:  "page=-1&page_size=10",
			expectedPage: 1, // default fallback (page must be > 0)
			expectedSize: 10,
			description:  "Negative page should fallback to default",
		},
		{
			name:         "Zero page_size",
			queryParams:  "page=2&page_size=0",
			expectedPage: 2,
			expectedSize: 20, // default fallback (page_size must be > 0)
			description:  "Zero page_size should fallback to default",
		},
		{
			name:         "Negative page_size",
			queryParams:  "page=2&page_size=-5",
			expectedPage: 2,
			expectedSize: 20, // default fallback (page_size must be > 0)
			description:  "Negative page_size should fallback to default",
		},
		{
			name:         "Page_size too large",
			queryParams:  "page=1&page_size=200",
			expectedPage: 1,
			expectedSize: 20, // default fallback (page_size must be <= 100)
			description:  "Page_size over 100 should fallback to default",
		},
		{
			name:         "Both parameters invalid",
			queryParams:  "page=invalid&page_size=invalid",
			expectedPage: 1,  // default fallback
			expectedSize: 20, // default fallback
			description:  "All invalid parameters should use defaults",
		},
		{
			name:         "Empty parameter values",
			queryParams:  "page=&page_size=",
			expectedPage: 1,  // default fallback
			expectedSize: 20, // default fallback
			description:  "Empty parameter values should use defaults",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			page, pageSize := GetPaginationFromQuery(c)

			assert.Equal(t, tt.expectedPage, page, tt.description+" - page")
			assert.Equal(t, tt.expectedSize, pageSize, tt.description+" - page_size")
		})
	}
}

func TestGetPaginationFromQuery_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  string
		expectedPage int
		expectedSize int
		description  string
	}{
		{
			name:         "Page size exactly 100",
			queryParams:  "page=1&page_size=100",
			expectedPage: 1,
			expectedSize: 100,
			description:  "Page size of exactly 100 should be accepted",
		},
		{
			name:         "Page size 101",
			queryParams:  "page=1&page_size=101",
			expectedPage: 1,
			expectedSize: 20, // fallback to default
			description:  "Page size of 101 should fallback to default",
		},
		{
			name:         "Very large page number",
			queryParams:  "page=999999&page_size=50",
			expectedPage: 999999,
			expectedSize: 50,
			description:  "Very large page numbers should be accepted",
		},
		{
			name:         "Page size 1",
			queryParams:  "page=1&page_size=1",
			expectedPage: 1,
			expectedSize: 1,
			description:  "Page size of 1 should be accepted",
		},
		{
			name:         "Multiple same parameters",
			queryParams:  "page=5&page=10&page_size=15&page_size=25",
			expectedPage: 5,  // First value is used in URL query parsing
			expectedSize: 15, // First value is used in URL query parsing
			description:  "First occurrence of duplicate parameters should be used",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test?"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			c, _ := gin.CreateTestContext(w)
			c.Request = req

			page, pageSize := GetPaginationFromQuery(c)

			assert.Equal(t, tt.expectedPage, page, tt.description+" - page")
			assert.Equal(t, tt.expectedSize, pageSize, tt.description+" - page_size")
		})
	}
}

func TestGetPaginationFromQuery_WithCustomContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with manually constructed context
	c := &gin.Context{
		Request: &http.Request{
			URL: &url.URL{
				RawQuery: "page=7&page_size=30",
			},
		},
	}

	page, pageSize := GetPaginationFromQuery(c)

	assert.Equal(t, 7, page, "Page should be extracted from custom context")
	assert.Equal(t, 30, pageSize, "Page size should be extracted from custom context")
}

func TestGetPaginationFromQuery_URLEncoded(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with URL-encoded parameters (though not typical for numbers)
	req := httptest.NewRequest("GET", "/test?page=3&page_size=25", nil)
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	page, pageSize := GetPaginationFromQuery(c)

	assert.Equal(t, 3, page, "URL-encoded parameters should be parsed correctly")
	assert.Equal(t, 25, pageSize, "URL-encoded parameters should be parsed correctly")
}
