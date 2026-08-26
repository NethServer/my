/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cron

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHeartbeatMonitor(t *testing.T) {
	// Initialize configuration for testing
	configuration.Config.HeartbeatTimeoutMinutes = 10
	configuration.Config.HeartbeatCheckIntervalSeconds = 120

	monitor := NewHeartbeatMonitor()

	if monitor == nil {
		t.Fatal("NewHeartbeatMonitor returned nil")
	}

	if monitor.timeoutMinutes <= 0 {
		t.Errorf("Expected positive timeout, got %d", monitor.timeoutMinutes)
	}

	if monitor.checkIntervalSec != 120 {
		t.Errorf("Expected check interval 120 seconds (from config), got %d", monitor.checkIntervalSec)
	}

	if monitor.timeoutMinutes != configuration.Config.HeartbeatTimeoutMinutes {
		t.Errorf("Expected timeout to match config value %d, got %d", configuration.Config.HeartbeatTimeoutMinutes, monitor.timeoutMinutes)
	}
}

func TestNewHeartbeatMonitor_DefaultIntervalWhenUnset(t *testing.T) {
	configuration.Config.HeartbeatCheckIntervalSeconds = 0 // unconfigured

	monitor := NewHeartbeatMonitor()

	if monitor.checkIntervalSec != defaultHeartbeatCheckIntervalSec {
		t.Errorf("Expected fallback interval %d seconds, got %d", defaultHeartbeatCheckIntervalSec, monitor.checkIntervalSec)
	}
}

func TestHeartbeatMonitor_Structure(t *testing.T) {
	monitor := &HeartbeatMonitor{
		db:               nil,
		timeoutMinutes:   10,
		checkIntervalSec: 60,
	}

	if monitor.timeoutMinutes != 10 {
		t.Errorf("Expected timeout 10 minutes, got %d", monitor.timeoutMinutes)
	}

	if monitor.checkIntervalSec != 60 {
		t.Errorf("Expected interval 60 seconds, got %d", monitor.checkIntervalSec)
	}

}

// TestCheckAndUpdateStatuses_ResolvesRecoveredSystem verifies the recovery path:
// inactive->active systems are returned, their context is looked up, and a
// resolved LinkFailed alert is posted for them.
func TestCheckAndUpdateStatuses_ResolvesRecoveredSystem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	posted := map[string][]models.AlertmanagerPostAlert{}
	monitor := &HeartbeatMonitor{
		db:               db,
		timeoutMinutes:   10,
		checkIntervalSec: 60,
		observedChecks:   observedWindowMet,
		postAlerts: func(orgID string, alerts []models.AlertmanagerPostAlert) error {
			posted[orgID] = append(posted[orgID], alerts...)
			return nil
		},
	}

	// 1. inactive -> active, returns one recovered id
	mock.ExpectQuery(`RETURNING s\.id::text`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("sys-uuid-1"))
	// 2. unknown -> active (no recoveries to resolve)
	mock.ExpectExec(`s\.status = 'unknown'`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	// 3. resolveRecovered -> LookupSystemAlertContext for the recovered id
	mock.ExpectQuery(`WHERE s\.id = \$1`).
		WithArgs("sys-uuid-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "organization_id", "system_key", "name", "type", "fqdn", "ipv4",
			"org_name", "org_vat", "org_type", "reseller_org_id",
		}).AddRow("sys-uuid-1", "org-1", "SYS-001", "web-01", "ns8", "", "", "Reseller X", "", "reseller", "reseller-1"))
	// 4. active -> inactive
	expectFleetObserved(mock)
	mock.ExpectExec(`SET status = 'inactive'`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	monitor.checkAndUpdateStatuses(context.Background())

	require.NoError(t, mock.ExpectationsWereMet())
	// Resolved alert is posted to the RESELLER tenant, but its organization_id
	// label stays the customer org.
	require.Len(t, posted["reseller-1"], 1)
	alert := posted["reseller-1"][0]
	assert.Equal(t, "LinkFailed", alert.Labels["alertname"])
	assert.Equal(t, "SYS-001", alert.Labels["system_key"])
	assert.Equal(t, "org-1", alert.Labels["organization_id"])
	assert.False(t, alert.EndsAt.After(time.Now().UTC().Add(time.Second)),
		"resolve alert EndsAt must be <= now")
}

