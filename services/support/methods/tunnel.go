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
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/session"
	"github.com/nethesis/my/services/support/tunnel"
)

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

// acceptControlStream accepts the control stream from the tunnel client
// and reads the service manifest. It continues to listen for manifest updates.
func acceptControlStream(t *tunnel.Tunnel, systemID, sessionID string) {
	log := logger.ComponentLogger("tunnel")
	var lastManifest time.Time

	for {
		stream, err := t.Session.Accept()
		if err != nil {
			return // session closed
		}

		// Rate limit manifest updates (max 1 per 10 seconds, first always accepted)
		if !lastManifest.IsZero() && time.Since(lastManifest) < 10*time.Second {
			_ = stream.Close()
			continue
		}

		// Decode manifest with size limit to prevent memory exhaustion
		var manifest tunnel.ServiceManifest
		decoder := json.NewDecoder(io.LimitReader(stream, 1<<20)) // 1 MB max
		if err := decoder.Decode(&manifest); err != nil {
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
