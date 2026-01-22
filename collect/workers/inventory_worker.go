/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package workers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// InventoryWorker handles high-throughput batch operations
type InventoryWorker struct {
	inventoryBatch chan *models.InventoryData
	batchSize      int
	flushInterval  time.Duration
	stopCh         chan struct{}
	isHealthy      bool
	mu             sync.RWMutex
	processedCount int64
	failedCount    int64
	lastFlush      time.Time
	queueManager   *queue.QueueManager
}

// NewInventoryWorker creates a new inventory worker
func NewInventoryWorker(batchSize int, flushInterval time.Duration) *InventoryWorker {
	return &InventoryWorker{
		inventoryBatch: make(chan *models.InventoryData, batchSize*2), // Buffer 2x batch size
		batchSize:      batchSize,
		flushInterval:  flushInterval,
		stopCh:         make(chan struct{}),
		isHealthy:      true,
		lastFlush:      time.Now(),
		queueManager:   queue.NewQueueManager(),
	}
}

// Start starts the inventory worker
func (iw *InventoryWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	wg.Add(1)
	go iw.batchWorker(ctx, wg)
	return nil
}

// Name returns the worker name
func (iw *InventoryWorker) Name() string {
	return "inventory-worker"
}

// IsHealthy returns health status
func (iw *InventoryWorker) IsHealthy() bool {
	iw.mu.RLock()
	defer iw.mu.RUnlock()
	return iw.isHealthy
}

// AddInventory adds inventory data to the batch for processing
func (iw *InventoryWorker) AddInventory(ctx context.Context, inventory *models.InventoryData) error {
	select {
	case iw.inventoryBatch <- inventory:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
		return fmt.Errorf("inventory worker queue full, dropping inventory")
	}
}

// batchWorker processes inventory data in batches
func (iw *InventoryWorker) batchWorker(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	logger := logger.ComponentLogger("inventory-worker")
	logger.Info().
		Int("batch_size", iw.batchSize).
		Dur("flush_interval", iw.flushInterval).
		Msg("Inventory worker started")

	ticker := time.NewTicker(iw.flushInterval)
	defer ticker.Stop()

	batch := make([]*models.InventoryData, 0, iw.batchSize)

	for {
		select {
		case <-ctx.Done():
			// Process remaining items in batch before stopping
			if len(batch) > 0 {
				iw.processBatch(ctx, batch, *logger)
			}
			logger.Info().Msg("Inventory worker stopped")
			return

		case inventory := <-iw.inventoryBatch:
			batch = append(batch, inventory)

			// Process batch when it reaches target size
			if len(batch) >= iw.batchSize {
				iw.processBatch(ctx, batch, *logger)
				batch = batch[:0] // Reset batch
				iw.updateLastFlush()
			}

		case <-ticker.C:
			// Process batch on timer if it has items
			if len(batch) > 0 {
				iw.processBatch(ctx, batch, *logger)
				batch = batch[:0] // Reset batch
				iw.updateLastFlush()
			}
		}
	}
}

// processBatch processes a batch of inventory data
func (iw *InventoryWorker) processBatch(ctx context.Context, batch []*models.InventoryData, logger zerolog.Logger) {
	if len(batch) == 0 {
		return
	}

	start := time.Now()
	logger.Info().
		Int("batch_size", len(batch)).
		Msg("Processing inventory batch")

	// Get managed connection
	conn, err := database.GetManagedConnection(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Int("batch_size", len(batch)).
			Msg("Failed to acquire database connection for batch")
		iw.recordFailure(int64(len(batch)))
		return
	}
	defer conn.Release()

	// Process batch in transaction
	if err := iw.processBatchInTransaction(ctx, conn, batch, logger); err != nil {
		logger.Error().
			Err(err).
			Int("batch_size", len(batch)).
			Msg("Failed to process batch")
		iw.recordFailure(int64(len(batch)))
		return
	}

	duration := time.Since(start)
	iw.recordSuccess(int64(len(batch)))

	logger.Info().
		Int("batch_size", len(batch)).
		Dur("duration", duration).
		Float64("items_per_second", float64(len(batch))/duration.Seconds()).
		Msg("Batch processed successfully")
}

