/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package middleware

import (
	"context"
	"strings"

	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/queue"
)

// systemAuthInvalidationChannel is the cross-service Redis pub/sub channel
// backend publishes on when a system is deleted, destroyed, or has its
// secret regenerated. The payload is the plain system_key string.
const systemAuthInvalidationChannel = "my:auth:invalidate"

// StartAuthInvalidator subscribes to the cross-service auth invalidation
// channel and purges every cached credential entry for a system as soon
// as backend signals the change. Without this hook the cache naturally
// expires within SystemAuthCacheTTL (default 10 minutes); the bus makes
// the propagation sub-second.
//
// The subscriber runs in its own goroutine and stops when ctx is done.
// A Redis outage logs a warning and disables the bus — natural TTL
// expiration still catches up with the change eventually.
func StartAuthInvalidator(ctx context.Context) {
	rdb := queue.GetClient()
	if rdb == nil {
		logger.Warn().Msg("Redis unavailable, auth invalidation bus disabled")
		return
	}

	pubsub := rdb.Subscribe(ctx, systemAuthInvalidationChannel)
	go func() {
		defer func() {
			if err := pubsub.Close(); err != nil {
				logger.Warn().Err(err).Msg("failed to close auth invalidation subscription")
			}
		}()
		ch := pubsub.Channel()
		logger.Info().Str("channel", systemAuthInvalidationChannel).Msg("auth invalidation bus started")
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				systemKey := strings.TrimSpace(msg.Payload)
				if systemKey == "" {
					continue
				}
				purgeSystemAuthCache(ctx, rdb, systemKey)
			}
		}
	}()
}

// purgeSystemAuthCache drops every cached credential entry for the given
// system_key from both the in-process sync.Map (see auth.go) and from
// Redis. In-process entries are keyed as `<systemKey>:<secretHash>`;
// Redis entries use `auth:system:<systemKey>:<secretHash>`.
func purgeSystemAuthCache(ctx context.Context, rdb *redis.Client, systemKey string) {
	// In-process: iterate and delete every entry whose key starts with
	// the system_key prefix. sync.Map.Range is safe against concurrent
	// modification.
	prefix := systemKey + ":"
	inProcessAuthCache.Range(func(key, _ any) bool {
		if k, ok := key.(string); ok && strings.HasPrefix(k, prefix) {
			inProcessAuthCache.Delete(k)
		}
		return true
	})

	// Redis: SCAN through every auth cache entry for this system_key and
	// delete it. Use a modest batch size so the scan does not block
	// other commands in a busy Redis.
	pattern := "auth:system:" + systemKey + ":*"
	iter := rdb.Scan(ctx, 0, pattern, 100).Iterator()
	count := 0
	for iter.Next(ctx) {
		if err := rdb.Del(ctx, iter.Val()).Err(); err != nil {
			logger.Warn().Err(err).Str("key", iter.Val()).Msg("auth invalidation: delete failed")
			continue
		}
		count++
	}
	if err := iter.Err(); err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("auth invalidation: scan failed")
	}

	logger.Info().
		Str("system_key", systemKey).
		Int("redis_keys_deleted", count).
		Msg("auth cache invalidated")
}
