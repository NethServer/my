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

const (
	supportProxyCodePrefix = "support_proxy_code:"
	supportProxyCodeTTL    = 30 * time.Second
)

// supportProxyCode is what a one-time proxy code stands for
type supportProxyCode struct {
	Token string `json:"token"`
}

// IssueSupportProxyCode stores a proxy token behind a one-time code that is
// short-lived, so the token itself never travels in a URL (access logs,
// history): the browser hands the code to the proxy, which swaps it for the
// token in an HttpOnly cookie.
func IssueSupportProxyCode(token string) (string, error) {
	rc := GetRedisClient()
	if rc == nil {
		return "", errSupportProxyAccessRedisUnavailable
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := hex.EncodeToString(b)
	if err := rc.Set(supportProxyCodePrefix+code, supportProxyCode{Token: token}, supportProxyCodeTTL); err != nil {
		return "", err
	}
	return code, nil
}

// ConsumeSupportProxyCode returns the token behind a one-time code and deletes
// it, or "" when the code is unknown, used or expired.
func ConsumeSupportProxyCode(code string) (string, error) {
	rc := GetRedisClient()
	if rc == nil {
		return "", errSupportProxyAccessRedisUnavailable
	}
	var entry supportProxyCode
	if err := rc.GetDel(supportProxyCodePrefix+code, &entry); err != nil {
		if err == ErrCacheMiss {
			return "", nil
		}
		return "", err
	}
	return entry.Token, nil
}
