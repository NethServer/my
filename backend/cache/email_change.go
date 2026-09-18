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
	"errors"
	"time"
)

// A self-service email change is applied only after the user proves control of
// the new mailbox: the address is parked here while Logto's one-time code is in
// flight, and the verify step reads it back from here rather than from the
// request, so the address that gets verified is the address that gets written.
//
// Every failure to read or write the record is an error, never a silent miss:
// a store that cannot answer must not let an unverified address through.

const (
	pendingEmailChangeKeyPrefix = "email_change:"

	// PendingEmailChangeTTL bounds the whole exchange. Logto's codes live ten
	// minutes; the record must not outlive the code it refers to.
	PendingEmailChangeTTL = 10 * time.Minute

	// PendingEmailChangeResendCooldown is the minimum gap between two codes
	// requested by the same user, so one account cannot be turned into a mail
	// cannon aimed at somebody else's inbox.
	PendingEmailChangeResendCooldown = 60 * time.Second

	// PendingEmailChangeMaxAttempts caps the guesses at one six-digit code.
	// After that the record is dropped and a fresh code has to be requested.
	PendingEmailChangeMaxAttempts = 5
)

// ErrPendingEmailChangeUnavailable is returned when the store cannot be
// reached. Callers answer 503 and apply nothing.
var ErrPendingEmailChangeUnavailable = errors.New("pending email change store unavailable")

// PendingEmailChange is the parked address and the state of the exchange.
type PendingEmailChange struct {
	Email       string    `json:"email"`
	RequestedAt time.Time `json:"requested_at"`
	Attempts    int       `json:"attempts"`
}

func pendingEmailChangeKey(logtoID string) string {
	return pendingEmailChangeKeyPrefix + logtoID
}

// SetPendingEmailChange parks an address for the user, replacing any earlier
// one, and restarts the attempt counter.
func SetPendingEmailChange(logtoID, email string) error {
	rc := GetRedisClient()
	if rc == nil {
		return ErrPendingEmailChangeUnavailable
	}
	rec := PendingEmailChange{Email: email, RequestedAt: time.Now().UTC()}
	if err := rc.Set(pendingEmailChangeKey(logtoID), rec, PendingEmailChangeTTL); err != nil {
		return ErrPendingEmailChangeUnavailable
	}
	return nil
}

// GetPendingEmailChange returns the parked address, or (nil, nil) when the
// user has none in flight.
func GetPendingEmailChange(logtoID string) (*PendingEmailChange, error) {
	rc := GetRedisClient()
	if rc == nil {
		return nil, ErrPendingEmailChangeUnavailable
	}
	var rec PendingEmailChange
	err := rc.Get(pendingEmailChangeKey(logtoID), &rec)
	if err == nil {
		return &rec, nil
	}
	if errors.Is(err, ErrCacheMiss) {
		return nil, nil
	}
	return nil, ErrPendingEmailChangeUnavailable
}

// RecordPendingEmailChangeAttempt counts one failed guess. Once the cap is
// reached the record is dropped and the function reports it, so the caller can
// tell the user to start over. The TTL is preserved: a wrong guess must not
// extend the life of the code.
func RecordPendingEmailChangeAttempt(logtoID string, rec *PendingEmailChange) (exhausted bool, err error) {
	rc := GetRedisClient()
	if rc == nil {
		return false, ErrPendingEmailChangeUnavailable
	}
	rec.Attempts++
	if rec.Attempts >= PendingEmailChangeMaxAttempts {
		_ = rc.Delete(pendingEmailChangeKey(logtoID))
		return true, nil
	}
	remaining := PendingEmailChangeTTL - time.Since(rec.RequestedAt)
	if remaining <= 0 {
		_ = rc.Delete(pendingEmailChangeKey(logtoID))
		return true, nil
	}
	if err := rc.Set(pendingEmailChangeKey(logtoID), *rec, remaining); err != nil {
		return false, ErrPendingEmailChangeUnavailable
	}
	return false, nil
}

// DeletePendingEmailChange forgets the parked address (applied, or abandoned).
func DeletePendingEmailChange(logtoID string) error {
	rc := GetRedisClient()
	if rc == nil {
		return ErrPendingEmailChangeUnavailable
	}
	if err := rc.Delete(pendingEmailChangeKey(logtoID)); err != nil {
		return ErrPendingEmailChangeUnavailable
	}
	return nil
}
