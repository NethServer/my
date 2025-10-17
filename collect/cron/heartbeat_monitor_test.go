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
	"testing"

	"github.com/nethesis/my/collect/configuration"
)

func TestNewHeartbeatMonitor(t *testing.T) {
	// Initialize configuration for testing
	configuration.Config.HeartbeatTimeoutMinutes = 10

	monitor := NewHeartbeatMonitor()

	if monitor == nil {
		t.Fatal("NewHeartbeatMonitor returned nil")
	}

	if monitor.timeoutMinutes <= 0 {
		t.Errorf("Expected positive timeout, got %d", monitor.timeoutMinutes)
	}

	if monitor.checkIntervalSec != 60 {
		t.Errorf("Expected check interval 60 seconds, got %d", monitor.checkIntervalSec)
	}
}

func TestHeartbeatMonitor_Structure(t *testing.T) {
	monitor := &HeartbeatMonitor{
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
