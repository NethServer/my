/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

// Package sshserver serves the SSH sessions operators open through the SSH
// gateway: shells, commands, scp, rsync and sftp run with the system's own
// programs, like the web terminal does, without touching the local sshd.
package sshserver

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/support/cmd/tunnel-client/internal/terminal"
)

// maxSessionsPerConn bounds the channels one connection can run at once
const maxSessionsPerConn = 10

// sftpServers are the system sftp-server binaries, tried in order; without
// one (e.g. dropbear only) sftp is served in-process
var sftpServers = []string{
	"/usr/libexec/openssh/sftp-server",
	"/usr/lib/openssh/sftp-server",
	"/usr/libexec/sftp-server",
	"/usr/lib/sftp-server",
}

// Only locale and timezone variables from the client reach the processes
var allowedEnv = regexp.MustCompile(`^(LANG|LANGUAGE|LC_[A-Z]+|TZ)$`)

var signals = map[string]syscall.Signal{
	"HUP": syscall.SIGHUP, "INT": syscall.SIGINT, "QUIT": syscall.SIGQUIT,
	"KILL": syscall.SIGKILL, "TERM": syscall.SIGTERM,
	"USR1": syscall.SIGUSR1, "USR2": syscall.SIGUSR2,
}

var (
	hostKeyOnce sync.Once
	hostKey     ssh.Signer
	hostKeyErr  error
)

// The stream is reachable only through the authenticated tunnel, after the
// gateway and the backend checked the operator: the SSH layer here needs no
// client authentication and an ephemeral host key
func serverConfig() (*ssh.ServerConfig, error) {
	hostKeyOnce.Do(func() {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			hostKeyErr = err
			return
		}
		hostKey, hostKeyErr = ssh.NewSignerFromKey(priv)
	})
	if hostKeyErr != nil {
		return nil, hostKeyErr
	}
	cfg := &ssh.ServerConfig{NoClientAuth: true}
	cfg.AddHostKey(hostKey)
	return cfg, nil
}

// Serve runs the SSH protocol on a tunnel stream until it closes
func Serve(conn net.Conn) {
	cfg, err := serverConfig()
	if err != nil {
		log.Printf("SSH: cannot create host key: %v", err)
		return
	}
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		log.Printf("SSH: handshake failed: %v", err)
		return
	}
	defer func() { _ = sshConn.Close() }()
	log.Println("SSH: session connected")

	// global requests (port forwarding included) are all refused
	go ssh.DiscardRequests(reqs)

	slots := make(chan struct{}, maxSessionsPerConn)
	for newCh := range chans {
		if newCh.ChannelType() != "session" {
			_ = newCh.Reject(ssh.Prohibited, "only sessions are allowed")
			continue
		}
		select {
		case slots <- struct{}{}:
		default:
			_ = newCh.Reject(ssh.ResourceShortage, "too many sessions")
			continue
		}
		go func() {
			defer func() { <-slots }()
			handleSession(newCh)
		}()
	}
	log.Println("SSH: session disconnected")
}

type session struct {
	ch      ssh.Channel
	env     []string
	pty     *ptyRequest
	ptmx    *os.File
	cmd     *exec.Cmd
	mu      sync.Mutex
	started bool
}

type ptyRequest struct {
	term       string
	cols, rows uint32
}

func handleSession(newCh ssh.NewChannel) {
	ch, reqs, err := newCh.Accept()
	if err != nil {
		return
	}
	s := &session{ch: ch}
	defer s.kill()
	defer func() { _ = ch.Close() }()

	for req := range reqs {
		ok := false
		switch req.Type {
		case "env":
			name, rest, okName := readString(req.Payload)
			value, _, okValue := readString(rest)
			if okName && okValue && allowedEnv.MatchString(name) {
				s.env = append(s.env, name+"="+value)
				ok = true
			}
		case "pty-req":
			term, rest, okTerm := readString(req.Payload)
			if okTerm && len(rest) >= 8 {
				s.pty = &ptyRequest{term: term, cols: binary.BigEndian.Uint32(rest), rows: binary.BigEndian.Uint32(rest[4:])}
				ok = true
			}
		case "window-change":
			if len(req.Payload) >= 8 {
				s.resize(binary.BigEndian.Uint32(req.Payload), binary.BigEndian.Uint32(req.Payload[4:]))
			}
		case "shell":
			ok = s.start("", req)
		case "exec":
			if command, _, okCmd := readString(req.Payload); okCmd {
				ok = s.start(command, req)
			}
		case "subsystem":
			if name, _, okName := readString(req.Payload); okName && name == "sftp" {
				ok = s.startSFTP(req)
			}
		case "signal":
			if name, _, okName := readString(req.Payload); okName {
				s.signal(name)
			}
		}
		// start replies itself, before the output flows
		if req.WantReply && req.Type != "shell" && req.Type != "exec" && req.Type != "subsystem" {
			_ = req.Reply(ok, nil)
		} else if req.WantReply && !ok {
			_ = req.Reply(false, nil)
		}
	}
}

func (s *session) begin() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return false
	}
	s.started = true
	return true
}

