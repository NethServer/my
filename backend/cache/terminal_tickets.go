/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	terminalTicketPrefix = "terminal_ticket:"
	terminalTicketTTL    = 30 * time.Second
)

// TerminalTicket represents a one-time ticket for WebSocket terminal authentication.
// The ticket is stored in Redis with a short TTL and consumed on first use.
type TerminalTicket struct {
	SessionID      string `json:"session_id"`
	UserID         string `json:"user_id"`
	UserLogtoID    string `json:"user_logto_id"`
	Username       string `json:"username"`
	Name           string `json:"name"`
	OrgRole        string `json:"org_role"`
	OrganizationID string `json:"organization_id"`
}

// GenerateTerminalTicket creates a one-time ticket for terminal WebSocket authentication.
// Returns the ticket string that the client uses as ?ticket= query parameter.
func GenerateTerminalTicket(ticket *TerminalTicket) (string, error) {
	rc := GetRedisClient()
	if rc == nil {
		return "", fmt.Errorf("redis not available")
	}

	// Generate a cryptographically random ticket ID
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate ticket: %w", err)
	}
	ticketID := hex.EncodeToString(b)

	key := terminalTicketPrefix + ticketID
	if err := rc.Set(key, ticket, terminalTicketTTL); err != nil {
		return "", fmt.Errorf("failed to store ticket: %w", err)
	}

	log.Debug().
		Str("component", "terminal_ticket").
		Str("session_id", ticket.SessionID).
		Str("user_id", ticket.UserID).
		Msg("Terminal ticket generated")

	return ticketID, nil
}

// ConsumeTerminalTicket atomically retrieves and deletes a one-time terminal ticket
// using Redis GETDEL to prevent race conditions (TOCTOU).
// Returns nil if the ticket does not exist or has expired.
func ConsumeTerminalTicket(ticketID string) (*TerminalTicket, error) {
	rc := GetRedisClient()
	if rc == nil {
		return nil, fmt.Errorf("redis not available")
	}

	key := terminalTicketPrefix + ticketID
	var ticket TerminalTicket
	if err := rc.GetDel(key, &ticket); err != nil {
		if err == ErrCacheMiss {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to consume ticket: %w", err)
	}

	log.Debug().
		Str("component", "terminal_ticket").
		Str("session_id", ticket.SessionID).
		Str("user_id", ticket.UserID).
		Msg("Terminal ticket consumed")

	return &ticket, nil
}
