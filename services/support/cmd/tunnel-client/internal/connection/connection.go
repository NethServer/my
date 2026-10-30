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
	"errors"
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

// discovered returns the services that were not injected via COMMAND.
func (s *serviceStore) discovered() map[string]models.ServiceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]models.ServiceInfo, len(s.services))
	for k, v := range s.services {
		if _, ok := s.injected[k]; !ok {
			result[k] = v
		}
	}
	return result
}

func (s *serviceStore) len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.services)
}

// maxServicesTotal caps the total number of services in the tunnel-client store.
// Must match the server-side maxServicesPerManifest in tunnel/manager.go.
const maxServicesTotal = 500

// closeCodeSessionClosed matches the server's CloseCodeSessionClosed.
// When the operator closes a session, the server sends this code
// to tell the client to exit without reconnecting.
const closeCodeSessionClosed = 4000

const (
	// maxAuthFailures is how many consecutive 401/403 handshakes are tolerated
	// before giving up: the system credentials were revoked or rotated.
	maxAuthFailures = 3
	// maxConcurrentStreams caps the yamux streams handled at the same time.
	maxConcurrentStreams = 256
	// headerReadTimeout bounds the wait for the CONNECT/COMMAND header line.
	headerReadTimeout = 10 * time.Second
	// fetchTimeout and maxFetchSize bound a single USERS_FETCH exchange.
	fetchTimeout = 15 * time.Second
	maxFetchSize = 256 * 1024
)

// errSessionGone is returned when the server answers the handshake with 410:
// the session was closed or expired and must not be resumed.
var errSessionGone = errors.New("support session closed or expired (HTTP 410)")

type authError struct{ status int }

func (e *authError) Error() string {
	return fmt.Sprintf("authentication rejected (HTTP %d)", e.status)
}

var streamSlots = make(chan struct{}, maxConcurrentStreams)

// runState is the state carried across reconnections.
type runState struct {
	provisioned    bool
	provisioner    users.Provisioner
	owned          *users.SessionUsers // identities created by this client
	leftover       *users.SessionUsers // identities of a previous run still pending deletion
	sessionUsers   *users.SessionUsers // what is reported and passed to plugins
	lastServices   map[string]models.ServiceInfo
	reconnectToken string
}

// RunWithReconnect connects to the support service and reconnects on failure
// with exponential backoff. User provisioning happens once on first successful
// connection and cleanup happens when the session is permanently closed.
// leftover holds identities of a previous run whose deletion failed; they are
// kept in the state file and deleted together with this session's ones.
// Returns the process exit code.
func RunWithReconnect(ctx context.Context, cfg *config.ClientConfig, leftover *users.SessionUsers) int {
	delay := cfg.ReconnectDelay
	st := &runState{leftover: leftover}
	authFailures := 0

	defer st.teardown(cfg)

	for {
		start := time.Now()
		err := connect(ctx, cfg, st)
		if ctx.Err() != nil {
			return 0 // context cancelled, clean shutdown
		}

		// Check if the server sent a "session closed" close frame
		if websocket.IsCloseError(err, closeCodeSessionClosed) {
			log.Println("Session closed by operator. Exiting.")
			return 0
		}
		if errors.Is(err, errSessionGone) {
			log.Println("Session closed or expired on the server. Exiting.")
			return 0
		}
		var authErr *authError
		if errors.As(err, &authErr) {
			authFailures++
			if authFailures >= maxAuthFailures {
				log.Printf("Giving up after %d rejected handshakes: %v", authFailures, err)
				return 1
			}
		} else {
			authFailures = 0
		}

		log.Printf("Connection lost: %v", err)

		// Reset backoff if connection lasted longer than 60 seconds
		if time.Since(start) > 60*time.Second {
			delay = cfg.ReconnectDelay
		}

		log.Printf("Reconnecting in %v...", delay)

		select {
		case <-ctx.Done():
			return 0
		case <-time.After(delay):
		}

		// Exponential backoff
		delay = delay * 2
		if delay > cfg.MaxReconnectDelay {
			delay = cfg.MaxReconnectDelay
		}
	}
}

