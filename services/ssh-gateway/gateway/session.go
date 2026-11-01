/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package gateway

import (
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

// Requests that would expose the operator's machine to the system
var deniedRequests = map[string]bool{
	"auth-agent-req@openssh.com": true,
	"x11-req":                    true,
}

// upstream is the SSH connection to the system's built-in endpoint, opened
// through the backend on the first session and shared by the others
type upstream struct {
	perms    *gossh.Permissions
	clientIP string
	once     sync.Once
	mu       sync.Mutex
	closed   bool
	client   *gossh.Client
	err      string
}

func (u *upstream) get() (*gossh.Client, string) {
	u.once.Do(func() {
		client, failure := dialSystem(u.perms)
		u.mu.Lock()
		defer u.mu.Unlock()
		// the operator left while the system was being reached
		if u.closed && client != nil {
			_ = client.Close()
			client, failure = nil, "connection closed"
		}
		u.client, u.err = client, failure
	})
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.client, u.err
}

func (u *upstream) close() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.closed = true
	if u.client != nil {
		_ = u.client.Close()
	}
}

// dialSystem opens the shell bridge on the backend, which checks the grant
// and its operator, and runs SSH over it. The bytes travel through the
// authenticated tunnel only, so the system's ephemeral host key is not pinned.
func dialSystem(perms *gossh.Permissions) (*gossh.Client, string) {
	log := logger.ComponentLogger("session")
	url := strings.Replace(configuration.Config.BackendURL, "http", "ws", 1) + gatewayPath + "/terminal"
	header := http.Header{}
	header.Set(gatewaySecretHeader, configuration.Config.SSHGatewaySecret)
	header.Set(grantHeader, perms.Extensions[extGrant])

	// the backend checks the operator on Logto first: leave it room
	dialer := websocket.Dialer{HandshakeTimeout: 45 * time.Second, ReadBufferSize: 32 * 1024, WriteBufferSize: 32 * 1024}
	ws, resp, err := dialer.Dial(url, header)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
			_ = resp.Body.Close()
		}
		log.Warn().Err(err).Int("status", status).Str("system", perms.Extensions[extSystem]).Msg("cannot reach the system")
		switch status {
		case http.StatusUnauthorized, http.StatusForbidden:
			// the grant expired or its operator or session is no longer valid:
			// the backend forgets the approved key with it
			return nil, "Autorizzazione scaduta o revocata: copia di nuovo il comando dal pulsante SSH in my."
		case http.StatusNotFound:
			return nil, "Nessuna sessione di supporto attiva su questo sistema."
		case http.StatusTooManyRequests:
			return nil, "Troppe connessioni aperte su questo sistema, riprova tra poco."
		case http.StatusServiceUnavailable:
			return nil, "Servizio momentaneamente non disponibile, riprova tra poco."
		case http.StatusNotImplemented:
			return nil, "Il sistema non supporta ancora l'accesso SSH: va aggiornato il client del supporto."
		}
		return nil, "Impossibile raggiungere il sistema."
	}

	conn := newWSConn(ws, ws.RemoteAddr())
	c, chans, reqs, err := gossh.NewClientConn(conn, "system", &gossh.ClientConfig{
		User:            "support",
		HostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec // see above
		Timeout:         15 * time.Second,
	})
	if err != nil {
		_ = conn.Close()
		log.Warn().Err(err).Str("system", perms.Extensions[extSystem]).Msg("ssh handshake with the system failed")
		return nil, "Impossibile aprire la sessione sul sistema."
	}
	log.Info().Str("system", perms.Extensions[extSystem]).Str("operator", perms.Extensions[extOperator]).Msg("ssh connected to the system")
	return gossh.NewClient(c, chans, reqs), ""
}

// proxyChannel relays one session channel as is: shell, commands, sftp,
// pipes, exit codes and signals behave as with a direct ssh
func proxyChannel(newCh gossh.NewChannel, up *upstream) {
	log := logger.ComponentLogger("session")
	client, failure := up.get()
	if client == nil {
		ch, reqs, err := newCh.Accept()
		if err != nil {
			return
		}
		go gossh.DiscardRequests(reqs)
		_, _ = fmt.Fprintf(ch.Stderr(), "%s\r\n", failure)
		sendExitStatus(ch, 255)
		_ = ch.Close()
		return
	}

	upCh, upReqs, err := client.OpenChannel(newCh.ChannelType(), newCh.ExtraData())
	if err != nil {
		reason, msg := gossh.ConnectionFailed, err.Error()
		if openErr, ok := err.(*gossh.OpenChannelError); ok {
			reason, msg = openErr.Reason, openErr.Message
		}
		_ = newCh.Reject(reason, msg)
		return
	}
	ch, reqs, err := newCh.Accept()
	if err != nil {
		_ = upCh.Close()
		return
	}

	// operator -> system: requests, then stdin with its EOF
	go func() {
		for req := range reqs {
			if deniedRequests[req.Type] {
				if req.WantReply {
					_ = req.Reply(false, nil)
				}
				continue
			}
			// what the operator runs is recorded with who and where
			switch req.Type {
			case "shell", "exec", "subsystem":
				event := log.Info().
					Str("operator", up.perms.Extensions[extOperator]).
					Str("system", up.perms.Extensions[extSystem]).
					Str("client_ip", up.clientIP).
					Str("request", req.Type)
				if req.Type != "shell" {
					event = event.Str("command", truncate(sshString(req.Payload), 1024))
				}
				event.Msg("ssh session request")
			}
			ok, err := upCh.SendRequest(req.Type, req.WantReply, req.Payload)
			if req.WantReply {
				_ = req.Reply(ok && err == nil, nil)
			}
		}
		_ = upCh.Close()
	}()
	go func() {
		_, _ = io.Copy(upCh, ch)
		_ = upCh.CloseWrite()
	}()

	// system -> operator: output, then exit status, then close
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); _, _ = io.Copy(ch, upCh) }()
	go func() { defer wg.Done(); _, _ = io.Copy(ch.Stderr(), upCh.Stderr()) }()
	go func() {
		defer wg.Done()
		for req := range upReqs {
			ok, err := ch.SendRequest(req.Type, req.WantReply, req.Payload)
			if req.WantReply {
				_ = req.Reply(ok && err == nil, nil)
			}
		}
	}()
	wg.Wait()
	_ = ch.Close()
}

func sendExitStatus(ch gossh.Channel, code uint32) {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, code)
	_, _ = ch.SendRequest("exit-status", false, payload)
}

// sshString reads the leading RFC 4251 string of a request payload
func sshString(payload []byte) string {
	if len(payload) < 4 {
		return ""
	}
	n := binary.BigEndian.Uint32(payload)
	if uint32(len(payload)-4) < n {
		return ""
	}
	return string(payload[4 : 4+n])
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
