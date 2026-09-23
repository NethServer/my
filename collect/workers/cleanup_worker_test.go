/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package workers

import (
	"testing"
	"time"
)

func TestVacuumDue(t *testing.T) {
	cw := NewCleanupWorker(1)
	at := func(hour int) time.Time {
		return time.Date(2026, 9, 23, hour, 28, 0, 0, time.UTC)
	}

	if cw.vacuumDue(at(14)) {
		t.Fatalf("vacuum must not be due outside hour %d UTC", vacuumHourUTC)
	}
	if !cw.vacuumDue(at(vacuumHourUTC)) {
		t.Fatal("vacuum must be due in the configured hour on a fresh worker")
	}

	// A second run in the same hour of the same day (e.g. the initial run
	// after a restart) must not vacuum twice.
	cw.lastVacuumDay = at(vacuumHourUTC).Format("2006-01-02")
	if cw.vacuumDue(at(vacuumHourUTC).Add(20 * time.Minute)) {
		t.Fatal("vacuum must run at most once per UTC day")
	}

	// The next day it is due again.
	if !cw.vacuumDue(at(vacuumHourUTC).Add(24 * time.Hour)) {
		t.Fatal("vacuum must be due again the following day")
	}

	// Local wall-clock time must not matter: 03:28 UTC expressed in another zone.
	rome := time.FixedZone("CEST", 2*3600)
	if !cw.vacuumDue(time.Date(2026, 9, 25, vacuumHourUTC+2, 28, 0, 0, rome)) {
		t.Fatal("vacuumDue must compare the hour in UTC")
	}
}