// processBatchInTransaction processes a batch within a single transaction
func (iw *InventoryWorker) processBatchInTransaction(ctx context.Context, conn *database.ManagedConnection, batch []*models.InventoryData, logger zerolog.Logger) error {
	// Start transaction with timeout
	txCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tx, err := conn.BeginTx(txCtx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			logger.Error().Err(err).Msg("Failed to rollback transaction")
		}
	}()

	// Prepare statement for batch insert
	stmt, err := tx.PrepareContext(txCtx, `
		INSERT INTO inventory_records 
		(system_id, timestamp, data, data_hash, data_size, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		ON CONFLICT (system_id, data_hash) 
		DO UPDATE SET 
			timestamp = EXCLUDED.timestamp,
			updated_at = NOW()
		RETURNING id
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare batch insert statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			logger.Error().Err(err).Msg("Failed to close prepared statement")
		}
	}()

	// Track inserted records for diff processing
	var insertedRecords []models.InventoryRecord

	// Process each item in batch
	for _, inventory := range batch {
		dataHash := iw.calculateDataHash(inventory.Data)
		dataSize := int64(len(inventory.Data))

		var recordID int64
		err := stmt.QueryRowContext(txCtx,
			inventory.SystemID,
			inventory.Timestamp,
			inventory.Data,
			dataHash,
			dataSize,
		).Scan(&recordID)

		if err != nil {
			return fmt.Errorf("failed to insert inventory for system %s: %w", inventory.SystemID, err)
		}

		// Create record for diff processing
		insertedRecords = append(insertedRecords, models.InventoryRecord{
			ID:        recordID,
			SystemID:  inventory.SystemID,
			Timestamp: inventory.Timestamp,
			Data:      inventory.Data,
			DataHash:  dataHash,
			DataSize:  dataSize,
		})

		// Log large payloads for monitoring
		if dataSize > 1024*1024 { // 1MB
			logger.Warn().
				Str("system_id", inventory.SystemID).
				Int64("data_size", dataSize).
				Int64("record_id", recordID).
				Msg("Large inventory payload processed")
		}
	}

	// Update system fields and extract applications from inventory data before committing
	for _, record := range insertedRecords {
		if err := iw.updateSystemFieldsFromInventory(txCtx, tx, &record, logger); err != nil {
			logger.Warn().
				Err(err).
				Str("system_id", record.SystemID).
				Int64("record_id", record.ID).
				Msg("Failed to update system fields from inventory")
			// Continue processing other records even if one fails
		}

		// Extract applications from NS8 inventory
		if err := iw.extractApplicationsFromInventory(txCtx, tx, &record, logger); err != nil {
			logger.Warn().
				Err(err).
				Str("system_id", record.SystemID).
				Int64("record_id", record.ID).
				Msg("Failed to extract applications from inventory")
			// Continue processing other records even if one fails
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	// After successful commit, trigger diff processing asynchronously
	go iw.triggerDiffProcessingAsync(ctx, insertedRecords, logger)

	return nil
}

// calculateDataHash calculates SHA-256 hash of inventory data
func (iw *InventoryWorker) calculateDataHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// updateLastFlush updates the last flush timestamp
func (iw *InventoryWorker) updateLastFlush() {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.lastFlush = time.Now()
}

// recordSuccess records successful batch processing
func (iw *InventoryWorker) recordSuccess(count int64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.processedCount += count
	iw.isHealthy = true
}

// recordFailure records failed batch processing
func (iw *InventoryWorker) recordFailure(count int64) {
	iw.mu.Lock()
	defer iw.mu.Unlock()
	iw.failedCount += count
	iw.isHealthy = false
}

// GetStats returns inventory worker statistics
func (iw *InventoryWorker) GetStats() map[string]interface{} {
	iw.mu.RLock()
	defer iw.mu.RUnlock()

	return map[string]interface{}{
		"processed_count": iw.processedCount,
		"failed_count":    iw.failedCount,
		"queue_length":    len(iw.inventoryBatch),
		"queue_capacity":  cap(iw.inventoryBatch),
		"last_flush":      iw.lastFlush,
		"is_healthy":      iw.isHealthy,
		"batch_size":      iw.batchSize,
		"flush_interval":  iw.flushInterval,
	}
}

// triggerDiffProcessingAsync triggers diff processing asynchronously for inserted inventory records
func (iw *InventoryWorker) triggerDiffProcessingAsync(ctx context.Context, insertedRecords []models.InventoryRecord, logger zerolog.Logger) {
	// Create a new context with timeout to prevent goroutine leaks
	asyncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	successCount := 0
	failedCount := 0

	for _, record := range insertedRecords {
		// Check if there's a previous record for this system
		previousRecord, err := iw.getPreviousInventoryRecord(asyncCtx, record.SystemID, record.ID)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("system_id", record.SystemID).
				Int64("record_id", record.ID).
				Msg("Failed to get previous inventory record for diff processing")
			failedCount++
			continue
		}

		// Only trigger diff processing if there's a previous record
		if previousRecord != nil {
			processingJob := &models.InventoryProcessingJob{
				InventoryRecord: &record,
				SystemID:        record.SystemID,
				ForceProcess:    false,
			}

			// Use a timeout context for each enqueue operation
			enqueueCtx, enqueueCancel := context.WithTimeout(asyncCtx, 30*time.Second)
			err := iw.queueManager.EnqueueProcessing(enqueueCtx, processingJob)
			enqueueCancel()

			if err != nil {
				logger.Warn().
					Err(err).
					Str("system_id", record.SystemID).
					Int64("inventory_id", record.ID).
					Msg("Failed to enqueue processing job for diff computation (non-blocking)")
				failedCount++
			} else {
				logger.Debug().
					Str("system_id", record.SystemID).
					Int64("inventory_id", record.ID).
					Int64("previous_id", previousRecord.ID).
					Msg("Queued diff processing job")
				successCount++
			}
		}
	}

	logger.Info().
		Int("total_records", len(insertedRecords)).
		Int("diff_jobs_queued", successCount).
		Int("diff_jobs_failed", failedCount).
		Msg("Async diff processing completed")
}

// getPreviousInventoryRecord gets the most recent previous inventory record for a system
func (iw *InventoryWorker) getPreviousInventoryRecord(ctx context.Context, systemID string, currentID int64) (*models.InventoryRecord, error) {
	query := `
		SELECT id, system_id, timestamp, data, data_hash, data_size,
		       processed_at, has_changes, change_count, created_at, updated_at
		FROM inventory_records
		WHERE system_id = $1 AND id < $2
		ORDER BY timestamp DESC, id DESC
		LIMIT 1
	`

	record := &models.InventoryRecord{}
	err := database.DB.QueryRowContext(ctx, query, systemID, currentID).Scan(
		&record.ID,
		&record.SystemID,
		&record.Timestamp,
		&record.Data,
		&record.DataHash,
		&record.DataSize,
		&record.ProcessedAt,
		&record.HasChanges,
		&record.ChangeCount,
		&record.CreatedAt,
		&record.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No previous record found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get previous inventory record: %w", err)
	}

	return record, nil
}

// updateSystemFieldsFromInventory extracts relevant fields from inventory and updates the systems table
// Supports both NS8 (nethserver) and NSEC (nethsecurity) inventory structures
func (iw *InventoryWorker) updateSystemFieldsFromInventory(ctx context.Context, tx *sql.Tx, record *models.InventoryRecord, logger zerolog.Logger) error {
	// Parse inventory data
	var inventoryData map[string]interface{}
	if err := json.Unmarshal(record.Data, &inventoryData); err != nil {
		return fmt.Errorf("failed to unmarshal inventory data: %w", err)
	}

	// Detect installation type
	installation, _ := inventoryData["installation"].(string)

	// Extract system fields based on installation type
	var name, fqdn, version, systemType, ipv4 *string

	switch installation {
	case "nethserver": // NS8
		systemType = strPtr("ns8")

		// Get facts object
		facts, ok := inventoryData["facts"].(map[string]interface{})
		if !ok {
			return nil // No facts, nothing to extract
		}

		// Extract cluster info
		if cluster, ok := facts["cluster"].(map[string]interface{}); ok {
			// System name from cluster.label (when available)
			if label, ok := cluster["label"].(string); ok && label != "" {
				name = &label
			}
			// FQDN from cluster.fqdn (when available)
			if fqdnVal, ok := cluster["fqdn"].(string); ok && fqdnVal != "" {
				fqdn = &fqdnVal
			}
			// Public IP from cluster.public_ip (when available)
			if publicIP, ok := cluster["public_ip"].(string); ok && publicIP != "" {
				ipv4 = &publicIP
			}
		}

		// Extract version from leader node (node "1" or first available)
		if nodes, ok := facts["nodes"].(map[string]interface{}); ok {
			// Try node "1" first (leader), then any available node
			nodeIDs := []string{"1"}
			for nodeID := range nodes {
				if nodeID != "1" {
					nodeIDs = append(nodeIDs, nodeID)
				}
			}

			for _, nodeID := range nodeIDs {
				if nodeData, ok := nodes[nodeID].(map[string]interface{}); ok {
					if nodeVersion, ok := nodeData["version"].(string); ok && nodeVersion != "" {
						version = &nodeVersion
						break
					}
				}
			}
		}

	case "nethsecurity": // NSEC
		systemType = strPtr("nsec")

		// Get facts object
		facts, ok := inventoryData["facts"].(map[string]interface{})
		if !ok {
			return nil
		}

		// Extract from distro
		if distro, ok := facts["distro"].(map[string]interface{}); ok {
			if distroVersion, ok := distro["version"].(string); ok && distroVersion != "" {
				version = &distroVersion
			}
		}

		// FQDN and public_ip will be added when available in NSEC inventory

	default:
		// Unknown installation type, try legacy structure
		// Extract IPv4 from data.public_ip (legacy)
		if publicIP, ok := inventoryData["public_ip"].(string); ok && publicIP != "" {
			ipv4 = &publicIP
		}

		// Extract FQDN from data.networking.fqdn (legacy)
		if networking, ok := inventoryData["networking"].(map[string]interface{}); ok {
			if fqdnVal, ok := networking["fqdn"].(string); ok && fqdnVal != "" {
				fqdn = &fqdnVal
			}
		}

		// Extract version from data.os.release.full (legacy)
		if os, ok := inventoryData["os"].(map[string]interface{}); ok {
			if release, ok := os["release"].(map[string]interface{}); ok {
				if fullVersion, ok := release["full"].(string); ok && fullVersion != "" {
					version = &fullVersion
				}
			}

			// Extract type from data.os.type (legacy)
			if osType, ok := os["type"].(string); ok && osType != "" {
				var productName string
				switch osType {
				case "nethserver":
					productName = "ns8"
				case "nethsecurity":
					productName = "nsec"
				default:
					productName = osType
				}
				systemType = &productName
			}
		}
	}

	// Build UPDATE query dynamically for non-null fields
	updates := []string{}
	args := []interface{}{}
	argPos := 1

	if name != nil {
		updates = append(updates, fmt.Sprintf("name = $%d", argPos))
		args = append(args, *name)
		argPos++
	}
	if fqdn != nil {
		updates = append(updates, fmt.Sprintf("fqdn = $%d", argPos))
		args = append(args, *fqdn)
		argPos++
	}
	if version != nil {
		updates = append(updates, fmt.Sprintf("version = $%d", argPos))
		args = append(args, *version)
		argPos++
	}
	if systemType != nil {
		updates = append(updates, fmt.Sprintf("type = $%d", argPos))
		args = append(args, *systemType)
		argPos++
	}
	if ipv4 != nil {
		updates = append(updates, fmt.Sprintf("ipv4_address = $%d", argPos))
		args = append(args, *ipv4)
		argPos++
	}

	// Always update updated_at timestamp
	updates = append(updates, "updated_at = NOW()")

	// If no fields to update, skip
	if len(updates) == 1 { // Only updated_at
		return nil
	}

	// Add system_id as last argument
	args = append(args, record.SystemID)

	// Execute UPDATE
	query := fmt.Sprintf(`
		UPDATE systems
		SET %s
		WHERE id = $%d AND deleted_at IS NULL
	`, strings.Join(updates, ", "), argPos)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update system fields: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		logger.Debug().
			Str("system_id", record.SystemID).
			Int("fields_updated", len(updates)-1).
			Msg("System fields updated from inventory")
	}

	return nil
}

// strPtr returns a pointer to a string
func strPtr(s string) *string {
	return &s
}

// extractApplicationsFromInventory extracts modules from NS8 inventory and upserts them into applications table
func (iw *InventoryWorker) extractApplicationsFromInventory(ctx context.Context, tx *sql.Tx, record *models.InventoryRecord, logger zerolog.Logger) error {
	// Parse inventory data
	var inventoryData map[string]interface{}
	if err := json.Unmarshal(record.Data, &inventoryData); err != nil {
		return fmt.Errorf("failed to unmarshal inventory data: %w", err)
	}

	// Only process NS8 inventories (nethserver)
	installation, _ := inventoryData["installation"].(string)
	if installation != "nethserver" {
		return nil // NSEC doesn't have modules
	}

	// Get facts object
	facts, ok := inventoryData["facts"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Get modules array
	modulesRaw, ok := facts["modules"].([]interface{})
	if !ok || len(modulesRaw) == 0 {
		return nil
	}

	// Get FQDN for URL generation (from cluster or system record)
	var systemFQDN string
	if cluster, ok := facts["cluster"].(map[string]interface{}); ok {
		if fqdn, ok := cluster["fqdn"].(string); ok {
			systemFQDN = fqdn
		}
	}
	// If no FQDN in cluster, try to get from system record
	if systemFQDN == "" {
		var fqdn sql.NullString
		err := tx.QueryRowContext(ctx, "SELECT fqdn FROM systems WHERE id = $1", record.SystemID).Scan(&fqdn)
		if err == nil && fqdn.Valid {
			systemFQDN = fqdn.String
		}
	}

	// Get nodes info for node_label lookup
	nodesData := make(map[string]map[string]interface{})
	if nodes, ok := facts["nodes"].(map[string]interface{}); ok {
		for nodeID, nodeInfo := range nodes {
			if nodeMap, ok := nodeInfo.(map[string]interface{}); ok {
				nodesData[nodeID] = nodeMap
			}
		}
	}

	// Track which module IDs we've seen in this inventory
	seenModuleIDs := make(map[string]bool)

	// Process each module
	for _, moduleRaw := range modulesRaw {
		module, ok := moduleRaw.(map[string]interface{})
		if !ok {
			continue
		}

		// Extract fixed fields
		moduleID, _ := module["id"].(string)
		moduleName, _ := module["name"].(string) // This is instance_of
		moduleVersion, _ := module["version"].(string)
		moduleNodeStr, _ := module["node"].(string)
		moduleLabel, _ := module["label"].(string) // display_name (when available)

		if moduleID == "" || moduleName == "" {
			continue // Skip invalid modules
		}

		seenModuleIDs[moduleID] = true

		// Parse node ID
		var nodeID *int
		if moduleNodeStr != "" {
			if n, err := strconv.Atoi(moduleNodeStr); err == nil {
				nodeID = &n
			}
		}

		// Get node label from nodes data
		var nodeLabel *string
		if moduleNodeStr != "" {
			if nodeInfo, ok := nodesData[moduleNodeStr]; ok {
				if label, ok := nodeInfo["label"].(string); ok && label != "" {
					nodeLabel = &label
				}
			}
		}

		// Determine if user-facing
		isUserFacing := configuration.IsUserFacingModule(moduleName)

		// Generate application URL
		var appURL *string
		if systemFQDN != "" && isUserFacing {
			url := configuration.GetApplicationURL(systemFQDN, moduleID)
			if url != "" {
				appURL = &url
			}
		}

		// Extract dynamic fields for inventory_data JSONB
		// Remove fixed fields and keep everything else
		inventoryDataJSON := make(map[string]interface{})
		fixedFields := map[string]bool{
			"id": true, "name": true, "version": true, "node": true, "label": true,
		}
		for key, value := range module {
			if !fixedFields[key] {
				inventoryDataJSON[key] = value
			}
		}

		inventoryDataBytes, err := json.Marshal(inventoryDataJSON)
		if err != nil {
			logger.Warn().Err(err).Str("module_id", moduleID).Msg("Failed to marshal inventory_data")
			inventoryDataBytes = []byte("{}")
		}

		// Generate application ID
		appID := fmt.Sprintf("%s-%s", record.SystemID, moduleID)

		// Upsert application
		query := `
			INSERT INTO applications (
				id, system_id, module_id, instance_of, display_name,
				node_id, node_label, version, url, inventory_data,
				is_user_facing, status, first_seen_at, last_inventory_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5,
				$6, $7, $8, $9, $10,
				$11, 'unassigned', NOW(), NOW(), NOW(), NOW()
			)
			ON CONFLICT (id) DO UPDATE SET
				instance_of = EXCLUDED.instance_of,
				display_name = COALESCE(EXCLUDED.display_name, applications.display_name),
				node_id = EXCLUDED.node_id,
				node_label = COALESCE(EXCLUDED.node_label, applications.node_label),
				version = EXCLUDED.version,
				url = COALESCE(EXCLUDED.url, applications.url),
				inventory_data = EXCLUDED.inventory_data,
				is_user_facing = EXCLUDED.is_user_facing,
				last_inventory_at = NOW(),
				updated_at = NOW(),
				deleted_at = NULL
		`

		_, err = tx.ExecContext(ctx, query,
			appID,                     // $1
			record.SystemID,           // $2
			moduleID,                  // $3
			moduleName,                // $4 (instance_of)
			nilIfEmpty(moduleLabel),   // $5 (display_name)
			nodeID,                    // $6
			nodeLabel,                 // $7
			nilIfEmpty(moduleVersion), // $8
			appURL,                    // $9
			inventoryDataBytes,        // $10
			isUserFacing,              // $11
		)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("app_id", appID).
				Str("module_id", moduleID).
				Msg("Failed to upsert application")
			continue
		}
	}

	// Soft-delete applications that are no longer in inventory
	if len(seenModuleIDs) > 0 {
		// Build list of module IDs we've seen
		moduleIDList := make([]string, 0, len(seenModuleIDs))
		for moduleID := range seenModuleIDs {
			moduleIDList = append(moduleIDList, moduleID)
		}

		// Create placeholders for the IN clause
		placeholders := make([]string, len(moduleIDList))
		args := make([]interface{}, len(moduleIDList)+1)
		args[0] = record.SystemID
		for i, moduleID := range moduleIDList {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args[i+1] = moduleID
		}

		// Soft-delete applications not in the current inventory
		softDeleteQuery := fmt.Sprintf(`
			UPDATE applications
			SET deleted_at = NOW(), updated_at = NOW()
			WHERE system_id = $1
			  AND module_id NOT IN (%s)
			  AND deleted_at IS NULL
		`, strings.Join(placeholders, ", "))

		result, err := tx.ExecContext(ctx, softDeleteQuery, args...)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("system_id", record.SystemID).
				Msg("Failed to soft-delete removed applications")
		} else if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
			logger.Info().
				Str("system_id", record.SystemID).
				Int64("deleted_count", rowsAffected).
				Msg("Soft-deleted applications no longer in inventory")
		}
	}

	logger.Debug().
		Str("system_id", record.SystemID).
		Int("modules_count", len(seenModuleIDs)).
		Msg("Applications extracted from inventory")

	return nil
}

// nilIfEmpty returns nil if the string is empty, otherwise returns a pointer to the string
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
