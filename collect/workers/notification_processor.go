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

	"github.com/rs/zerolog"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/database"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
)

// NotificationProcessor handles sending notifications for inventory changes
type NotificationProcessor struct {
	id            int
	workerCount   int
	queueManager  *queue.QueueManager
	isHealthy     int32
	processedJobs int64
	failedJobs    int64
	lastActivity  time.Time
	mu            sync.RWMutex
}

// NewNotificationProcessor creates a new notification processor
func NewNotificationProcessor(id, workerCount int) *NotificationProcessor {
	return &NotificationProcessor{
		id:           id,
		workerCount:  workerCount,
		queueManager: queue.NewQueueManager(),
		isHealthy:    1,
		lastActivity: time.Now(),
	}
}

// Start starts the notification processor workers
func (np *NotificationProcessor) Start(ctx context.Context, wg *sync.WaitGroup) error {
	// Start multiple worker goroutines
	for i := 0; i < np.workerCount; i++ {
		wg.Add(1)
		go np.worker(ctx, wg, i+1)
	}

	// Start health monitor
	wg.Add(1)
	go np.healthMonitor(ctx, wg)

	return nil
}

// Name returns the worker name
func (np *NotificationProcessor) Name() string {
	return fmt.Sprintf("notification-processor-%d", np.id)
}

// IsHealthy returns the health status
func (np *NotificationProcessor) IsHealthy() bool {
	return atomic.LoadInt32(&np.isHealthy) == 1
}

// worker processes notification messages from the queue
func (np *NotificationProcessor) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("notification-processor").
		With().
		Int("worker_id", workerID).
		Logger()

	workerLogger.Info().Msg("Notification processor worker started")

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Notification processor worker stopping")
			return
		default:
			// Process messages from queue
			if err := np.processNextMessage(ctx, &workerLogger); err != nil {
				workerLogger.Error().Err(err).Msg("Error processing notification message")
				atomic.AddInt64(&np.failedJobs, 1)
				
				// Brief pause on error to prevent tight error loops
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processNextMessage processes the next message from the notification queue
func (np *NotificationProcessor) processNextMessage(ctx context.Context, workerLogger *zerolog.Logger) error {
	// Get message from queue with timeout
	message, err := np.queueManager.DequeueMessage(ctx, configuration.Config.QueueNotificationName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue message: %w", err)
	}

	if message == nil {
		// No message available, this is normal
		return nil
	}

	// Update activity timestamp
	np.mu.Lock()
	np.lastActivity = time.Now()
	np.mu.Unlock()

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
	if err := np.processNotification(ctx, &job, workerLogger); err != nil {
		// Requeue the message for retry
		if requeueErr := np.queueManager.RequeueMessage(ctx, configuration.Config.QueueNotificationName, message, err); requeueErr != nil {
			workerLogger.Error().
				Err(requeueErr).
				Str("message_id", message.ID).
				Msg("Failed to requeue message")
		}
		return fmt.Errorf("failed to process notification: %w", err)
	}

	atomic.AddInt64(&np.processedJobs, 1)
	workerLogger.Debug().
		Str("system_id", job.SystemID).
		Str("message_id", message.ID).
		Str("notification_type", job.Type).
		Msg("Notification processed successfully")

	return nil
}

// processNotification processes a single notification job
func (np *NotificationProcessor) processNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	switch job.Type {
	case "diff":
		return np.processDiffNotification(ctx, job, workerLogger)
	case "alert":
		return np.processAlertNotification(ctx, job, workerLogger)
	case "system_status":
		return np.processSystemStatusNotification(ctx, job, workerLogger)
	default:
		return fmt.Errorf("unknown notification type: %s", job.Type)
	}
}

// processDiffNotification processes notifications for inventory differences
func (np *NotificationProcessor) processDiffNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	if len(job.Diffs) == 0 {
		return fmt.Errorf("no diffs provided for diff notification")
	}

	// Group diffs by category for better notification organization
	categoryGroups := make(map[string][]models.InventoryDiff)
	for _, diff := range job.Diffs {
		categoryGroups[diff.Category] = append(categoryGroups[diff.Category], diff)
	}

	// Format notification message
	message := np.formatDiffNotificationMessage(job.SystemID, categoryGroups, job.Severity)

	// Log the notification (for now, we'll just log instead of actually sending)
	workerLogger.Info().
		Str("system_id", job.SystemID).
		Str("severity", job.Severity).
		Int("total_diffs", len(job.Diffs)).
		Int("categories", len(categoryGroups)).
		Str("message", message).
		Msg("Inventory change notification")

	// Mark diffs as notification sent
	if err := np.markDiffsNotificationSent(ctx, job.Diffs); err != nil {
		return fmt.Errorf("failed to mark diffs as notified: %w", err)
	}

	// TODO: Implement actual notification sending (email, webhook, etc.)
	// This could be extended to support different notification channels
	return np.sendNotification(ctx, job.SystemID, message, job.Severity, workerLogger)
}

