/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// LocalAlertHistoryRepository implements alert history queries against the shared database.
type LocalAlertHistoryRepository struct {
	db *sql.DB
}

// NewLocalAlertHistoryRepository creates a new alert history repository.
func NewLocalAlertHistoryRepository() *LocalAlertHistoryRepository {
	return &LocalAlertHistoryRepository{db: database.DB}
}

// GetAlertHistoryBySystemKey returns paginated alert history for a given system_key.
// Valid sortBy values: id, alertname, severity, status, starts_at, ends_at, created_at.
// sortDirection must be "asc" or "desc".
func (r *LocalAlertHistoryRepository) GetAlertHistoryBySystemKey(systemKey string, page, pageSize int, sortBy, sortDirection string) ([]models.AlertHistoryRecord, int, error) {
	// Allowlist for sortBy to prevent SQL injection.
	allowedSortBy := map[string]string{
		"id":         "id",
		"alertname":  "alertname",
		"severity":   "severity",
		"status":     "status",
		"starts_at":  "starts_at",
		"ends_at":    "ends_at",
		"created_at": "created_at",
	}
	col, ok := allowedSortBy[sortBy]
	if !ok {
		col = "created_at"
	}
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	countQuery := `SELECT COUNT(*) FROM alert_history WHERE system_key = $1`
	var totalCount int
	if err := r.db.QueryRow(countQuery, systemKey).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count alert history: %w", err)
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, system_key, alertname, severity, status, fingerprint,
		       starts_at, ends_at, summary, labels, annotations, receiver, created_at
		FROM alert_history
		WHERE system_key = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3
	`, col, sortDirection)

	rows, err := r.db.Query(query, systemKey, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alert history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []models.AlertHistoryRecord
	for rows.Next() {
		var rec models.AlertHistoryRecord
		var labelsRaw, annotationsRaw []byte
		var severity, summary, receiver sql.NullString
		var endsAt sql.NullTime

		err := rows.Scan(
			&rec.ID,
			&rec.SystemKey,
			&rec.Alertname,
			&severity,
			&rec.Status,
			&rec.Fingerprint,
			&rec.StartsAt,
			&endsAt,
			&summary,
			&labelsRaw,
			&annotationsRaw,
			&receiver,
			&rec.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan alert history row: %w", err)
		}

		if severity.Valid {
			rec.Severity = &severity.String
		}
		if summary.Valid {
			rec.Summary = &summary.String
		}
		if receiver.Valid {
			rec.Receiver = &receiver.String
		}
		if endsAt.Valid {
			rec.EndsAt = &endsAt.Time
		}

		if err := json.Unmarshal(labelsRaw, &rec.Labels); err != nil {
			rec.Labels = map[string]string{}
		}
		if err := json.Unmarshal(annotationsRaw, &rec.Annotations); err != nil {
			rec.Annotations = map[string]string{}
		}

		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate alert history rows: %w", err)
	}

	if records == nil {
		records = []models.AlertHistoryRecord{}
	}

	return records, totalCount, nil
}
