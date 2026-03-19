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
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/config"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/diagnostics"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/discovery"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/stream"
)

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
	services := discovery.DiscoverServices(ctx, cfg)

	// Send initial manifest
	if err := sendManifest(session, services); err != nil {
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
						services = newServices
						log.Printf("Manifest updated with %d services", len(services))
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
		go stream.HandleStream(yamuxStream, services)
	}
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
