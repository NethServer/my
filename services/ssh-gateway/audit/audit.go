/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package audit

import (
	"encoding/json"

	"github.com/nethesis/my/services/ssh-gateway/database"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

// SSHAccessMetadata holds metadata stored in the access log's jsonb column
type SSHAccessMetadata struct {
	AuthMethod  string `json:"auth_method"`       // "browser" or "cached_key"
	Command     string `json:"command,omitempty"` // command executed (empty for interactive shell)
	SessionType string `json:"session_type"`      // "interactive", "exec", "scp"
	ClientIP    string `json:"client_ip"`
	Fingerprint string `json:"fingerprint,omitempty"` // SSH public key fingerprint
}

// LogConnect inserts an access log entry for an SSH connection and returns the log ID
func LogConnect(sessionID, operatorID, operatorName string, meta SSHAccessMetadata) string {
	if database.DB == nil {
		return ""
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		logger.Error().Err(err).Msg("failed to marshal SSH access metadata")
		return ""
	}

	var logID string
	err = database.DB.QueryRow(
		`INSERT INTO support_access_logs (session_id, operator_id, operator_name, access_type, connected_at, metadata)
		 VALUES ($1, $2, $3, 'ssh', NOW(), $4) RETURNING id`,
		sessionID, operatorID, operatorName, metaJSON,
	).Scan(&logID)
	if err != nil {
		logger.Error().Err(err).
			Str("session_id", sessionID).
			Str("operator_id", operatorID).
			Msg("failed to insert SSH access log")
		return ""
	}

	logger.Info().
		Str("log_id", logID).
		Str("session_id", sessionID).
		Str("operator", operatorName).
		Str("auth_method", meta.AuthMethod).
		Str("session_type", meta.SessionType).
		Str("client_ip", meta.ClientIP).
		Msg("SSH access log created")

	return logID
}

// LogDisconnect sets disconnected_at on an access log entry
func LogDisconnect(logID string) {
	if database.DB == nil || logID == "" {
		return
	}

	_, err := database.DB.Exec(
		`UPDATE support_access_logs SET disconnected_at = NOW() WHERE id = $1 AND disconnected_at IS NULL`,
		logID,
	)
	if err != nil {
		logger.Error().Err(err).Str("log_id", logID).Msg("failed to update SSH access log disconnect")
	}
}
