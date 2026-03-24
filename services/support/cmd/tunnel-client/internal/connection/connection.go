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
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/users"
)

// validServiceName matches safe service names: lowercase alphanumeric, hyphens, underscores.
var validServiceName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// commandPayload is the JSON body of a COMMAND stream sent by the support service.
type commandPayload struct {
	Action       string                        `json:"action"`
	Services     map[string]models.ServiceInfo `json:"services,omitempty"`
	ServiceNames []string                      `json:"service_names,omitempty"` // for remove_services
}

// serviceStore is a goroutine-safe holder for the current service map.
// It tracks which services were injected via COMMAND so they survive re-discovery.
type serviceStore struct {
	mu       sync.RWMutex
	services map[string]models.ServiceInfo
	injected map[string]models.ServiceInfo // services added via COMMAND (preserved across re-discovery)
}

func newServiceStore(initial map[string]models.ServiceInfo) *serviceStore {
	return &serviceStore{
		services: initial,
		injected: make(map[string]models.ServiceInfo),
	}
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

// setDiscovered replaces discovered services but preserves injected (COMMAND) services.
func (s *serviceStore) setDiscovered(discovered map[string]models.ServiceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	merged := make(map[string]models.ServiceInfo, len(discovered)+len(s.injected))
	for k, v := range discovered {
		merged[k] = v
	}
	// Re-add injected services (COMMAND-added) that weren't in the new discovery
	for k, v := range s.injected {
		if _, exists := merged[k]; !exists {
			merged[k] = v
		}
	}
	s.services = merged
}

// addInjected adds services via COMMAND and marks them as injected.
func (s *serviceStore) addInjected(additional map[string]models.ServiceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range additional {
		s.services[k] = v
		s.injected[k] = v
	}
}

// removeInjected removes an injected service by name.
func (s *serviceStore) removeInjected(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.injected[name]; !ok {
		return false
	}
	delete(s.injected, name)
	delete(s.services, name)
	return true
}

func (s *serviceStore) len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.services)
}

// maxServicesTotal caps the total number of services in the tunnel-client store.
// Must match the server-side maxServicesPerManifest in tunnel/manager.go.
const maxServicesTotal = 500

// dangerousHostnames mirrors the server-side list in tunnel/manager.go.
var dangerousHostnames = map[string]bool{
	"metadata.google.internal":     true,
	"metadata":                     true,
	"metadata.azure.internal":      true,
	"instance-data":                true,
	"metadata.platformequinix.com": true,
}

// validateTarget rejects service targets pointing to dangerous addresses.
// Fix #1: mirrors the server-side validateServiceTarget in tunnel/manager.go so that
// COMMAND-injected services are SSRF-validated on the tunnel-client side as well.
func validateTarget(target string) error {
	if target == "" {
		return fmt.Errorf("empty target")
	}
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	if dangerousHostnames[strings.ToLower(host)] {
		return fmt.Errorf("cloud metadata hostname blocked: %s", host)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		ips, lookupErr := net.LookupIP(host)
		if lookupErr != nil {
			return fmt.Errorf("DNS resolution failed for %s: %w", host, lookupErr)
		}
		for _, resolvedIP := range ips {
			if err := validateTargetIP(resolvedIP); err != nil {
				return fmt.Errorf("hostname %s resolves to blocked address: %w", host, err)
			}
		}
		return nil
	}
	return validateTargetIP(ip)
}

func validateTargetIP(ip net.IP) error {
	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified address blocked: %s", ip)
	}
	if ip.To4() != nil {
		linkLocal := net.IPNet{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}
		if linkLocal.Contains(ip) {
			return fmt.Errorf("link-local/cloud metadata address blocked: %s", ip)
		}
		multicast := net.IPNet{IP: net.IPv4(224, 0, 0, 0), Mask: net.CIDRMask(4, 32)}
		if multicast.Contains(ip) {
			return fmt.Errorf("multicast address blocked: %s", ip)
		}
		if ip.Equal(net.IPv4bcast) {
			return fmt.Errorf("broadcast address blocked: %s", ip)
		}
	} else {
		if ip.IsLinkLocalUnicast() {
			return fmt.Errorf("IPv6 link-local address blocked: %s", ip)
		}
		if ip.IsMulticast() {
			return fmt.Errorf("IPv6 multicast address blocked: %s", ip)
		}
	}
	return nil
}

// closeCodeSessionClosed matches the server's CloseCodeSessionClosed.
// When the operator closes a session, the server sends this code
// to tell the client to exit without reconnecting.
const closeCodeSessionClosed = 4000

