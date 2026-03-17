/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package bridge

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

// Connect establishes a raw TCP stream to the support service's internal
// stream endpoint for the given system. The support service hijacks the
// HTTP connection and bridges it to the yamux tunnel's shell service.
//
// Uses system_id (stable) instead of session_id (changes on reconnect)
// to find the active tunnel.
func Connect(systemID string) (net.Conn, error) {
	log := logger.ComponentLogger("bridge")

	url := fmt.Sprintf("%s/api/internal/stream-by-system/%s/shell", configuration.Config.SupportServiceURL, systemID)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Internal-Secret", configuration.Config.SupportInternalSecret)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "raw-stream")

	// Dial TCP directly and send the HTTP request manually to hijack the connection
	host := req.URL.Host
	if req.URL.Port() == "" {
		host = net.JoinHostPort(host, "80")
	}

	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to support service: %w", err)
	}

	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		_ = conn.Close()
		body := make([]byte, 512)
		n, _ := resp.Body.Read(body)
		log.Warn().
			Int("status", resp.StatusCode).
			Str("system_id", systemID).
			Str("body", string(body[:n])).
			Msg("support service rejected stream request")
		return nil, fmt.Errorf("support service returned %d: %s", resp.StatusCode, string(body[:n]))
	}

	log.Debug().
		Str("system_id", systemID).
		Msg("bridge connection established")

	return conn, nil
}
