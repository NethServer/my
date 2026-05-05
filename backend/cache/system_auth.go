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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// systemAuthInvalidationChannelBase is the namespace prefix for the cross-service
// Redis pub/sub channel backend uses to signal collect that a system_key's
// credentials have been revoked or rotated. The actual channel name is suffixed
// with APP_ENV ("dev"/"qa"/"prod") so a Redis instance shared across deployments
// stops cross-pollinating invalidations.
const systemAuthInvalidationChannelBase = "my:auth:invalidate"

// SystemAuthInvalidationChannel returns the env-namespaced channel name. Exported
// so tests and the consumer subscribe to the same string deterministically.
func SystemAuthInvalidationChannel() string {
	env := configuration.Config.AppEnv
	if env == "" {
		env = "dev"
	}
	return systemAuthInvalidationChannelBase + ":" + env
}

// signInvalidationPayload returns the wire payload sent on the channel.
// When INTERNAL_HMAC_SECRET is set the payload is `<systemKey>|<hex(hmac)>` so
// a network-adjacent attacker with PUBLISH access cannot forge invalidations.
// When the secret is empty (dev) the bare system_key is sent and the consumer
// accepts it on a best-effort basis.
func signInvalidationPayload(systemKey string) string {
	secret := configuration.Config.InternalHMACSecret
	if secret == "" {
		return systemKey
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(systemKey))
	return systemKey + "|" + hex.EncodeToString(mac.Sum(nil))
}

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
	channel := SystemAuthInvalidationChannel()
	payload := signInvalidationPayload(systemKey)
	if err := rc.client.Publish(ctx, channel, payload).Err(); err != nil {
		logger.Warn().Err(err).Str("system_key", systemKey).Msg("failed to publish auth invalidation")
		return
	}
	logger.Debug().Str("system_key", systemKey).Str("channel", channel).Msg("auth cache invalidation published")
}
