/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"fmt"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

// RecordAPIKeyEvent appends an audit row. Best-effort: a failure is logged but
// never propagated, so auditing can't break the operation being audited.
func (s *APIKeysService) RecordAPIKeyEvent(rec models.APIKeyAuditRecord) {
	_, err := database.DB.Exec(`
		INSERT INTO api_key_audit
			(id, api_key_id, user_id, organization_id, event, reason, key_name, key_mode, ip, method, path)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		nullString(rec.APIKeyID), nullString(rec.UserID), nullString(rec.OrganizationID),
		rec.Event, nullString(rec.Reason), nullString(rec.KeyName), nullString(rec.KeyMode),
		nullString(rec.IP), nullString(rec.Method), nullString(rec.Path),
	)
	if err != nil {
		logger.Logger.Warn().
			Err(err).
			Str("component", "api-keys").
			Str("event", rec.Event).
			Msg("Failed to record api key audit event")
	}
}

// ListAPIKeyAudit returns a page of audit entries owned by the user, newest
// first, optionally filtered by event and/or key id. Returns entries + total.
func (s *APIKeysService) ListAPIKeyAudit(userID, eventFilter, apiKeyIDFilter string, page, pageSize int) ([]models.APIKeyAuditEntry, int, error) {
	where := "WHERE user_id = $1"
	args := []any{userID}
	idx := 2
	if eventFilter != "" {
		where += fmt.Sprintf(" AND event = $%d", idx)
		args = append(args, eventFilter)
		idx++
	}
	if apiKeyIDFilter != "" {
		where += fmt.Sprintf(" AND api_key_id = $%d", idx)
		args = append(args, apiKeyIDFilter)
		idx++
	}

	var total int
	if err := database.DB.QueryRow("SELECT COUNT(*) FROM api_key_audit "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count api key audit: %w", err)
	}

	listQuery := "SELECT id, api_key_id, user_id, organization_id, event, reason, key_name, key_mode, ip, method, path, created_at " +
		"FROM api_key_audit " + where +
		fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := database.DB.Query(listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list api key audit: %w", err)
	}
	defer func() { _ = rows.Close() }()

	entries := make([]models.APIKeyAuditEntry, 0)
	for rows.Next() {
		var e models.APIKeyAuditEntry
		var apiKeyID, userIDCol, orgID, reason, keyName, keyMode, ip, method, path sql.NullString
		if err := rows.Scan(&e.ID, &apiKeyID, &userIDCol, &orgID, &e.Event, &reason,
			&keyName, &keyMode, &ip, &method, &path, &e.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan api key audit: %w", err)
		}
		e.APIKeyID = nullStringPtr(apiKeyID)
		e.UserID = nullStringPtr(userIDCol)
		e.OrganizationID = nullStringPtr(orgID)
		e.Reason = nullStringPtr(reason)
		e.KeyName = nullStringPtr(keyName)
		e.KeyMode = nullStringPtr(keyMode)
		e.IP = nullStringPtr(ip)
		e.Method = nullStringPtr(method)
		e.Path = nullStringPtr(path)
		entries = append(entries, e)
	}
	return entries, total, rows.Err()
}

func nullStringPtr(ns sql.NullString) *string {
	if ns.Valid {
		v := ns.String
		return &v
	}
	return nil
}
