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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nethesis/my/collect/configuration"
	"github.com/nethesis/my/collect/logger"
	"github.com/nethesis/my/collect/queue"
)

// systemKeyFormat pins the shape of a legitimate system_key so a malicious or
// misconfigured publisher cannot smuggle Redis SCAN globs (`*`, `?`, `[…]`) or
// other metacharacters into the invalidation payload. A `*` payload would
// otherwise expand to a SCAN pattern matching every cached credential and
// produce a thundering-herd auth lookup against Postgres on the next request
// from each appliance.
var systemKeyFormat = regexp.MustCompile(`^NETH(-[A-F0-9]{4}){9}$`)

// systemAuthInvalidationChannelBase is the channel namespace; APP_ENV is
// appended at runtime so a Redis instance shared across deployments stops
// cross-pollinating invalidations.
const systemAuthInvalidationChannelBase = "my:auth:invalidate"

// lastInvalidatedAt tracks the wall-clock moment when each system_key was last
// invalidated. checkInProcessCache (auth.go) consults this map to drop entries
// inserted before the invalidation arrived: sync.Map.Range used by
// purgeSystemAuthCache is not a snapshot, so a request that lands while the
// purge is in flight could install a stale entry that Range never sees. The
// timestamp closes that race without taking a global lock.
var lastInvalidatedAt sync.Map

// LastInvalidatedAt returns the most recent invalidation time for the given
// system_key, or the zero time if none was ever recorded.
func LastInvalidatedAt(systemKey string) time.Time {
	if v, ok := lastInvalidatedAt.Load(systemKey); ok {
		if t, ok := v.(time.Time); ok {
			return t
		}
	}
	return time.Time{}
}

// authInvalidationChannel returns the env-namespaced channel name. Must match
// what backend publishes on (see backend/cache/system_auth.go).
func authInvalidationChannel() string {
	env := configuration.Config.AppEnv
	if env == "" {
		env = "dev"
	}
	return systemAuthInvalidationChannelBase + ":" + env
}

// verifyInvalidationPayload accepts either a bare system_key (when no shared
// HMAC secret is configured locally) or `<systemKey>|<hex(hmac)>` and returns
// the validated system_key. Mismatched HMAC, missing HMAC when one is
// expected, or invalid system_key shape all return ("", false) so the caller
// drops the message.
func verifyInvalidationPayload(payload string) (string, bool) {
	secret := configuration.Config.InternalHMACSecret
	parts := strings.SplitN(strings.TrimSpace(payload), "|", 2)
	systemKey := parts[0]
	if !systemKeyFormat.MatchString(systemKey) {
		return "", false
	}
	if secret == "" {
		// No verification key configured — accept legacy bare-system_key
		// payloads. A future hardening can flip this to "reject if no HMAC".
		return systemKey, true
	}
	if len(parts) != 2 {
		return "", false
	}
	expected, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(systemKey))
	if !hmac.Equal(expected, mac.Sum(nil)) {
		return "", false
	}
	return systemKey, true
}

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

	channel := authInvalidationChannel()
	pubsub := rdb.Subscribe(ctx, channel)
	go func() {
		defer func() {
			if err := pubsub.Close(); err != nil {
				logger.Warn().Err(err).Msg("failed to close auth invalidation subscription")
			}
		}()
		ch := pubsub.Channel()
		logger.Info().Str("channel", channel).Msg("auth invalidation bus started")
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				systemKey, ok := verifyInvalidationPayload(msg.Payload)
				if !ok {
					logger.Warn().
						Int("payload_bytes", len(msg.Payload)).
						Msg("auth invalidation: dropping payload with bad shape or signature")
					continue
				}
				// Stamp the invalidation moment BEFORE purging so any
				// concurrent cache insert is detectable by checkInProcessCache.
				lastInvalidatedAt.Store(systemKey, time.Now())
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
