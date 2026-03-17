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
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/response"
	"github.com/nethesis/my/services/support/tunnel"
)

// HandleStream handles GET /api/internal/stream/:session_id/:service
// Finds the tunnel by session ID.
func HandleStream(c *gin.Context) {
	sessionID := c.Param("session_id")
	serviceName := c.Param("service")

	t := TunnelManager.GetBySessionID(sessionID)
	if t == nil {
		logger.ComponentLogger("stream").Warn().
			Str("session_id", sessionID).
			Str("service", serviceName).
			Int("active_tunnels", TunnelManager.Count()).
			Msg("tunnel not found for session")
		c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for session", nil))
		return
	}

	bridgeStream(c, t, serviceName)
}

// HandleStreamBySystem handles GET /api/internal/stream-by-system/:system_id/:service
// Finds the tunnel by system ID (any node). Used by the SSH gateway
// because the cached auth result may reference a stale session_id.
func HandleStreamBySystem(c *gin.Context) {
	systemID := c.Param("system_id")
	serviceName := c.Param("service")

	t := TunnelManager.GetBySystemID(systemID)
	if t == nil {
		logger.ComponentLogger("stream").Warn().
			Str("system_id", systemID).
			Str("service", serviceName).
			Int("active_tunnels", TunnelManager.Count()).
			Msg("tunnel not found for system")
		c.JSON(http.StatusNotFound, response.NotFound("tunnel not found for system", nil))
		return
	}

	bridgeStream(c, t, serviceName)
}

// bridgeStream opens a yamux stream to the tunnel, sends a CONNECT header,
// hijacks the HTTP connection, and bridges them bidirectionally.
func bridgeStream(c *gin.Context, t *tunnel.Tunnel, serviceName string) {
	log := logger.ComponentLogger("stream")

	if !t.AcquireStream() {
		c.JSON(http.StatusTooManyRequests, response.Error(http.StatusTooManyRequests, "too many concurrent streams on this tunnel", nil))
		return
	}

	stream, err := t.Session.Open()
	if err != nil {
		t.ReleaseStream()
		log.Error().Err(err).Str("system_id", t.SystemID).Msg("failed to open yamux stream")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to open stream to remote system", nil))
		return
	}

	if err := tunnel.WriteConnectHeader(stream, serviceName); err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		log.Error().Err(err).Str("system_id", t.SystemID).Msg("failed to send CONNECT header")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "failed to connect to remote service", nil))
		return
	}

	if err := tunnel.ReadConnectResponse(stream); err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		log.Error().Err(err).Str("system_id", t.SystemID).Str("service", serviceName).Msg("remote service rejected connection")
		c.JSON(http.StatusBadGateway, response.Error(http.StatusBadGateway, "remote service rejected connection: "+err.Error(), nil))
		return
	}

	hijacker, ok := c.Writer.(http.Hijacker)
	if !ok {
		t.ReleaseStream()
		_ = stream.Close()
		c.JSON(http.StatusInternalServerError, response.InternalServerError("server does not support connection hijacking", nil))
		return
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		log.Error().Err(err).Str("system_id", t.SystemID).Msg("failed to hijack connection")
		return
	}

	_, _ = buf.WriteString("HTTP/1.1 101 Switching Protocols\r\nUpgrade: raw-stream\r\nConnection: Upgrade\r\n\r\n")
	_ = buf.Flush()

	log.Info().
		Str("system_id", t.SystemID).
		Str("session_id", t.SessionID).
		Str("service", serviceName).
		Msg("stream bridge established")

	bridgeConnections(conn, stream)

	t.ReleaseStream()

	log.Info().
		Str("system_id", t.SystemID).
		Str("service", serviceName).
		Msg("stream bridge closed")
}

// bridgeConnections copies data bidirectionally between two connections
func bridgeConnections(a net.Conn, b io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(b, a)
		_ = b.Close()
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(a, b)
		_ = a.Close()
	}()

	wg.Wait()
}
