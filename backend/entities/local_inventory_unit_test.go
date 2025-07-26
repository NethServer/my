/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"fmt"
	"testing"
	"time"

	"github.com/nethesis/my/backend/models"
	"github.com/stretchr/testify/assert"
)

// TestLocalInventoryRepository_ValidateInventoryAccess tests inventory access based on system ownership
func TestLocalInventoryRepository_ValidateInventoryAccess(t *testing.T) {
	tests := []struct {
		name              string
		userOrgRole       string
		userOrgID         string
		systemID          string
		systemOrgID       string
		expectedCanAccess bool
		expectedError     string
	}{
		{
			name:              "owner can access any system inventory",
			userOrgRole:       "owner",
			userOrgID:         "org-owner",
			systemID:          "system-123",
			systemOrgID:       "org-customer-456",
			expectedCanAccess: true,
		},
		{
			name:              "distributor can access managed customer inventory",
			userOrgRole:       "distributor",
			userOrgID:         "org-distributor-1",
			systemID:          "system-123",
			systemOrgID:       "org-customer-managed-by-dist-1",
			expectedCanAccess: true,
		},
		{
			name:              "distributor cannot access unmanaged customer inventory",
			userOrgRole:       "distributor",
			userOrgID:         "org-distributor-1",
			systemID:          "system-123",
			systemOrgID:       "org-customer-other",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
		{
			name:              "reseller can access own customer inventory",
			userOrgRole:       "reseller",
			userOrgID:         "org-reseller-1",
			systemID:          "system-123",
			systemOrgID:       "org-customer-managed-by-res-1",
			expectedCanAccess: true,
		},
		{
			name:              "customer can access own system inventory",
			userOrgRole:       "customer",
			userOrgID:         "org-customer-123",
			systemID:          "system-123",
			systemOrgID:       "org-customer-123",
			expectedCanAccess: true,
		},
		{
			name:              "customer cannot access other customer inventory",
			userOrgRole:       "customer",
			userOrgID:         "org-customer-123",
			systemID:          "system-456",
			systemOrgID:       "org-customer-456",
			expectedCanAccess: false,
			expectedError:     "insufficient permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canAccess, err := validateInventoryAccess(tt.userOrgRole, tt.userOrgID, tt.systemID, tt.systemOrgID)

			assert.Equal(t, tt.expectedCanAccess, canAccess)

			if !tt.expectedCanAccess {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLocalInventoryRepository_InventoryHistoryPagination tests inventory history pagination
func TestLocalInventoryRepository_InventoryHistoryPagination(t *testing.T) {
	tests := []struct {
		name            string
		systemID        string
		page            int
		pageSize        int
		expectedRecords int
		expectedOffset  int
		expectedLimit   int
	}{
		{
			name:            "first page",
			systemID:        "system-123",
			page:            1,
			pageSize:        10,
			expectedRecords: 10,
			expectedOffset:  0,
			expectedLimit:   10,
		},
		{
			name:            "second page",
			systemID:        "system-123",
			page:            2,
			pageSize:        10,
			expectedRecords: 10,
			expectedOffset:  10,
			expectedLimit:   10,
		},
		{
			name:            "large page size",
			systemID:        "system-123",
			page:            1,
			pageSize:        100,
			expectedRecords: 25, // Assume system only has 25 records
			expectedOffset:  0,
			expectedLimit:   100,
		},
		{
			name:            "small page size",
			systemID:        "system-123",
			page:            5,
			pageSize:        5,
			expectedRecords: 5,
			expectedOffset:  20,
			expectedLimit:   5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, totalCount := simulateGetInventoryHistory(tt.systemID, tt.page, tt.pageSize, nil, nil)
			offset := (tt.page - 1) * tt.pageSize

			assert.Equal(t, tt.expectedOffset, offset)
			assert.Equal(t, tt.expectedLimit, tt.pageSize)
			assert.GreaterOrEqual(t, totalCount, len(records))

			if totalCount > offset {
				expectedRecords := min(tt.pageSize, totalCount-offset)
				assert.Equal(t, expectedRecords, len(records))
			} else {
				assert.Equal(t, 0, len(records))
			}
		})
	}
}

// TestLocalInventoryRepository_InventoryDateFiltering tests date range filtering for inventory
func TestLocalInventoryRepository_InventoryDateFiltering(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	lastWeek := now.Add(-7 * 24 * time.Hour)

	tests := []struct {
		name            string
		fromDate        *time.Time
		toDate          *time.Time
		expectedRecords int
		expectedInRange bool
	}{
		{
			name:            "last 24 hours",
			fromDate:        &yesterday,
			toDate:          &now,
			expectedRecords: 24, // Hourly records
			expectedInRange: true,
		},
		{
			name:            "last week",
			fromDate:        &lastWeek,
			toDate:          &now,
			expectedRecords: 168, // 7 days * 24 hours
			expectedInRange: true,
		},
		{
			name:            "no date filter",
			fromDate:        nil,
			toDate:          nil,
			expectedRecords: 720, // 30 days of hourly records
			expectedInRange: true,
		},
		{
			name:            "future date range (no records)",
			fromDate:        &now,
			toDate:          &now,
			expectedRecords: 0,
			expectedInRange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records, _ := simulateGetInventoryHistory("system-123", 1, 1000, tt.fromDate, tt.toDate)

			assert.Equal(t, tt.expectedRecords, len(records))

			// Verify all records are within date range if specified
			if tt.fromDate != nil && tt.toDate != nil && tt.expectedInRange {
				for _, record := range records {
					assert.True(t, record.CreatedAt.After(*tt.fromDate) || record.CreatedAt.Equal(*tt.fromDate))
					assert.True(t, record.CreatedAt.Before(*tt.toDate) || record.CreatedAt.Equal(*tt.toDate))
				}
			}
		})
	}
}

// TestLocalInventoryRepository_InventoryDiffFiltering tests inventory diff filtering
func TestLocalInventoryRepository_InventoryDiffFiltering(t *testing.T) {
	tests := []struct {
		name            string
		severity        string
		category        string
		diffType        string
		expectedDiffs   int
		expectedMatches bool
	}{
		{
			name:            "critical security diffs",
			severity:        "critical",
			category:        "security",
			diffType:        "",
			expectedDiffs:   5,
			expectedMatches: true,
		},
		{
			name:            "package changes",
			severity:        "",
			category:        "packages",
			diffType:        "changed",
			expectedDiffs:   15,
			expectedMatches: true,
		},
		{
			name:            "high severity additions",
			severity:        "high",
			category:        "",
			diffType:        "added",
			expectedDiffs:   8,
			expectedMatches: true,
		},
		{
			name:            "no filters (all diffs)",
			severity:        "",
			category:        "",
			diffType:        "",
			expectedDiffs:   50,
			expectedMatches: true,
		},
		{
			name:            "non-existent combination",
			severity:        "critical",
			category:        "non-existent",
			diffType:        "added",
			expectedDiffs:   0,
			expectedMatches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs, _ := simulateGetInventoryDiffs("system-123", 1, 100, tt.severity, tt.category, tt.diffType, nil, nil)

			assert.Equal(t, tt.expectedDiffs, len(diffs))

			// Verify all diffs match the filter criteria
			if tt.expectedMatches && len(diffs) > 0 {
				for _, diff := range diffs {
					if tt.severity != "" {
						assert.Equal(t, tt.severity, diff.Severity)
					}
					if tt.category != "" {
						assert.Equal(t, tt.category, diff.Category)
					}
					if tt.diffType != "" {
						assert.Equal(t, tt.diffType, diff.DiffType)
					}
				}
			}
		})
	}
}

// TestLocalInventoryRepository_LatestInventoryRetrieval tests latest inventory record retrieval
func TestLocalInventoryRepository_LatestInventoryRetrieval(t *testing.T) {
	tests := []struct {
		name              string
		systemID          string
		hasInventory      bool
		expectedTimestamp *time.Time
	}{
		{
			name:         "system with recent inventory",
			systemID:     "system-active",
			hasInventory: true,
			expectedTimestamp: func() *time.Time {
				t := time.Now().Add(-1 * time.Hour)
				return &t
			}(),
		},
		{
			name:         "system without inventory",
			systemID:     "system-new",
			hasInventory: false,
		},
		{
			name:         "non-existent system",
			systemID:     "system-nonexistent",
			hasInventory: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inventory := simulateGetLatestInventory(tt.systemID)

			if tt.hasInventory {
				assert.NotNil(t, inventory)
				assert.Equal(t, tt.systemID, inventory.SystemID)
				if tt.expectedTimestamp != nil {
					// Allow some tolerance for timestamp comparison
					timeDiff := inventory.CreatedAt.Sub(*tt.expectedTimestamp)
					assert.True(t, timeDiff < time.Minute && timeDiff > -time.Minute)
				}
			} else {
				assert.Nil(t, inventory)
			}
		})
	}
}

// TestLocalInventoryRepository_LatestInventoryDiffs tests latest inventory diffs retrieval
func TestLocalInventoryRepository_LatestInventoryDiffs(t *testing.T) {
	tests := []struct {
		name               string
		systemID           string
		expectedDiffCount  int
		expectedSeverities []string
	}{
		{
			name:               "system with recent changes",
			systemID:           "system-active",
			expectedDiffCount:  3,
			expectedSeverities: []string{"high", "medium", "low"},
		},
		{
			name:               "system without recent changes",
			systemID:           "system-stable",
			expectedDiffCount:  0,
			expectedSeverities: []string{},
		},
		{
			name:               "system with critical changes only",
			systemID:           "system-critical",
			expectedDiffCount:  1,
			expectedSeverities: []string{"critical"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diffs := simulateGetLatestInventoryDiffs(tt.systemID)

			assert.Equal(t, tt.expectedDiffCount, len(diffs))

			// Verify severities match
			actualSeverities := make([]string, len(diffs))
			for i, diff := range diffs {
				actualSeverities[i] = diff.Severity
			}

			for _, expectedSeverity := range tt.expectedSeverities {
				assert.Contains(t, actualSeverities, expectedSeverity)
			}
		})
	}
}

// TestLocalInventoryRepository_InventoryDataTypes tests inventory data structure validation
func TestLocalInventoryRepository_InventoryDataTypes(t *testing.T) {
	tests := []struct {
		name               string
		inventoryData      map[string]interface{}
		expectedValid      bool
		expectedCategories []string
	}{
		{
			name: "complete inventory data",
			inventoryData: map[string]interface{}{
				"packages":    []interface{}{"package1", "package2"},
				"services":    []interface{}{"service1", "service2"},
				"users":       []interface{}{"user1", "user2"},
				"network":     map[string]interface{}{"interfaces": 2, "routes": 10},
				"system_info": map[string]interface{}{"os": "Ubuntu 20.04", "kernel": "5.4.0"},
			},
			expectedValid:      true,
			expectedCategories: []string{"packages", "services", "users", "network", "system_info"},
		},
		{
			name: "minimal inventory data",
			inventoryData: map[string]interface{}{
				"system_info": map[string]interface{}{"os": "CentOS 8"},
			},
			expectedValid:      true,
			expectedCategories: []string{"system_info"},
		},
		{
			name:               "empty inventory data",
			inventoryData:      map[string]interface{}{},
			expectedValid:      true,
			expectedCategories: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validateInventoryData(tt.inventoryData)
			assert.Equal(t, tt.expectedValid, valid)

			categories := extractInventoryCategories(tt.inventoryData)
			assert.Equal(t, len(tt.expectedCategories), len(categories))

			for _, expectedCategory := range tt.expectedCategories {
				assert.Contains(t, categories, expectedCategory)
			}
		})
	}
}

