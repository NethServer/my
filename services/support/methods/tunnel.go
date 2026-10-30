/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"bufio"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/models"
	"github.com/nethesis/my/services/support/session"
	"github.com/nethesis/my/services/support/tunnel"
)

// nodeIDPattern validates node_id query parameter (numeric, max 10 digits)
var nodeIDPattern = regexp.MustCompile(`^[0-9]{1,10}$`)

// reconnectTokenPattern matches tokens produced by session.GenerateToken
var reconnectTokenPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

var (
	// TunnelManager is the global tunnel manager instance
	TunnelManager *tunnel.Manager

	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Only non-browser clients (no Origin header) connect to the tunnel.
			// Reject browser-originated requests to prevent CSRF with cached credentials.
			return r.Header.Get("Origin") == ""
		},
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}
)

// HandleTunnel handles WebSocket tunnel connections from systems.
// A client reconnecting with the reconnect_token of its live session keeps
// that session; without a valid token the old session is replaced; a token
// of a session that has ended is answered with 410 Gone (terminal for clients).
func HandleTunnel(c *gin.Context) {
	systemID, exists := c.Get("system_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "system not authenticated"})
		return
	}

	sysID := systemID.(string)
	nodeID := c.Query("node_id")

	// Validate node_id format to prevent memory abuse from crafted values (#20)
	if nodeID != "" {
		if len(nodeID) > 10 || !nodeIDPattern.MatchString(nodeID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node_id format"})
			return
		}
	}

	log := logger.RequestLogger(c, "tunnel")

	reconnectToken := c.Query("reconnect_token")
	if !reconnectTokenPattern.MatchString(reconnectToken) {
		reconnectToken = ""
	}

	existing, err := session.GetActiveSession(sysID, nodeID)
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Str("node_id", nodeID).Msg("failed to get session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
		return
	}

	decision, err := decideTunnelSession(existing, reconnectToken, func() (bool, error) {
		return session.IsReconnectTokenEnded(sysID, reconnectToken)
	})
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Msg("failed to check reconnect token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
		return
	}
	if decision == sessionGone {
		log.Info().Str("system_id", sysID).Str("node_id", nodeID).Msg("reconnect refused: session has ended")
		c.JSON(http.StatusGone, gin.H{"error": "support session has ended"})
		return
	}

	var sess *models.SupportSession
	if decision == sessionResume {
		sess = existing
	} else {
		ok, capErr := session.HasSessionCapacity(sysID, nodeID)
		if capErr != nil {
			log.Error().Err(capErr).Str("system_id", sysID).Msg("failed to check session capacity")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
			return
		}
		if !ok {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "maximum active sessions per system reached"})
			return
		}
	}

	responseToken := ""
	if sess != nil {
		responseToken = sess.ReconnectToken
	} else if responseToken, err = session.GenerateToken(); err != nil {
		log.Error().Err(err).Msg("failed to generate reconnect token")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	// Upgrade to WebSocket before any session row is created
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, http.Header{"X-Reconnect-Token": []string{responseToken}})
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Msg("websocket upgrade failed")
		return
	}

	created := false
	if sess == nil {
		if decision == sessionReplace {
			log.Warn().
				Str("system_id", sysID).
				Str("node_id", nodeID).
				Str("session_id", existing.ID).
				Msg("missing or invalid reconnect token, replacing existing session")
			if err := session.CloseSession(existing.ID, "replaced"); err != nil {
				log.Warn().Err(err).Str("session_id", existing.ID).Msg("failed to close replaced session")
			}
			TunnelManager.CloseBySessionID(existing.ID)
		}
		sess, err = session.CreateSession(sysID, nodeID, responseToken)
		if err != nil {
			log.Error().Err(err).Str("system_id", sysID).Str("node_id", nodeID).Msg("failed to create session")
			closeWebSocket(wsConn, websocket.CloseTryAgainLater, "session unavailable")
			return
		}
		created = true
	}
	abandon := func() {
		if created {
			_ = session.CloseSession(sess.ID, "disconnect")
		}
	}

	if key, ok := c.Get("system_key"); ok {
		systemKeys.Store(key.(string), sysID)
	}

	log.Info().
		Str("system_id", sysID).
		Str("node_id", nodeID).
		Str("session_id", sess.ID).
		Bool("resumed", !created).
		Str("remote_addr", c.Request.RemoteAddr).
		Msg("websocket connection established")

	// Wrap WebSocket as net.Conn for yamux
	wsNetConn := tunnel.NewWebSocketConn(wsConn)

	// Create yamux server session over the WebSocket connection
	// #11: Explicit keepalive, write timeout and resource limits
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 15 * time.Second
	yamuxConfig.ConnectionWriteTimeout = 10 * time.Second
	yamuxConfig.MaxStreamWindowSize = 256 * 1024
	yamuxConfig.AcceptBacklog = 64
	yamuxConfig.LogOutput = io.Discard

	yamuxSession, err := yamux.Server(wsNetConn, yamuxConfig)
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Msg("yamux session creation failed")
		_ = wsConn.Close()
		abandon()
		return
	}

	// Register the tunnel
	t, regErr := TunnelManager.Register(sysID, nodeID, sess.ID, yamuxSession, wsConn)
	if regErr != nil {
		log.Error().Err(regErr).Str("system_id", sysID).Str("node_id", nodeID).Msg("failed to register tunnel")
		_ = yamuxSession.Close()
		abandon()
		return
	}

	// Activate the session in the database
	if err := session.ActivateSession(sess.ID); err != nil {
		log.Error().Err(err).Str("session_id", sess.ID).Msg("failed to activate session")
	}

	// The session may have been closed (grace expiry, operator, revocation)
	// between the token check and the registration
	if current, err := session.GetSessionByID(sess.ID); err != nil || current == nil || current.Status != "active" {
		log.Warn().Str("session_id", sess.ID).Msg("session no longer active after registration, closing tunnel")
		t.GracefulClose()
		TunnelManager.Unregister(t)
		return
	}

	// Accept the control stream (first stream from client with service manifest)
	go acceptControlStream(t, sysID, sess.ID)

	// Handle the tunnel lifecycle in a goroutine
	go handleTunnelLifecycle(t)
}

