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
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
	"github.com/rs/zerolog"
)

// NotificationWorker handles sending notifications for inventory changes
type NotificationWorker struct {
	id            int
	workerCount   int
	queueManager  *queue.QueueManager
	isHealthy     int32
	processedJobs int64
	failedJobs    int64
	lastActivity  time.Time
	mu            sync.RWMutex
}

// NewNotificationWorker creates a new notification worker
func NewNotificationWorker(id, workerCount int) *NotificationWorker {
	return &NotificationWorker{
		id:           id,
		workerCount:  workerCount,
		queueManager: queue.NewQueueManager(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the notification worker workers
func (nw *NotificationWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Start multiple worker goroutines
	for i := 0; i < nw.workerCount; i++ {
		wg.Add(1)
		go nw.worker(ctx, wg, i+1)
	}

	// Start health monitor
	wg.Add(1)
	go nw.healthMonitor(ctx, wg)

	return nil
}

// Name returns the worker name
func (nw *NotificationWorker) Name() string {
	return fmt.Sprintf("notification-worker-%d", nw.id)
}

// IsHealthy returns the health status
func (nw *NotificationWorker) IsHealthy() bool {
	return atomic.LoadInt32(&nw.isHealthy) == 1
}

// worker processes notification messages from the queue
func (nw *NotificationWorker) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("notification-worker").
		With().
		Int("worker_id", workerID).
		Logger()

	workerLogger.Info().Msg("Notification worker started")

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Notification worker stopping")
			return
		default:
			// Process messages from queue
			if err := nw.processNextMessage(ctx, &workerLogger); err != nil {
				workerLogger.Error().Err(err).Msg("Error processing notification message")
				atomic.AddInt64(&nw.failedJobs, 1)

				// Brief pause on error to prevent tight error loops
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNextMessage processes the next message from the notification queue
func (nw *NotificationWorker) processNextMessage(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Update activity timestamp
	nw.mu.Lock()
	nw.lastActivity = time.Now()
	nw.mu.Unlock()

	// Get message from queue with timeout
	message, err := nw.queueManager.DequeueMessage(ctx, configuration.Config.QueueNotificationName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue message: %w", err)
	}

	if message == nil {
		// No message available, this is normal
		return nil
	}

	// Parse notification job
	var job models.NotificationJob
	if err := json.Unmarshal(message.Data, &job); err != nil {
		workerLogger.Error().
			Err(err).
			Str("message_id", message.ID).
			Msg("Failed to unmarshal notification job")
		return fmt.Errorf("failed to unmarshal notification job: %w", err)
	}

	// Process the notification
	if err := nw.processNotification(ctx, &job, workerLogger); err != nil {
		// Requeue the message for retry
		if requeueErr := nw.queueManager.RequeueMessage(ctx, configuration.Config.QueueNotificationName, message, err); requeueErr != nil {
			workerLogger.Error().
				Err(requeueErr).
				Str("message_id", message.ID).
				Msg("Failed to requeue message")
		}
		return fmt.Errorf("failed to process notification: %w", err)
	}

	atomic.AddInt64(&nw.processedJobs, 1)
	workerLogger.Debug().
		Str("system_id", job.SystemID).
		Str("message_id", message.ID).
		Str("notification_type", job.Type).
		Msg("Notification processed successfully")

	return nil
}

// processNotification processes a single notification job
func (nw *NotificationWorker) processNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	switch job.Type {
	case "diff":
		return nw.processDiffNotification(ctx, job, workerLogger)
	case "alert":
		return nw.processAlertNotification(ctx, job, workerLogger)
	case "system_status":
		return nw.processSystemStatusNotification(ctx, job, workerLogger)
	default:
		return fmt.Errorf("unknown notification type: %s", job.Type)
	}
}

// processDiffNotification processes notifications for inventory differences
func (nw *NotificationWorker) processDiffNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	if len(job.Diffs) == 0 {
		return fmt.Errorf("no diffs provided for diff notification")
	}

	// Group diffs by category for better notification organization
	categoryGroups := make(map[string][]models.InventoryDiff)
	for _, diff := range job.Diffs {
		categoryGroups[diff.Category] = append(categoryGroups[diff.Category], diff)
	}

	// Format notification message
	message := nw.formatDiffNotificationMessage(job.SystemID, categoryGroups, job.Severity)

	// Log the notification (for now, we'll just log instead of actually sending)
	workerLogger.Info().
		Str("system_id", job.SystemID).
		Str("severity", job.Severity).
		Int("total_diffs", len(job.Diffs)).
		Int("categories", len(categoryGroups)).
		Str("message", message).
		Msg("Inventory change notification")

	// Mark diffs as notification sent
	if err := nw.markDiffsNotificationSent(ctx, job.Diffs); err != nil {
		return fmt.Errorf("failed to mark diffs as notified: %w", err)
	}

	// This could be extended to support different notification channels
	return nw.sendNotification(ctx, job.SystemID, message, job.Severity, workerLogger)
}

// processAlertNotification processes alert notifications
func (nw *NotificationWorker) processAlertNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	if job.Alert == nil {
		return fmt.Errorf("no alert provided for alert notification")
	}

	message := fmt.Sprintf("Alert for system %s: %s (Severity: %s)",
		job.SystemID, job.Alert.Message, job.Alert.Severity)

	workerLogger.Info().
		Str("system_id", job.SystemID).
		Str("alert_type", job.Alert.AlertType).
		Str("severity", job.Alert.Severity).
		Str("message", message).
		Msg("Alert notification")

	return nw.sendNotification(ctx, job.SystemID, message, job.Alert.Severity, workerLogger)
}

// processSystemStatusNotification processes system status change notifications
func (nw *NotificationWorker) processSystemStatusNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	message := fmt.Sprintf("System status change for %s: %s", job.SystemID, job.Message)

	workerLogger.Info().
		Str("system_id", job.SystemID).
		Str("severity", job.Severity).
		Str("message", message).
		Msg("System status notification")

	return nw.sendNotification(ctx, job.SystemID, message, job.Severity, workerLogger)
}

