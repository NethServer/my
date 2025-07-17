/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/models"
)

var (
	client *redis.Client
)

// Init initializes the Redis queue client
func Init() error {
	opt, err := redis.ParseURL(configuration.Config.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	// Override with configuration values
	opt.DB = configuration.Config.RedisDB
	opt.Password = configuration.Config.RedisPassword
	opt.MaxRetries = configuration.Config.RedisMaxRetries
	opt.DialTimeout = configuration.Config.RedisDialTimeout
	opt.ReadTimeout = configuration.Config.RedisReadTimeout
	opt.WriteTimeout = configuration.Config.RedisWriteTimeout

	client = redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().
		Str("redis_url", logger.SanitizeConnectionURL(configuration.Config.RedisURL)).
		Int("redis_db", opt.DB).
		Msg("Redis queue client initialized successfully")

	return nil
}

// GetClient returns the Redis client instance
func GetClient() *redis.Client {
	return client
}

// QueueManager manages different types of queues
type QueueManager struct {
	client *redis.Client
}

// NewQueueManager creates a new queue manager
func NewQueueManager() *QueueManager {
	return &QueueManager{
		client: client,
	}
}

// EnqueueInventory adds an inventory processing job to the queue
func (qm *QueueManager) EnqueueInventory(ctx context.Context, inventoryData *models.InventoryData) error {
	message := &models.QueueMessage{
		ID:          uuid.New().String(),
		Type:        "inventory",
		SystemID:    inventoryData.SystemID,
		Attempts:    0,
		MaxAttempts: configuration.Config.QueueRetryAttempts,
		CreatedAt:   time.Now(),
	}

	// Serialize inventory data
	data, err := json.Marshal(inventoryData)
	if err != nil {
		return fmt.Errorf("failed to marshal inventory data: %w", err)
	}
	message.Data = data

	return qm.enqueueMessage(ctx, configuration.Config.QueueInventoryName, message)
}

// EnqueueProcessing adds a processing job to the queue
func (qm *QueueManager) EnqueueProcessing(ctx context.Context, job *models.InventoryProcessingJob) error {
	message := &models.QueueMessage{
		ID:          uuid.New().String(),
		Type:        "processing",
		SystemID:    job.SystemID,
		Attempts:    0,
		MaxAttempts: configuration.Config.QueueRetryAttempts,
		CreatedAt:   time.Now(),
	}

	// Serialize job data
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal processing job: %w", err)
	}
	message.Data = data

	return qm.enqueueMessage(ctx, configuration.Config.QueueProcessingName, message)
}

// EnqueueNotification adds a notification job to the queue
func (qm *QueueManager) EnqueueNotification(ctx context.Context, job *models.NotificationJob) error {
	message := &models.QueueMessage{
		ID:          uuid.New().String(),
		Type:        "notification",
		SystemID:    job.SystemID,
		Attempts:    0,
		MaxAttempts: configuration.Config.NotificationRetryAttempts,
		CreatedAt:   time.Now(),
	}

	// Serialize job data
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal notification job: %w", err)
	}
	message.Data = data

	return qm.enqueueMessage(ctx, configuration.Config.QueueNotificationName, message)
}

// enqueueMessage adds a message to the specified queue
func (qm *QueueManager) enqueueMessage(ctx context.Context, queueName string, message *models.QueueMessage) error {
	// Serialize message
	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal queue message: %w", err)
	}

	// Add to Redis list (FIFO queue)
	err = qm.client.LPush(ctx, queueName, messageData).Err()
	if err != nil {
		return fmt.Errorf("failed to enqueue message: %w", err)
	}

	logger.Debug().
		Str("queue", queueName).
		Str("message_id", message.ID).
		Str("message_type", message.Type).
		Str("system_id", message.SystemID).
		Msg("Message enqueued successfully")

	return nil
}

// DequeueMessage retrieves and removes a message from the specified queue
func (qm *QueueManager) DequeueMessage(ctx context.Context, queueName string, timeout time.Duration) (*models.QueueMessage, error) {
	// Use BRPOP for blocking pop with timeout
	result, err := qm.client.BRPop(ctx, timeout, queueName).Result()
	if err == redis.Nil {
		return nil, nil // No message available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue message: %w", err)
	}

	if len(result) != 2 {
		return nil, fmt.Errorf("unexpected Redis response format")
	}

	messageData := result[1]
	var message models.QueueMessage
	if err := json.Unmarshal([]byte(messageData), &message); err != nil {
		return nil, fmt.Errorf("failed to unmarshal queue message: %w", err)
	}

	logger.Debug().
		Str("queue", queueName).
		Str("message_id", message.ID).
		Str("message_type", message.Type).
		Str("system_id", message.SystemID).
		Msg("Message dequeued successfully")

	return &message, nil
}

