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
	"errors"
	"time"
)

const supportProxyAccessKeyPrefix = "support_proxy_access:"

var errSupportProxyAccessRedisUnavailable = errors.New("redis unavailable")

// MarkSupportProxyAccess records the use of a proxy token on a service and
// reports whether it is the first one. On a Redis failure it reports true, so
// the access is logged twice rather than not at all.
func MarkSupportProxyAccess(tokenHash, service string, ttl time.Duration) (bool, error) {
	rc := GetRedisClient()
	if rc == nil || rc.client == nil {
		return true, errSupportProxyAccessRedisUnavailable
	}
	if ttl <= 0 {
		ttl = time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), rc.defaultTimeout)
	defer cancel()
	first, err := rc.client.SetNX(ctx, supportProxyAccessKeyPrefix+tokenHash+":"+service, 1, ttl).Result()
	if err != nil {
		return true, err
	}
	return first, nil
}
