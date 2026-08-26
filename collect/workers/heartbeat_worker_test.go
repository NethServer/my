/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package workers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// The summary exists to make a stalled ingest visible, so it must report a
// window in which nothing was flushed. Driving it from the loop rather than from
// processBatch is what makes that possible: an empty queue returns early, and a
// summary living at the end of processBatch would go quiet exactly when the
// ingest stops.
func TestReportThroughput_ReportsAWindowWithNoBatches(t *testing.T) {
	hw := &HeartbeatWorker{summarySince: time.Now().Add(-heartbeatSummaryInterval - time.Second)}

	hw.reportThroughput(context.Background())

	assert.Zero(t, hw.summaryBatches)
	assert.WithinDuration(t, time.Now(), hw.summarySince, time.Second,
		"a reported window starts a new one")
}

// Below the interval nothing is emitted and the window keeps accumulating.
func TestReportThroughput_HoldsUntilTheIntervalElapses(t *testing.T) {
	started := time.Now().Add(-time.Second)
	hw := &HeartbeatWorker{summarySince: started, summaryBatches: 3, summaryUpserted: 120}

	hw.reportThroughput(context.Background())

	assert.Equal(t, 3, hw.summaryBatches, "the window is still open")
	assert.Equal(t, 120, hw.summaryUpserted)
	assert.Equal(t, started, hw.summarySince)
}

// A reported window resets the counters so the next one measures only itself.
func TestReportThroughput_StartsAFreshWindow(t *testing.T) {
	hw := &HeartbeatWorker{
		summarySince:    time.Now().Add(-heartbeatSummaryInterval - time.Second),
		summaryBatches:  42,
		summaryUpserted: 9600,
	}

	hw.reportThroughput(context.Background())

	assert.Zero(t, hw.summaryBatches)
	assert.Zero(t, hw.summaryUpserted)
}
