/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package tunnel

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// validServiceNamePattern validates service names against injection attacks.
// Rejects names with newlines or control characters that could desync the CONNECT protocol.
var validServiceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// WriteConnectHeader writes a CONNECT request header to the stream.
// Format: "CONNECT <serviceName>\n"
// Validates the service name to prevent newline injection (#17).
func WriteConnectHeader(w io.Writer, serviceName string) error {
	if !validServiceNamePattern.MatchString(serviceName) {
		return fmt.Errorf("invalid service name: %q", serviceName)
	}
	_, err := fmt.Fprintf(w, "CONNECT %s\n", serviceName)
	return err
}

// ReadConnectHeader reads a CONNECT request header from the stream.
// Returns the service name requested.
func ReadConnectHeader(r io.Reader) (string, error) {
	line, err := readLine(r)
	if err != nil {
		return "", fmt.Errorf("failed to read CONNECT header: %w", err)
	}

	if !strings.HasPrefix(line, "CONNECT ") {
		return "", fmt.Errorf("invalid CONNECT header: %q", line)
	}

	serviceName := strings.TrimPrefix(line, "CONNECT ")
	if serviceName == "" {
		return "", fmt.Errorf("empty service name in CONNECT header")
	}

	return serviceName, nil
}

// WriteConnectResponse writes a CONNECT response to the stream.
// If err is nil, writes "OK\n"; otherwise writes "ERROR <message>\n".
func WriteConnectResponse(w io.Writer, err error) error {
	if err == nil {
		_, writeErr := fmt.Fprint(w, "OK\n")
		return writeErr
	}
	_, writeErr := fmt.Fprintf(w, "ERROR %s\n", err.Error())
	return writeErr
}

// ReadConnectResponse reads a CONNECT response from the stream.
// Returns nil on "OK", or an error with the message on "ERROR".
func ReadConnectResponse(r io.Reader) error {
	line, err := readLine(r)
	if err != nil {
		return fmt.Errorf("failed to read CONNECT response: %w", err)
	}

	if line == "OK" {
		return nil
	}

	if strings.HasPrefix(line, "ERROR ") {
		return fmt.Errorf("%s", strings.TrimPrefix(line, "ERROR "))
	}

	return fmt.Errorf("unexpected CONNECT response: %q", line)
}

// readLine reads a single line from the reader byte-by-byte until '\n'.
// Returns the line without the trailing newline.
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
			// Prevent unbounded reads
			if len(buf) > 1024 {
				return "", fmt.Errorf("line too long")
			}
		}
		if err != nil {
			if err == io.EOF && len(buf) > 0 {
				return "", fmt.Errorf("unexpected EOF: incomplete line")
			}
			return "", err
		}
	}
}
