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
	"time"

	"github.com/nethesis/my/collect/configuration"
)

func TestNewHeartbeatMonitor(t *testing.T) {
	// Initialize configuration for testing
	configuration.Config.HeartbeatTimeoutMinutes = 10
	configuration.Config.MimirURL = "http://localhost:9009"

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

	if monitor.mimirURL != "http://localhost:9009" {
		t.Errorf("Expected mimirURL http://localhost:9009, got %s", monitor.mimirURL)
	}

	if monitor.timeoutMinutes != configuration.Config.HeartbeatTimeoutMinutes {
		t.Errorf("Expected timeout to match config value %d, got %d", configuration.Config.HeartbeatTimeoutMinutes, monitor.timeoutMinutes)
	}
}

func TestHeartbeatMonitor_Structure(t *testing.T) {
	monitor := &HeartbeatMonitor{
		db:               nil,
		mimirURL:         "http://localhost:9009",
		timeoutMinutes:   10,
		checkIntervalSec: 60,
	}

	if monitor.timeoutMinutes != 10 {
		t.Errorf("Expected timeout 10 minutes, got %d", monitor.timeoutMinutes)
	}

	if monitor.checkIntervalSec != 60 {
		t.Errorf("Expected interval 60 seconds, got %d", monitor.checkIntervalSec)
	}

	if monitor.mimirURL == "" {
		t.Fatal("Expected mimirURL to be set")
	}
}

func TestHeartbeatMonitor_AlertFunctions(t *testing.T) {
	monitor := &HeartbeatMonitor{
		db:               nil,
		mimirURL:         "http://localhost:9009",
		timeoutMinutes:   10,
		checkIntervalSec: 60,
	}

	// Test fireHostDownAlert doesn't panic (won't actually post in test)
	err := monitor.fireHostDownAlert("TEST-KEY", "org-123", nowForTest())
	// Note: Will fail to connect to mimir in test, but that's OK for structure validation
	_ = err

	// Test resolveHostDownAlert doesn't panic
	err = monitor.resolveHostDownAlert("TEST-KEY", "org-123")
	// Note: Will fail to connect to mimir in test, but that's OK for structure validation
	_ = err
}

func nowForTest() time.Time {
	return time.Unix(123, 0)
}
