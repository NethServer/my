/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package connection

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/config"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
)

func TestBuildConnectURL(t *testing.T) {
	got, err := buildConnectURL("wss://support.example.com/api/tunnel", "2", "tok/en+=")
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(got)
	if u.Query().Get("node_id") != "2" || u.Query().Get("reconnect_token") != "tok/en+=" {
		t.Errorf("unexpected query: %s", got)
	}

	got, _ = buildConnectURL("wss://support.example.com/api/tunnel", "", "")
	if strings.Contains(got, "reconnect_token") || strings.Contains(got, "node_id") {
		t.Errorf("empty params must be omitted: %s", got)
	}
}

func testConfig(serverURL string) *config.ClientConfig {
	return &config.ClientConfig{
		URL:               "ws" + strings.TrimPrefix(serverURL, "http") + "/api/tunnel",
		Key:               "NETH-TEST",
		Secret:            "secret",
		NodeID:            "1",
		ReconnectDelay:    time.Millisecond,
		MaxReconnectDelay: time.Millisecond,
	}
}

func TestDialTunnelReconnectToken(t *testing.T) {
	var mu sync.Mutex
	var queries []url.Values
	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		queries = append(queries, r.URL.Query())
		n := len(queries)
		mu.Unlock()
		if n == 3 {
			w.WriteHeader(http.StatusGone)
			return
		}
		hdr := http.Header{}
		hdr.Set("X-Reconnect-Token", "token-"+string(rune('0'+n)))
		conn, err := upgrader.Upgrade(w, r, hdr)
		if err != nil {
			return
		}
		_ = conn.Close()
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	st := &runState{}

	conn, err := dialTunnel(context.Background(), cfg, st)
	if err != nil {
		t.Fatalf("first dial: %v", err)
	}
	_ = conn.Close()
	if st.reconnectToken != "token-1" {
		t.Fatalf("token not stored: %q", st.reconnectToken)
	}

	conn, err = dialTunnel(context.Background(), cfg, st)
	if err != nil {
		t.Fatalf("second dial: %v", err)
	}
	_ = conn.Close()
	if st.reconnectToken != "token-2" {
		t.Errorf("token not refreshed: %q", st.reconnectToken)
	}

	if _, err := dialTunnel(context.Background(), cfg, st); !errors.Is(err, errSessionGone) {
		t.Errorf("410 must map to errSessionGone, got %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if queries[0].Has("reconnect_token") {
		t.Error("first handshake must not carry a reconnect token")
	}
	if queries[1].Get("reconnect_token") != "token-1" || queries[2].Get("reconnect_token") != "token-2" {
		t.Errorf("reconnect token not sent: %v", queries)
	}
	if queries[1].Get("node_id") != "1" {
		t.Errorf("node_id missing: %v", queries[1])
	}
}

func TestRunWithReconnectTerminalStatuses(t *testing.T) {
	cases := []struct {
		status    int
		wantCode  int
		wantCalls int
	}{
		{http.StatusGone, 0, 1},
		{http.StatusUnauthorized, 1, maxAuthFailures},
		{http.StatusForbidden, 1, maxAuthFailures},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			var mu sync.Mutex
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				calls++
				mu.Unlock()
				w.WriteHeader(tc.status)
			}))
			defer srv.Close()

			cfg := testConfig(srv.URL)
			cfg.UsersStateFile = filepath.Join(t.TempDir(), "users-state.json")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if code := RunWithReconnect(ctx, cfg, nil); code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
			if ctx.Err() != nil {
				t.Fatal("RunWithReconnect did not stop on a terminal status")
			}
			mu.Lock()
			defer mu.Unlock()
			if calls != tc.wantCalls {
				t.Errorf("handshakes = %d, want %d", calls, tc.wantCalls)
			}
		})
	}
}

func stubResolver(t *testing.T, records map[string][]net.IP, local []net.IP) {
	t.Helper()
	origLookup, origLocal := lookupIP, localAddrs
	lookupIP = func(_ context.Context, host string) ([]net.IP, error) {
		if ips, ok := records[host]; ok {
			return ips, nil
		}
		return nil, errors.New("no such host")
	}
	localAddrs = func() []net.IP { return local }
	t.Cleanup(func() { lookupIP, localAddrs = origLookup, origLocal })
}

