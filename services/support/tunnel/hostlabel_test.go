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
	"regexp"
	"testing"
)

func TestServiceHostLabel(t *testing.T) {
	// Same vectors as backend/helpers/servicehostlabel_test.go
	for name, want := range map[string]string{
		"cluster-admin":                       "cluster-admin-6e5263a7",
		"nethsecurity-controller4-prometheus": "nethsecurity-control-87b27f47",
		"":                                    "svc-e3b0c442",
	} {
		if got := ServiceHostLabel(name); got != want {
			t.Errorf("ServiceHostLabel(%q) = %q, want %q", name, got, want)
		}
	}

	label := regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*-[0-9a-f]{8}$`)
	for _, name := range []string{"cluster-admin", "nethvoice103-cti", "Web.Top_5", "a.b.c", "--x--", "nethsecurity-controller4-prometheus-very-long-route-name", "_"} {
		l := ServiceHostLabel(name)
		if len(l+"--0123456789abcdef0123456789abcdef") > 63 || !label.MatchString(l) {
			t.Errorf("ServiceHostLabel(%q) = %q is not a valid host label", name, l)
		}
	}
}

func TestResolveSessionServiceName(t *testing.T) {
	m := NewManager(10, 10)
	tun := &Tunnel{SystemID: "sys", SessionID: "sess", services: map[string]ServiceInfo{"nethvoice1-cti": {Target: "127.0.0.1:80"}}}
	m.tunnels[TunnelKey("sys", "")] = tun

	if name, ok := m.ResolveSessionServiceName("sess", ServiceHostLabel("nethvoice1-cti")); !ok || name != "nethvoice1-cti" {
		t.Fatalf("label not resolved: %q %v", name, ok)
	}
	if name, ok := m.ResolveSessionServiceName("sess", "nethvoice1-cti"); !ok || name != "nethvoice1-cti" {
		t.Fatalf("exact name not resolved: %q %v", name, ok)
	}
	if _, ok := m.ResolveSessionServiceName("sess", ServiceHostLabel("webtop1")); ok {
		t.Fatal("unknown label resolved")
	}
}