// teardown undoes plugin configuration and deletes every identity this client
// created. Identities whose deletion fails stay in the state file.
func (st *runState) teardown(cfg *config.ClientConfig) {
	if st.sessionUsers != nil {
		users.RunTeardown(context.Background(), cfg.UsersDir, st.sessionUsers, st.lastServices, cfg.RedisAddr, cfg.UsersPluginTimeout)
	}
	pending := users.MergeState(st.leftover, st.owned)
	if !users.HasIdentities(pending) {
		if err := users.RemoveState(cfg.UsersStateFile); err != nil {
			log.Printf("Failed to remove users state file: %v", err)
		}
		return
	}
	deleteErr := users.ProvisionerForPlatform(pending.Platform, cfg.RedisAddr).Delete(pending)
	if err := users.PersistState(cfg.UsersStateFile, pending); err != nil {
		log.Printf("Failed to update users state file: %v", err)
	}
	if deleteErr != nil {
		log.Printf("Support users cleanup incomplete, pending identities kept in %s for retry: %v", cfg.UsersStateFile, deleteErr)
		return
	}
	log.Println("Support users cleaned up")
}

// buildConnectURL appends the node_id and reconnect_token query parameters.
func buildConnectURL(base, nodeID, reconnectToken string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	q := parsed.Query()
	if nodeID != "" {
		q.Set("node_id", nodeID)
	}
	if reconnectToken != "" {
		q.Set("reconnect_token", reconnectToken)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String(), nil
}

// handshakeError maps a failed WebSocket handshake to a terminal error when
// the server's answer says retrying is pointless.
func handshakeError(resp *http.Response, err error) error {
	if resp != nil {
		switch resp.StatusCode {
		case http.StatusGone:
			return errSessionGone
		case http.StatusUnauthorized, http.StatusForbidden:
			return &authError{status: resp.StatusCode}
		}
	}
	return fmt.Errorf("websocket dial failed: %w", err)
}

// dialTunnel performs the WebSocket handshake, resuming the session with the
// reconnect token when one was issued, and stores the token the server returns.
func dialTunnel(ctx context.Context, cfg *config.ClientConfig, st *runState) (*websocket.Conn, error) {
	// Build Basic Auth header
	creds := base64.StdEncoding.EncodeToString([]byte(cfg.Key + ":" + cfg.Secret))
	header := http.Header{}
	header.Set("Authorization", "Basic "+creds)

	connectURL, err := buildConnectURL(cfg.URL, cfg.NodeID, st.reconnectToken)
	if err != nil {
		return nil, err
	}

	log.Printf("Connecting to %s (node_id=%q, resume=%v) ...", cfg.URL, cfg.NodeID, st.reconnectToken != "")

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.TLSInsecure && cfg.Dev, //nolint:gosec // Only with --dev, refused otherwise at startup
		},
	}
	wsConn, resp, err := dialer.DialContext(ctx, connectURL, header)
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, handshakeError(resp, err)
	}
	if token := resp.Header.Get("X-Reconnect-Token"); token != "" {
		st.reconnectToken = token
	}
	return wsConn, nil
}

