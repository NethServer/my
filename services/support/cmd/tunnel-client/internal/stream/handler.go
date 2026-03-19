/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package stream

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/models"
	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/terminal"
)

const maxLineLength = 1024

// ReadLine reads a newline-terminated line from r, byte by byte.
// Returns the line without the trailing newline. Returns an error if the line
// exceeds maxLineLength bytes or the reader fails.
func ReadLine(r io.Reader) (string, error) {
	return readLine(r)
}

// HandleStreamWithFirstLine processes an incoming yamux stream when the caller
// has already consumed the first line (e.g. to determine the stream type).
// It behaves identically to HandleStream but skips reading the header line,
// using firstLine instead.
func HandleStreamWithFirstLine(stream net.Conn, firstLine string, services map[string]models.ServiceInfo) {
	defer func() { _ = stream.Close() }()

	if !strings.HasPrefix(firstLine, "CONNECT ") {
		log.Printf("Invalid CONNECT header: %q", firstLine)
		return
	}
	serviceName := strings.TrimPrefix(firstLine, "CONNECT ")
	if serviceName == "" {
		log.Printf("Empty service name in CONNECT header")
		return
	}
	dispatchStream(stream, serviceName, services)
}

// HandleStream processes an incoming yamux stream by reading a CONNECT header,
// resolving the target service, and proxying traffic bidirectionally.
func HandleStream(stream net.Conn, services map[string]models.ServiceInfo) {
	defer func() { _ = stream.Close() }()

	// Read CONNECT header
	serviceName, err := readConnectHeader(stream)
	if err != nil {
		log.Printf("Failed to read CONNECT header: %v", err)
		return
	}

	dispatchStream(stream, serviceName, services)
}

// dispatchStream routes an already-identified service name to the correct handler.
func dispatchStream(stream net.Conn, serviceName string, services map[string]models.ServiceInfo) {
	// Built-in terminal service: spawn a PTY instead of dialing TCP
	if serviceName == "terminal" {
		if err := writeConnectResponse(stream, nil); err != nil {
			return
		}
		log.Println("CONNECT terminal -> PTY")
		terminal.HandleTerminal(stream)
		return
	}

	// Look up service
	svc, ok := services[serviceName]
	if !ok {
		_ = writeConnectResponse(stream, fmt.Errorf("service not found: %s", serviceName))
		return
	}

	// Connect to local target
	var err error
	var targetConn net.Conn
	if svc.TLS {
		targetConn, err = tls.Dial("tcp", svc.Target, &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec // Local services use self-signed certs
		})
	} else {
		targetConn, err = net.DialTimeout("tcp", svc.Target, 10*time.Second)
	}
	if err != nil {
		_ = writeConnectResponse(stream, fmt.Errorf("failed to connect to %s: %v", svc.Target, err))
		return
	}

	// Send OK response
	if err := writeConnectResponse(stream, nil); err != nil {
		_ = targetConn.Close()
		return
	}

	log.Printf("CONNECT %s -> %s", serviceName, svc.Target)

	// Bidirectional copy with proper cleanup to prevent goroutine leaks
	var once sync.Once
	done := make(chan struct{})
	closeBoth := func() {
		once.Do(func() {
			close(done)
			_ = targetConn.Close()
			_ = stream.Close()
		})
	}

	go func() {
		defer closeBoth()
		_, _ = io.Copy(targetConn, stream)
	}()

	go func() {
		defer closeBoth()
		_, _ = io.Copy(stream, targetConn)
	}()

	<-done
}

// readConnectHeader reads "CONNECT <service>\n" from the stream byte-by-byte
func readConnectHeader(r io.Reader) (string, error) {
	line, err := readLine(r)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(line, "CONNECT ") {
		return "", fmt.Errorf("invalid CONNECT header: %q", line)
	}
	name := strings.TrimPrefix(line, "CONNECT ")
	if name == "" {
		return "", fmt.Errorf("empty service name")
	}
	return name, nil
}

func writeConnectResponse(w io.Writer, err error) error {
	if err == nil {
		_, writeErr := fmt.Fprint(w, "OK\n")
		return writeErr
	}
	_, writeErr := fmt.Fprintf(w, "ERROR %s\n", err.Error())
	return writeErr
}

func readLine(r io.Reader) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				return string(buf), nil
			}
			buf = append(buf, b[0])
			if len(buf) > maxLineLength {
				return "", fmt.Errorf("line too long")
			}
		}
		if err != nil {
			if err == io.EOF && len(buf) > 0 {
				return string(buf), nil
			}
			return "", err
		}
	}
}
