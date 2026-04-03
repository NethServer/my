/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package tunnel

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	m := NewManager(0, 64)
	if m == nil {
		t.Fatal("expected non-nil manager")
	}
	if m.Count() != 0 {
		t.Fatalf("expected 0 tunnels, got %d", m.Count())
	}
}

func TestManagerRegisterUnregister(t *testing.T) {
	m := NewManager(0, 64)

	// Register a tunnel with nil yamux session (just for registry test)
	tun, err := m.Register("sys1", "", "sess1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tun == nil {
		t.Fatal("expected non-nil tunnel")
	}
	if m.Count() != 1 {
		t.Fatalf("expected 1 tunnel, got %d", m.Count())
	}

	// Get by system ID (no node)
	got := m.Get("sys1", "")
	if got == nil {
		t.Fatal("expected to find tunnel by system ID")
	}
	if got.SystemID != "sys1" {
		t.Fatalf("expected system ID sys1, got %s", got.SystemID)
	}

	// Get by session ID
	got = m.GetBySessionID("sess1")
	if got == nil {
		t.Fatal("expected to find tunnel by session ID")
	}

	// Unregister
	m.Unregister("sys1", "")
	if m.Count() != 0 {
		t.Fatalf("expected 0 tunnels after unregister, got %d", m.Count())
	}
}

func TestManagerCloseBySessionID(t *testing.T) {
	m := NewManager(0, 64)
	_, _ = m.Register("sys1", "", "sess1", nil, nil)

	closed := m.CloseBySessionID("sess1")
	if !closed {
		t.Fatal("expected tunnel to be closed")
	}
	if m.Count() != 0 {
		t.Fatalf("expected 0 tunnels, got %d", m.Count())
	}

	// Closing non-existent session
	closed = m.CloseBySessionID("nonexistent")
	if closed {
		t.Fatal("expected false for non-existent session")
	}
}

func TestManagerReplaceExisting(t *testing.T) {
	m := NewManager(0, 64)
	_, _ = m.Register("sys1", "", "sess1", nil, nil)
	_, _ = m.Register("sys1", "", "sess2", nil, nil)

	if m.Count() != 1 {
		t.Fatalf("expected 1 tunnel after replacement, got %d", m.Count())
	}

	got := m.Get("sys1", "")
	if got.SessionID != "sess2" {
		t.Fatalf("expected session sess2, got %s", got.SessionID)
	}
}

func TestManagerList(t *testing.T) {
	m := NewManager(0, 64)
	_, _ = m.Register("sys1", "", "sess1", nil, nil)
	_, _ = m.Register("sys2", "", "sess2", nil, nil)

	list := m.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 tunnels in list, got %d", len(list))
	}
}

func TestManagerCloseAll(t *testing.T) {
	m := NewManager(0, 64)
	_, _ = m.Register("sys1", "", "sess1", nil, nil)
	_, _ = m.Register("sys2", "", "sess2", nil, nil)

	m.CloseAll()
	if m.Count() != 0 {
		t.Fatalf("expected 0 tunnels after CloseAll, got %d", m.Count())
	}
}

func TestManagerMaxTunnelsLimit(t *testing.T) {
	m := NewManager(2, 64)
	_, err := m.Register("sys1", "", "sess1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = m.Register("sys2", "", "sess2", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Third tunnel should be rejected
	_, err = m.Register("sys3", "", "sess3", nil, nil)
	if err == nil {
		t.Fatal("expected error when exceeding max tunnels")
	}

	// Replacing existing system should still work
	_, err = m.Register("sys1", "", "sess1b", nil, nil)
	if err != nil {
		t.Fatalf("replacement should work even at limit: %v", err)
	}
}

func TestManagerMultiNode(t *testing.T) {
	m := NewManager(0, 64)

	// Register tunnels for the same system but different nodes
	_, err := m.Register("sys1", "1", "sess-1", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = m.Register("sys1", "2", "sess-2", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err = m.Register("sys1", "3", "sess-3", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m.Count() != 3 {
		t.Fatalf("expected 3 tunnels for multi-node, got %d", m.Count())
	}

	// Each node is independently addressable
	got := m.Get("sys1", "1")
	if got == nil || got.SessionID != "sess-1" {
		t.Fatal("expected to find tunnel for node 1")
	}
	got = m.Get("sys1", "2")
	if got == nil || got.SessionID != "sess-2" {
		t.Fatal("expected to find tunnel for node 2")
	}

	// GetBySessionID still works
	got = m.GetBySessionID("sess-3")
	if got == nil || got.NodeID != "3" {
		t.Fatal("expected to find tunnel by session ID with node 3")
	}

	// Unregister one node
	m.Unregister("sys1", "2")
	if m.Count() != 2 {
		t.Fatalf("expected 2 tunnels after unregister, got %d", m.Count())
	}

	// Replace a node tunnel
	_, err = m.Register("sys1", "1", "sess-1b", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error replacing node tunnel: %v", err)
	}
	got = m.Get("sys1", "1")
	if got.SessionID != "sess-1b" {
		t.Fatalf("expected sess-1b, got %s", got.SessionID)
	}
}

func TestTunnelKeyFunction(t *testing.T) {
	if TunnelKey("sys1", "") != "sys1" {
		t.Fatal("expected plain systemID for empty nodeID")
	}
	if TunnelKey("sys1", "2") != "sys1:2" {
		t.Fatal("expected systemID:nodeID for non-empty nodeID")
	}
}

func TestTunnelStreamLimits(t *testing.T) {
	m := NewManager(0, 2) // max 2 streams per tunnel
	tun, _ := m.Register("sys1", "", "sess1", nil, nil)

	if !tun.AcquireStream() {
		t.Fatal("expected first stream to be acquired")
	}
	if !tun.AcquireStream() {
		t.Fatal("expected second stream to be acquired")
	}
	if tun.AcquireStream() {
		t.Fatal("expected third stream to be rejected (limit 2)")
	}

	tun.ReleaseStream()
	if !tun.AcquireStream() {
		t.Fatal("expected stream to be acquired after release")
	}
}

func TestValidateServiceTarget(t *testing.T) {
	tests := []struct {
		target  string
		wantErr bool
	}{
		{"localhost:8080", false},
		{"10.0.0.1:443", false},
		{"192.168.1.1:80", false},
		{"169.254.169.254:80", true},              // AWS metadata
		{"169.254.0.1:80", true},                  // link-local
		{"metadata.google.internal:80", true},     // GCP metadata
		{"metadata.azure.internal:80", true},      // Azure metadata
		{"instance-data:80", true},                // Oracle Cloud
		{"metadata.platformequinix.com:80", true}, // Equinix
		{"0.0.0.0:80", true},                      // unspecified
		{"224.0.0.1:80", true},                    // multicast
		{"255.255.255.255:80", true},              // broadcast
		{"", true},
	}

	for _, tt := range tests {
		err := validateServiceTarget(tt.target)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateServiceTarget(%q) = %v, wantErr %v", tt.target, err, tt.wantErr)
		}
	}
}
