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
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"net"
	"os"
	"time"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

// NewServer creates and configures the SSH server
func NewServer(auth *AuthHandler) *ssh.Server {
	connLimiter := newRateLimiter(10, time.Minute)

	server := &ssh.Server{
		Addr: configuration.Config.SSHListenAddress,
		Handler: func(s ssh.Session) {
			HandleSession(s, auth)
		},
		PublicKeyHandler:           auth.PublicKeyAuth,
		KeyboardInteractiveHandler: auth.KeyboardInteractive,
		// Disable port forwarding for security
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			return false
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, bindHost string, bindPort uint32) bool {
			return false
		},
		ConnCallback: func(ctx ssh.Context, conn net.Conn) net.Conn {
			host, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
			if !connLimiter.Allow(host) {
				logger.Warn().Str("client_ip", host).Msg("SSH connection rate limit exceeded")
				conn.Close() //nolint:errcheck
				return nil
			}
			return conn
		},
	}

	// Load or generate host key
	hostKey, err := loadOrGenerateHostKey(configuration.Config.SSHHostKeyPath)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to load or generate SSH host key")
	}
	server.AddHostKey(hostKey)

	return server
}

// loadOrGenerateHostKey loads an existing host key or generates a new one
func loadOrGenerateHostKey(path string) (gossh.Signer, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		signer, err := gossh.ParsePrivateKey(data)
		if err != nil {
			return nil, err
		}
		logger.Info().Str("path", path).Msg("SSH host key loaded")
		return signer, nil
	}

	// Generate new ed25519 key
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	pemBlock, err := gossh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return nil, err
	}

	keyData := pem.EncodeToMemory(pemBlock)
	if err := os.WriteFile(path, keyData, 0600); err != nil {
		return nil, err
	}

	signer, err := gossh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, err
	}

	logger.Info().Str("path", path).Msg("SSH host key generated")
	return signer, nil
}
