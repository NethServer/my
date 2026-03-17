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
	"encoding/binary"
	"fmt"
	"io"
)

const maxFrameSize = 1024 * 1024 // 1 MB

// writeFrame writes a length-prefixed frame: [4 bytes big-endian length][payload]
func writeFrame(w io.Writer, data []byte) error {
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))
	if _, err := w.Write(header); err != nil {
		return err
	}
	_, err := w.Write(data)
	return err
}

// readFrame reads a length-prefixed frame: [4 bytes big-endian length][payload]
func readFrame(r io.Reader) ([]byte, error) {
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

// writeDataFrame writes a type-0 (data) frame
func writeDataFrame(w io.Writer, data []byte) error {
	frame := make([]byte, 1+len(data))
	frame[0] = 0 // type 0 = data
	copy(frame[1:], data)
	return writeFrame(w, frame)
}

// writeResizeFrame writes a type-1 (resize) frame with JSON payload
func writeResizeFrame(w io.Writer, payload []byte) error {
	frame := make([]byte, 1+len(payload))
	frame[0] = 1 // type 1 = resize
	copy(frame[1:], payload)
	return writeFrame(w, frame)
}
