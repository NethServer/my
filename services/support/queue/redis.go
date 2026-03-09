/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/logger"
)

var client *redis.Client

// Init initializes the Redis client
func Init() error {
	opt, err := redis.ParseURL(configuration.Config.RedisURL)
	if err != nil {
		return fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	opt.DB = configuration.Config.RedisDB
	opt.Password = configuration.Config.RedisPassword
	opt.PoolSize = 20
	opt.MinIdleConns = 5
	opt.ConnMaxIdleTime = 5 * time.Minute
	opt.ConnMaxLifetime = 30 * time.Minute

	client = redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info().
		Str("redis_url", logger.SanitizeConnectionURL(configuration.Config.RedisURL)).
		Int("redis_db", opt.DB).
		Msg("Redis client initialized")

	return nil
}

// GetClient returns the Redis client instance
func GetClient() *redis.Client {
	return client
}

// Subscribe subscribes to a Redis pub/sub channel
func Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return client.Subscribe(ctx, channel)
}

// Close closes the Redis connection
func Close() error {
	if client != nil {
		return client.Close()
	}
	return nil
}
