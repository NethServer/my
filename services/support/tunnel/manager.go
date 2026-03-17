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
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/logger"
)

// validHostname matches valid FQDN hostnames and IP addresses.
// Prevents hostname rewrite injection by rejecting special characters.
var validHostname = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9._:-]*[a-zA-Z0-9])?$`)

// ServiceInfo describes a service available through the tunnel
type ServiceInfo struct {
	Target     string `json:"target"`
	Host       string `json:"host"`
	TLS        bool   `json:"tls"`
	Label      string `json:"label"`
	Path       string `json:"path,omitempty"`        // Traefik route path (for display)
	PathPrefix string `json:"path_prefix,omitempty"` // Traefik PathPrefix to strip when proxying
	ModuleID   string `json:"module_id,omitempty"`   // NS8 module ID for grouping (e.g., "nethvoice103")
	NodeID     string `json:"node_id,omitempty"`     // NS8 node ID (e.g., "1", "2")
}

// ServiceManifest is the JSON manifest sent by the tunnel client
type ServiceManifest struct {
	Version  int                    `json:"version"`
	Services map[string]ServiceInfo `json:"services"`
}

// CloseCodeSessionClosed is the WebSocket close code sent when an operator closes a session.
// Tunnel clients receiving this code should exit without reconnecting.
const CloseCodeSessionClosed = 4000

// Tunnel represents an active WebSocket tunnel connection
type Tunnel struct {
	SystemID        string
	NodeID          string
	SessionID       string
	Session         *yamux.Session
	WsConn          WsCloser // underlying WebSocket for sending close frames
	ConnectedAt     time.Time
	done            chan struct{}
	closeOnce       sync.Once
	services        map[string]ServiceInfo
	servicesMu      sync.RWMutex
	activeStreams   int64
	activeStreamsMu sync.Mutex
	maxStreams      int
}

// WsCloser allows sending a WebSocket close frame with a status code and reason
type WsCloser interface {
	WriteControl(messageType int, data []byte, deadline time.Time) error
}

// TunnelKey builds the map key for a tunnel. For multi-node clusters,
// each node has its own tunnel keyed by "systemID:nodeID".
// Single-node systems use just "systemID".
func TunnelKey(systemID, nodeID string) string {
	if nodeID == "" {
		return systemID
	}
	return systemID + ":" + nodeID
}

// GraceExpiredCallback is called when a grace period expires without reconnection
type GraceExpiredCallback func(systemID, sessionID string)

// graceTimer tracks a pending grace period for a disconnected tunnel
type graceTimer struct {
	sessionID string
	timer     *time.Timer
}

// Manager manages active tunnel connections
type Manager struct {
	mu            sync.RWMutex
	tunnels       map[string]*Tunnel     // keyed by systemID
	graceTimers   map[string]*graceTimer // keyed by systemID
	graceCallback GraceExpiredCallback
	maxTunnels    int
	maxStreams    int
}

// NewManager creates a new tunnel manager with a maximum tunnel limit
func NewManager(maxTunnels, maxStreams int) *Manager {
	return &Manager{
		tunnels:     make(map[string]*Tunnel),
		graceTimers: make(map[string]*graceTimer),
		maxTunnels:  maxTunnels,
		maxStreams:  maxStreams,
	}
}

// SetGraceCallback sets the callback for grace period expiration
func (m *Manager) SetGraceCallback(cb GraceExpiredCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.graceCallback = cb
}

// Register adds a tunnel to the manager. Returns an error if the maximum tunnel limit is reached.
// nodeID identifies the cluster node; empty for single-node systems.
// wsConn is the underlying WebSocket connection used for sending close frames.
func (m *Manager) Register(systemID, nodeID, sessionID string, yamuxSession *yamux.Session, wsConn WsCloser) (*Tunnel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := TunnelKey(systemID, nodeID)

	// Enforce maximum tunnel limit (existing tunnel for this key doesn't count as it gets replaced)
	_, isReplacement := m.tunnels[key]
	if !isReplacement && m.maxTunnels > 0 && len(m.tunnels) >= m.maxTunnels {
		logger.ComponentLogger("tunnel_manager").Warn().
			Int("max_tunnels", m.maxTunnels).
			Int("current_tunnels", len(m.tunnels)).
			Str("system_id", systemID).
			Str("node_id", nodeID).
			Msg("maximum tunnel limit reached, rejecting connection")
		return nil, fmt.Errorf("maximum tunnel limit reached (%d)", m.maxTunnels)
	}

	// Cancel any pending grace period for this key
	if gt, ok := m.graceTimers[key]; ok {
		gt.timer.Stop()
		delete(m.graceTimers, key)
		logger.ComponentLogger("tunnel_manager").Info().
			Str("system_id", systemID).
			Str("node_id", nodeID).
			Str("session_id", gt.sessionID).
			Msg("grace period cancelled: system reconnected")
	}

	// Close existing tunnel for this key if any
	if existing, ok := m.tunnels[key]; ok {
		logger.ComponentLogger("tunnel_manager").Warn().
			Str("system_id", systemID).
			Str("node_id", nodeID).
			Str("old_session_id", existing.SessionID).
			Str("new_session_id", sessionID).
			Msg("replacing existing tunnel")
		existing.Close()
	}

	t := &Tunnel{
		SystemID:    systemID,
		NodeID:      nodeID,
		SessionID:   sessionID,
		Session:     yamuxSession,
		WsConn:      wsConn,
		ConnectedAt: time.Now(),
		done:        make(chan struct{}),
		maxStreams:  m.maxStreams,
	}
	if t.maxStreams == 0 {
		t.maxStreams = 64
	}

	m.tunnels[key] = t

	logger.ComponentLogger("tunnel_manager").Info().
		Str("system_id", systemID).
		Str("node_id", nodeID).
		Str("session_id", sessionID).
		Int("active_tunnels", len(m.tunnels)).
		Msg("tunnel registered")

	return t, nil
}

// Unregister removes a tunnel from the manager
func (m *Manager) Unregister(systemID, nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := TunnelKey(systemID, nodeID)
	if t, ok := m.tunnels[key]; ok {
		t.Close()
		delete(m.tunnels, key)

		logger.ComponentLogger("tunnel_manager").Info().
			Str("system_id", systemID).
			Str("node_id", nodeID).
			Str("session_id", t.SessionID).
			Msg("tunnel unregistered")
	}
}

// Get returns a tunnel by system ID and node ID
func (m *Manager) Get(systemID, nodeID string) *Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tunnels[TunnelKey(systemID, nodeID)]
}

// GetBySystemID returns the first tunnel for a system (any node).
// For single-node systems this is the only tunnel.
// For multi-node clusters this returns an arbitrary node's tunnel.
func (m *Manager) GetBySystemID(systemID string) *Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tunnels {
		if t.SystemID == systemID {
			return t
		}
	}
	return nil
}

// GetBySessionID returns a tunnel by session ID
func (m *Manager) GetBySessionID(sessionID string) *Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, t := range m.tunnels {
		if t.SessionID == sessionID {
			return t
		}
	}
	return nil
}

// Count returns the number of active tunnels
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tunnels)
}

// List returns info about all active tunnels
func (m *Manager) List() []TunnelInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]TunnelInfo, 0, len(m.tunnels))
	for _, t := range m.tunnels {
		infos = append(infos, TunnelInfo{
			SystemID:    t.SystemID,
			NodeID:      t.NodeID,
			SessionID:   t.SessionID,
			ConnectedAt: t.ConnectedAt,
		})
	}
	return infos
}

// CloseBySessionID closes a tunnel by session ID, sending a graceful close
// frame so the tunnel-client knows not to reconnect.
func (m *Manager) CloseBySessionID(sessionID string) bool {
	m.mu.Lock()
	var found *Tunnel
	for key, t := range m.tunnels {
		if t.SessionID == sessionID {
			found = t
			delete(m.tunnels, key)
			break
		}
	}
	m.mu.Unlock()

	if found == nil {
		return false
	}

	found.GracefulClose()
	logger.ComponentLogger("tunnel_manager").Info().
		Str("system_id", found.SystemID).
		Str("session_id", sessionID).
		Msg("tunnel gracefully closed by session ID")
	return true
}

// StartGracePeriod begins a grace period for a disconnected tunnel.
// If the system reconnects before the grace period expires, the timer is cancelled.
// If it expires, the callback is invoked to close the session.
func (m *Manager) StartGracePeriod(systemID, nodeID, sessionID string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := TunnelKey(systemID, nodeID)

	// Cancel any existing grace timer
	if gt, ok := m.graceTimers[key]; ok {
		gt.timer.Stop()
		delete(m.graceTimers, key)
	}

	timer := time.AfterFunc(duration, func() {
		m.mu.Lock()
		cb := m.graceCallback
		delete(m.graceTimers, key)
		m.mu.Unlock()

		logger.ComponentLogger("tunnel_manager").Info().
			Str("system_id", systemID).
			Str("node_id", nodeID).
			Str("session_id", sessionID).
			Msg("grace period expired: closing session")

		if cb != nil {
			cb(systemID, sessionID)
		}
	})

	m.graceTimers[key] = &graceTimer{
		sessionID: sessionID,
		timer:     timer,
	}

	logger.ComponentLogger("tunnel_manager").Info().
		Str("system_id", systemID).
		Str("node_id", nodeID).
		Str("session_id", sessionID).
		Dur("grace_period", duration).
		Msg("grace period started")
}

// HasGracePeriod returns true if there is an active grace period for this system+node
func (m *Manager) HasGracePeriod(systemID, nodeID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.graceTimers[TunnelKey(systemID, nodeID)]
	return ok
}

// CloseAll closes all active tunnels and cancels all grace timers
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for systemID, t := range m.tunnels {
		t.Close()
		delete(m.tunnels, systemID)
	}

	for systemID, gt := range m.graceTimers {
		gt.timer.Stop()
		delete(m.graceTimers, systemID)
	}

	logger.ComponentLogger("tunnel_manager").Info().Msg("all tunnels closed")
}

// Close closes the tunnel's yamux session (concurrency-safe via sync.Once)
func (t *Tunnel) Close() {
	t.closeOnce.Do(func() {
		close(t.done)
		if t.Session != nil {
			_ = t.Session.Close()
		}
	})
}

// GracefulClose sends a WebSocket close frame with CloseCodeSessionClosed
// before closing the tunnel. This tells the tunnel-client to exit without reconnecting.
func (t *Tunnel) GracefulClose() {
	if t.WsConn != nil {
		msg := websocket.FormatCloseMessage(CloseCodeSessionClosed, "session_closed")
		_ = t.WsConn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(5*time.Second))
		// Give the client a moment to process the close frame
		time.Sleep(100 * time.Millisecond)
	}
	t.Close()
}

// Done returns a channel that is closed when the tunnel is done
func (t *Tunnel) Done() <-chan struct{} {
	return t.done
}

// maxServicesPerManifest caps the number of services a tunnel client can advertise
const maxServicesPerManifest = 500

// SetServices updates the services available through this tunnel.
// Services with dangerous targets (cloud metadata, link-local) are rejected.
// The manifest is capped at maxServicesPerManifest entries.
func (t *Tunnel) SetServices(services map[string]ServiceInfo) {
	t.servicesMu.Lock()
	defer t.servicesMu.Unlock()

	validated := make(map[string]ServiceInfo, len(services))
	for name, svc := range services {
		if len(validated) >= maxServicesPerManifest {
			logger.ComponentLogger("tunnel_manager").Warn().
				Str("system_id", t.SystemID).
				Int("max", maxServicesPerManifest).
				Msg("service manifest truncated at max limit")
			break
		}
		if err := validateServiceTarget(svc.Target); err != nil {
			logger.ComponentLogger("tunnel_manager").Warn().
				Str("system_id", t.SystemID).
				Str("service", name).
				Str("target", svc.Target).
				Err(err).
				Msg("rejected service with dangerous target")
			continue
		}
		// Validate Host field to prevent hostname rewrite injection (#7):
		// a malicious Host value could hijack rewrite mappings for other services.
		if svc.Host != "" && !validHostname.MatchString(svc.Host) {
			logger.ComponentLogger("tunnel_manager").Warn().
				Str("system_id", t.SystemID).
				Str("service", name).
				Str("host", svc.Host).
				Msg("rejected service with invalid host value")
			continue
		}
		validated[name] = svc
	}
	t.services = validated
}

// dangerousHostnames contains cloud metadata and other dangerous hostnames
var dangerousHostnames = map[string]bool{
	"metadata.google.internal":     true,
	"metadata":                     true,
	"metadata.azure.internal":      true,
	"instance-data":                true, // Oracle Cloud
	"metadata.platformequinix.com": true,
}

// validateServiceTarget rejects targets pointing to dangerous addresses.
// For non-IP hostnames, DNS is resolved to block DNS rebinding attacks (#2).
func validateServiceTarget(target string) error {
	if target == "" {
		return fmt.Errorf("empty target")
	}

	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}

	// Block known dangerous hostnames
	if dangerousHostnames[strings.ToLower(host)] {
		return fmt.Errorf("cloud metadata hostname blocked: %s", host)
	}

	ip := net.ParseIP(host)
	if ip == nil {
		// Resolve DNS to prevent rebinding attacks: a hostname that resolves
		// to a benign IP at registration time could later resolve to a
		// dangerous IP (e.g., 169.254.169.254) when the operator connects.
		ips, lookupErr := net.LookupIP(host)
		if lookupErr != nil {
			return fmt.Errorf("DNS resolution failed for %s: %w", host, lookupErr)
		}
		for _, resolvedIP := range ips {
			if err := validateIP(resolvedIP); err != nil {
				return fmt.Errorf("hostname %s resolves to blocked address: %w", host, err)
			}
		}
		return nil
	}

	return validateIP(ip)
}

// validateIP checks a single IP address against blocked ranges
func validateIP(ip net.IP) error {
	// Block unspecified address (0.0.0.0, ::)
	if ip.IsUnspecified() {
		return fmt.Errorf("unspecified address blocked: %s", ip)
	}

	if ip.To4() != nil {
		// Block link-local (169.254.0.0/16) — cloud metadata lives here
		linkLocal := net.IPNet{IP: net.IPv4(169, 254, 0, 0), Mask: net.CIDRMask(16, 32)}
		if linkLocal.Contains(ip) {
			return fmt.Errorf("link-local/cloud metadata address blocked: %s", ip)
		}

		// Block multicast (224.0.0.0/4)
		multicast := net.IPNet{IP: net.IPv4(224, 0, 0, 0), Mask: net.CIDRMask(4, 32)}
		if multicast.Contains(ip) {
			return fmt.Errorf("multicast address blocked: %s", ip)
		}

		// Block broadcast
		if ip.Equal(net.IPv4bcast) {
			return fmt.Errorf("broadcast address blocked: %s", ip)
		}
	} else {
		// Block IPv6 link-local (fe80::/10)
		if ip.IsLinkLocalUnicast() {
			return fmt.Errorf("IPv6 link-local address blocked: %s", ip)
		}

		// Block IPv6 multicast (ff00::/8)
		if ip.IsMulticast() {
			return fmt.Errorf("IPv6 multicast address blocked: %s", ip)
		}
	}

	return nil
}

// GetService returns the service info for a given service name
func (t *Tunnel) GetService(name string) (ServiceInfo, bool) {
	t.servicesMu.RLock()
	defer t.servicesMu.RUnlock()
	svc, ok := t.services[name]
	return svc, ok
}

// GetServices returns all services available through this tunnel
func (t *Tunnel) GetServices() map[string]ServiceInfo {
	t.servicesMu.RLock()
	defer t.servicesMu.RUnlock()
	result := make(map[string]ServiceInfo, len(t.services))
	for k, v := range t.services {
		result[k] = v
	}
	return result
}

// AcquireStream increments the active stream count and returns true if within limits (#10).
func (t *Tunnel) AcquireStream() bool {
	t.activeStreamsMu.Lock()
	defer t.activeStreamsMu.Unlock()
	if t.maxStreams > 0 && int(t.activeStreams) >= t.maxStreams {
		return false
	}
	t.activeStreams++
	return true
}

// ReleaseStream decrements the active stream count.
func (t *Tunnel) ReleaseStream() {
	t.activeStreamsMu.Lock()
	defer t.activeStreamsMu.Unlock()
	if t.activeStreams > 0 {
		t.activeStreams--
	}
}

// TunnelInfo represents basic tunnel information
type TunnelInfo struct {
	SystemID    string    `json:"system_id"`
	NodeID      string    `json:"node_id,omitempty"`
	SessionID   string    `json:"session_id"`
	ConnectedAt time.Time `json:"connected_at"`
}
