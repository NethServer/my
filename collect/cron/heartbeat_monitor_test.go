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
	mock.ExpectExec(`SET status = 'inactive'`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	monitor.checkAndUpdateStatuses(context.Background())

	require.NoError(t, mock.ExpectationsWereMet())
	assert.False(t, called, "postAlerts must not be called when nothing recovered")
}
