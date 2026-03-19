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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/config"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/diagnostics"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/discovery"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/stream"
)

// validServiceName matches safe service names: lowercase alphanumeric, hyphens, underscores.
var validServiceName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// commandPayload is the JSON body of a COMMAND stream sent by the support service.
type commandPayload struct {
	Action   string                        `json:"action"`
	Services map[string]models.ServiceInfo `json:"services,omitempty"`
}

// serviceStore is a goroutine-safe holder for the current service map.
type serviceStore struct {
	mu       sync.RWMutex
	services map[string]models.ServiceInfo
}

func newServiceStore(initial map[string]models.ServiceInfo) *serviceStore {
	return &serviceStore{services: initial}
}

func (s *serviceStore) get() map[string]models.ServiceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]models.ServiceInfo, len(s.services))
	for k, v := range s.services {
		result[k] = v
	}
	return result
}

func (s *serviceStore) set(m map[string]models.ServiceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services = m
}

func (s *serviceStore) merge(additional map[string]models.ServiceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range additional {
		s.services[k] = v
	}
}

// closeCodeSessionClosed matches the server's CloseCodeSessionClosed.
// When the operator closes a session, the server sends this code
// to tell the client to exit without reconnecting.
const closeCodeSessionClosed = 4000

// RunWithReconnect connects to the support service and reconnects on failure
// with exponential backoff.
func RunWithReconnect(ctx context.Context, cfg *config.ClientConfig) {
	delay := cfg.ReconnectDelay

	for {
		start := time.Now()
		err := connect(ctx, cfg)
		if ctx.Err() != nil {
			return // context cancelled, clean shutdown
		}

		// Check if the server sent a "session closed" close frame
		if websocket.IsCloseError(err, closeCodeSessionClosed) {
			log.Println("Session closed by operator. Exiting.")
			os.Exit(0)
		}

		log.Printf("Connection lost: %v", err)

		// Reset backoff if connection lasted longer than 60 seconds
		if time.Since(start) > 60*time.Second {
			delay = cfg.ReconnectDelay
		}

		log.Printf("Reconnecting in %v...", delay)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		// Exponential backoff
		delay = delay * 2
		if delay > cfg.MaxReconnectDelay {
			delay = cfg.MaxReconnectDelay
		}
	}
}

func connect(ctx context.Context, cfg *config.ClientConfig) error {
	// Build Basic Auth header
	creds := base64.StdEncoding.EncodeToString([]byte(cfg.Key + ":" + cfg.Secret))
	header := http.Header{}
	header.Set("Authorization", "Basic "+creds)

	// Append node_id query parameter for multi-node clusters
	connectURL := cfg.URL
	if cfg.NodeID != "" {
		parsed, err := url.Parse(connectURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		q := parsed.Query()
		q.Set("node_id", cfg.NodeID)
		parsed.RawQuery = q.Encode()
		connectURL = parsed.String()
	}

	// Start diagnostics collection in background (runs while connecting)
	diagCh := make(chan diagnostics.DiagnosticsReport, 1)
	go func() {
		report := diagnostics.Collect(cfg.DiagnosticsDir, cfg.DiagnosticsPluginTimeout)
		diagCh <- report
	}()

	log.Printf("Connecting to %s ...", connectURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.TLSInsecure, //nolint:gosec // Configurable: disabled by default, enable for dev/self-signed certs
		},
	}
	wsConn, _, err := dialer.Dial(connectURL, header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	log.Println("WebSocket connected")

	// Wrap as net.Conn
	netConn := &WsNetConn{conn: wsConn}

	// Create yamux client session
	yamuxCfg := yamux.DefaultConfig()
	yamuxCfg.EnableKeepAlive = true
	yamuxCfg.KeepAliveInterval = config.DefaultYamuxKeepAlive
	yamuxCfg.LogOutput = io.Discard

	session, err := yamux.Client(netConn, yamuxCfg)
	if err != nil {
		_ = wsConn.Close()
		return fmt.Errorf("yamux client creation failed: %w", err)
	}
	log.Println("yamux session established")

	// Discover services
	initialServices := discovery.DiscoverServices(ctx, cfg)
	store := newServiceStore(initialServices)

	// Send initial manifest
	if err := sendManifest(session, store.get()); err != nil {
		_ = session.Close()
		return fmt.Errorf("failed to send manifest: %w", err)
	}

	// Send diagnostics report if ready within the total timeout
	diagDeadline := time.NewTimer(cfg.DiagnosticsTotalTimeout)
	defer diagDeadline.Stop()
	select {
	case report := <-diagCh:
		if err := sendDiagnostics(session, report); err != nil {
			log.Printf("Failed to send diagnostics: %v", err)
		}
	case <-diagDeadline.C:
		log.Printf("Diagnostics timed out after %v, skipping", cfg.DiagnosticsTotalTimeout)
	}

	// Start periodic re-discovery
	go func() {
		ticker := time.NewTicker(cfg.DiscoveryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-session.CloseChan():
				return
			case <-ticker.C:
				newServices := discovery.DiscoverServices(ctx, cfg)
				if len(newServices) > 0 {
					if err := sendManifest(session, newServices); err != nil {
						log.Printf("Failed to send updated manifest: %v", err)
					} else {
						store.set(newServices)
						log.Printf("Manifest updated with %d services", len(newServices))
					}
				}
			}
		}
	}()

	// Close session when context is cancelled to unblock Accept()
	go func() {
		<-ctx.Done()
		_ = session.Close()
	}()

	// Accept incoming streams
	for {
		yamuxStream, err := session.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			// If the underlying WebSocket received a close frame, return that error
			// so the reconnect loop can inspect the close code
			netConn.mu.Lock()
			closeErr := netConn.closeErr
			netConn.mu.Unlock()
			if closeErr != nil {
				return closeErr
			}
			return fmt.Errorf("stream accept error: %w", err)
		}
		go func() {
			// Read the first line to determine stream type
			firstLine, lineErr := stream.ReadLine(yamuxStream)
			if lineErr != nil {
				_ = yamuxStream.Close()
				return
			}
			if strings.HasPrefix(firstLine, "COMMAND ") {
				handleCommandStream(yamuxStream, firstLine, store, session)
			} else {
				stream.HandleStreamWithFirstLine(yamuxStream, firstLine, store.get())
			}
		}()
	}
}

