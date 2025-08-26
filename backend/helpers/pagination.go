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
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetPaginationFromQuery extracts page and pageSize from query parameters
func GetPaginationFromQuery(c *gin.Context) (int, int) {
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20 // Default page size
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	return page, pageSize
}

// GetSortingFromQuery extracts sortBy and sortDirection from query parameters
func GetSortingFromQuery(c *gin.Context) (string, string) {
	sortBy := c.Query("sort_by")

	sortDirection := "asc" // Default sort direction
	if sortDir := c.Query("sort_direction"); sortDir != "" {
		if sortDir == "desc" || sortDir == "asc" {
			sortDirection = sortDir
		}
	}

	return sortBy, sortDirection
}

// GetPaginationAndSortingFromQuery extracts page, pageSize, sortBy and sortDirection from query parameters
func GetPaginationAndSortingFromQuery(c *gin.Context) (int, int, string, string) {
	page, pageSize := GetPaginationFromQuery(c)
	sortBy, sortDirection := GetSortingFromQuery(c)

	return page, pageSize, sortBy, sortDirection
}
