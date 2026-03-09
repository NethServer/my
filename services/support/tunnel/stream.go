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
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConn wraps a gorilla/websocket.Conn to implement net.Conn
// for use with yamux, which requires a net.Conn interface.
type WebSocketConn struct {
	conn   *websocket.Conn
	reader io.Reader
}

// NewWebSocketConn wraps a WebSocket connection as a net.Conn
func NewWebSocketConn(conn *websocket.Conn) *WebSocketConn {
	return &WebSocketConn{conn: conn}
}

// Read reads data from the WebSocket connection
func (wsc *WebSocketConn) Read(b []byte) (int, error) {
	for {
		if wsc.reader == nil {
			_, reader, err := wsc.conn.NextReader()
			if err != nil {
				return 0, err
			}
			wsc.reader = reader
		}

		n, err := wsc.reader.Read(b)
		if err == io.EOF {
			wsc.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

// Write writes data to the WebSocket connection
func (wsc *WebSocketConn) Write(b []byte) (int, error) {
	err := wsc.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

// Close closes the underlying WebSocket connection
func (wsc *WebSocketConn) Close() error {
	return wsc.conn.Close()
}

// LocalAddr returns the local network address (not applicable for WebSocket)
func (wsc *WebSocketConn) LocalAddr() net.Addr {
	return wsc.conn.LocalAddr()
}

// RemoteAddr returns the remote network address
func (wsc *WebSocketConn) RemoteAddr() net.Addr {
	return wsc.conn.RemoteAddr()
}

// SetDeadline sets read and write deadlines on the underlying WebSocket connection
func (wsc *WebSocketConn) SetDeadline(t time.Time) error {
	if err := wsc.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return wsc.conn.SetWriteDeadline(t)
}

// SetReadDeadline sets the read deadline on the underlying WebSocket connection
func (wsc *WebSocketConn) SetReadDeadline(t time.Time) error {
	return wsc.conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the write deadline on the underlying WebSocket connection
func (wsc *WebSocketConn) SetWriteDeadline(t time.Time) error {
	return wsc.conn.SetWriteDeadline(t)
}
