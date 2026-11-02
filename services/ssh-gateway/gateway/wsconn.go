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
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsConn carries a byte stream over binary WebSocket messages, so the SSH
// server can run on a connection that reached us over HTTPS
type wsConn struct {
	ws         *websocket.Conn
	remoteAddr net.Addr
	reader     io.Reader
	writeMu    sync.Mutex
	done       chan struct{}
	doneOnce   sync.Once
}

// maxMessageSize bounds a single WebSocket message from the peer
const maxMessageSize = 1 << 20

func newWSConn(ws *websocket.Conn, remoteAddr net.Addr) *wsConn {
	ws.SetReadLimit(maxMessageSize)
	return &wsConn{ws: ws, remoteAddr: remoteAddr, done: make(chan struct{})}
}

// Done closes once the peer is gone or the connection is closed
func (c *wsConn) Done() <-chan struct{} { return c.done }

func (c *wsConn) markDone() { c.doneOnce.Do(func() { close(c.done) }) }

func (c *wsConn) Read(p []byte) (int, error) {
	for {
		if c.reader == nil {
			kind, r, err := c.ws.NextReader()
			if err != nil {
				c.markDone()
				return 0, io.EOF
			}
			if kind != websocket.BinaryMessage {
				continue
			}
			c.reader = r
		}
		n, err := c.reader.Read(p)
		if err == io.EOF {
			c.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (c *wsConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := c.ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// keepAlive pings the client so idle sessions survive the proxies in between
func (c *wsConn) keepAlive(interval time.Duration, done <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				c.markDone()
				return
			}
		}
	}
}

// Close says goodbye with a WebSocket close frame first: without it a client
// such as websocat keeps waiting on a connection that is gone
func (c *wsConn) Close() error {
	c.markDone()
	c.writeMu.Lock()
	_ = c.ws.WriteControl(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	c.writeMu.Unlock()
	return c.ws.Close()
}

func (c *wsConn) LocalAddr() net.Addr                { return c.ws.LocalAddr() }
func (c *wsConn) RemoteAddr() net.Addr               { return c.remoteAddr }
func (c *wsConn) SetDeadline(t time.Time) error      { return c.ws.NetConn().SetDeadline(t) }
func (c *wsConn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *wsConn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }
