/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/backend/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupAlertActivityMock(t *testing.T) (*LocalAlertActivityRepository, sqlmock.Sqlmock, func()) {
	originalDB := database.DB
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	database.DB = mockDB
	repo := NewLocalAlertActivityRepository()

	cleanup := func() {
		database.DB = originalDB
		_ = mockDB.Close()
	}
	return repo, mock, cleanup
}

func TestWithNotesByFingerprints_EmptyInputNoQuery(t *testing.T) {
	repo, mock, cleanup := setupAlertActivityMock(t)
	defer cleanup()

	got, err := repo.WithNotesByFingerprints(nil, nil, "silenced from my")

	assert.NoError(t, err)
	assert.Empty(t, got)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithNotesByFingerprints_KeysMatchingPairs(t *testing.T) {
	repo, mock, cleanup := setupAlertActivityMock(t)
	defer cleanup()

	// The query must count standalone notes, assignment notes and non-default
	// silence comments, and nothing else.
	mock.ExpectQuery(`FROM alert_activity a.*a\.action = \$3.*a\.action = \$4 AND btrim\(COALESCE\(a\.details->>'note', ''\)\) <> ''.*a\.action IN \(\$5, \$6\) AND btrim\(COALESCE\(a\.details->>'comment', ''\)\) NOT IN \('', \$7\)`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(),
			AlertActivityNoteAdded, AlertActivityAssigned,
			AlertActivitySilenced, AlertActivitySilenceUpdated, "silenced from my").
		WillReturnRows(sqlmock.NewRows([]string{"organization_id", "fingerprint"}).
			AddRow("org-1", "fp-a"))

	got, err := repo.WithNotesByFingerprints([]string{"org-1", "org-1"}, []string{"fp-a", "fp-b"}, "silenced from my")

	require.NoError(t, err)
	assert.True(t, got[AssignmentKey("org-1", "fp-a")])
	assert.False(t, got[AssignmentKey("org-1", "fp-b")])
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestWithNotesByFingerprints_QueryError(t *testing.T) {
	repo, mock, cleanup := setupAlertActivityMock(t)
	defer cleanup()

	mock.ExpectQuery(`FROM alert_activity`).WillReturnError(errors.New("boom"))

	_, err := repo.WithNotesByFingerprints([]string{"org-1"}, []string{"fp-a"}, "silenced from my")

	assert.ErrorContains(t, err, "boom")
	assert.NoError(t, mock.ExpectationsWereMet())
}
