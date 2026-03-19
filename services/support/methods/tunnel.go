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
	"net/http"
	"regexp"
	"strings"
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

// HandleTunnel handles WebSocket tunnel connections from systems
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

	// #8: Check for reconnect token when reusing an existing session during grace period
	reconnectToken := c.Query("reconnect_token")

	// Create or reuse a session for this system+node
	sess, err := session.GetActiveSession(sysID, nodeID)
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Str("node_id", nodeID).Msg("failed to get session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get session"})
		return
	}

	if sess != nil && TunnelManager.HasGracePeriod(sysID, nodeID) {
		// Session exists during grace period — validate reconnect token (#8)
		if !session.ValidateReconnectToken(sess.ID, reconnectToken) {
			log.Warn().
				Str("system_id", sysID).
				Str("node_id", nodeID).
				Str("session_id", sess.ID).
				Msg("reconnect token mismatch during grace period, closing old session")
			// Close the old session to prevent orphaned active sessions
			if err := session.CloseSession(sess.ID, "replaced"); err != nil {
				log.Warn().Err(err).Str("session_id", sess.ID).Msg("failed to close replaced session")
			}
			sess = nil // force new session
		}
	}

	if sess == nil {
		// Create a new session
		sess, err = session.CreateSession(sysID, nodeID)
		if err != nil {
			log.Error().Err(err).Str("system_id", sysID).Str("node_id", nodeID).Msg("failed to create session")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
			return
		}
	}

	// Upgrade to WebSocket
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Msg("websocket upgrade failed")
		return
	}

	log.Info().
		Str("system_id", sysID).
		Str("node_id", nodeID).
		Str("session_id", sess.ID).
		Str("remote_addr", c.Request.RemoteAddr).
		Msg("websocket connection established")

	// Wrap WebSocket as net.Conn for yamux
	wsNetConn := tunnel.NewWebSocketConn(wsConn)

	// Create yamux server session over the WebSocket connection
	// #11: Explicit keepalive and write timeout configuration
	yamuxConfig := yamux.DefaultConfig()
	yamuxConfig.EnableKeepAlive = true
	yamuxConfig.KeepAliveInterval = 15 // seconds (more aggressive than default 30)
	yamuxConfig.ConnectionWriteTimeout = 10 * time.Second
	yamuxConfig.LogOutput = io.Discard

	yamuxSession, err := yamux.Server(wsNetConn, yamuxConfig)
	if err != nil {
		log.Error().Err(err).Str("system_id", sysID).Msg("yamux session creation failed")
		_ = wsConn.Close()
		return
	}

	// Activate the session in the database
	if err := session.ActivateSession(sess.ID); err != nil {
		log.Error().Err(err).Str("session_id", sess.ID).Msg("failed to activate session")
	}

	// Register the tunnel
	t, regErr := TunnelManager.Register(sysID, nodeID, sess.ID, yamuxSession, wsConn)
	if regErr != nil {
		log.Error().Err(regErr).Str("system_id", sysID).Str("node_id", nodeID).Msg("failed to register tunnel")
		_ = yamuxSession.Close()
		return
	}

	// Accept the control stream (first stream from client with service manifest)
	go acceptControlStream(t, sysID, sess.ID)

	// Handle the tunnel lifecycle in a goroutine
	go handleTunnelLifecycle(t, sysID, nodeID, sess.ID)
}