// RunWithReconnect connects to the support service and reconnects on failure
// with exponential backoff. User provisioning happens once on first successful
// connection and cleanup happens when the session is permanently closed.
func RunWithReconnect(ctx context.Context, cfg *config.ClientConfig) {
	delay := cfg.ReconnectDelay
	var sessionUsers *users.SessionUsers
	var provisioner users.Provisioner
	var lastServices map[string]models.ServiceInfo
	usersProvisioned := false

	// Cleanup users on final exit (context cancel or session closed)
	defer func() {
		if sessionUsers != nil {
			cleanupCtx := context.Background()
			users.RunTeardown(cleanupCtx, cfg.UsersDir, sessionUsers, lastServices, cfg.RedisAddr, cfg.UsersPluginTimeout)
			_ = provisioner.Delete(sessionUsers)
			users.RemoveState(cfg.UsersStateFile)
			log.Println("Support users cleaned up")
		}
	}()

	for {
		start := time.Now()
		err := connect(ctx, cfg, &sessionUsers, &provisioner, &usersProvisioned, &lastServices)
		if ctx.Err() != nil {
			return // context cancelled, clean shutdown
		}

		// Check if the server sent a "session closed" close frame
		if websocket.IsCloseError(err, closeCodeSessionClosed) {
			log.Println("Session closed by operator. Exiting.")
			return
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

func connect(ctx context.Context, cfg *config.ClientConfig, sessionUsers **users.SessionUsers, provisioner *users.Provisioner, usersProvisioned *bool, lastServices *map[string]models.ServiceInfo) error {
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

	// Fix #10: start diagnostics collection only after the connection is established.
	// Running plugins during a failed dial attempt wastes resources and slows reconnect loops.
	diagCh := make(chan diagnostics.DiagnosticsReport, 1)
	go func() {
		report := diagnostics.Collect(cfg.DiagnosticsDir, cfg.DiagnosticsPluginTimeout)
		diagCh <- report
	}()

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

	// Provision ephemeral support users (only on first successful connection)
	if !*usersProvisioned {
		*provisioner = users.NewProvisioner(cfg.RedisAddr)
		su, provisionErr := (*provisioner).Create(cfg.Key)
		if provisionErr != nil {
			log.Printf("User provisioning failed: %v", provisionErr)
		}

		// If provisioner didn't create credentials (e.g., worker node),
		// fetch them from the server (created by the leader node)
		hasCredentials := su != nil && (su.ClusterAdmin != nil || len(su.DomainUsers) > 0 || len(su.LocalUsers) > 0)
		if !hasCredentials {
			log.Println("No local credentials created, fetching from server...")
			fetched := fetchUsersFromServer(session, 3, 10*time.Second)
			if fetched != nil {
				if su == nil {
					su = fetched
				} else {
					// Merge fetched credentials into the local result
					su.ClusterAdmin = fetched.ClusterAdmin
					su.DomainUsers = fetched.DomainUsers
					su.LocalUsers = fetched.LocalUsers
				}
			}
		}

		if su != nil {
			// Run users.d/ setup plugins, passing discovered services for module context
			setupCtx, setupCancel := context.WithTimeout(ctx, cfg.UsersTotalTimeout)
			currentServices := store.get()
			apps, pluginErrors := users.RunSetup(setupCtx, cfg.UsersDir, su, currentServices, cfg.RedisAddr, cfg.UsersPluginTimeout)
			setupCancel()
			su.Apps = apps
			su.Errors = pluginErrors
			*lastServices = currentServices

			// Save state for crash recovery
			if stateErr := users.SaveState(cfg.UsersStateFile, su); stateErr != nil {
				log.Printf("Failed to save users state: %v", stateErr)
			}

			*sessionUsers = su
		}
		*usersProvisioned = true

		// Send users report to support service
		if *sessionUsers != nil {
			if sendErr := sendUsersReport(session, *sessionUsers); sendErr != nil {
				log.Printf("Failed to send users report: %v", sendErr)
			}
		}
	} else if *sessionUsers != nil {
		// Re-send users report on reconnect so the new session gets the data
		if sendErr := sendUsersReport(session, *sessionUsers); sendErr != nil {
			log.Printf("Failed to re-send users report: %v", sendErr)
		}
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
					store.setDiscovered(newServices)
					if err := sendManifest(session, store.get()); err != nil {
						log.Printf("Failed to send updated manifest: %v", err)
					} else {
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
	case "remove_services":
		if len(payload.ServiceNames) == 0 {
			_, _ = fmt.Fprint(s, "ERROR no service names provided\n")
			return
		}
		removed := 0
		for _, name := range payload.ServiceNames {
			if store.removeInjected(name) {
				removed++
			}
		}
		if removed > 0 {
			if err := sendManifest(session, store.get()); err != nil {
				log.Printf("Failed to resend manifest after remove: %v", err)
			}
		}
		log.Printf("remove_services: removed %d service(s)", removed)
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
	// Fix #9: enforce total services cap to prevent unbounded store growth
	// via repeated add_services calls.
	if store.len()+len(newSvcs) > maxServicesTotal {
		return fmt.Errorf("service limit exceeded: max %d total services", maxServicesTotal)
	}

	validated := make(map[string]models.ServiceInfo, len(newSvcs))
	for name, svc := range newSvcs {
		if !validServiceName.MatchString(name) {
			return fmt.Errorf("invalid service name %q: must match [a-z0-9][a-z0-9_-]{0,63}", name)
		}
		// Fix #1: SSRF validation on injected targets — mirrors server-side validateServiceTarget.
		if err := validateTarget(svc.Target); err != nil {
			return fmt.Errorf("service %q rejected: %w", name, err)
		}
		validated[name] = svc
	}

	store.addInjected(validated)

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

// fetchUsersFromServer asks the support service for credentials already created
// by another node (e.g., the leader). Returns nil if no credentials are available.
// Retries up to maxAttempts with a delay between attempts to handle the case
// where the leader hasn't connected yet.
func fetchUsersFromServer(session *yamux.Session, maxAttempts int, retryDelay time.Duration) *users.SessionUsers {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		yamuxStream, err := session.Open()
		if err != nil {
			return nil
		}

		if _, err := fmt.Fprintf(yamuxStream, "USERS_FETCH 1\n"); err != nil {
			_ = yamuxStream.Close()
			return nil
		}

		// Read response (up to 256 KB)
		data, readErr := io.ReadAll(io.LimitReader(yamuxStream, 256*1024))
		_ = yamuxStream.Close()

		if readErr != nil || len(data) == 0 {
			return nil
		}

		// Parse the response — it's a UsersReport JSON (same format as USERS 1)
		var report users.UsersReport
		if err := json.Unmarshal(data, &report); err != nil {
			// Try parsing as empty object
			if string(data) == "{}\n" || string(data) == "{}" {
				if attempt < maxAttempts {
					log.Printf("No credentials available from server yet (attempt %d/%d), retrying in %v...", attempt, maxAttempts, retryDelay)
					time.Sleep(retryDelay)
					continue
				}
				return nil
			}
			log.Printf("Failed to parse fetched users: %v", err)
			return nil
		}

		// Check if there are actual credentials
		su := &report.Users
		if su.ClusterAdmin == nil && len(su.DomainUsers) == 0 && len(su.LocalUsers) == 0 {
			if attempt < maxAttempts {
				log.Printf("No credentials available from server yet (attempt %d/%d), retrying in %v...", attempt, maxAttempts, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return nil
		}

		log.Printf("Fetched credentials from server: cluster_admin=%v, domain_users=%d, local_users=%d",
			su.ClusterAdmin != nil, len(su.DomainUsers), len(su.LocalUsers))
		return su
	}
	return nil
}

func sendUsersReport(session *yamux.Session, sessionUsers *users.SessionUsers) error {
	yamuxStream, err := session.Open()
	if err != nil {
		return fmt.Errorf("failed to open users stream: %w", err)
	}
	defer func() { _ = yamuxStream.Close() }()

	// Write header line to identify this as a users stream
	if _, err := fmt.Fprintf(yamuxStream, "USERS 1\n"); err != nil {
		return fmt.Errorf("failed to write users header: %w", err)
	}

	report := users.UsersReport{
		CreatedAt:  sessionUsers.CreatedAt,
		DurationMs: time.Since(sessionUsers.CreatedAt).Milliseconds(),
		Users:      *sessionUsers,
	}

	if err := json.NewEncoder(yamuxStream).Encode(report); err != nil {
		return fmt.Errorf("failed to encode users report: %w", err)
	}

	log.Printf("Users report sent: platform=%s, domain_users=%d, local_users=%d, apps=%d",
		sessionUsers.Platform, len(sessionUsers.DomainUsers), len(sessionUsers.LocalUsers), len(sessionUsers.Apps))
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
