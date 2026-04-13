/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

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

// GetAlertHistoryBySystemKey returns paginated alert history for a given system_key
// scoped to an organization. Both filters are mandatory to prevent cross-tenant
// disclosure if two systems ever share a system_key.
// Valid sortBy values: id, alertname, severity, status, starts_at, ends_at, created_at.
// sortDirection must be "asc" or "desc".
func (r *LocalAlertHistoryRepository) GetAlertHistoryBySystemKey(systemKey, organizationID string, page, pageSize int, sortBy, sortDirection string) ([]models.AlertHistoryRecord, int, error) {
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

	countQuery := `SELECT COUNT(*) FROM alert_history WHERE system_key = $1 AND organization_id = $2`
	var totalCount int
	if err := r.db.QueryRow(countQuery, systemKey, organizationID).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count alert history: %w", err)
	}

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(`
		SELECT id, system_key, alertname, severity, status, fingerprint,
		       starts_at, ends_at, summary, labels, annotations, receiver, created_at
		FROM alert_history
		WHERE system_key = $1 AND organization_id = $2
		ORDER BY %s %s
		LIMIT $3 OFFSET $4
	`, col, sortDirection)

	rows, err := r.db.Query(query, systemKey, organizationID, pageSize, offset)
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

// GetAlertHistoryTotals returns the total count of alert history records.
// When orgID is non-empty, results are scoped to that organization; the caller
// is expected to have validated hierarchy access. An empty orgID returns the
// aggregate across all tenants and is reserved for callers (Owner) that have
// already cleared that authorization gate.
func (r *LocalAlertHistoryRepository) GetAlertHistoryTotals(orgID string) (int, error) {
	var total int
	if orgID == "" {
		if err := r.db.QueryRow(`SELECT COUNT(*) FROM alert_history`).Scan(&total); err != nil {
			return 0, fmt.Errorf("failed to count alert history: %w", err)
		}
		return total, nil
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM alert_history WHERE organization_id = $1`, orgID).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to count alert history: %w", err)
	}
	return total, nil
}

// GetAlertHistoryTrend returns trend data for resolved alerts over a specified
// period (days). When orgID is non-empty, results are scoped to that
// organization; an empty orgID returns the aggregate across all tenants and is
// reserved for Owner-only callers that have cleared authorization upstream.
func (r *LocalAlertHistoryRepository) GetAlertHistoryTrend(period int, orgID string) (*models.TrendResponse, error) {
	now := time.Now().UTC()
	currentStart := now.AddDate(0, 0, -period)
	previousStart := currentStart.AddDate(0, 0, -period)

	scopeAll := orgID == ""

	var currentTotal int
	{
		query := `SELECT COUNT(*) FROM alert_history WHERE created_at >= $1`
		args := []interface{}{currentStart}
		if !scopeAll {
			query += ` AND organization_id = $2`
			args = append(args, orgID)
		}
		if err := r.db.QueryRow(query, args...).Scan(&currentTotal); err != nil {
			return nil, fmt.Errorf("failed to count current period: %w", err)
		}
	}

	var previousTotal int
	{
		query := `SELECT COUNT(*) FROM alert_history WHERE created_at >= $1 AND created_at < $2`
		args := []interface{}{previousStart, currentStart}
		if !scopeAll {
			query += ` AND organization_id = $3`
			args = append(args, orgID)
		}
		if err := r.db.QueryRow(query, args...).Scan(&previousTotal); err != nil {
			return nil, fmt.Errorf("failed to count previous period: %w", err)
		}
	}

	dpQuery := `SELECT DATE(created_at) AS day, COUNT(*) AS count FROM alert_history WHERE created_at >= $1`
	dpArgs := []interface{}{currentStart}
	if !scopeAll {
		dpQuery += ` AND organization_id = $2`
		dpArgs = append(dpArgs, orgID)
	}
	dpQuery += ` GROUP BY day ORDER BY day`
	rows, err := r.db.Query(dpQuery, dpArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query data points: %w", err)
	}
	defer func() { _ = rows.Close() }()

	pointMap := make(map[string]int)
	for rows.Next() {
		var day time.Time
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			return nil, fmt.Errorf("failed to scan data point: %w", err)
		}
		pointMap[day.Format("2006-01-02")] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate data points: %w", err)
	}

	// Build full data points array (one per day, zero-fill gaps)
	dataPoints := make([]models.TrendDataPoint, 0, period)
	for i := 0; i < period; i++ {
		date := currentStart.AddDate(0, 0, i+1).Format("2006-01-02")
		count := pointMap[date]
		dataPoints = append(dataPoints, models.TrendDataPoint{Date: date, Count: count})
	}

	delta := currentTotal - previousTotal
	var deltaPercentage float64
	if previousTotal > 0 {
		deltaPercentage = math.Round(float64(delta)/float64(previousTotal)*10000) / 100
	}

	trend := "stable"
	if delta > 0 {
		trend = "up"
	} else if delta < 0 {
		trend = "down"
	}

	periodLabels := map[int]string{7: "7 days", 30: "30 days", 180: "6 months", 365: "1 year"}
	label := periodLabels[period]
	if label == "" {
		label = fmt.Sprintf("%d days", period)
	}

	return &models.TrendResponse{
		Period:          period,
		PeriodLabel:     label,
		CurrentTotal:    currentTotal,
		PreviousTotal:   previousTotal,
		Delta:           delta,
		DeltaPercentage: deltaPercentage,
		Trend:           trend,
		DataPoints:      dataPoints,
	}, nil
}
