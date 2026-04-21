/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"context"

	"github.com/nethesis/my/backend/logger"
)

// systemAuthInvalidationChannel is the cross-service Redis pub/sub channel
// backend uses to signal collect (and any future subscriber) that a
// system_key's credentials have been revoked or rotated. The payload is the
// plain system_key string.
const systemAuthInvalidationChannel = "my:auth:invalidate"

// InvalidateSystemAuth publishes a system_key on the global auth
// invalidation channel so collect can purge its in-memory and Redis-cached
// credentials for that system immediately, instead of waiting up to the
// full SystemAuthCacheTTL window.
//
// Best effort: a Redis outage is logged but does not block the caller —
// the cached entries still expire naturally within SystemAuthCacheTTL.
func InvalidateSystemAuth(ctx context.Context, systemKey string) {
	if systemKey == "" {
		return
	}
	rc := GetRedisClient()
	if rc == nil || rc.client == nil {
		logger.Warn().Str("system_key", systemKey).Msg("Redis unavailable, skipping auth cache invalidation")
		return
	}
	if err := rc.client.Publish(ctx, systemAuthInvalidationChannel, systemKey).Err(); err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("failed to publish auth invalidation")
		return
	}
	logger.Debug().Str("system_key", systemKey).Msg("auth cache invalidation published")
}
