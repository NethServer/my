/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package terminal

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/creack/pty"
)

const (
	defaultShell   = "/bin/bash"
	defaultTermEnv = "TERM=xterm-256color"
	maxFrameSize   = 1024 * 1024 // 1 MB
)

// sensitiveEnvPrefixes lists environment variable prefixes that are stripped
// from the PTY shell to prevent operators from extracting credentials (#8).
var sensitiveEnvPrefixes = []string{
	"SYSTEM_KEY=",
	"SYSTEM_SECRET=",
	"SUPPORT_URL=",
	"DATABASE_URL=",
	"REDIS_ADDR=",
	"REDIS_PASSWORD=",
	"REDIS_URL=",
	"INTERNAL_SECRET=",
	"TUNNEL_CONFIG=",
}

// HandleTerminal spawns a shell with a PTY and bridges it to the yamux stream
// using length-prefixed binary frames:
//   - Type 0 (data): raw terminal bytes (bidirectional)
//   - Type 1 (resize): JSON {"cols": N, "rows": N} (stream -> PTY)
func HandleTerminal(stream net.Conn) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = defaultShell
	}

	cmd := exec.Command(shell)
	cmd.Env = append(SanitizeEnv(os.Environ()), defaultTermEnv)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY: %v", err)
		return
	}
	var once sync.Once
	done := make(chan struct{})
	closeAll := func() {
		once.Do(func() {
			close(done)
			_ = ptmx.Close()
			_ = stream.Close()
		})
	}
	defer func() {
		closeAll()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	// PTY -> stream: read from PTY, send as type-0 length-prefixed frames
	go func() {
		defer closeAll()
		buf := make([]byte, 4096)
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				frame := make([]byte, 1+n)
				frame[0] = 0 // data frame
				copy(frame[1:], buf[:n])
				if writeErr := WriteFrame(stream, frame); writeErr != nil {
					return
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Stream -> PTY: read length-prefixed frames, dispatch by type
	go func() {
		defer closeAll()
		for {
			frame, readErr := ReadFrame(stream)
			if readErr != nil {
				return
			}
			if len(frame) < 1 {
				continue
			}

			frameType := frame[0]
			payload := frame[1:]

			switch frameType {
			case 0: // data -> write to PTY
				if _, writeErr := ptmx.Write(payload); writeErr != nil {
					return
				}
			case 1: // resize -> set PTY window size
				var size struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				}
				if jsonErr := json.Unmarshal(payload, &size); jsonErr != nil {
					continue
				}
				if size.Cols > 0 && size.Rows > 0 {
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(size.Rows),
						Cols: uint16(size.Cols),
					})
				}
			}
		}
	}()

	<-done
}

// WriteFrame writes a length-prefixed frame: [4 bytes big-endian length][payload]
func WriteFrame(w io.Writer, data []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// ReadFrame reads a length-prefixed frame: [4 bytes big-endian length][payload]
func ReadFrame(r io.Reader) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header)
	if length > maxFrameSize {
		return nil, fmt.Errorf("frame too large: %d", length)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, err
	}
	return data, nil
}

// SanitizeEnv filters out sensitive environment variables before spawning a shell
func SanitizeEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		sensitive := false
		for _, prefix := range sensitiveEnvPrefixes {
			if strings.HasPrefix(e, prefix) {
				sensitive = true
				break
			}
		}
		if !sensitive {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
