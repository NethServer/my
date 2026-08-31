/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unregisterUpdateRegex matches the CTE that stamps unregistered_at while
// returning the status the row carried before the write.
const unregisterUpdateRegex = `WITH previous AS.+UPDATE systems SET unregistered_at`

// alertContextRegex matches the per-system label lookup used to clear a firing
// LinkFailed alert.
const alertContextRegex = `FROM systems s LEFT JOIN distributors d`

func callUnregister(authenticated bool) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/systems/unregister", nil)
	if authenticated {
		c.Set("system_id", "sys-1")
		c.Set("system_key", "NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0D")
	}
	UnregisterSystem(c)
	return w
}

func TestUnregisterSystem_StampsUnregisteredAt(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(unregisterUpdateRegex).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))

	w := callUnregister(true)

	require.Equal(t, http.StatusOK, w.Code)

	var body struct {
		Message string `json:"message"`
		Data    struct {
			SystemKey      string `json:"system_key"`
			UnregisteredAt string `json:"unregistered_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "system unregistered", body.Message)
	assert.Equal(t, "NETH-D49F-F40A-8650-4786-9CE1-E6DE-6E6C-2B0D", body.Data.SystemKey)
	assert.NotEmpty(t, body.Data.UnregisteredAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// A system that stopped reporting carries a firing LinkFailed alert, and the
// monitor that refreshes it skips unregistered rows: the handler has to clear
// it here or it hangs until its TTL.
func TestUnregisterSystem_ClearsLinkFailedWhenSystemWasInactive(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(unregisterUpdateRegex).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("inactive"))
	mock.ExpectQuery(alertContextRegex).
		WithArgs("sys-1").
		WillReturnError(sql.ErrNoRows)

	w := callUnregister(true)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// An active system has no LinkFailed alert to clear, so no alert lookup runs.
func TestUnregisterSystem_SkipsAlertLookupForActiveSystem(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(unregisterUpdateRegex).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("active"))

	callUnregister(true)

	assert.NoError(t, mock.ExpectationsWereMet())
}

// Two calls racing: one wins the update, the other matches no row. The
// credentials are revoked either way, which is what the caller asked for.
func TestUnregisterSystem_ConcurrentCallSucceeds(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(unregisterUpdateRegex).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnError(sql.ErrNoRows)

	w := callUnregister(true)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// The credentials must never be reported as revoked while the row still
// accepts them.
func TestUnregisterSystem_DatabaseFailureIsNotSilent(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	mock.ExpectQuery(unregisterUpdateRegex).
		WithArgs("sys-1", sqlmock.AnyArg()).
		WillReturnError(errors.New("connection reset"))

	w := callUnregister(true)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUnregisterSystem_RequiresAuthenticatedSystem(t *testing.T) {
	mock, cleanup := setupMockDB(t)
	defer cleanup()

	w := callUnregister(false)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.NoError(t, mock.ExpectationsWereMet())
}
