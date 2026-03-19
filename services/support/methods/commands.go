/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package methods

import (
	"context"
	"encoding/json"

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/queue"
	"github.com/nethesis/my/services/support/session"
	"github.com/nethesis/my/services/support/tunnel"
)

// SupportCommand represents a command received via Redis pub/sub
type SupportCommand struct {
	Action    string                        `json:"action"`
	SessionID string                        `json:"session_id"`
	Services  map[string]tunnel.ServiceInfo `json:"services,omitempty"`
}

// StartCommandListener listens for commands from the backend via Redis pub/sub
func StartCommandListener(ctx context.Context) {
	log := logger.ComponentLogger("commands")

	pubsub := queue.Subscribe(ctx, "support:commands")
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	log.Info().Msg("command listener started on support:commands channel")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("command listener stopped")
			return
		case msg, ok := <-ch:
			if !ok {
				log.Warn().Msg("command channel closed")
				return
			}

			var cmd SupportCommand
			if err := json.Unmarshal([]byte(msg.Payload), &cmd); err != nil {
				log.Error().Err(err).Str("payload", msg.Payload).Msg("invalid command payload")
				continue
			}

			log.Info().
				Str("action", cmd.Action).
				Str("session_id", cmd.SessionID).
				Msg("command received")

			switch cmd.Action {
			case "close":
				handleCloseCommand(cmd.SessionID)
			case "add_services":
				handleAddServicesCommand(cmd)
			default:
				log.Warn().Str("action", cmd.Action).Msg("unknown command action")
			}
		}
	}
}

func handleAddServicesCommand(cmd SupportCommand) {
	log := logger.ComponentLogger("commands")

	payload := tunnel.CommandPayload{
		Action:   "add_services",
		Services: cmd.Services,
	}

	if err := TunnelManager.SendCommandToSession(cmd.SessionID, payload); err != nil {
		log.Error().Err(err).Str("session_id", cmd.SessionID).Msg("failed to send add_services command to tunnel")
	} else {
		log.Info().Str("session_id", cmd.SessionID).Int("count", len(cmd.Services)).Msg("add_services command sent")
	}
}

func handleCloseCommand(sessionID string) {
	log := logger.ComponentLogger("commands")

	// Close the tunnel
	if TunnelManager.CloseBySessionID(sessionID) {
		log.Info().Str("session_id", sessionID).Msg("tunnel closed by command")
	}

	// Close the session in the database
	if err := session.CloseSession(sessionID, "operator"); err != nil {
		log.Error().Err(err).Str("session_id", sessionID).Msg("failed to close session")
	}
}
