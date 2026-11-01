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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/logger"
)

const (
	// handshakeTimeout covers the whole authentication, browser wait included
	handshakeExtra = time.Minute
	pingInterval   = 30 * time.Second
)

// Server accepts SSH connections tunnelled in WebSockets: Render only lets
// HTTPS in, so clients reach it through a ProxyCommand such as websocat
type Server struct {
	auth     *Authenticator
	hostKey  gossh.Signer
	upgrader websocket.Upgrader
	timeout  time.Duration
}

func NewServer(auth *Authenticator, hostKey gossh.Signer, authTimeout time.Duration) *Server {
	return &Server{
		auth:    auth,
		hostKey: hostKey,
		timeout: authTimeout + handshakeExtra,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  32 * 1024,
			WriteBufferSize: 32 * 1024,
			// SSH clients send no Origin: a browser page has no business here
			CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" },
		},
	}
}

// ServeHTTP handles GET /api/ssh
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	clientIP := clientIP(r)
	conn := newWSConn(ws, &net.TCPAddr{IP: net.ParseIP(clientIP)})
	done := make(chan struct{})
	defer close(done)
	go conn.keepAlive(pingInterval, done)
	s.serveConn(conn, clientIP)
}

func (s *Server) serveConn(conn net.Conn, clientIP string) {
	log := logger.ComponentLogger("server")
	defer conn.Close() //nolint:errcheck

	timer := time.AfterFunc(s.timeout, func() { _ = conn.Close() })
	sshConn, chans, reqs, err := gossh.NewServerConn(conn, s.auth.ServerConfig(clientIP, s.hostKey))
	timer.Stop()
	if err != nil {
		log.Debug().Err(err).Str("client_ip", clientIP).Msg("ssh handshake failed")
		return
	}
	defer sshConn.Close() //nolint:errcheck
	log.Info().Str("client_ip", clientIP).Str("user", sshConn.User()).Str("operator", sshConn.Permissions.Extensions[extOperator]).Msg("ssh connection established")

	// global requests (remote port forwarding included) are refused
	go gossh.DiscardRequests(reqs)

	up := &upstream{auth: s.auth, perms: sshConn.Permissions}
	defer up.close()
	// the backend closes the bridge when the operator is no longer allowed:
	// the operator's connection goes with it
	go func() {
		if client, _ := up.get(); client != nil {
			_ = client.Wait()
			_ = sshConn.Close()
		}
	}()

	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			// no port forwarding through the systems
			_ = newCh.Reject(gossh.Prohibited, "only sessions are allowed")
			continue
		}
		go proxyChannel(newCh, up)
	}
}

// clientIP takes the address the proxy computed: X-Forwarded-For starts with
// whatever the client wrote there
func clientIP(r *http.Request) string {
	if ip := strings.TrimSpace(r.Header.Get("X-Client-IP")); net.ParseIP(ip) != nil {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// LoadOrGenerateHostKey keeps the gateway's identity stable across restarts:
// a changed key makes every operator's ssh warn of a man in the middle
func LoadOrGenerateHostKey(pemKey, path string) (gossh.Signer, error) {
	if pemKey != "" {
		return gossh.ParsePrivateKey([]byte(pemKey))
	}
	if data, err := os.ReadFile(path); err == nil {
		return gossh.ParsePrivateKey(data)
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	block, err := gossh.MarshalPrivateKey(priv, "")
	if err != nil {
		return nil, err
	}
	data := pem.EncodeToMemory(block)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, err
	}
	logger.Info().Str("path", path).Msg("ssh host key generated")
	return gossh.ParsePrivateKey(data)
}
