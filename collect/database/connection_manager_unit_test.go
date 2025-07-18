/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConnectionManager(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	assert.NotNil(t, manager)
	assert.Equal(t, 10, manager.maxConnections)
	assert.Equal(t, 0, manager.currentConnections)
	assert.NotNil(t, manager.connectionSemaphore)
	assert.NotNil(t, manager.metrics)
}

func TestConnectionManagerWithDifferentSizes(t *testing.T) {
	tests := []struct {
		name           string
		maxConnections int
	}{
		{
			name:           "small pool",
			maxConnections: 5,
		},
		{
			name:           "medium pool",
			maxConnections: 20,
		},
		{
			name:           "large pool",
			maxConnections: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewConnectionManager(nil, tt.maxConnections)
			assert.NotNil(t, manager)
			assert.Equal(t, tt.maxConnections, manager.maxConnections)
		})
	}
}

func TestConnectionManagerGetCurrentConnections(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	// Initially should be 0
	assert.Equal(t, 0, manager.GetCurrentConnections())

	// Simulate connection acquisition
	manager.mu.Lock()
	manager.currentConnections = 5
	manager.mu.Unlock()

	assert.Equal(t, 5, manager.GetCurrentConnections())
}

func TestConnectionManagerGetMetrics(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	metrics := manager.GetMetrics()
	assert.Equal(t, int64(0), metrics.AcquiredConnections)
	assert.Equal(t, int64(0), metrics.ReleasedConnections)
	assert.Equal(t, int64(0), metrics.WaitingForConnection)
	assert.Equal(t, time.Duration(0), metrics.MaxWaitTime)
	assert.Equal(t, time.Duration(0), metrics.TotalWaitTime)
}

func TestConnectionManagerReleaseConnection(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	// Simulate having connections
	manager.mu.Lock()
	manager.currentConnections = 1
	manager.mu.Unlock()

	// Add to semaphore to simulate acquired connection
	manager.connectionSemaphore <- struct{}{}

	// Release connection
	manager.ReleaseConnection()

	// Check that connection count decreased
	assert.Equal(t, 0, manager.GetCurrentConnections())

	// Check that metrics were updated
	metrics := manager.GetMetrics()
	assert.Equal(t, int64(1), metrics.ReleasedConnections)
}

func TestConnectionMetricsStruct(t *testing.T) {
	metrics := &ConnectionMetrics{
		AcquiredConnections:  5,
		ReleasedConnections:  3,
		WaitingForConnection: 2,
		MaxWaitTime:          100 * time.Millisecond,
		TotalWaitTime:        500 * time.Millisecond,
	}

	assert.Equal(t, int64(5), metrics.AcquiredConnections)
	assert.Equal(t, int64(3), metrics.ReleasedConnections)
	assert.Equal(t, int64(2), metrics.WaitingForConnection)
	assert.Equal(t, 100*time.Millisecond, metrics.MaxWaitTime)
	assert.Equal(t, 500*time.Millisecond, metrics.TotalWaitTime)
}

func TestManagedConnectionRelease(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	// Create managed connection
	conn := &ManagedConnection{
		db:       nil,
		manager:  manager,
		acquired: time.Now(),
		released: false,
	}

	// Simulate connection being in semaphore
	manager.connectionSemaphore <- struct{}{}
	manager.mu.Lock()
	manager.currentConnections = 1
	manager.mu.Unlock()

	// Release connection
	conn.Release()

	// Check that connection was released
	assert.True(t, conn.released)
	assert.Equal(t, 0, manager.GetCurrentConnections())
}

func TestManagedConnectionDoubleRelease(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	// Create managed connection
	conn := &ManagedConnection{
		db:       nil,
		manager:  manager,
		acquired: time.Now(),
		released: false,
	}

	// Simulate connection being in semaphore
	manager.connectionSemaphore <- struct{}{}
	manager.mu.Lock()
	manager.currentConnections = 1
	manager.mu.Unlock()

	// Release connection twice
	conn.Release()
	conn.Release()

	// Should still be released and not cause issues
	assert.True(t, conn.released)
	assert.Equal(t, 0, manager.GetCurrentConnections())
}

func TestContextTimeout(t *testing.T) {
	// Test context timeout handling
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Verify timeout is set
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, time.Until(deadline) <= 100*time.Millisecond)

	// Test context cancellation
	select {
	case <-ctx.Done():
		assert.Error(t, ctx.Err())
	case <-time.After(200 * time.Millisecond):
		t.Error("Context should have timed out")
	}
}

func TestConnectionManagerSemaphore(t *testing.T) {
	manager := NewConnectionManager(nil, 2)

	// Test semaphore capacity
	assert.Equal(t, 2, cap(manager.connectionSemaphore))
	assert.Equal(t, 0, len(manager.connectionSemaphore))

	// Add to semaphore
	manager.connectionSemaphore <- struct{}{}
	assert.Equal(t, 1, len(manager.connectionSemaphore))

	manager.connectionSemaphore <- struct{}{}
	assert.Equal(t, 2, len(manager.connectionSemaphore))

	// Remove from semaphore
	<-manager.connectionSemaphore
	assert.Equal(t, 1, len(manager.connectionSemaphore))
}

func TestConnectionManagerMetricsThreadSafety(t *testing.T) {
	manager := NewConnectionManager(nil, 10)

	// Test concurrent metric updates
	done := make(chan bool)

	go func() {
		for i := 0; i < 100; i++ {
			manager.metrics.mu.Lock()
			manager.metrics.AcquiredConnections++
			manager.metrics.mu.Unlock()
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			manager.metrics.mu.Lock()
			manager.metrics.ReleasedConnections++
			manager.metrics.mu.Unlock()
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	metrics := manager.GetMetrics()
	assert.Equal(t, int64(100), metrics.AcquiredConnections)
	assert.Equal(t, int64(100), metrics.ReleasedConnections)
}

func TestGetManagedConnection(t *testing.T) {
	// Test global connection manager functions
	// This will return error since no database is initialized
	ctx := context.Background()
	conn, err := GetManagedConnection(ctx)
	assert.Error(t, err)
	assert.Nil(t, conn)
	assert.Contains(t, err.Error(), "connection manager not initialized")
}

func TestGetConnectionMetrics(t *testing.T) {
	// Test global connection metrics function
	// This will return nil since no connection manager is initialized
	metrics := GetConnectionMetrics()
	assert.Nil(t, metrics)
}
