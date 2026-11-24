/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package session

import (
	"context"
	"time"

	"github.com/nethesis/my/services/support/configuration"
	"github.com/nethesis/my/services/support/logger"
)

// CleanerCallback is called when sessions are expired so tunnels can be disconnected
type CleanerCallback func(expiredSessionIDs []string)

// StartCleaner runs the session cleanup goroutine
func StartCleaner(ctx context.Context, callback CleanerCallback) {
	log := logger.ComponentLogger("session_cleaner")
	ticker := time.NewTicker(configuration.Config.SessionCleanerInterval)
	defer ticker.Stop()

	log.Info().
		Dur("interval", configuration.Config.SessionCleanerInterval).
		Msg("session cleaner started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("session cleaner stopped")
			return
		case <-ticker.C:
			expiredIDs, err := ExpireSessions()
			if err != nil {
				log.Error().Err(err).Msg("failed to expire sessions")
				continue
			}

			if len(expiredIDs) > 0 {
				log.Info().Int("expired_count", len(expiredIDs)).Msg("sessions expired")
				if callback != nil {
					callback(expiredIDs)
				}
			}
		}
	}
}
