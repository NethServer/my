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
	"time"

	"github.com/nethesis/my/services/support/logger"
	"github.com/nethesis/my/services/support/session"
	"github.com/nethesis/my/services/support/tunnel"
)

// restoreDelay lets the first manifest's rate-limit window pass: the client
// answers the restored services with a new manifest, which must be accepted
const restoreDelay = 11 * time.Second

// restoreBatch matches the tunnel client's add_services limit per command
const restoreBatch = 10

// restoreSavedServices hands a freshly connected tunnel the services operators
// added to its system node in earlier sessions
func restoreSavedServices(t *tunnel.Tunnel) {
	log := logger.ComponentLogger("tunnel")
	saved, err := session.GetSavedServices(t.SystemID, t.NodeID)
	if err != nil {
		log.Error().Err(err).Str("system_id", t.SystemID).Msg("cannot read the saved services")
		return
	}
	if len(saved) == 0 {
		return
	}

	batch := make(map[string]tunnel.ServiceInfo, restoreBatch)
	send := func() {
		if len(batch) == 0 {
			return
		}
		payload := tunnel.CommandPayload{Action: "add_services", Services: batch}
		if err := TunnelManager.SendCommandToSession(t.SessionID, payload); err != nil {
			log.Warn().Err(err).Str("session_id", t.SessionID).Msg("cannot restore the saved services")
		}
		batch = make(map[string]tunnel.ServiceInfo, restoreBatch)
	}
	for _, s := range saved {
		svc := tunnel.ServiceInfo{Target: s.Target, Label: s.Label, TLS: s.TLS}
		// the same checks as a service added live
		if err := tunnel.ValidateService(svc); err != nil {
			log.Warn().Err(err).Str("system_id", t.SystemID).Str("service", s.Name).Msg("skipping an invalid saved service")
			continue
		}
		batch[s.Name] = svc
		if len(batch) == restoreBatch {
			send()
		}
	}
	send()
	log.Info().Str("system_id", t.SystemID).Str("node_id", t.NodeID).Int("count", len(saved)).Msg("saved services restored")
}