// handleCommandStream processes a COMMAND stream sent by the support service.
// It reads the JSON payload, applies the command, and writes OK or ERROR response.
func handleCommandStream(s net.Conn, firstLine string, store *serviceStore, session *yamux.Session) {
	defer func() { _ = s.Close() }()

	version := strings.TrimPrefix(firstLine, "COMMAND ")
	if version != "1" {
		log.Printf("Unsupported COMMAND version: %q", version)
		_, _ = fmt.Fprintf(s, "ERROR unsupported command version %q\n", version)
		return
	}

	// Read JSON payload (limit to 64 KB)
	var payload commandPayload
	dec := json.NewDecoder(io.LimitReader(s, 64*1024))
	if err := dec.Decode(&payload); err != nil {
		log.Printf("Failed to decode command payload: %v", err)
		_, _ = fmt.Fprintf(s, "ERROR invalid json: %v\n", err)
		return
	}

	switch payload.Action {
	case "add_services":
		if err := applyAddServices(payload.Services, store, session); err != nil {
			log.Printf("add_services failed: %v", err)
			_, _ = fmt.Fprintf(s, "ERROR %v\n", err)
			return
		}
		log.Printf("add_services: added %d static service(s)", len(payload.Services))
		_, _ = fmt.Fprint(s, "OK\n")
	default:
		log.Printf("Unknown command action: %q", payload.Action)
		_, _ = fmt.Fprintf(s, "ERROR unknown action %q\n", payload.Action)
	}
}

// applyAddServices validates and merges new static services into the store,
// then re-sends the manifest to the support service.
func applyAddServices(newSvcs map[string]models.ServiceInfo, store *serviceStore, session *yamux.Session) error {
	if len(newSvcs) == 0 {
		return fmt.Errorf("no services provided")
	}
	if len(newSvcs) > 10 {
		return fmt.Errorf("too many services: max 10 per call")
	}

	validated := make(map[string]models.ServiceInfo, len(newSvcs))
	for name, svc := range newSvcs {
		if !validServiceName.MatchString(name) {
			return fmt.Errorf("invalid service name %q: must match [a-z0-9][a-z0-9_-]{0,63}", name)
		}
		// Validate target format: must be host:port
		if svc.Target == "" {
			return fmt.Errorf("service %q has empty target", name)
		}
		validated[name] = svc
	}

	store.merge(validated)

	// Re-send manifest so the support service registers the new services
	if err := sendManifest(session, store.get()); err != nil {
		return fmt.Errorf("failed to resend manifest: %w", err)
	}
	return nil
}

func sendManifest(session *yamux.Session, services map[string]models.ServiceInfo) error {
	yamuxStream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open control stream: %w", err)
	}
	defer func() { _ = yamuxStream.Close() }()

	manifest := models.ServiceManifest{
		Version:  1,
		Services: services,
	}

	if err := json.NewEncoder(yamuxStream).Encode(manifest); err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	log.Printf("Manifest sent with %d services", len(services))
	return nil
}

func sendDiagnostics(session *yamux.Session, report diagnostics.DiagnosticsReport) error {
	yamuxStream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open diagnostics stream: %w", err)
	}
	defer func() { _ = yamuxStream.Close() }()

	// Write header line to identify this as a diagnostics stream
	if _, err := fmt.Fprintf(yamuxStream, "DIAGNOSTICS 1\n"); err != nil {
		return fmt.Errorf("failed to write diagnostics header: %w", err)
	}

	if err := json.NewEncoder(yamuxStream).Encode(report); err != nil {
		return fmt.Errorf("failed to encode diagnostics: %w", err)
	}

	log.Printf("Diagnostics sent: overall_status=%s, plugins=%d", report.OverallStatus, len(report.Plugins))
	return nil
}
