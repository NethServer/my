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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

const (
	// waiting for the browser gets this much on top of the approval timeout
	approvalExtra = time.Minute
	pingInterval  = 30 * time.Second
)

// Server accepts SSH connections tunnelled in WebSockets: Render only lets
// HTTPS in, so clients reach it through a ProxyCommand such as websocat
type Server struct {
	auth     *Authenticator
	hostKey  gossh.Signer
	upgrader websocket.Upgrader

	// connections still authenticating, overall and per client IP
	pending chan struct{}
	mu      sync.Mutex
	perIP   map[string]int
}

func NewServer(auth *Authenticator, hostKey gossh.Signer) *Server {
	return &Server{
		auth:    auth,
		hostKey: hostKey,
		pending: make(chan struct{}, configuration.Config.MaxPendingConns),
		perIP:   make(map[string]int),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  32 * 1024,
			WriteBufferSize: 32 * 1024,
			// SSH clients send no Origin: a browser page has no business here
			CheckOrigin: func(r *http.Request) bool { return r.Header.Get("Origin") == "" },
		},
	}
}

func (s *Server) acquire(ip string) bool {
	select {
	case s.pending <- struct{}{}:
	default:
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.perIP[ip] >= configuration.Config.MaxConnsPerIP {
		<-s.pending
		return false
	}
	s.perIP[ip]++
	return true
}

func (s *Server) release(ip string) {
	<-s.pending
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.perIP[ip]--; s.perIP[ip] <= 0 {
		delete(s.perIP, ip)
	}
}

// ServeHTTP handles GET /api/ssh
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	clientIP := clientIP(r)
	if !s.acquire(clientIP) {
		logger.Warn().Str("client_ip", clientIP).Msg("ssh connection refused: too many pending connections")
		http.Error(w, "too many connections", http.StatusTooManyRequests)
		return
	}
	released := false
	release := func() {
		if !released {
			released = true
			s.release(clientIP)
		}
	}
	defer release()

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	conn := newWSConn(ws, &net.TCPAddr{IP: net.ParseIP(clientIP)})
	go conn.keepAlive(pingInterval, conn.Done())
	s.serveConn(conn, clientIP, release)
}

func (s *Server) serveConn(conn *wsConn, clientIP string, authenticated func()) {
	log := logger.ComponentLogger("server")
	defer conn.Close() //nolint:errcheck

	// a short limit until the authentication starts, a longer one while an
	// operator approves in the browser
	timer := time.AfterFunc(configuration.Config.HandshakeTimeout, func() { _ = conn.Close() })
	st := &connState{
		clientIP: clientIP,
		done:     conn.Done(),
		authStarted: func() {
			timer.Reset(configuration.Config.AuthTimeout + approvalExtra)
		},
	}
	sshConn, chans, reqs, err := gossh.NewServerConn(conn, s.auth.ServerConfig(st, s.hostKey))
	timer.Stop()
	if err != nil {
		log.Debug().Err(err).Str("client_ip", clientIP).Msg("ssh handshake failed")
		return
	}
	authenticated()
	defer sshConn.Close() //nolint:errcheck
	perms := sshConn.Permissions
	log.Info().Str("client_ip", clientIP).Str("user", stripToken(sshConn.User())).Str("operator", perms.Extensions[extOperator]).Msg("ssh connection established")

	// global requests (remote port forwarding included) are refused
	go gossh.DiscardRequests(reqs)

	up := &upstream{perms: perms, clientIP: clientIP}
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

// stripToken keeps the one-time token out of the logs
func stripToken(user string) string {
	if i := strings.LastIndex(user, "+"); i >= 0 {
		return user[:i]
	}
	return user
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
