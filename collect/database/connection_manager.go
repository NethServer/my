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
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// ConnectionManager manages database connections with intelligent pooling
type ConnectionManager struct {
	db                  *sql.DB
	maxConnections      int
	currentConnections  int
	connectionSemaphore chan struct{}
	metrics             *ConnectionMetrics
	mu                  sync.RWMutex
}

// ConnectionMetrics tracks connection pool usage
type ConnectionMetrics struct {
	AcquiredConnections  int64
	ReleasedConnections  int64
	WaitingForConnection int64
	MaxWaitTime          time.Duration
	TotalWaitTime        time.Duration
	mu                   sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager(db *sql.DB, maxConnections int) *ConnectionManager {
	return &ConnectionManager{
		db:                  db,
		maxConnections:      maxConnections,
		connectionSemaphore: make(chan struct{}, maxConnections),
		metrics:             &ConnectionMetrics{},
	}
}

// AcquireConnection acquires a connection from the pool with timeout
func (cm *ConnectionManager) AcquireConnection(ctx context.Context) (*ManagedConnection, error) {
	start := time.Now()

	// Update waiting metrics
	cm.metrics.mu.Lock()
	cm.metrics.WaitingForConnection++
	cm.metrics.mu.Unlock()

	// Wait for available connection slot
	select {
	case cm.connectionSemaphore <- struct{}{}:
		// Got slot, continue
	case <-ctx.Done():
		cm.metrics.mu.Lock()
		cm.metrics.WaitingForConnection--
		cm.metrics.mu.Unlock()
		return nil, ctx.Err()
	}

	waitTime := time.Since(start)

	// Update metrics
	cm.metrics.mu.Lock()
	cm.metrics.AcquiredConnections++
	cm.metrics.WaitingForConnection--
	cm.metrics.TotalWaitTime += waitTime
	if waitTime > cm.metrics.MaxWaitTime {
		cm.metrics.MaxWaitTime = waitTime
	}
	cm.metrics.mu.Unlock()

	cm.mu.Lock()
	cm.currentConnections++
	cm.mu.Unlock()

	return &ManagedConnection{
		db:       cm.db,
		manager:  cm,
		acquired: time.Now(),
	}, nil
}

// ReleaseConnection releases a connection back to the pool
func (cm *ConnectionManager) ReleaseConnection() {
	cm.mu.Lock()
	cm.currentConnections--
	cm.mu.Unlock()

	cm.metrics.mu.Lock()
	cm.metrics.ReleasedConnections++
	cm.metrics.mu.Unlock()

	// Release semaphore slot
	<-cm.connectionSemaphore
}

// GetMetrics returns current connection metrics
func (cm *ConnectionManager) GetMetrics() ConnectionMetrics {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()
	return ConnectionMetrics{
		AcquiredConnections:  cm.metrics.AcquiredConnections,
		ReleasedConnections:  cm.metrics.ReleasedConnections,
		WaitingForConnection: cm.metrics.WaitingForConnection,
		MaxWaitTime:          cm.metrics.MaxWaitTime,
		TotalWaitTime:        cm.metrics.TotalWaitTime,
	}
}

// GetCurrentConnections returns current active connections
func (cm *ConnectionManager) GetCurrentConnections() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.currentConnections
}

// ManagedConnection wraps a database connection with lifecycle management
type ManagedConnection struct {
	db       *sql.DB
	manager  *ConnectionManager
	acquired time.Time
	released bool
	mu       sync.Mutex
}

// QueryRowContext executes a query that returns a single row
func (mc *ManagedConnection) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return mc.db.QueryRowContext(ctx, query, args...)
}

// QueryContext executes a query that returns multiple rows
func (mc *ManagedConnection) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return mc.db.QueryContext(ctx, query, args...)
}

// ExecContext executes a query without returning rows
func (mc *ManagedConnection) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return mc.db.ExecContext(ctx, query, args...)
}

// BeginTx starts a transaction
func (mc *ManagedConnection) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return mc.db.BeginTx(ctx, opts)
}

// PrepareContext creates a prepared statement
func (mc *ManagedConnection) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return mc.db.PrepareContext(ctx, query)
}

// Release releases the connection back to the pool
func (mc *ManagedConnection) Release() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.released {
		return // Already released
	}

	mc.released = true
	mc.manager.ReleaseConnection()

	// Log slow connections for monitoring
	duration := time.Since(mc.acquired)
	if duration > 30*time.Second {
		logger.ComponentLogger("connection-manager").Warn().
			Dur("duration", duration).
			Msg("Long-running connection detected")
	}
}

// Global connection manager instance
var connectionManager *ConnectionManager

// InitConnectionManager initializes the global connection manager
func InitConnectionManager() {
	if DB == nil {
		logger.ComponentLogger("database").Error().Msg("Database not initialized before connection manager")
		return
	}

	// Use 80% of max connections for managed connections
	maxManaged := int(float64(50) * 0.8) // 40 connections for managed operations
	connectionManager = NewConnectionManager(DB, maxManaged)

	logger.ComponentLogger("database").Info().
		Int("max_managed_connections", maxManaged).
		Msg("Connection manager initialized")
}

// GetManagedConnection returns a managed connection from the global pool
func GetManagedConnection(ctx context.Context) (*ManagedConnection, error) {
	if connectionManager == nil {
		return nil, fmt.Errorf("connection manager not initialized")
	}
	return connectionManager.AcquireConnection(ctx)
}

// GetConnectionMetrics returns current connection metrics
func GetConnectionMetrics() *ConnectionMetrics {
	if connectionManager == nil {
		return nil
	}
	metrics := connectionManager.GetMetrics()
	return &metrics
}
