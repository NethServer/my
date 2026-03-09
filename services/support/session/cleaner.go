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
	"github.com/nethesis/my/services/support/database"
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
			// Get sessions that are about to expire
			expiredIDs, err := getExpiredSessionIDs()
			if err != nil {
				log.Error().Err(err).Msg("failed to get expired session IDs")
				continue
			}

			// Expire sessions in DB
			count, err := ExpireSessions()
			if err != nil {
				log.Error().Err(err).Msg("failed to expire sessions")
				continue
			}

			if count > 0 {
				log.Info().Int64("expired_count", count).Msg("sessions expired")
				if callback != nil && len(expiredIDs) > 0 {
					callback(expiredIDs)
				}
			}
		}
	}
}

// getExpiredSessionIDs returns IDs of sessions that are expired but not yet marked
func getExpiredSessionIDs() ([]string, error) {
	rows, err := database.DB.Query(
		`SELECT id FROM support_sessions
		 WHERE status IN ('pending', 'active') AND expires_at < NOW()`,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