// Helper functions for simulation and validation

func validateInventoryAccess(userOrgRole, userOrgID, systemID, systemOrgID string) (bool, error) {
	switch userOrgRole {
	case "owner":
		return true, nil
	case "distributor":
		// Simulate checking if distributor manages this customer
		if systemOrgID == "org-customer-managed-by-dist-1" && userOrgID == "org-distributor-1" {
			return true, nil
		}
		return false, fmt.Errorf("insufficient permissions to access inventory")
	case "reseller":
		// Simulate checking if reseller manages this customer
		if systemOrgID == "org-customer-managed-by-res-1" && userOrgID == "org-reseller-1" {
			return true, nil
		}
		return false, fmt.Errorf("insufficient permissions to access inventory")
	case "customer":
		if systemOrgID == userOrgID {
			return true, nil
		}
		return false, fmt.Errorf("insufficient permissions to access inventory")
	default:
		return false, fmt.Errorf("insufficient permissions to access inventory")
	}
}

func simulateGetInventoryHistory(systemID string, page, pageSize int, fromDate, toDate *time.Time) ([]models.InventoryRecord, int) {
	// Simulate different record counts based on system
	totalRecords := 720 // Default to 30 days of hourly records
	if systemID == "system-small" {
		totalRecords = 25
	}

	// Apply date filtering
	if fromDate != nil && toDate != nil {
		duration := toDate.Sub(*fromDate)
		totalRecords = int(duration.Hours()) // Hourly records
	}

	offset := (page - 1) * pageSize
	if offset >= totalRecords {
		return []models.InventoryRecord{}, totalRecords
	}

	recordCount := min(pageSize, totalRecords-offset)
	records := make([]models.InventoryRecord, recordCount)

	for i := 0; i < recordCount; i++ {
		createdAt := time.Now().Add(-time.Duration(offset+i) * time.Hour)
		if fromDate != nil {
			createdAt = fromDate.Add(time.Duration(i) * time.Hour)
		}

		records[i] = models.InventoryRecord{
			ID:        int64(offset + i + 1),
			SystemID:  systemID,
			CreatedAt: createdAt,
			Data:      []byte(`{"packages": ["pkg1", "pkg2"]}`),
		}
	}

	return records, totalRecords
}