// TestCheckAndUpdateStatuses_NoRecovery_NoResolve verifies that when nothing
// recovers, no context lookup and no resolve post happen.
func TestCheckAndUpdateStatuses_NoRecovery_NoResolve(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	called := false
	monitor := &HeartbeatMonitor{
		db:               db,
		timeoutMinutes:   10,
		checkIntervalSec: 60,
		observedChecks:   observedWindowMet,
		postAlerts: func(string, []models.AlertmanagerPostAlert) error {
			called = true
			return nil
		},
	}

	mock.ExpectQuery(`RETURNING s\.id::text`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"})) // none recovered
	mock.ExpectExec(`s\.status = 'unknown'`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	expectFleetObserved(mock)
	mock.ExpectExec(`SET status = 'inactive'`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	monitor.checkAndUpdateStatuses(context.Background())

	require.NoError(t, mock.ExpectationsWereMet())
	assert.False(t, called, "postAlerts must not be called when nothing recovered")
}

// expectFleetObserved queues the one fleet-freshness read a warmed-up monitor
// makes, so a test can get straight to the transition it is really about.
func expectFleetObserved(mock sqlmock.Sqlmock) {
	mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+s\.status = 'active'`).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(time.Now()))
}

// observedWindowMet is an observedChecks value past any required count, for
// tests whose subject is not the observation window itself.
const observedWindowMet = 1 << 20

func TestRequiredObservedChecks(t *testing.T) {
	tests := []struct {
		name        string
		timeoutMin  int
		intervalSec int
		want        int
	}{
		{"production defaults", 20, 300, 4},
		{"the PR timeout", 30, 300, 6},
		{"rounds up rather than short-changing the window", 7, 300, 2},
		{"interval longer than the timeout still checks once", 1, 300, 1},
		{"unconfigured interval falls back", 20, 0, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &HeartbeatMonitor{timeoutMinutes: tt.timeoutMin, checkIntervalSec: tt.intervalSec}
			assert.Equal(t, tt.want, m.requiredObservedChecks())
		})
	}
}

// The requirement in one test: while My is not recording, nothing is marked
// offline, so no LinkFailed can be raised for a fault on our side.
func TestCanJudgeSilence_RefusesWhileTheFleetIsUnrecorded(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	monitor := &HeartbeatMonitor{db: db, timeoutMinutes: 20, checkIntervalSec: 300, observedChecks: 4}

	mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+s\.status = 'active'`).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(time.Now().Add(-9 * time.Minute)))

	assert.False(t, monitor.canJudgeSilence(context.Background()))
	assert.Zero(t, monitor.observedChecks, "an unobserved check restarts the window")
	require.NoError(t, mock.ExpectationsWereMet())
}

// Recording alone is not enough: the window must be as long as the one a
// LinkFailed claims to have watched, which is what stops the storm on recovery.
func TestCanJudgeSilence_WaitsForAFullTimeoutOfRecording(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	monitor := &HeartbeatMonitor{db: db, timeoutMinutes: 20, checkIntervalSec: 300}
	require.Equal(t, 4, monitor.requiredObservedChecks())

	for i := 0; i < 5; i++ {
		mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+s\.status = 'active'`).
			WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(time.Now()))
	}

	var verdicts []bool
	for i := 0; i < 5; i++ {
		verdicts = append(verdicts, monitor.canJudgeSilence(context.Background()))
	}

	assert.Equal(t, []bool{false, false, false, true, true}, verdicts)
	require.NoError(t, mock.ExpectationsWereMet())
}

// A single unrecorded check restarts the window: a flapping ingest never
// accumulates the evidence it needs.
func TestCanJudgeSilence_OneUnrecordedCheckRestartsTheWindow(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	monitor := &HeartbeatMonitor{db: db, timeoutMinutes: 20, checkIntervalSec: 300}

	fresh := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"max"}).AddRow(time.Now())
	}
	stale := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"max"}).AddRow(time.Now().Add(-10 * time.Minute))
	}
	for _, rows := range []*sqlmock.Rows{fresh(), fresh(), fresh(), stale(), fresh()} {
		mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+s\.status = 'active'`).WillReturnRows(rows)
	}

	for i := 0; i < 5; i++ {
		assert.False(t, monitor.canJudgeSilence(context.Background()), "check %d", i)
	}
	assert.Equal(t, 1, monitor.observedChecks, "counting starts over after the gap")
	require.NoError(t, mock.ExpectationsWereMet())
}

// With no active system that has ever beaten there is nothing to judge and
// nothing the transition could touch. Withholding here would be a trap: once
// every remaining beat belongs to a system that is already offline, the reading
// can never come back and the check would hold the transition for good.
func TestCanJudgeSilence_NoActiveSystemToJudge(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	monitor := &HeartbeatMonitor{db: db, timeoutMinutes: 20, checkIntervalSec: 300}

	mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+s\.status = 'active'`).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(nil))

	assert.True(t, monitor.canJudgeSilence(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

// The reading must ignore systems that are already offline: their beats are
// frozen by definition, and letting them set the newest-beat reading is what
// turns a fleet of dead systems into a permanently closed gate.
func TestCanJudgeSilence_IgnoresAlreadyOfflineSystems(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	monitor := &HeartbeatMonitor{db: db, timeoutMinutes: 20, checkIntervalSec: 300}

	// An active system beat a moment ago; whatever the offline ones last did is
	// none of this check's business, so the query filters them out in SQL.
	mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+INNER JOIN systems.+s\.status = 'active'.+s\.deleted_at IS NULL`).
		WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(time.Now()))

	assert.False(t, monitor.canJudgeSilence(context.Background()), "first observed check of a fresh window")
	assert.Equal(t, 1, monitor.observedChecks)
	require.NoError(t, mock.ExpectationsWereMet())
}

// Losing the reading is not evidence of an outage. A spurious alert can be
// resolved; one that never fires cannot, so the check fails open.
func TestCanJudgeSilence_FailsOpenOnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()

	monitor := &HeartbeatMonitor{db: db, timeoutMinutes: 20, checkIntervalSec: 300}

	mock.ExpectQuery(`MAX\(h\.last_heartbeat\).+s\.status = 'active'`).WillReturnError(errors.New("connection reset"))

	assert.True(t, monitor.canJudgeSilence(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}