// RequeueMessage re-adds a failed message to the queue with incremented attempt count
func (qm *QueueManager) RequeueMessage(ctx context.Context, queueName string, message *models.QueueMessage, err error) error {
	message.Attempts++
	if message.Error != nil {
		*message.Error = err.Error()
	} else {
		errStr := err.Error()
		message.Error = &errStr
	}

	if message.Attempts >= message.MaxAttempts {
		// Move to dead letter queue
		deadLetterQueue := queueName + ":dead"
		return qm.enqueueMessage(ctx, deadLetterQueue, message)
	}

	// Calculate exponential backoff delay
	delay := time.Duration(message.Attempts) * configuration.Config.QueueRetryDelay
	if delay > 5*time.Minute {
		delay = 5 * time.Minute // Cap at 5 minutes
	}

	// Use Redis sorted set for delayed processing
	delayedQueue := queueName + ":delayed"
	score := float64(time.Now().Add(delay).Unix())

	messageData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message for requeue: %w", err)
	}

	err = qm.client.ZAdd(ctx, delayedQueue, redis.Z{
		Score:  score,
		Member: messageData,
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to requeue message: %w", err)
	}

	logger.Warn().
		Str("queue", queueName).
		Str("message_id", message.ID).
		Int("attempts", message.Attempts).
		Int("max_attempts", message.MaxAttempts).
		Dur("delay", delay).
		Err(err).
		Msg("Message requeued with delay")

	return nil
}

// ProcessDelayedMessages moves delayed messages back to main queue when ready
func (qm *QueueManager) ProcessDelayedMessages(ctx context.Context, queueName string) error {
	delayedQueue := queueName + ":delayed"
	now := float64(time.Now().Unix())

	// Get messages ready to be processed
	result, err := qm.client.ZRangeByScoreWithScores(ctx, delayedQueue, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%f", now),
		Count: int64(configuration.Config.QueueBatchSize),
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to get delayed messages: %w", err)
	}

	for _, item := range result {
		messageData := item.Member.(string)

		// Move message back to main queue
		pipe := qm.client.TxPipeline()
		pipe.ZRem(ctx, delayedQueue, messageData)
		pipe.LPush(ctx, queueName, messageData)

		if _, err := pipe.Exec(ctx); err != nil {
			logger.Error().
				Err(err).
				Str("queue", queueName).
				Msg("Failed to move delayed message to main queue")
			continue
		}

		logger.Debug().
			Str("queue", queueName).
			Msg("Delayed message moved to main queue")
	}

	return nil
}

// GetQueueStats returns statistics about queue lengths and health
func (qm *QueueManager) GetQueueStats(ctx context.Context) (*models.InventoryStats, error) {
	stats := &models.InventoryStats{}

	// Get queue lengths
	inventoryLen, err := qm.client.LLen(ctx, configuration.Config.QueueInventoryName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory queue length: %w", err)
	}

	processingLen, err := qm.client.LLen(ctx, configuration.Config.QueueProcessingName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get processing queue length: %w", err)
	}

	notificationLen, err := qm.client.LLen(ctx, configuration.Config.QueueNotificationName).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification queue length: %w", err)
	}

	// Get delayed queue lengths
	delayedInventory, err := qm.client.ZCard(ctx, configuration.Config.QueueInventoryName+":delayed").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get delayed inventory queue length: %w", err)
	}

	delayedProcessing, err := qm.client.ZCard(ctx, configuration.Config.QueueProcessingName+":delayed").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get delayed processing queue length: %w", err)
	}

	delayedNotification, err := qm.client.ZCard(ctx, configuration.Config.QueueNotificationName+":delayed").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get delayed notification queue length: %w", err)
	}

	// Get dead letter queue lengths
	deadInventory, err := qm.client.LLen(ctx, configuration.Config.QueueInventoryName+":dead").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get dead inventory queue length: %w", err)
	}

	deadProcessing, err := qm.client.LLen(ctx, configuration.Config.QueueProcessingName+":dead").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get dead processing queue length: %w", err)
	}

	deadNotification, err := qm.client.LLen(ctx, configuration.Config.QueueNotificationName+":dead").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get dead notification queue length: %w", err)
	}

	stats.PendingJobs = inventoryLen + processingLen + notificationLen + delayedInventory + delayedProcessing + delayedNotification
	stats.ProcessingJobs = 0 // This would be tracked by workers
	stats.FailedJobs = deadInventory + deadProcessing + deadNotification

	// Determine queue health
	if stats.FailedJobs > 100 || stats.PendingJobs > 10000 {
		stats.QueueHealth = "critical"
	} else if stats.FailedJobs > 10 || stats.PendingJobs > 1000 {
		stats.QueueHealth = "warning"
	} else {
		stats.QueueHealth = "healthy"
	}

	return stats, nil
}

// Close closes the Redis connection
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
