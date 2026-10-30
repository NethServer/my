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
	"bytes"
	"fmt"
	"testing"
)

func TestWriteReadConnectHeader(t *testing.T) {
	var buf bytes.Buffer

	err := WriteConnectHeader(&buf, "cluster-admin")
	if err != nil {
		t.Fatalf("WriteConnectHeader failed: %v", err)
	}

	if buf.String() != "CONNECT cluster-admin\n" {
		t.Fatalf("unexpected header: %q", buf.String())
	}

	serviceName, err := ReadConnectHeader(&buf)
	if err != nil {
		t.Fatalf("ReadConnectHeader failed: %v", err)
	}
	if serviceName != "cluster-admin" {
		t.Fatalf("expected service name 'cluster-admin', got %q", serviceName)
	}
}

func TestReadConnectHeader_InvalidPrefix(t *testing.T) {
	buf := bytes.NewBufferString("GET /foo\n")
	_, err := ReadConnectHeader(buf)
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
}

func TestReadConnectHeader_EmptyServiceName(t *testing.T) {
	buf := bytes.NewBufferString("CONNECT \n")
	_, err := ReadConnectHeader(buf)
	if err == nil {
		t.Fatal("expected error for empty service name")
	}
}

func TestWriteReadConnectResponse_OK(t *testing.T) {
	var buf bytes.Buffer

	err := WriteConnectResponse(&buf, nil)
	if err != nil {
		t.Fatalf("WriteConnectResponse failed: %v", err)
	}

	if buf.String() != "OK\n" {
		t.Fatalf("unexpected response: %q", buf.String())
	}

	err = ReadConnectResponse(&buf)
	if err != nil {
		t.Fatalf("ReadConnectResponse returned error for OK: %v", err)
	}
}

func TestWriteReadConnectResponse_Error(t *testing.T) {
	var buf bytes.Buffer

	err := WriteConnectResponse(&buf, fmt.Errorf("service not found"))
	if err != nil {
		t.Fatalf("WriteConnectResponse failed: %v", err)
	}

	if buf.String() != "ERROR service not found\n" {
		t.Fatalf("unexpected response: %q", buf.String())
	}

	err = ReadConnectResponse(&buf)
	if err == nil {
		t.Fatal("expected error from ReadConnectResponse")
	}
	if err.Error() != "service not found" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestReadConnectResponse_UnexpectedResponse(t *testing.T) {
	buf := bytes.NewBufferString("WHAT\n")
	err := ReadConnectResponse(buf)
	if err == nil {
		t.Fatal("expected error for unexpected response")
	}
}

func TestReadLine_TooLong(t *testing.T) {
	// Create a line longer than 1024 bytes without newline
	longData := make([]byte, 2000)
	for i := range longData {
		longData[i] = 'a'
	}
	buf := bytes.NewBuffer(longData)
	_, err := readLine(buf)
	if err == nil {
		t.Fatal("expected error for line too long")
	}
}
