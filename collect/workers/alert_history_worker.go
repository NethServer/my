/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 */

package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/methods"
	"github.com/nethesis/my/collect/models"
	"github.com/nethesis/my/collect/queue"
)

// alertHistoryStoreTimeout bounds each retry attempt's DB work. Wider than
// the webhook handler's 8s budget: here there is no Alertmanager client
// timeout to stay under, and the queue exists precisely for the moments the
// database is slow.
const alertHistoryStoreTimeout = 30 * time.Second

// AlertHistoryWorker drains the alert-history retry queue: webhook payloads
// whose synchronous DB write failed (see methods.ReceiveAlertHistory) are
// retried here until Postgres recovers, instead of depending on Alertmanager
// retries that die with its process.
type AlertHistoryWorker struct {
	BaseWorker
	queueManager *queue.QueueManager
}

// NewAlertHistoryWorker creates a new alert-history retry worker
func NewAlertHistoryWorker(id, workerCount int, queueManager *queue.QueueManager) *AlertHistoryWorker {
	return &AlertHistoryWorker{
		BaseWorker:   NewBaseWorker(id, fmt.Sprintf("alert-history-worker-%d", id), workerCount),
		queueManager: queueManager,
	}
}

// Start starts the alert-history worker goroutines
func (ahw *AlertHistoryWorker) Start(ctx context.Context, wg *sync.WaitGroup) error {
	for i := 0; i < ahw.workerCount; i++ {
		wg.Add(1)
		go ahw.worker(ctx, wg, i+1)
	}

	wg.Add(1)
	go ahw.HealthMonitor(ctx, wg)

	return nil
}

// worker processes alert-history retry messages from the queue
func (ahw *AlertHistoryWorker) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	workerLogger := logger.ComponentLogger("alert-history-worker").
		With().
		Int("worker_id", workerID).
		Logger()

	workerLogger.Info().Msg("Alert history worker started")

	for {
		select {
		case <-ctx.Done():
			workerLogger.Info().Msg("Alert history worker stopping")
			return
		default:
			if err := ahw.processNextMessage(ctx, &workerLogger); err != nil {
				workerLogger.Error().Err(err).Msg("Error processing alert history message")
				ahw.RecordFailure()

				// Brief pause on error to prevent tight error loops
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// GetStats returns alert-history worker statistics
func (ahw *AlertHistoryWorker) GetStats() map[string]interface{} {
	return ahw.GetBaseStats()
}

// processNextMessage retries the next parked webhook payload
func (ahw *AlertHistoryWorker) processNextMessage(ctx context.Context, workerLogger *zerolog.Logger) error {
	ahw.UpdateActivity()

	message, err := ahw.queueManager.DequeueMessage(ctx, configuration.Config.QueueAlertHistoryName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to dequeue message: %w", err)
	}

	if message == nil {
		// No message available, this is normal
		return nil
	}

	var payload models.AlertmanagerWebhookPayload
	if err := json.Unmarshal(message.Data, &payload); err != nil {
		// Malformed payloads can never succeed: log and drop instead of
		// requeueing them forever.
		workerLogger.Error().
			Err(err).
			Str("message_id", message.ID).
			Msg("Failed to unmarshal alert history payload; dropping message")
		return nil
	}

	storeCtx, cancel := context.WithTimeout(ctx, alertHistoryStoreTimeout)
	saved, err := methods.StoreAlertHistory(storeCtx, &payload)
	cancel()
	if err != nil {
		// Requeue with backoff; RequeueMessage moves it to the dead-letter
		// queue once MaxAttempts is exhausted.
		if requeueErr := ahw.queueManager.RequeueMessage(ctx, configuration.Config.QueueAlertHistoryName, message, err); requeueErr != nil {
			workerLogger.Error().
				Err(requeueErr).
				Str("message_id", message.ID).
				Msg("Failed to requeue alert history message")
		}
		return fmt.Errorf("failed to store alert history: %w", err)
	}

	ahw.RecordSuccess()
	workerLogger.Info().
		Str("message_id", message.ID).
		Int("attempts", message.Attempts).
		Int("saved", saved).
		Msg("Alert history payload recovered from retry queue")

	return nil
}