func connect(ctx context.Context, cfg *config.ClientConfig, st *runState) error {
	wsConn, err := dialTunnel(ctx, cfg, st)
	if err != nil {
		return err
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
	if !st.provisioned {
		st.provisioner = users.NewProvisioner(cfg.RedisAddr)
		record := func(created *users.SessionUsers) error {
			return users.PersistState(cfg.UsersStateFile, users.MergeState(st.leftover, created))
		}
		su, provisionErr := st.provisioner.Create(cfg.Key, record)
		if provisionErr != nil {
			log.Printf("User provisioning failed: %v", provisionErr)
		}
		if users.HasIdentities(su) {
			st.owned = su
		}

		// NS8 worker nodes cannot create users: adopt the credentials the
		// leader created, fetched through the support service. They are
		// never recorded in the state file nor deleted by this client.
		if ns, ok := st.provisioner.(*users.NethServerProvisioner); ok && ns.FetchesFromLeader() && !users.HasIdentities(su) {
			log.Println("No local credentials created, fetching from server...")
			if fetched := users.SanitizeFetched(fetchUsersFromServer(ctx, session, 3, 10*time.Second), cfg.Key); fetched != nil {
				su = fetched
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
			st.lastServices = currentServices
			st.sessionUsers = su
		}
		st.provisioned = true

		// Send users report to support service
		if st.sessionUsers != nil {
			if sendErr := sendUsersReport(session, st.sessionUsers); sendErr != nil {
				log.Printf("Failed to send users report: %v", sendErr)
			}
		}
	} else if st.sessionUsers != nil {
		// Re-send users report on reconnect so the new session gets the data
		if sendErr := sendUsersReport(session, st.sessionUsers); sendErr != nil {
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
					// A module installed after session start (e.g. NethVoice
					// configured later) brings a new user-domain binding that
					// the frontend needs to show domain credentials. Refresh
					// users.ModuleDomains and resend the report when changed.
					if st.sessionUsers != nil &&
						users.RefreshModuleDomains(st.sessionUsers, newServices, cfg.RedisAddr) {
						if err := sendUsersReport(session, st.sessionUsers); err != nil {
							log.Printf("Failed to send updated users report: %v", err)
						} else {
							log.Printf("Users report updated (module_domains: %d)", len(st.sessionUsers.ModuleDomains))
						}
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
		select {
		case streamSlots <- struct{}{}:
		default:
			log.Printf("Rejecting stream: %d concurrent streams already open", maxConcurrentStreams)
			_ = yamuxStream.Close()
			continue
		}
		go func() {
			defer func() { <-streamSlots }()
			// Read the first line to determine stream type
			_ = yamuxStream.SetReadDeadline(time.Now().Add(headerReadTimeout))
			firstLine, lineErr := stream.ReadLine(yamuxStream)
			if lineErr != nil {
				_ = yamuxStream.Close()
				return
			}
			if strings.HasPrefix(firstLine, "COMMAND ") {
				handleCommandStream(yamuxStream, firstLine, store, session)
			} else {
				_ = yamuxStream.SetReadDeadline(time.Time{})
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

	discovered := store.discovered()
	validated := make(map[string]models.ServiceInfo, len(newSvcs))
	for name, svc := range newSvcs {
		if !validServiceName.MatchString(name) {
			return fmt.Errorf("invalid service name %q: must match [a-z0-9][a-z0-9_-]{0,63}", name)
		}
		if _, exists := discovered[name]; exists || name == "terminal" {
			return fmt.Errorf("service %q rejected: name already used by a discovered service", name)
		}
		dialAddr, err := resolveInjectedTarget(svc.Target, discovered)
		if err != nil {
			return fmt.Errorf("service %q rejected: %w", name, err)
		}
		svc.DialAddr = dialAddr
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
// where the leader hasn't connected yet. The result is untrusted and must go
// through users.SanitizeFetched.
func fetchUsersFromServer(ctx context.Context, session *yamux.Session, maxAttempts int, retryDelay time.Duration) *users.SessionUsers {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		su, retry := fetchUsersOnce(session)
		if su != nil || !retry {
			return su
		}
		if attempt < maxAttempts {
			log.Printf("No credentials available from server yet (attempt %d/%d), retrying in %v...", attempt, maxAttempts, retryDelay)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(retryDelay):
			}
		}
	}
	return nil
}

// fetchUsersOnce performs a single bounded USERS_FETCH exchange. retry is true
// when the server answered but had no credentials yet.
func fetchUsersOnce(session *yamux.Session) (su *users.SessionUsers, retry bool) {
	yamuxStream, err := session.Open()
	if err != nil {
		return nil, false
	}
	defer func() { _ = yamuxStream.Close() }()
	_ = yamuxStream.SetDeadline(time.Now().Add(fetchTimeout))

	if _, err := fmt.Fprintf(yamuxStream, "USERS_FETCH 1\n"); err != nil {
		return nil, false
	}

	data, readErr := io.ReadAll(io.LimitReader(yamuxStream, maxFetchSize+1))
	if readErr != nil || len(data) == 0 {
		return nil, false
	}
	if len(data) > maxFetchSize {
		log.Printf("Fetched users response exceeds %d bytes, ignoring", maxFetchSize)
		return nil, false
	}

	// Parse the response — it's a UsersReport JSON (same format as USERS 1)
	var report users.UsersReport
	if err := json.Unmarshal(data, &report); err != nil {
		log.Printf("Failed to parse fetched users: %v", err)
		return nil, false
	}

	if !users.HasIdentities(&report.Users) {
		return nil, true
	}

	log.Printf("Fetched credentials from server: cluster_admin=%v, domain_users=%d, local_users=%d",
		report.Users.ClusterAdmin != nil, len(report.Users.DomainUsers), len(report.Users.LocalUsers))
	return &report.Users, false
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