func TestResolveInjectedTarget(t *testing.T) {
	stubResolver(t, map[string][]net.IP{
		"phone.lan":    {net.ParseIP("192.168.1.100")},
		"rebind.evil":  {net.ParseIP("127.0.0.1")},
		"mixed.evil":   {net.ParseIP("192.168.1.5"), net.ParseIP("169.254.169.254")},
		"self.lan":     {net.ParseIP("192.168.1.1")},
		"metadata.lan": {net.ParseIP("192.168.1.9")},
	}, []net.IP{net.ParseIP("192.168.1.1")})

	discovered := map[string]models.ServiceInfo{
		"cluster-admin": {Target: "127.0.0.1:9090"},
		"nethsec-ui":    {Target: "localhost:443"},
	}

	cases := []struct {
		target   string
		wantAddr string
		wantErr  bool
	}{
		{"192.168.1.100:443", "192.168.1.100:443", false},
		{"phone.lan:80", "192.168.1.100:80", false},
		{"127.0.0.1:9090", "127.0.0.1:9090", false},
		{"127.0.0.1:443", "127.0.0.1:443", false},
		{"127.0.0.1:6379", "", true},
		{"[::1]:22", "", true},
		{"rebind.evil:6379", "", true},
		{"192.168.1.1:22", "", true},
		{"self.lan:22", "", true},
		{"mixed.evil:80", "", true},
		{"169.254.169.254:80", "", true},
		{"0.0.0.0:80", "", true},
		{"metadata.google.internal:80", "", true},
		{"unknown.lan:80", "", true},
		{"192.168.1.100", "", true},
		{"192.168.1.100:0", "", true},
		{"192.168.1.100:http", "", true},
	}
	for _, tc := range cases {
		addr, err := resolveInjectedTarget(tc.target, discovered)
		if (err != nil) != tc.wantErr {
			t.Errorf("resolveInjectedTarget(%q) err=%v, wantErr=%v", tc.target, err, tc.wantErr)
			continue
		}
		if addr != tc.wantAddr {
			t.Errorf("resolveInjectedTarget(%q) = %q, want %q", tc.target, addr, tc.wantAddr)
		}
	}
}

// yamuxPair returns a client session whose peer discards every stream.
func yamuxPair(t *testing.T) *yamux.Session {
	t.Helper()
	c1, c2 := net.Pipe()
	cfg := yamux.DefaultConfig()
	cfg.LogOutput = io.Discard
	client, err := yamux.Client(c1, cfg)
	if err != nil {
		t.Fatal(err)
	}
	server, err := yamux.Server(c2, cfg)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			s, err := server.Accept()
			if err != nil {
				return
			}
			go func() { _, _ = io.Copy(io.Discard, s); _ = s.Close() }()
		}
	}()
	t.Cleanup(func() { _ = client.Close(); _ = server.Close() })
	return client
}

func TestApplyAddServicesPinsAndRejectsShadowing(t *testing.T) {
	stubResolver(t, map[string][]net.IP{"phone.lan": {net.ParseIP("192.168.1.100")}}, nil)
	session := yamuxPair(t)
	store := newServiceStore(map[string]models.ServiceInfo{
		"cluster-admin": {Target: "127.0.0.1:9090"},
	})

	for _, name := range []string{"cluster-admin", "terminal"} {
		err := applyAddServices(map[string]models.ServiceInfo{name: {Target: "192.168.1.100:443"}}, store, session)
		if err == nil {
			t.Errorf("injecting %q must be rejected", name)
		}
	}
	if store.get()["cluster-admin"].Target != "127.0.0.1:9090" {
		t.Fatal("discovered service was overwritten")
	}

	if err := applyAddServices(map[string]models.ServiceInfo{"phone": {Target: "phone.lan:443", TLS: true}}, store, session); err != nil {
		t.Fatalf("add phone: %v", err)
	}
	phone := store.get()["phone"]
	if phone.DialAddr != "192.168.1.100:443" || phone.Target != "phone.lan:443" {
		t.Errorf("unexpected injected service: %+v", phone)
	}

	// Re-discovery keeps the injected service and its pinned address
	store.setDiscovered(map[string]models.ServiceInfo{"cluster-admin": {Target: "127.0.0.1:9090"}})
	if store.get()["phone"].DialAddr != "192.168.1.100:443" {
		t.Error("pinned address lost after re-discovery")
	}
}
