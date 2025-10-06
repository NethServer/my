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
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetPaginationFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		queryParams  map[string]string
		expectedPage int
		expectedSize int
	}{
		{
			name:         "default values when no query params",
			queryParams:  map[string]string{},
			expectedPage: 1,
			expectedSize: 20,
		},
		{
			name: "valid page and page_size",
			queryParams: map[string]string{
				"page":      "3",
				"page_size": "50",
			},
			expectedPage: 3,
			expectedSize: 50,
		},
		{
			name: "invalid page defaults to 1",
			queryParams: map[string]string{
				"page":      "invalid",
				"page_size": "25",
			},
			expectedPage: 1,
			expectedSize: 25,
		},
		{
			name: "zero page defaults to 1",
			queryParams: map[string]string{
				"page":      "0",
				"page_size": "25",
			},
			expectedPage: 1,
			expectedSize: 25,
		},
		{
			name: "negative page defaults to 1",
			queryParams: map[string]string{
				"page":      "-1",
				"page_size": "25",
			},
			expectedPage: 1,
			expectedSize: 25,
		},
		{
			name: "invalid page_size defaults to 20",
			queryParams: map[string]string{
				"page":      "2",
				"page_size": "invalid",
			},
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name: "zero page_size defaults to 20",
			queryParams: map[string]string{
				"page":      "2",
				"page_size": "0",
			},
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name: "negative page_size defaults to 20",
			queryParams: map[string]string{
				"page":      "2",
				"page_size": "-10",
			},
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name: "page_size too large defaults to 20",
			queryParams: map[string]string{
				"page":      "2",
				"page_size": "200",
			},
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name: "maximum valid page_size",
			queryParams: map[string]string{
				"page":      "1",
				"page_size": "100",
			},
			expectedPage: 1,
			expectedSize: 100,
		},
		{
			name: "minimum valid page and page_size",
			queryParams: map[string]string{
				"page":      "1",
				"page_size": "1",
			},
			expectedPage: 1,
			expectedSize: 1,
		},
		{
			name: "large page number",
			queryParams: map[string]string{
				"page":      "9999",
				"page_size": "10",
			},
			expectedPage: 9999,
			expectedSize: 10,
		},
		{
			name: "empty string values",
			queryParams: map[string]string{
				"page":      "",
				"page_size": "",
			},
			expectedPage: 1,
			expectedSize: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up request with query parameters
			req := httptest.NewRequest("GET", "/test", nil)
			q := url.Values{}
			for key, value := range tt.queryParams {
				q.Set(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			// Test the function
			page, pageSize := GetPaginationFromQuery(c)

			assert.Equal(t, tt.expectedPage, page, "Page should match expected value")
			assert.Equal(t, tt.expectedSize, pageSize, "Page size should match expected value")
		})
	}
}

func TestGetSortingFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		queryParams       map[string]string
		expectedSortBy    string
		expectedDirection string
	}{
		{
			name:              "default values when no query params",
			queryParams:       map[string]string{},
			expectedSortBy:    "",
			expectedDirection: "asc",
		},
		{
			name: "valid sort_by and sort_direction",
			queryParams: map[string]string{
				"sort_by":        "name",
				"sort_direction": "desc",
			},
			expectedSortBy:    "name",
			expectedDirection: "desc",
		},
		{
			name: "valid ascending direction",
			queryParams: map[string]string{
				"sort_by":        "created_at",
				"sort_direction": "asc",
			},
			expectedSortBy:    "created_at",
			expectedDirection: "asc",
		},
		{
			name: "invalid sort_direction defaults to asc",
			queryParams: map[string]string{
				"sort_by":        "name",
				"sort_direction": "invalid",
			},
			expectedSortBy:    "name",
			expectedDirection: "asc",
		},
		{
			name: "empty sort_direction defaults to asc",
			queryParams: map[string]string{
				"sort_by":        "name",
				"sort_direction": "",
			},
			expectedSortBy:    "name",
			expectedDirection: "asc",
		},
		{
			name: "only sort_by provided",
			queryParams: map[string]string{
				"sort_by": "email",
			},
			expectedSortBy:    "email",
			expectedDirection: "asc",
		},
		{
			name: "only sort_direction provided",
			queryParams: map[string]string{
				"sort_direction": "desc",
			},
			expectedSortBy:    "",
			expectedDirection: "desc",
		},
		{
			name: "case sensitive sort_direction - uppercase DESC invalid",
			queryParams: map[string]string{
				"sort_by":        "name",
				"sort_direction": "DESC",
			},
			expectedSortBy:    "name",
			expectedDirection: "asc", // Should default to asc since DESC is not lowercase
		},
		{
			name: "case sensitive sort_direction - uppercase ASC invalid",
			queryParams: map[string]string{
				"sort_by":        "name",
				"sort_direction": "ASC",
			},
			expectedSortBy:    "name",
			expectedDirection: "asc", // Should default to asc since ASC is not lowercase
		},
		{
			name: "complex sort_by field name",
			queryParams: map[string]string{
				"sort_by":        "user.created_at",
				"sort_direction": "desc",
			},
			expectedSortBy:    "user.created_at",
			expectedDirection: "desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up request with query parameters
			req := httptest.NewRequest("GET", "/test", nil)
			q := url.Values{}
			for key, value := range tt.queryParams {
				q.Set(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			// Test the function
			sortBy, sortDirection := GetSortingFromQuery(c)

			assert.Equal(t, tt.expectedSortBy, sortBy, "Sort by should match expected value")
			assert.Equal(t, tt.expectedDirection, sortDirection, "Sort direction should match expected value")
		})
	}
}

func TestGetPaginationAndSortingFromQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		queryParams       map[string]string
		expectedPage      int
		expectedSize      int
		expectedSortBy    string
		expectedDirection string
	}{
		{
			name:              "all default values",
			queryParams:       map[string]string{},
			expectedPage:      1,
			expectedSize:      20,
			expectedSortBy:    "",
			expectedDirection: "asc",
		},
		{
			name: "all valid parameters",
			queryParams: map[string]string{
				"page":           "2",
				"page_size":      "50",
				"sort_by":        "name",
				"sort_direction": "desc",
			},
			expectedPage:      2,
			expectedSize:      50,
			expectedSortBy:    "name",
			expectedDirection: "desc",
		},
		{
			name: "mixed valid and invalid parameters",
			queryParams: map[string]string{
				"page":           "invalid", // Should default to 1
				"page_size":      "25",      // Valid
				"sort_by":        "email",   // Valid
				"sort_direction": "invalid", // Should default to asc
			},
			expectedPage:      1,
			expectedSize:      25,
			expectedSortBy:    "email",
			expectedDirection: "asc",
		},
		{
			name: "pagination only",
			queryParams: map[string]string{
				"page":      "3",
				"page_size": "15",
			},
			expectedPage:      3,
			expectedSize:      15,
			expectedSortBy:    "",
			expectedDirection: "asc",
		},
		{
			name: "sorting only",
			queryParams: map[string]string{
				"sort_by":        "created_at",
				"sort_direction": "desc",
			},
			expectedPage:      1,
			expectedSize:      20,
			expectedSortBy:    "created_at",
			expectedDirection: "desc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up request with query parameters
			req := httptest.NewRequest("GET", "/test", nil)
			q := url.Values{}
			for key, value := range tt.queryParams {
				q.Set(key, value)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			// Test the function
			page, pageSize, sortBy, sortDirection := GetPaginationAndSortingFromQuery(c)

			assert.Equal(t, tt.expectedPage, page, "Page should match expected value")
			assert.Equal(t, tt.expectedSize, pageSize, "Page size should match expected value")
			assert.Equal(t, tt.expectedSortBy, sortBy, "Sort by should match expected value")
			assert.Equal(t, tt.expectedDirection, sortDirection, "Sort direction should match expected value")
		})
	}
}

func TestPaginationHelpers_EdgeCases(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		queryString string
		testFunc    func(*gin.Context)
	}{
		{
			name:        "very long query string",
			queryString: "page=1&page_size=20&sort_by=" + strings.Repeat("a", 1000) + "&sort_direction=asc",
			testFunc: func(c *gin.Context) {
				page, pageSize := GetPaginationFromQuery(c)
				assert.Equal(t, 1, page)
				assert.Equal(t, 20, pageSize)
			},
		},
		{
			name:        "query with special characters",
			queryString: "page=1&page_size=20&sort_by=user%2Ename&sort_direction=desc",
			testFunc: func(c *gin.Context) {
				sortBy, sortDirection := GetSortingFromQuery(c)
				assert.Equal(t, "user.name", sortBy) // URL decoded
				assert.Equal(t, "desc", sortDirection)
			},
		},
		{
			name:        "duplicate query parameters",
			queryString: "page=1&page=2&page_size=10&page_size=30",
			testFunc: func(c *gin.Context) {
				// Gin should use the first value
				page, pageSize := GetPaginationFromQuery(c)
				assert.Equal(t, 1, page)      // First page value
				assert.Equal(t, 10, pageSize) // First page_size value
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest("GET", "/test?"+tt.queryString, nil)
			c.Request = req

			tt.testFunc(c)
		})
	}
}

func TestPaginationHelpers_NilContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test with nil request (should not panic)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = nil

	// These should not panic even with nil request
	assert.NotPanics(t, func() {
		page, pageSize := GetPaginationFromQuery(c)
		assert.Equal(t, 1, page)      // Default values
		assert.Equal(t, 20, pageSize) // Default values
	})

	assert.NotPanics(t, func() {
		sortBy, sortDirection := GetSortingFromQuery(c)
		assert.Equal(t, "", sortBy)           // Default values
		assert.Equal(t, "asc", sortDirection) // Default values
	})
}

func TestPaginationHelpers_Performance(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test performance with many iterations
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := httptest.NewRequest("GET", "/test?page=5&page_size=25&sort_by=name&sort_direction=desc", nil)
	c.Request = req

	const iterations = 1000
	for i := 0; i < iterations; i++ {
		page, pageSize, sortBy, sortDirection := GetPaginationAndSortingFromQuery(c)
		assert.Equal(t, 5, page)
		assert.Equal(t, 25, pageSize)
		assert.Equal(t, "name", sortBy)
		assert.Equal(t, "desc", sortDirection)
	}
}
