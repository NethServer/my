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
	"github.com/nethesis/my/backend/models"
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

// CalculateTotalPages calculates the total number of pages based on total count and page size
func CalculateTotalPages(totalCount int, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	if totalCount == 0 {
		return 0
	}
	return (totalCount + pageSize - 1) / pageSize
}

// BuildPaginationInfo creates a standard pagination info object
func BuildPaginationInfo(page, pageSize, totalCount int) models.PaginationInfo {
	totalPages := CalculateTotalPages(totalCount, pageSize)
	hasNext := page < totalPages
	hasPrev := page > 1

	paginationInfo := models.PaginationInfo{
		Page:       page,
		PageSize:   pageSize,
		TotalCount: totalCount,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	if hasNext {
		nextPage := page + 1
		paginationInfo.NextPage = &nextPage
	}

	if hasPrev {
		prevPage := page - 1
		paginationInfo.PrevPage = &prevPage
	}

	return paginationInfo
}