type sessionDecision int

const (
	sessionNew     sessionDecision = iota // no live session: create one
	sessionResume                         // live session + matching token: keep it
	sessionReplace                        // live session, missing/invalid token: close it and create one
	sessionGone                           // token of an ended session: 410, client stops
)

// decideTunnelSession applies the reconnect rules to the live session of a
// system+node (nil if none) and the token presented by the client.
func decideTunnelSession(existing *models.SupportSession, token string, tokenEnded func() (bool, error)) (sessionDecision, error) {
	if session.ReconnectTokenMatches(existing, token) {
		return sessionResume, nil
	}
	if token != "" {
		ended, err := tokenEnded()
		if err != nil {
			return sessionNew, err
		}
		if ended {
			return sessionGone, nil
		}
	}
	if existing != nil {
		return sessionReplace, nil
	}
	return sessionNew, nil
}

// closeWebSocket sends a close frame and closes the connection
func closeWebSocket(wsConn *websocket.Conn, code int, reason string) {
	msg := websocket.FormatCloseMessage(code, reason)
	_ = wsConn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(5*time.Second))
	_ = wsConn.Close()
}

// systemKeys maps the system_key of each tunnel-connected system to its id,
// so a revocation still finds the tunnels of a hard-deleted system.
var systemKeys sync.Map

// RevokeSystem closes every live session and tunnel of a system whose
// credentials have been invalidated (deleted, suspended, secret rotated).
func RevokeSystem(systemKey string) {
	log := logger.ComponentLogger("tunnel")

	sysID := ""
	if v, ok := systemKeys.Load(systemKey); ok {
		sysID = v.(string)
	}
	if sysID == "" {
		id, err := session.SystemIDByKey(systemKey)
		if err != nil {
			log.Warn().Err(err).Str("system_key", systemKey).Msg("failed to resolve system for revocation")
			return
		}
		sysID = id
	}
	if sysID == "" {
		return
	}

	ids, err := session.CloseSessionsBySystem(sysID, "system")
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Msg("failed to close sessions of revoked system")
	}
	closed := TunnelManager.CloseBySystemID(sysID)
	if len(ids) > 0 || closed > 0 {
		log.Info().
			Str("system_id", sysID).
			Int("sessions_closed", len(ids)).
			Int("tunnels_closed", closed).
			Msg("support access revoked after auth invalidation")
	}
}

const (
	// maxControlStreams caps concurrent control streams handled per tunnel
	maxControlStreams = 8
	// controlStreamTimeout bounds how long a control stream may stay silent
	controlStreamTimeout = 30 * time.Second
)

// controlState holds the per-tunnel rate-limit timestamps shared by control stream handlers
type controlState struct {
	mu              sync.Mutex
	lastManifest    time.Time
	lastDiagnostics time.Time // fast in-memory pre-check; DB enforces cross-reconnect rate limit
}