// acceptControlStream accepts control streams from the tunnel client.
// Each stream carries either a service manifest (raw JSON starting with '{')
// or a diagnostics report (header line "DIAGNOSTICS 1\n" followed by JSON).
func acceptControlStream(t *tunnel.Tunnel, systemID, sessionID string) {
	log := logger.ComponentLogger("tunnel")
	var lastManifest time.Time
	var lastDiagnostics time.Time // fast in-memory pre-check; DB enforces cross-reconnect rate limit

	for {
		stream, err := t.Session.Accept()
		if err != nil {
			return // session closed
		}

		// Wrap stream with a size-limited buffered reader (1 MB overall limit)
		br := bufio.NewReaderSize(io.LimitReader(stream, 1<<20), 4096)

		firstByte, err := br.ReadByte()
		if err != nil {
			_ = stream.Close()
			continue
		}

		if firstByte == 'D' {
			// Read the rest of the header line (br already consumed the 'D')
			rest, _ := br.ReadString('\n')
			headerLine := "D" + rest
			// Fix #8: validate exact version to reject unknown protocol versions
			diagParts := strings.Fields(strings.TrimSpace(headerLine))
			if len(diagParts) == 2 && diagParts[0] == "DIAGNOSTICS" && diagParts[1] == "1" {
				// Fast in-memory rate-limit (pre-check before hitting the DB)
				if !lastDiagnostics.IsZero() && time.Since(lastDiagnostics) < 30*time.Second {
					_ = stream.Close()
					continue
				}

				// Read remaining bytes as JSON
				rawJSON, readErr := io.ReadAll(br)
				_ = stream.Close()
				if readErr != nil {
					log.Warn().Err(readErr).
						Str("system_id", systemID).
						Str("session_id", sessionID).
						Msg("failed to read diagnostics payload from control stream")
					continue
				}

				// Fix #4: explicit size guard — reject payloads larger than 512 KB
				if len(rawJSON) > 512*1024 {
					log.Warn().
						Str("system_id", systemID).
						Str("session_id", sessionID).
						Int("bytes", len(rawJSON)).
						Msg("diagnostics payload exceeds 512 KB limit, skipping")
					continue
				}

				// Fix #4: schema validation — unmarshal into typed struct and re-serialize
				// to reject malformed JSON and strip unknown fields (prevents stored XSS).
				var report models.DiagnosticsReport
				if parseErr := json.Unmarshal(rawJSON, &report); parseErr != nil {
					log.Warn().Err(parseErr).
						Str("system_id", systemID).
						Str("session_id", sessionID).
						Msg("invalid diagnostics JSON schema, skipping")
					continue
				}
				sanitized, marshalErr := json.Marshal(report)
				if marshalErr != nil {
					log.Warn().Err(marshalErr).
						Str("system_id", systemID).
						Str("session_id", sessionID).
						Msg("failed to re-serialize diagnostics, skipping")
					continue
				}

				var raw json.RawMessage = sanitized
				// Fix #5: SaveDiagnostics enforces the rate limit in the DB
				// to handle reconnect-based bypass of the in-memory check above.
				saved, jsonErr := session.SaveDiagnostics(sessionID, raw)
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
					lastDiagnostics = time.Now()
					log.Info().
						Str("system_id", systemID).
						Str("session_id", sessionID).
						Int("payload_bytes", len(rawJSON)).
						Msg("diagnostics received")
				}
				continue
			}
			// Unknown or unsupported header starting with 'D' — skip stream
			_ = stream.Close()
			continue
		}

		// Manifest path: put the peeked byte back so the JSON decoder sees it
		_ = br.UnreadByte()

		// Rate-limit manifest updates (max 1 per 10 seconds, first always accepted)
		if !lastManifest.IsZero() && time.Since(lastManifest) < 10*time.Second {
			_ = stream.Close()
			continue
		}

		// Decode manifest with the buffered reader (preserves already-read bytes)
		var manifest tunnel.ServiceManifest
		if err := json.NewDecoder(br).Decode(&manifest); err != nil {
			log.Warn().Err(err).
				Str("system_id", systemID).
				Str("session_id", sessionID).
				Msg("failed to decode service manifest from control stream")
			_ = stream.Close()
			continue
		}
		_ = stream.Close()

		if manifest.Services != nil {
			t.SetServices(manifest.Services)
			lastManifest = time.Now()
			log.Info().
				Str("system_id", systemID).
				Str("session_id", sessionID).
				Int("service_count", len(manifest.Services)).
				Msg("service manifest received")
		}
	}
}

func handleTunnelLifecycle(t *tunnel.Tunnel, systemID, nodeID, sessionID string) {
	log := logger.ComponentLogger("tunnel")

	// Wait for the yamux session to close (either side disconnects)
	<-t.Session.CloseChan()

	log.Info().
		Str("system_id", systemID).
		Str("node_id", nodeID).
		Str("session_id", sessionID).
		Msg("tunnel disconnected, starting grace period")

	// Unregister the tunnel (yamux session is dead)
	TunnelManager.Unregister(systemID, nodeID)

	// Start a grace period instead of immediately closing the session.
	// If the client reconnects before the grace period expires,
	// GetActiveSession finds the still-active session and reuses it.
	TunnelManager.StartGracePeriod(systemID, nodeID, sessionID, configuration.Config.TunnelGracePeriod)
}