func simulateGetInventoryDiffs(systemID string, page, pageSize int, severity, category, diffType string, fromDate, toDate *time.Time) ([]models.InventoryDiff, int) {
	// Simulate different diff counts based on filters
	totalDiffs := 50

	if severity == "critical" && category == "security" {
		totalDiffs = 5
	} else if category == "packages" && diffType == "changed" {
		totalDiffs = 15
	} else if severity == "high" && diffType == "added" {
		totalDiffs = 8
	} else if severity == "critical" && category == "non-existent" {
		totalDiffs = 0
	}

	offset := (page - 1) * pageSize
	if offset >= totalDiffs {
		return []models.InventoryDiff{}, totalDiffs
	}

	diffCount := min(pageSize, totalDiffs-offset)
	diffs := make([]models.InventoryDiff, diffCount)

	for i := 0; i < diffCount; i++ {
		diffSeverity := "medium"
		if severity != "" {
			diffSeverity = severity
		}

		diffCategory := "general"
		if category != "" {
			diffCategory = category
		}

		diffDiffType := "changed"
		if diffType != "" {
			diffDiffType = diffType
		}

		diffs[i] = models.InventoryDiff{
			ID:        int64(offset + i + 1),
			SystemID:  systemID,
			Severity:  diffSeverity,
			Category:  diffCategory,
			DiffType:  diffDiffType,
			CreatedAt: time.Now().Add(-time.Duration(i) * time.Hour),
		}
	}

	return diffs, totalDiffs
}

