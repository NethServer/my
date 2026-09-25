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

func TestNextCleanupTime(t *testing.T) {
	at := func(h, m, s int) time.Time { return time.Date(2026, 9, 25, h, m, s, 0, time.UTC) }

	// Well before the minute: same hour.
	if got := nextCleanupTime(at(7, 40, 0)); !got.Equal(at(8, 2, 0)) {
		t.Fatalf("07:40 -> %s, want 08:02", got)
	}
	// Too close to the minute (less than the startup delay): next hour.
	if got := nextCleanupTime(at(8, 1, 30)); !got.Equal(at(9, 2, 0)) {
		t.Fatalf("08:01:30 -> %s, want 09:02", got)
	}
	// Just after the run: next hour.
	if got := nextCleanupTime(at(8, 8, 0)); !got.Equal(at(9, 2, 0)) {
		t.Fatalf("08:08 -> %s, want 09:02", got)
	}
	// Day boundary.
	if got := nextCleanupTime(at(23, 30, 0)); !got.Equal(time.Date(2026, 9, 26, 0, 2, 0, 0, time.UTC)) {
		t.Fatalf("23:30 -> %s, want 00:02 next day", got)
	}
	// Local zones must not shift the anchor: 09:40 CEST is 07:40 UTC.
	rome := time.FixedZone("CEST", 2*3600)
	if got := nextCleanupTime(time.Date(2026, 9, 25, 9, 40, 0, 0, rome)); !got.Equal(at(8, 2, 0)) {
		t.Fatalf("09:40 CEST -> %s, want 08:02 UTC", got)
	}
	// Whatever the hour, the anchor is always hh:02:00.
	for h := 0; h < 24; h++ {
		got := nextCleanupTime(at(h, 30, 0))
		if got.Minute() != cleanupMinute || got.Second() != 0 {
			t.Fatalf("anchor drifted: %s", got)
		}
	}
}

func TestHeartbeatBurstEnd(t *testing.T) {
	at := func(h, m, s int) time.Time { return time.Date(2026, 9, 25, h, m, s, 0, time.UTC) }
	cases := []struct {
		now  time.Time
		want time.Time // zero = outside a burst
	}{
		{at(8, 0, 0), time.Time{}},
		{at(8, 0, 29), time.Time{}},
		{at(8, 0, 30), at(8, 1, 40)},
		{at(8, 1, 0), at(8, 1, 40)},
		{at(8, 1, 39), at(8, 1, 40)},
		{at(8, 1, 40), time.Time{}},
		{at(8, 5, 0), time.Time{}},
		{at(8, 20, 45), at(8, 21, 40)},
		{at(23, 59, 50), time.Time{}},
		{time.Date(2026, 9, 26, 0, 0, 35, 0, time.UTC), time.Date(2026, 9, 26, 0, 1, 40, 0, time.UTC)},
	}
	for _, c := range cases {
		if got := heartbeatBurstEnd(c.now); !got.Equal(c.want) {
			t.Errorf("%s -> %s, want %s", c.now.Format("15:04:05"), got.Format("15:04:05"), c.want.Format("15:04:05"))
		}
	}
	// Zone independence: 10:00:45 CEST is 08:00:45 UTC, inside a burst.
	rome := time.FixedZone("CEST", 2*3600)
	if got := heartbeatBurstEnd(time.Date(2026, 9, 25, 10, 0, 45, 0, rome)); !got.Equal(at(8, 1, 40)) {
		t.Errorf("CEST input -> %s, want 08:01:40 UTC", got)
	}
}