// formatDiffNotificationMessage formats a human-readable notification message for diffs
func (nw *NotificationWorker) formatDiffNotificationMessage(systemID string, categoryGroups map[string][]models.InventoryDiff, severity string) string {
	message := fmt.Sprintf("System %s has %d inventory changes (Severity: %s):\n",
		systemID, nw.countTotalDiffs(categoryGroups), severity)

	for category, diffs := range categoryGroups {
		message += fmt.Sprintf("\n%s (%d changes):\n", category, len(diffs))

		for _, diff := range diffs {
			var changeDesc string
			switch diff.DiffType {
			case "create":
				changeDesc = fmt.Sprintf("  + Added %s", diff.FieldPath)
				if diff.CurrentValueRaw != nil {
					changeDesc += fmt.Sprintf(": %s", *diff.CurrentValueRaw)
				}
			case "update":
				changeDesc = fmt.Sprintf("  ~ Changed %s", diff.FieldPath)
				if diff.PreviousValueRaw != nil && diff.CurrentValueRaw != nil {
					changeDesc += fmt.Sprintf(": %s → %s", *diff.PreviousValueRaw, *diff.CurrentValueRaw)
				}
			case "delete":
				changeDesc = fmt.Sprintf("  - Removed %s", diff.FieldPath)
				if diff.PreviousValueRaw != nil {
					changeDesc += fmt.Sprintf(": %s", *diff.PreviousValueRaw)
				}
			default:
				changeDesc = fmt.Sprintf("  ? %s %s", diff.DiffType, diff.FieldPath)
			}

			message += changeDesc + "\n"
		}
	}

	return message
}

// countTotalDiffs counts total diffs across all categories
func (nw *NotificationWorker) countTotalDiffs(categoryGroups map[string][]models.InventoryDiff) int {
	total := 0
	for _, diffs := range categoryGroups {
		total += len(diffs)
	}
	return total
}

// markDiffsNotificationSent marks diffs as having notifications sent
func (nw *NotificationWorker) markDiffsNotificationSent(ctx context.Context, diffs []models.InventoryDiff) error {
	if len(diffs) == 0 {
		return nil
	}

	// Add timeout to prevent connection hanging
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Process in batches to avoid connection issues with large lists
	const batchSize = 50
	for i := 0; i < len(diffs); i += batchSize {
		end := i + batchSize
		if end > len(diffs) {
			end = len(diffs)
		}

		batch := diffs[i:end]
		if err := nw.markDiffBatch(ctx, batch); err != nil {
			return fmt.Errorf("failed to mark diff batch %d-%d: %w", i, end, err)
		}
	}

	return nil
}

// markDiffBatch marks a batch of diffs as notified using prepared statement
func (nw *NotificationWorker) markDiffBatch(ctx context.Context, diffs []models.InventoryDiff) error {
	if len(diffs) == 0 {
		return nil
	}

	// Use prepared statement for better performance and safety
	stmt, err := database.DB.PrepareContext(ctx, "UPDATE inventory_diffs SET notification_sent = true WHERE id = $1")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		_ = stmt.Close() // Ignore error on close
	}()

	for _, diff := range diffs {
		if _, err := stmt.ExecContext(ctx, diff.ID); err != nil {
			return fmt.Errorf("failed to update diff %d: %w", diff.ID, err)
		}
	}

	return nil
}

// sendNotification sends the actual notification (placeholder implementation)
func (nw *NotificationWorker) sendNotification(ctx context.Context, systemID, message, severity string, workerLogger *zerolog.Logger) error {
	// This could include:
	// - Email notifications
	// - Webhook calls
	// - Push notifications
	// - Integration with external systems (Slack, Teams, etc.)

	workerLogger.Info().
		Str("system_id", systemID).
		Str("severity", severity).
		Str("notification_method", "log").
		Str("notification_message", message).
		Msg("Notification sent")

	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	return nil
}

// healthMonitor monitors the health of the notification worker
func (nw *NotificationWorker) healthMonitor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(configuration.Config.WorkerHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nw.checkHealth()
		}
	}
}

// checkHealth checks the health of the worker
func (nw *NotificationWorker) checkHealth() {
	nw.mu.RLock()
	lastActivity := nw.lastActivity
	nw.mu.RUnlock()

	// Consider unhealthy if no activity for too long
	if time.Since(lastActivity) > 5*configuration.Config.WorkerHeartbeatInterval {
		atomic.StoreInt32(&nw.isHealthy, 0)
		logger.Warn().
			Str("worker", nw.Name()).
			Time("last_activity", lastActivity).
			Msg("Worker marked as unhealthy due to inactivity")
	} else {
		atomic.StoreInt32(&nw.isHealthy, 1)
	}
}

// GetStats returns worker statistics
func (nw *NotificationWorker) GetStats() map[string]interface{} {
	nw.mu.RLock()
	defer nw.mu.RUnlock()

	return map[string]interface{}{
		"worker_count":   nw.workerCount,
		"processed_jobs": atomic.LoadInt64(&nw.processedJobs),
		"failed_jobs":    atomic.LoadInt64(&nw.failedJobs),
		"last_activity":  nw.lastActivity,
		"is_healthy":     nw.IsHealthy(),
	}
}
