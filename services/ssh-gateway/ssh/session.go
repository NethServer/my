/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package ssh

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	glssh "github.com/gliderlabs/ssh"

	"github.com/nethesis/my/services/ssh-gateway/audit"
	"github.com/nethesis/my/services/ssh-gateway/bridge"
	"github.com/nethesis/my/services/ssh-gateway/logger"
	"github.com/nethesis/my/services/ssh-gateway/models"
)

// HandleSession handles an authenticated SSH session by bridging it to the remote system
func HandleSession(s glssh.Session, auth *AuthHandler) {
	log := logger.ComponentLogger("session")
	startTime := time.Now()

	// Get auth result from context
	val := s.Context().Value("auth_result")
	result, ok := val.(*models.AuthResult)
	if !ok || result == nil {
		log.Error().Msg("no auth result in session context")
		_, _ = io.WriteString(s, "authentication error\n")
		return
	}

	// Get the command if this is an exec request (e.g., scp -O, remote command)
	command := s.RawCommand()

	// Determine session type and auth method
	sessionType := "interactive"
	if command != "" {
		sessionType = "exec"
		if len(command) >= 3 && command[:3] == "scp" {
			sessionType = "scp"
		}
	}

	authMethod := "browser"
	if fp := s.Context().Value("ssh_auth_method"); fp != nil {
		if method, ok := fp.(string); ok {
			authMethod = method
		}
	}

	// Get client IP
	clientIP := ""
	if s.RemoteAddr() != nil {
		clientIP, _, _ = net.SplitHostPort(s.RemoteAddr().String())
	}

	// Get SSH key fingerprint from context
	fingerprint := ""
	if fp := s.Context().Value("ssh_pubkey_fingerprint"); fp != nil {
		if f, ok := fp.(string); ok {
			fingerprint = f
		}
	}

	log.Info().
		Str("session_id", result.SessionID).
		Str("system_id", result.SystemID).
		Str("system_name", result.SystemName).
		Str("system_type", result.SystemType).
		Str("user_id", result.UserID).
		Str("username", result.Username).
		Str("email", result.UserEmail).
		Str("organization", result.OrganizationName).
		Str("client_ip", clientIP).
		Str("auth_method", authMethod).
		Str("session_type", sessionType).
		Str("command", command).
		Msg("SSH session starting")

	// Write access log to database
	accessLogID := audit.LogConnect(result.SessionID, result.UserID, result.Username, audit.SSHAccessMetadata{
		AuthMethod:  authMethod,
		Command:     command,
		SessionType: sessionType,
		ClientIP:    clientIP,
		Fingerprint: fingerprint,
	})
	defer func() {
		audit.LogDisconnect(accessLogID)
		duration := time.Since(startTime)
		log.Info().
			Str("session_id", result.SessionID).
			Str("system_id", result.SystemID).
			Str("user_id", result.UserID).
			Str("username", result.Username).
			Str("client_ip", clientIP).
			Str("session_type", sessionType).
			Dur("duration", duration).
			Str("disconnect_reason", "client_closed").
			Msg("SSH session ended")
	}()

	// Connect to the remote system via the support service tunnel (by system_id, not session_id)
	stream, err := bridge.Connect(result.SystemID)
	if err != nil {
		log.Error().Err(err).
			Str("session_id", result.SessionID).
			Str("client_ip", clientIP).
			Msg("failed to connect to tunnel")
		_, _ = io.WriteString(s, "failed to connect to remote system\n")
		return
	}
	defer func() { _ = stream.Close() }()

	// Send the command as the first line over the stream
	_, err = fmt.Fprintf(stream, "%s\n", command)
	if err != nil {
		log.Error().Err(err).Msg("failed to send command to tunnel")
		_, _ = io.WriteString(s, "failed to connect to remote system\n")
		return
	}

	log.Info().
		Str("session_id", result.SessionID).
		Str("system_id", result.SystemID).
		Str("client_ip", clientIP).
		Msg("tunnel bridge established")

	if command == "" {
		handleInteractiveSession(s, stream)
	} else {
		handleRawSession(s, stream)
	}
}

// handleInteractiveSession bridges an interactive shell with framed protocol.
// Uses type-0 frames for data and type-1 frames for PTY resize events.
func handleInteractiveSession(s glssh.Session, stream net.Conn) {
	done := make(chan struct{}, 1)

	// Forward PTY resize events from SSH client to tunnel
	_, winCh, _ := s.Pty()
	go func() {
		for win := range winCh {
			payload, _ := json.Marshal(map[string]int{
				"cols": win.Width,
				"rows": win.Height,
			})
			if err := writeResizeFrame(stream, payload); err != nil {
				return
			}
		}
	}()

	// SSH client -> tunnel: read from session, send as type-0 data frames
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := s.Read(buf)
			if n > 0 {
				if writeErr := writeDataFrame(stream, buf[:n]); writeErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		_ = stream.Close()
		done <- struct{}{}
	}()

	// Tunnel -> SSH client: read type-0 data frames, write to session
	go func() {
		for {
			frame, err := readFrame(stream)
			if err != nil {
				break
			}
			if len(frame) < 1 {
				continue
			}
			if frame[0] == 0 { // data frame
				if _, writeErr := s.Write(frame[1:]); writeErr != nil {
					break
				}
			}
		}
		_ = s.Close()
		done <- struct{}{}
	}()

	<-done
}

// handleRawSession bridges exec/scp with raw bidirectional copy (no framing).
func handleRawSession(s glssh.Session, stream net.Conn) {
	done := make(chan struct{}, 1)

	go func() {
		_, _ = io.Copy(stream, s)
		_ = stream.Close()
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(s, stream)
		_ = s.Close()
		done <- struct{}{}
	}()

	<-done
}
