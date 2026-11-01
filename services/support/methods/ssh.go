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

// HandleSSH bridges the SSH gateway to the system's built-in SSH endpoint:
// binary WebSocket messages carry the raw SSH byte stream.
// Route: GET /api/ssh/:session_id (internal, session token auth)
func HandleSSH(c *gin.Context) {
	sessionID := c.Param("session_id")
	log := logger.ComponentLogger("ssh")

	t := TunnelManager.GetBySessionID(sessionID)
	if t == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tunnel not found for session"})
		return
	}
	if !t.AcquireStream() {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many concurrent streams on this tunnel"})
		return
	}

	stream, err := t.Session.Open()
	if err != nil {
		t.ReleaseStream()
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to open tunnel stream"})
		return
	}
	if err := tunnel.WriteConnectHeader(stream, "ssh"); err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		c.JSON(http.StatusBadGateway, gin.H{"error": "tunnel connect failed"})
		return
	}
	// an older client on the system has no SSH endpoint: say so before upgrading
	if err := tunnel.ReadConnectResponse(stream); err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		log.Warn().Err(err).Str("session_id", sessionID).Msg("ssh CONNECT rejected")
		c.JSON(http.StatusNotImplemented, gin.H{"error": "ssh not available on this system: " + err.Error()})
		return
	}

	wsConn, err := terminalUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		t.ReleaseStream()
		_ = stream.Close()
		return
	}
	// the gateway sends at most 32 KB a message: anything far larger is abuse
	wsConn.SetReadLimit(1 << 20)
	log.Info().Str("session_id", sessionID).Msg("ssh stream started")

	var bytesIn, bytesOut atomic.Int64
	lastActivity := &atomic.Value{}
	lastActivity.Store(time.Now())
	startTime := time.Now()

	var once sync.Once
	done := make(chan struct{})
	cleanup := func() {
		once.Do(func() {
			close(done)
			_ = wsConn.Close()
			_ = stream.Close()
			t.ReleaseStream()
			log.Info().
				Str("session_id", sessionID).
				Dur("duration", time.Since(startTime)).
				Int64("bytes_in", bytesIn.Load()).
				Int64("bytes_out", bytesOut.Load()).
				Msg("ssh stream ended")
		})
	}

	// same idle limit as the web terminal; SSH keepalives count as activity
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				if time.Since(lastActivity.Load().(time.Time)) > configuration.Config.TerminalInactivityTimeout {
					log.Warn().Str("session_id", sessionID).Msg("ssh inactivity timeout, closing")
					cleanup()
					return
				}
			}
		}
	}()

	go func() {
		defer cleanup()
		for {
			kind, msg, err := wsConn.ReadMessage()
			if err != nil {
				return
			}
			if kind != websocket.BinaryMessage {
				continue
			}
			lastActivity.Store(time.Now())
			bytesIn.Add(int64(len(msg)))
			if _, err := stream.Write(msg); err != nil {
				return
			}
		}
	}()

	go func() {
		defer cleanup()
		buf := make([]byte, 32*1024)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				lastActivity.Store(time.Now())
				bytesOut.Add(int64(n))
				if werr := wsConn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Debug().Err(err).Str("session_id", sessionID).Msg("ssh stream read ended")
				}
				return
			}
		}
	}()

	<-done
}
