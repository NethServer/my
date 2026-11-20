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
	"encoding/json"
	"sync"
	"time"
)

// Who is using the web interfaces of a support session: every proxied request
// refreshes the operator's mark on the service, and a mark older than
// SupportPresenceWindow means the operator left. Terminals are tracked by
// their access log rows, which stay open while connected.
const (
	supportPresencePrefix = "support_presence:"
	// how long a mark lasts without new requests
	SupportPresenceWindow = 2 * time.Minute
	// a busy page refreshes its mark at most this often
	supportPresenceThrottle = 30 * time.Second
)

// SupportPresence is an operator seen on a web interface of a session
type SupportPresence struct {
	SessionID    string `json:"session_id"`
	OperatorID   string `json:"operator_id"`
	OperatorName string `json:"operator_name"`
	// the proxy host label of the service: the client maps it to its name
	Service  string    `json:"service"`
	LastSeen time.Time `json:"last_seen"`
}

var presenceTouched sync.Map // session|operator|service -> time.Time

// TouchSupportPresence marks the operator as using the service now. Failures
// are ignored: presence is an indication, never a reason to fail a request.
func TouchSupportPresence(sessionID, operatorID, operatorName, service string) {
	field := operatorID + "|" + service
	local := sessionID + "|" + field
	now := time.Now()
	if last, ok := presenceTouched.Load(local); ok && now.Sub(last.(time.Time)) < supportPresenceThrottle {
		return
	}
	presenceTouched.Store(local, now)

	rc := GetRedisClient()
	if rc == nil || rc.client == nil {
		return
	}
	value, err := json.Marshal(SupportPresence{
		SessionID: sessionID, OperatorID: operatorID, OperatorName: operatorName, Service: service, LastSeen: now,
	})
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), rc.defaultTimeout)
	defer cancel()
	key := supportPresencePrefix + sessionID
	pipe := rc.client.TxPipeline()
	pipe.HSet(ctx, key, field, value)
	pipe.Expire(ctx, key, SupportPresenceWindow*2)
	_, _ = pipe.Exec(ctx)

	// the throttle map would otherwise grow with every session ever proxied
	presenceTouched.Range(func(k, v interface{}) bool {
		if now.Sub(v.(time.Time)) > SupportPresenceWindow {
			presenceTouched.Delete(k)
		}
		return true
	})
}

// GetSupportPresence returns the operators seen within SupportPresenceWindow
// on the web interfaces of the given sessions
func GetSupportPresence(sessionIDs []string) []SupportPresence {
	result := []SupportPresence{}
	rc := GetRedisClient()
	if rc == nil || rc.client == nil || len(sessionIDs) == 0 {
		return result
	}
	ctx, cancel := context.WithTimeout(context.Background(), rc.defaultTimeout)
	defer cancel()
	cutoff := time.Now().Add(-SupportPresenceWindow)
	for _, sessionID := range sessionIDs {
		entries, err := rc.client.HGetAll(ctx, supportPresencePrefix+sessionID).Result()
		if err != nil {
			continue
		}
		for _, raw := range entries {
			var p SupportPresence
			if json.Unmarshal([]byte(raw), &p) != nil || p.LastSeen.Before(cutoff) {
				continue
			}
			result = append(result, p)
		}
	}
	return result
}
