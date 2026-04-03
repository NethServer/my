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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/queue"
	"github.com/nethesis/my/services/support/session"
	"github.com/nethesis/my/services/support/tunnel"
)

// signedEnvelope wraps a Redis pub/sub payload with an HMAC-SHA256 signature.
// The backend signs all messages with SUPPORT_INTERNAL_SECRET; the support service verifies
// using INTERNAL_SECRET (the same shared secret) before processing any command.
type signedEnvelope struct {
	Payload string `json:"payload"`
	Sig     string `json:"sig"`
}

// verifyAndUnwrap authenticates a signed Redis message and returns the inner payload.
// INTERNAL_SECRET is required at startup, so HMAC verification is always enforced.
func verifyAndUnwrap(raw string) (string, bool) {
	var env signedEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		return "", false
	}
	secret := configuration.Config.InternalSecret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(env.Payload))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(env.Sig)) {
		return "", false
	}
	return env.Payload, true
}

// SupportCommand represents a command received via Redis pub/sub
type SupportCommand struct {
	Action       string                        `json:"action"`
	SessionID    string                        `json:"session_id"`
	Services     map[string]tunnel.ServiceInfo `json:"services,omitempty"`
	ServiceNames []string                      `json:"service_names,omitempty"` // for remove_services
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
		case msg, chanOk := <-ch:
			if !chanOk {
				log.Warn().Msg("command channel closed")
				return
			}

			// Fix #2: verify HMAC signature before processing any command
			payload, valid := verifyAndUnwrap(msg.Payload)
			if !valid {
				log.Error().Msg("rejected Redis command: invalid or missing HMAC signature")
				continue
			}

			var cmd SupportCommand
			if err := json.Unmarshal([]byte(payload), &cmd); err != nil {
				log.Error().Err(err).Str("payload", payload).Msg("invalid command payload")
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
			case "remove_services":
				handleRemoveServicesCommand(cmd)
			default:
				log.Warn().Str("action", cmd.Action).Msg("unknown command action")
			}
		}
	}
}

func handleAddServicesCommand(cmd SupportCommand) {
	log := logger.ComponentLogger("commands")

	// Fix #3: SSRF pre-check — validate each service target before forwarding to the tunnel-client.
	// This is a defense-in-depth layer; the tunnel-client also validates, but the server
	// should reject dangerous targets before they reach the customer's machine at all.
	for name, svc := range cmd.Services {
		if err := tunnel.ValidateServiceTarget(svc.Target); err != nil {
			log.Error().Err(err).
				Str("session_id", cmd.SessionID).
				Str("service", name).
				Str("target", svc.Target).
				Msg("rejected add_services command: dangerous service target")
			return
		}
	}

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

func handleRemoveServicesCommand(cmd SupportCommand) {
	log := logger.ComponentLogger("commands")

	// Remove services from the server-side tunnel registry immediately
	// so GET /services reflects the change without waiting for the tunnel-client manifest
	TunnelManager.RemoveServicesBySessionID(cmd.SessionID, cmd.ServiceNames)

	payload := tunnel.CommandPayload{
		Action:       "remove_services",
		ServiceNames: cmd.ServiceNames,
	}

	if err := TunnelManager.SendCommandToSession(cmd.SessionID, payload); err != nil {
		log.Error().Err(err).Str("session_id", cmd.SessionID).Msg("failed to send remove_services command to tunnel")
	} else {
		log.Info().Str("session_id", cmd.SessionID).Int("count", len(cmd.ServiceNames)).Msg("remove_services command sent")
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
