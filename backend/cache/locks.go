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
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/nethesis/my/backend/logger"
)

// ErrSystemReassignLocked is returned by AcquireSystemReassignLock when
// another concurrent reassignment for the same system holds the lock.
var ErrSystemReassignLocked = errors.New("system reassignment is already in progress")

// ErrCustomerTransferLocked is returned by AcquireCustomerTransferLock when
// another transfer of the same customer holds the lock.
var ErrCustomerTransferLocked = errors.New("customer transfer is already in progress")

const (
	systemReassignLockKeyPrefix   = "lock:system:reassign:"
	customerTransferLockKeyPrefix = "lock:customer:transfer:"
)

// AcquireSystemReassignLock takes a Redis SETNX-based advisory lock on the
// system_id for the duration of an org reassignment. The lock has a TTL so a
// crashed process cannot wedge a system forever; the TTL is set generously
// (default 5 m) above the realistic ceiling of the migration loop, which
// typically runs in single-digit seconds.
//
// Returns a release function that the caller MUST defer; it deletes the key
// only if the value still matches the random fence token written here, so a
// release after the TTL expired (and another holder may have taken the lock)
// does not free the wrong holder.
//
// If Redis is unreachable the lock fails open: a logged warning, no lock
// taken, the caller proceeds. This matches the established pattern for
// non-critical cache infrastructure (the worst case under Redis outage is
// the same as before this lock existed) and avoids tying a normal admin
// operation to Redis availability.
func AcquireSystemReassignLock(ctx context.Context, systemID string, ttl time.Duration) (func(), error) {
	return acquireAdvisoryLock(ctx, systemReassignLockKeyPrefix+systemID, ttl, ErrSystemReassignLocked)
}

// AcquireCustomerTransferLock serializes the "managed by" transfers of one
// customer, with the same fence-token and fail-open semantics as
// AcquireSystemReassignLock: two concurrent transfers to different targets
// would otherwise both pass validation and leave Logto and Postgres pointing
// at different parents.
func AcquireCustomerTransferLock(ctx context.Context, customerID string, ttl time.Duration) (func(), error) {
	return acquireAdvisoryLock(ctx, customerTransferLockKeyPrefix+customerID, ttl, ErrCustomerTransferLocked)
}

func acquireAdvisoryLock(ctx context.Context, key string, ttl time.Duration, lockedErr error) (func(), error) {
	rc := GetRedisClient()
	if rc == nil || rc.client == nil {
		logger.Warn().Str("lock_key", key).Msg("Redis unavailable, advisory lock disabled")
		return func() {}, nil
	}

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(tokenBytes)

	ok, err := rc.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		// Treat Redis errors as fail-open: log and let the caller proceed.
		logger.Warn().Err(err).Str("lock_key", key).Msg("advisory lock SETNX failed; proceeding without lock")
		return func() {}, nil
	}
	if !ok {
		return nil, lockedErr
	}

	release := func() {
		// Compare-and-delete via a small Lua script: only release the lock
		// when its value still matches our token, so an expired-then-retaken
		// lock is not freed by us.
		const releaseScript = `
			if redis.call("GET", KEYS[1]) == ARGV[1] then
				return redis.call("DEL", KEYS[1])
			else
				return 0
			end`
		ctxRelease, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rc.client.Eval(ctxRelease, releaseScript, []string{key}, token).Err(); err != nil {
			logger.Warn().Err(err).Str("lock_key", key).Msg("advisory lock release failed")
		}
	}
	return release, nil
}
