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

	if monitor.firedAlerts == nil {
		t.Fatal("Expected firedAlerts map to be initialized")
	}
}

func TestHeartbeatMonitor_Structure(t *testing.T) {
	monitor := &HeartbeatMonitor{
		mimirURL:         "http://localhost:9009",
		timeoutMinutes:   10,
		checkIntervalSec: 60,
		firedAlerts:      make(map[string]firedAlert),
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

func TestHeartbeatMonitor_FiredAlertsTracking(t *testing.T) {
	monitor := &HeartbeatMonitor{
		timeoutMinutes:   10,
		checkIntervalSec: 60,
		firedAlerts:      make(map[string]firedAlert),
	}

	systemKey := "NETH-TEST-KEY"
	monitor.firedAlerts[systemKey] = firedAlert{
		OrgID:    "org-123",
		StartsAt: nowForTest(),
	}

	if len(monitor.firedAlerts) != 1 {
		t.Fatalf("Expected 1 fired alert, got %d", len(monitor.firedAlerts))
	}

	alert, ok := monitor.firedAlerts[systemKey]
	if !ok {
		t.Fatal("Expected fired alert to be present")
	}

	if alert.OrgID != "org-123" {
		t.Fatalf("Expected OrgID org-123, got %s", alert.OrgID)
	}

	delete(monitor.firedAlerts, systemKey)

	if len(monitor.firedAlerts) != 0 {
		t.Fatalf("Expected fired alerts map to be empty, got %d entries", len(monitor.firedAlerts))
	}
}

func nowForTest() time.Time {
	return time.Unix(123, 0)
}
