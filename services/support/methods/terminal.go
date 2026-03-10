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
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/tunnel"
)

// terminalUpgrader is a separate WebSocket upgrader for terminal connections.
// Unlike the tunnel upgrader, this rejects cross-origin requests since terminal
// sessions are initiated by browsers on the MY domain.
var terminalUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return r.Header.Get("Origin") == "" // Internal endpoint: backend strips Origin before proxying
	},
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

// Terminal frame types (used on WebSocket between browser and this service)
const (
	frameTypeData   = 0
	frameTypeResize = 1
	frameTypeError  = 3
)

// HandleTerminal handles WebSocket terminal connections.
// It bridges the browser WebSocket to a yamux stream using CONNECT "terminal".
// The tunnel-client spawns a PTY on the remote system — no SSH involved.
// Route: GET /api/terminal/:session_id (internal, session token auth)
func HandleTerminal(c *gin.Context) {
	sessionID := c.Param("session_id")
	log := logger.ComponentLogger("terminal")

	// Find tunnel by session ID
	t := TunnelManager.GetBySessionID(sessionID)
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found for session"})
		return
	}

	// #10: Check stream limit
	if !t.AcquireStream() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many concurrent streams on this tunnel"})
		return
	}

	// Upgrade to WebSocket (using dedicated terminal upgrader)
	wsConn, err := terminalUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		t.ReleaseStream()
		log.Error().Err(err).Str("session_id", sessionID).Msg("websocket upgrade failed")
		return
	}

	// Open yamux stream and CONNECT to terminal service
	stream, err := t.Session.Open()
	if err != nil {
		t.ReleaseStream()
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to open yamux stream")
		sendErrorFrame(wsConn, "failed to open tunnel stream")
		_ = wsConn.Close()
		return
	}

	if err := tunnel.WriteConnectHeader(stream, "terminal"); err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to write CONNECT header")
		sendErrorFrame(wsConn, "tunnel connect failed")
		_ = wsConn.Close()
		return
	}

	if err := tunnel.ReadConnectResponse(stream); err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		log.Error().Err(err).Str("session_id", sessionID).Msg("CONNECT rejected")
		sendErrorFrame(wsConn, "terminal not available: "+err.Error())
		_ = wsConn.Close()
		return
	}

	log.Info().Str("session_id", sessionID).Msg("terminal session started")

	// #7/#1: Track activity for inactivity timeout and audit logging
	var bytesIn, bytesOut atomic.Int64
	lastActivity := &atomic.Value{}
	lastActivity.Store(time.Now())
	startTime := time.Now()

	inactivityTimeout := configuration.Config.TerminalInactivityTimeout

	var once sync.Once
	done := make(chan struct{})
	cleanup := func() {
		once.Do(func() {
			close(done)
			_ = wsConn.Close()
			_ = stream.Close()
			t.ReleaseStream()

			// #1: Log terminal session summary
			duration := time.Since(startTime)
			log.Info().
				Str("session_id", sessionID).
				Dur("duration", duration).
				Int64("bytes_in", bytesIn.Load()).
				Int64("bytes_out", bytesOut.Load()).
				Msg("terminal session ended")
		})
	}

	// #7: Inactivity timeout watchdog
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				last := lastActivity.Load().(time.Time)
				if time.Since(last) > inactivityTimeout {
					log.Warn().
						Str("session_id", sessionID).
						Dur("idle_time", time.Since(last)).
						Msg("terminal inactivity timeout, closing")
					sendErrorFrame(wsConn, "session closed due to inactivity")
					cleanup()
					return
				}
			}
		}
	}()

	// #13: Use configurable max frame size (default 64KB)
	maxFrameSize := configuration.Config.TerminalMaxFrameSize

	// WebSocket → stream: read WS binary messages, write as length-prefixed frames
	go func() {
		defer cleanup()
		for {
			_, msg, readErr := wsConn.ReadMessage()
			if readErr != nil {
				return
			}
			lastActivity.Store(time.Now())
			bytesIn.Add(int64(len(msg)))
			if writeErr := writeFrame(stream, msg); writeErr != nil {
				return
			}
		}
	}()

	// Stream → WebSocket: read length-prefixed frames, send as WS binary messages
	go func() {
		defer cleanup()
		for {
			frame, readErr := readFrameWithLimit(stream, maxFrameSize)
			if readErr != nil {
				return
			}
			lastActivity.Store(time.Now())
			bytesOut.Add(int64(len(frame)))
			if writeErr := wsConn.WriteMessage(websocket.BinaryMessage, frame); writeErr != nil {
				return
			}
		}
	}()

	<-done
}

// sendErrorFrame sends a type 3 error frame to the WebSocket client
func sendErrorFrame(wsConn *websocket.Conn, msg string) {
	frame := make([]byte, 1+len(msg))
	frame[0] = frameTypeError
	copy(frame[1:], msg)
	_ = wsConn.WriteMessage(websocket.BinaryMessage, frame)
}

// writeFrame writes a length-prefixed frame: [4 bytes big-endian length][payload]
func writeFrame(w io.Writer, data []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// readFrameWithLimit reads a length-prefixed frame with configurable max size (#13)
func readFrameWithLimit(r io.Reader, maxSize int) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if int(length) > maxSize {
		return nil, fmt.Errorf("frame too large: %d (max %d)", length, maxSize)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}
