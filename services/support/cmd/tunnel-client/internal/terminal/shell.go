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
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

// HandleShell spawns a shell with a raw PTY and bridges it to the stream.
//
// Protocol: the first line from the stream is the command to execute.
//   - empty → interactive shell with framed protocol (resize support)
//   - non-empty → sh -c "<command>" with raw I/O (for scp, sftp, remote commands)
func HandleShell(stream net.Conn) {
	reader := bufio.NewReader(stream)
	command, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Failed to read command line: %v", err)
		return
	}
	command = command[:len(command)-1] // strip newline

	if command == "" {
		handleInteractiveShell(stream, reader)
	} else {
		handleExecCommand(stream, reader, command)
	}
}

// handleInteractiveShell spawns an interactive shell with PTY and uses
// the framed protocol (same as HandleTerminal) for resize support.
func handleInteractiveShell(stream net.Conn, reader *bufio.Reader) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = defaultShell
	}

	cmd := exec.Command(shell)
	cmd.Env = append(SanitizeEnv(os.Environ()), defaultTermEnv)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("Failed to start PTY for shell: %v", err)
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

	// PTY -> stream: read from PTY, send as type-0 data frames
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

	// Stream -> PTY: read frames, dispatch by type
	go func() {
		defer closeAll()
		// Use buffered reader for initial data (may have buffered bytes from command line read)
		var frameReader io.Reader = reader
		if reader.Buffered() == 0 {
			frameReader = stream
		}

		for {
			frame, readErr := ReadFrame(frameReader)
			if readErr != nil {
				return
			}
			// After first read from buffered reader, switch to raw stream
			if frameReader != stream && reader.Buffered() == 0 {
				frameReader = stream
			}

			if len(frame) < 1 {
				continue
			}

			switch frame[0] {
			case 0: // data -> write to PTY
				if _, writeErr := ptmx.Write(frame[1:]); writeErr != nil {
					return
				}
			case 1: // resize -> set PTY window size
				var size struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				}
				if jsonErr := json.Unmarshal(frame[1:], &size); jsonErr != nil {
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

// handleExecCommand runs a command with raw stdin/stdout (no PTY, no framing).
func handleExecCommand(stream net.Conn, reader *bufio.Reader, command string) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Env = append(SanitizeEnv(os.Environ()), defaultTermEnv)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Printf("Failed to create stdin pipe: %v", err)
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Failed to create stdout pipe: %v", err)
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start command %q: %v", command, err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if reader.Buffered() > 0 {
			_, _ = io.Copy(stdin, reader)
		}
		_, _ = io.Copy(stdin, stream)
		_ = stdin.Close()
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(stream, stdout)
	}()

	wg.Wait()
	_ = cmd.Wait()
}