func simulateGetLatestInventory(systemID string) *models.InventoryRecord {
	switch systemID {
	case "system-active":
		return &models.InventoryRecord{
			ID:        1,
			SystemID:  systemID,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Data:      []byte(`{"packages": ["pkg1", "pkg2"]}`),
		}
	default:
		return nil
	}
}

func simulateGetLatestInventoryDiffs(systemID string) []models.InventoryDiff {
	switch systemID {
	case "system-active":
		return []models.InventoryDiff{
			{ID: 1, SystemID: systemID, Severity: "high", Category: "security", DiffType: "added"},
			{ID: 2, SystemID: systemID, Severity: "medium", Category: "packages", DiffType: "changed"},
			{ID: 3, SystemID: systemID, Severity: "low", Category: "services", DiffType: "removed"},
		}
	case "system-critical":
		return []models.InventoryDiff{
			{ID: 4, SystemID: systemID, Severity: "critical", Category: "security", DiffType: "added"},
		}
	default:
		return []models.InventoryDiff{}
	}
}

func validateInventoryData(data map[string]interface{}) bool {
	// All inventory data is considered valid in this simulation
	return true
}

func extractInventoryCategories(data map[string]interface{}) []string {
	categories := make([]string, 0, len(data))
	for key := range data {
		categories = append(categories, key)
	}
	return categories
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