// acceptControlStream accepts control streams from the tunnel client.
// Each stream carries either a service manifest (raw JSON starting with '{')
// or a diagnostics report (header line "DIAGNOSTICS 1\n" followed by JSON).
// Streams are handled concurrently up to maxControlStreams; extra ones are dropped.
func acceptControlStream(t *tunnel.Tunnel, systemID, sessionID string) {
	sem := make(chan struct{}, maxControlStreams)
	state := &controlState{}

	for {
		stream, err := t.Session.Accept()
		if err != nil {
			return // session closed
		}
		select {
		case sem <- struct{}{}:
		default:
			_ = stream.Close()
			continue
		}
		go func() {
			defer func() { <-sem }()
			defer func() { _ = stream.Close() }()
			_ = stream.SetDeadline(time.Now().Add(controlStreamTimeout))
			handleControlStream(t, stream, systemID, sessionID, state)
		}()
	}
}

// handleControlStream processes a single control stream
func handleControlStream(t *tunnel.Tunnel, stream net.Conn, systemID, sessionID string, state *controlState) {
	log := logger.ComponentLogger("tunnel")

	// Wrap stream with a size-limited buffered reader (1 MB overall limit)
	br := bufio.NewReaderSize(io.LimitReader(stream, 1<<20), 4096)

	firstByte, err := br.ReadByte()
	if err != nil {
		return
	}

	switch firstByte {
	case 'U':
		// Read the rest of the header line (br already consumed the 'U')
		rest, _ := br.ReadString('\n')
		headerParts := strings.Fields(strings.TrimSpace("U" + rest))
		if len(headerParts) != 2 || headerParts[1] != "1" {
			return
		}
		switch headerParts[0] {
		case "USERS_FETCH":
			handleUsersFetch(t, stream, systemID, sessionID)
		case "USERS":
			handleUsersReport(br, systemID, sessionID)
		}
		return

	case 'D':
		// Read the rest of the header line (br already consumed the 'D')
		rest, _ := br.ReadString('\n')
		// Fix #8: validate exact version to reject unknown protocol versions
		diagParts := strings.Fields(strings.TrimSpace("D" + rest))
		if len(diagParts) == 2 && diagParts[0] == "DIAGNOSTICS" && diagParts[1] == "1" {
			handleDiagnostics(br, systemID, sessionID, state)
		}
		return
	}

	// Manifest path: put the peeked byte back so the JSON decoder sees it
	_ = br.UnreadByte()

	// Rate-limit manifest updates (max 1 per 10 seconds, first always accepted)
	state.mu.Lock()
	limited := !state.lastManifest.IsZero() && time.Since(state.lastManifest) < 10*time.Second
	state.mu.Unlock()
	if limited {
		return
	}

	// Decode manifest with the buffered reader (preserves already-read bytes)
	var manifest tunnel.ServiceManifest
	if err := json.NewDecoder(br).Decode(&manifest); err != nil {
		log.Warn().Err(err).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to decode service manifest from control stream")
		return
	}

	if manifest.Services != nil {
		t.SetServices(manifest.Services)
		state.mu.Lock()
		state.lastManifest = time.Now()
		state.mu.Unlock()
		log.Info().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Int("service_count", len(manifest.Services)).
			Msg("service manifest received")
	}
}

// handleUsersFetch answers USERS_FETCH 1: a worker node asks for the
// credentials created by another node of the same system. Anything but a
// registered tunnel of an active multi-node session gets an empty answer.
func handleUsersFetch(t *tunnel.Tunnel, stream net.Conn, systemID, sessionID string) {
	log := logger.ComponentLogger("tunnel")
	answer := []byte("{}\n")

	if t.NodeID != "" && TunnelManager.IsRegistered(t) {
		usersData, err := session.GetUsersForNode(systemID, sessionID)
		if err != nil {
			log.Warn().Err(err).
				Str("system_id", systemID).
				Str("session_id", sessionID).
				Msg("failed to fetch users for system")
		} else if usersData != nil {
			answer = append(usersData, '\n')
			log.Info().
				Str("system_id", systemID).
				Str("session_id", sessionID).
				Str("node_id", t.NodeID).
				Int("payload_bytes", len(usersData)).
				Msg("users credentials sent to requesting node")
		}
	}
	_, _ = stream.Write(answer)
}