// processAlertNotification processes alert notifications
func (np *NotificationProcessor) processAlertNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
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

	return np.sendNotification(ctx, job.SystemID, message, job.Alert.Severity, workerLogger)
}

// processSystemStatusNotification processes system status change notifications
func (np *NotificationProcessor) processSystemStatusNotification(ctx context.Context, job *models.NotificationJob, workerLogger *zerolog.Logger) error {
	message := fmt.Sprintf("System status change for %s: %s", job.SystemID, job.Message)

	workerLogger.Info().
		Str("system_id", job.SystemID).
		Str("severity", job.Severity).
		Str("message", message).
		Msg("System status notification")

	return np.sendNotification(ctx, job.SystemID, message, job.Severity, workerLogger)
}

// formatDiffNotificationMessage formats a human-readable notification message for diffs
func (np *NotificationProcessor) formatDiffNotificationMessage(systemID string, categoryGroups map[string][]models.InventoryDiff, severity string) string {
	message := fmt.Sprintf("System %s has %d inventory changes (Severity: %s):\n", 
		systemID, np.countTotalDiffs(categoryGroups), severity)

	for category, diffs := range categoryGroups {
		message += fmt.Sprintf("\n%s (%d changes):\n", category, len(diffs))
		
		for _, diff := range diffs {
			var changeDesc string
			switch diff.DiffType {
			case "create":
				changeDesc = fmt.Sprintf("  + Added %s", diff.FieldPath)
				if diff.CurrentValue != nil {
					changeDesc += fmt.Sprintf(": %s", *diff.CurrentValue)
				}
			case "update":
				changeDesc = fmt.Sprintf("  ~ Changed %s", diff.FieldPath)
				if diff.PreviousValue != nil && diff.CurrentValue != nil {
					changeDesc += fmt.Sprintf(": %s → %s", *diff.PreviousValue, *diff.CurrentValue)
				}
			case "delete":
				changeDesc = fmt.Sprintf("  - Removed %s", diff.FieldPath)
				if diff.PreviousValue != nil {
					changeDesc += fmt.Sprintf(": %s", *diff.PreviousValue)
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
func (np *NotificationProcessor) countTotalDiffs(categoryGroups map[string][]models.InventoryDiff) int {
	total := 0
	for _, diffs := range categoryGroups {
		total += len(diffs)
	}
	return total
}

// markDiffsNotificationSent marks diffs as having notifications sent
func (np *NotificationProcessor) markDiffsNotificationSent(ctx context.Context, diffs []models.InventoryDiff) error {
	if len(diffs) == 0 {
		return nil
	}

	// Build list of diff IDs
	diffIDs := make([]interface{}, len(diffs))
	for i, diff := range diffs {
		diffIDs[i] = diff.ID
	}

	// Create placeholders for the IN clause
	placeholders := ""
	for i := range diffIDs {
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		UPDATE inventory_diffs 
		SET notification_sent = true 
		WHERE id IN (%s)
	`, placeholders)

	_, err := database.DB.ExecContext(ctx, query, diffIDs...)
	return err
}

// sendNotification sends the actual notification (placeholder implementation)
func (np *NotificationProcessor) sendNotification(ctx context.Context, systemID, message, severity string, workerLogger *zerolog.Logger) error {
	// TODO: Implement actual notification sending logic
	// This could include:
	// - Email notifications
	// - Webhook calls
	// - Push notifications
	// - Integration with external systems (Slack, Teams, etc.)
	
	// For now, we'll just log the notification
	workerLogger.Info().
		Str("system_id", systemID).
		Str("severity", severity).
		Str("notification_method", "log").
		Msg("Notification sent")

	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	return nil
}

// healthMonitor monitors the health of the notification processor
func (np *NotificationProcessor) healthMonitor(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	ticker := time.NewTicker(configuration.Config.WorkerHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			np.checkHealth()
		}
	}
}

// checkHealth checks the health of the processor
func (np *NotificationProcessor) checkHealth() {
	np.mu.RLock()
	lastActivity := np.lastActivity
	np.mu.RUnlock()

	// Consider unhealthy if no activity for too long
	if time.Since(lastActivity) > 5*configuration.Config.WorkerHeartbeatInterval {
		atomic.StoreInt32(&np.isHealthy, 0)
		logger.Warn().
			Str("worker", np.Name()).
			Time("last_activity", lastActivity).
			Msg("Worker marked as unhealthy due to inactivity")
	} else {
		atomic.StoreInt32(&np.isHealthy, 1)
	}
}

// GetStats returns processor statistics
func (np *NotificationProcessor) GetStats() map[string]interface{} {
	np.mu.RLock()
	defer np.mu.RUnlock()

	return map[string]interface{}{
		"worker_count":    np.workerCount,
		"processed_jobs":  atomic.LoadInt64(&np.processedJobs),
		"failed_jobs":     atomic.LoadInt64(&np.failedJobs),
		"last_activity":   np.lastActivity,
		"is_healthy":      np.IsHealthy(),
	}
}