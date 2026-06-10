/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package connection

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WsNetConn wraps gorilla/websocket.Conn as net.Conn for yamux.
// It captures WebSocket close errors so the reconnect loop can inspect the close code.
type WsNetConn struct {
	conn     *websocket.Conn
	reader   io.Reader
	mu       sync.Mutex
	closeErr error // stores the WebSocket close error if received
}

func (w *WsNetConn) Read(b []byte) (int, error) {
	for {
		if w.reader == nil {
			_, reader, err := w.conn.NextReader()
			if err != nil {
				w.mu.Lock()
				w.closeErr = err
				w.mu.Unlock()
				return 0, err
			}
			w.reader = reader
		}
		n, err := w.reader.Read(b)
		if err == io.EOF {
			w.reader = nil
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (w *WsNetConn) Write(b []byte) (int, error) {
	err := w.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func (w *WsNetConn) Close() error         { return w.conn.Close() }
func (w *WsNetConn) LocalAddr() net.Addr  { return w.conn.LocalAddr() }
func (w *WsNetConn) RemoteAddr() net.Addr { return w.conn.RemoteAddr() }
func (w *WsNetConn) SetDeadline(t time.Time) error {
	if err := w.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return w.conn.SetWriteDeadline(t)
}
func (w *WsNetConn) SetReadDeadline(t time.Time) error  { return w.conn.SetReadDeadline(t) }
func (w *WsNetConn) SetWriteDeadline(t time.Time) error { return w.conn.SetWriteDeadline(t) }