// start runs the login shell, or a command through it
func (s *session) start(command string, req *ssh.Request) bool {
	if !s.begin() {
		return false
	}
	shell, home := loginShell()
	var cmd *exec.Cmd
	if command == "" {
		cmd = exec.Command(shell, "-l")
	} else {
		cmd = exec.Command(shell, "-c", command)
	}
	cmd.Dir = home
	cmd.Env = append(terminal.SanitizeEnv(os.Environ()), s.env...)
	if home != "" {
		cmd.Env = append(cmd.Env, "HOME="+home)
	}
	if s.pty != nil {
		cmd.Env = append(cmd.Env, "TERM="+s.pty.term)
		return s.runPTY(cmd, req)
	}
	return s.runPipes(cmd, req)
}

func (s *session) runPTY(cmd *exec.Cmd, req *ssh.Request) bool {
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(s.pty.cols), Rows: uint16(s.pty.rows)})
	if err != nil {
		log.Printf("SSH: cannot start PTY: %v", err)
		return false
	}
	s.mu.Lock()
	s.cmd, s.ptmx = cmd, ptmx
	s.mu.Unlock()
	_ = req.Reply(true, nil)

	go func() { _, _ = io.Copy(ptmx, s.ch) }()
	go func() {
		// a PTY reports EIO once the shell is gone: the output is drained
		_, _ = io.Copy(s.ch, ptmx)
		s.finish(cmd.Wait())
		_ = ptmx.Close()
	}()
	return true
}

func (s *session) runPipes(cmd *exec.Cmd, req *ssh.Request) bool {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, errIn := cmd.StdinPipe()
	stdout, errOut := cmd.StdoutPipe()
	stderr, errErr := cmd.StderrPipe()
	if err := errors.Join(errIn, errOut, errErr); err != nil {
		return false
	}
	if err := cmd.Start(); err != nil {
		log.Printf("SSH: cannot start command: %v", err)
		return false
	}
	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()
	_ = req.Reply(true, nil)

	// stdin EOF from the client must reach the command: rsync and scp rely on it
	go func() {
		_, _ = io.Copy(stdin, s.ch)
		_ = stdin.Close()
	}()
	go func() {
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); _, _ = io.Copy(s.ch, stdout) }()
		go func() { defer wg.Done(); _, _ = io.Copy(s.ch.Stderr(), stderr) }()
		wg.Wait()
		s.finish(cmd.Wait())
	}()
	return true
}

func (s *session) startSFTP(req *ssh.Request) bool {
	for _, path := range sftpServers {
		if _, err := os.Stat(path); err == nil {
			if !s.begin() {
				return false
			}
			_, home := loginShell()
			cmd := exec.Command(path)
			cmd.Dir = home
			cmd.Env = terminal.SanitizeEnv(os.Environ())
			return s.runPipes(cmd, req)
		}
	}
	if !s.begin() {
		return false
	}
	_, home := loginShell()
	server, err := sftp.NewServer(s.ch, sftp.WithServerWorkingDirectory(home))
	if err != nil {
		return false
	}
	_ = req.Reply(true, nil)
	go func() {
		err := server.Serve()
		if errors.Is(err, io.EOF) {
			err = nil
		}
		s.sendExit(exitCode(err), "")
		_ = s.ch.Close()
	}()
	return true
}

// finish reports how the process ended and closes the channel
func (s *session) finish(waitErr error) {
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			s.sendExit(0, signalName(status.Signal()))
			_ = s.ch.Close()
			return
		}
	}
	s.sendExit(exitCode(waitErr), "")
	_ = s.ch.Close()
}

func (s *session) sendExit(code int, signal string) {
	if signal != "" {
		payload := ssh.Marshal(struct {
			Signal     string
			CoreDumped bool
			Error      string
			Lang       string
		}{Signal: signal})
		_, _ = s.ch.SendRequest("exit-signal", false, payload)
		return
	}
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(code))
	_, _ = s.ch.SendRequest("exit-status", false, payload)
}

func (s *session) resize(cols, rows uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pty != nil {
		s.pty.cols, s.pty.rows = cols, rows
	}
	if s.ptmx != nil && cols > 0 && rows > 0 {
		_ = pty.Setsize(s.ptmx, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	}
}

func (s *session) signal(name string) {
	sig, ok := signals[name]
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Signal(sig)
	}
}

// kill ends what is still running when the client goes away
func (s *session) kill() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil || s.cmd.ProcessState != nil {
		return
	}
	// the whole group: a pipeline started by sh -c must not outlive the session
	_ = syscall.Kill(-s.cmd.Process.Pid, syscall.SIGKILL)
	_ = s.cmd.Process.Kill()
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 255
}

func signalName(sig syscall.Signal) string {
	for name, s := range signals {
		if s == sig {
			return name
		}
	}
	return "KILL"
}

// loginShell picks the shell and home of the user the client runs as
func loginShell() (shell, home string) {
	shell = os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
		if _, err := os.Stat("/bin/bash"); err == nil {
			shell = "/bin/bash"
		}
	}
	if u, err := user.Current(); err == nil {
		home = u.HomeDir
	}
	if home == "" {
		home = "/"
	}
	return shell, home
}

// readString reads an RFC 4251 string from an SSH payload
func readString(b []byte) (string, []byte, bool) {
	if len(b) < 4 {
		return "", nil, false
	}
	n := binary.BigEndian.Uint32(b)
	if uint32(len(b)-4) < n {
		return "", nil, false
	}
	return string(b[4 : 4+n]), b[4+n:], true
}