// handleUsersReport stores the credentials sent with USERS 1
func handleUsersReport(br *bufio.Reader, systemID, sessionID string) {
	log := logger.ComponentLogger("tunnel")

	rawJSON, readErr := io.ReadAll(br)
	if readErr != nil {
		log.Warn().Err(readErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to read users payload from control stream")
		return
	}

	// Reject payloads larger than 256 KB
	if len(rawJSON) > 256*1024 {
		log.Warn().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Int("bytes", len(rawJSON)).
			Msg("users payload exceeds 256 KB limit, skipping")
		return
	}

	// Schema validation: unmarshal into typed struct and re-serialize
	var report models.UsersReport
	if parseErr := json.Unmarshal(rawJSON, &report); parseErr != nil {
		log.Warn().Err(parseErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("invalid users JSON schema, skipping")
		return
	}
	sanitized, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		log.Warn().Err(marshalErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to re-serialize users, skipping")
		return
	}

	saved, jsonErr := session.SaveUsers(sessionID, sanitized)
	if jsonErr != nil {
		log.Warn().Err(jsonErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to save users")
	} else if !saved {
		log.Debug().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("users update skipped: already present")
	} else {
		log.Info().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Int("payload_bytes", len(rawJSON)).
			Msg("users report received")
	}
}

// handleDiagnostics stores the report sent with DIAGNOSTICS 1
func handleDiagnostics(br *bufio.Reader, systemID, sessionID string, state *controlState) {
	log := logger.ComponentLogger("tunnel")

	// Fast in-memory rate-limit (pre-check before hitting the DB)
	state.mu.Lock()
	limited := !state.lastDiagnostics.IsZero() && time.Since(state.lastDiagnostics) < 30*time.Second
	state.mu.Unlock()
	if limited {
		return
	}

	// Read remaining bytes as JSON
	rawJSON, readErr := io.ReadAll(br)
	if readErr != nil {
		log.Warn().Err(readErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to read diagnostics payload from control stream")
		return
	}

	// Fix #4: explicit size guard — reject payloads larger than 512 KB
	if len(rawJSON) > 512*1024 {
		log.Warn().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Int("bytes", len(rawJSON)).
			Msg("diagnostics payload exceeds 512 KB limit, skipping")
		return
	}

	// Fix #4: schema validation — unmarshal into typed struct and re-serialize
	// to reject malformed JSON and strip unknown fields (prevents stored XSS).
	var report models.DiagnosticsReport
	if parseErr := json.Unmarshal(rawJSON, &report); parseErr != nil {
		log.Warn().Err(parseErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("invalid diagnostics JSON schema, skipping")
		return
	}
	sanitized, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		log.Warn().Err(marshalErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to re-serialize diagnostics, skipping")
		return
	}

	// Fix #5: SaveDiagnostics enforces the rate limit in the DB
	// to handle reconnect-based bypass of the in-memory check above.
	saved, jsonErr := session.SaveDiagnostics(sessionID, sanitized)
	if jsonErr != nil {
		log.Warn().Err(jsonErr).
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("failed to save diagnostics")
	} else if !saved {
		log.Debug().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Msg("diagnostics update skipped: rate-limited")
	} else {
		state.mu.Lock()
		state.lastDiagnostics = time.Now()
		state.mu.Unlock()
		log.Info().
			Str("system_id", systemID).
			Str("session_id", sessionID).
			Int("payload_bytes", len(rawJSON)).
			Msg("diagnostics received")
	}
}

func handleTunnelLifecycle(t *tunnel.Tunnel) {
	log := logger.ComponentLogger("tunnel")

	// Wait for the yamux session to close (either side disconnects)
	<-t.Session.CloseChan()

	// A tunnel that was replaced or closed on purpose must not start a grace
	// period: that would interfere with the tunnel now registered for the key
	if !TunnelManager.Unregister(t) {
		log.Debug().
			Str("system_id", t.SystemID).
			Str("node_id", t.NodeID).
			Str("session_id", t.SessionID).
			Msg("tunnel closed after being replaced or revoked")
		return
	}

	log.Info().
		Str("system_id", t.SystemID).
		Str("node_id", t.NodeID).
		Str("session_id", t.SessionID).
		Msg("tunnel disconnected, starting grace period")

	// Start a grace period instead of immediately closing the session.
	// A client reconnecting with its reconnect token before it expires
	// resumes the still-active session.
	TunnelManager.StartGracePeriod(t.SystemID, t.NodeID, t.SessionID, configuration.Config.TunnelGracePeriod)
}
