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
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
)

func TestCheckVATExists(t *testing.T) {
	// Store original DB
	originalDB := database.DB

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	// Set mock DB
	database.DB = mockDB

	// Restore original DB after test
	defer func() {
		database.DB = originalDB
	}()

	tests := []struct {
		name         string
		vat          string
		entityType   string
		excludeID    string
		mockSetup    func()
		expectedBool bool
		expectError  bool
	}{
		{
			name:         "Empty VAT returns false",
			vat:          "",
			entityType:   "customers",
			excludeID:    "",
			mockSetup:    func() {}, // No mock needed for empty VAT
			expectedBool: false,
			expectError:  false,
		},
		{
			name:       "VAT exists in customers",
			vat:        "12345678901",
			entityType: "customers",
			excludeID:  "",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
					WithArgs("12345678901", "").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedBool: true,
			expectError:  false,
		},
		{
			name:       "VAT does not exist in distributors",
			vat:        "98765432109",
			entityType: "distributors",
			excludeID:  "",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM distributors`).
					WithArgs("98765432109", "").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			expectedBool: false,
			expectError:  false,
		},
		{
			name:       "VAT exists but excluded by ID",
			vat:        "11111111111",
			entityType: "resellers",
			excludeID:  "reseller-123",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM resellers`).
					WithArgs("11111111111", "reseller-123").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
			},
			expectedBool: false,
			expectError:  false,
		},
		{
			name:       "VAT with whitespace is trimmed",
			vat:        "  22222222222  ",
			entityType: "customers",
			excludeID:  "",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
					WithArgs("22222222222", "").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			expectedBool: true,
			expectError:  false,
		},
		{
			name:       "Database error",
			vat:        "33333333333",
			entityType: "distributors",
			excludeID:  "",
			mockSetup: func() {
				mock.ExpectQuery(`SELECT COUNT\(\*\) FROM distributors`).
					WithArgs("33333333333", "").
					WillReturnError(sql.ErrConnDone)
			},
			expectedBool: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock expectations
			tt.mockSetup()

			// Call the function
			result, err := CheckVATExists(tt.vat, tt.entityType, tt.excludeID)

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to check VAT in")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBool, result)
			}

			// Verify all expectations were met
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCheckVATExists_EdgeCases(t *testing.T) {
	// Store original DB
	originalDB := database.DB

	// Create mock database
	mockDB, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	// Set mock DB
	database.DB = mockDB

	// Restore original DB after test
	defer func() {
		database.DB = originalDB
	}()

	t.Run("Only whitespace VAT", func(t *testing.T) {
		result, err := CheckVATExists("   ", "customers", "")
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("Multiple VAT matches", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM customers`).
			WithArgs("44444444444", "").
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(5))

		result, err := CheckVATExists("44444444444", "customers", "")
		assert.NoError(t, err)
		assert.True(t, result) // Any count > 0 should return true

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
