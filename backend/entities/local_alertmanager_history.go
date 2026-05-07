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
	"strings"
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

// AlertHistoryQuery captures all the optional filters and pagination params
// for QueryAlertHistory. OrgIDs is the only required field (callers resolve
// scope upstream via resolveOrgScope or equivalent). SystemKey limits to one
// system (used by /api/systems/:id/alerts/history); when empty the query is
// org-level (used by /api/alerts/history).
type AlertHistoryQuery struct {
	OrgIDs        []string
	SystemKey     string
	Alertnames    []string
	Severities    []string
	Statuses      []string
	From          *time.Time
	To            *time.Time
	Page          int
	PageSize      int
	SortBy        string
	SortDirection string
}

// QueryAlertHistory returns paginated, filtered alert_history records scoped
// to the given organization IDs. An empty OrgIDs slice returns no rows (the
// caller has nothing in scope). Multi-value label filters are matched with
// IN (...); the date range (From/To) is half-open on the left (>= From,
// < To). Valid SortBy values: id, alertname, severity, status, starts_at,
// ends_at, created_at — invalid values fall back to created_at.
func (r *LocalAlertHistoryRepository) QueryAlertHistory(q AlertHistoryQuery) ([]models.AlertHistoryRecord, int, error) {
	allowedSortBy := map[string]string{
		"id": "id", "alertname": "alertname", "severity": "severity",
		"status": "status", "starts_at": "starts_at", "ends_at": "ends_at",
		"created_at": "created_at",
	}
	col, ok := allowedSortBy[q.SortBy]
	if !ok {
		col = "created_at"
	}
	dir := q.SortDirection
	if dir != "asc" && dir != "desc" {
		dir = "desc"
	}

	if len(q.OrgIDs) == 0 {
		return []models.AlertHistoryRecord{}, 0, nil
	}

	// Build WHERE clause incrementally with positional placeholders so we can
	// reuse it for both COUNT and SELECT queries. argsBase holds the shared
	// args; LIMIT/OFFSET get appended at the end of the SELECT only.
	conds := make([]string, 0, 6)
	args := make([]interface{}, 0, 8)
	idx := 1

	// Org IDs: WHERE organization_id IN ($N, $N+1, ...)
	{
		ph := make([]string, len(q.OrgIDs))
		for i, id := range q.OrgIDs {
			ph[i] = fmt.Sprintf("$%d", idx)
			args = append(args, id)
			idx++
		}
		conds = append(conds, fmt.Sprintf("organization_id IN (%s)", strings.Join(ph, ",")))
	}
	if q.SystemKey != "" {
		conds = append(conds, fmt.Sprintf("system_key = $%d", idx))
		args = append(args, q.SystemKey)
		idx++
	}
	if len(q.Alertnames) > 0 {
		ph := make([]string, len(q.Alertnames))
		for i, v := range q.Alertnames {
			ph[i] = fmt.Sprintf("$%d", idx)
			args = append(args, v)
			idx++
		}
		conds = append(conds, fmt.Sprintf("alertname IN (%s)", strings.Join(ph, ",")))
	}
	if len(q.Severities) > 0 {
		ph := make([]string, len(q.Severities))
		for i, v := range q.Severities {
			ph[i] = fmt.Sprintf("$%d", idx)
			args = append(args, v)
			idx++
		}
		conds = append(conds, fmt.Sprintf("severity IN (%s)", strings.Join(ph, ",")))
	}
	if len(q.Statuses) > 0 {
		ph := make([]string, len(q.Statuses))
		for i, v := range q.Statuses {
			ph[i] = fmt.Sprintf("$%d", idx)
			args = append(args, v)
			idx++
		}
		conds = append(conds, fmt.Sprintf("status IN (%s)", strings.Join(ph, ",")))
	}
	if q.From != nil {
		conds = append(conds, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *q.From)
		idx++
	}
	if q.To != nil {
		conds = append(conds, fmt.Sprintf("created_at < $%d", idx))
		args = append(args, *q.To)
		idx++
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var totalCount int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM alert_history `+where, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count alert history: %w", err)
	}

	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.Page <= 0 {
		q.Page = 1
	}
	offset := (q.Page - 1) * q.PageSize

	selectArgs := append([]interface{}{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	query := fmt.Sprintf(`
		SELECT id, system_key, alertname, severity, status, fingerprint,
		       starts_at, ends_at, summary, labels, annotations, receiver, created_at
		FROM alert_history
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, where, col, dir, idx, idx+1)

	rows, err := r.db.Query(query, selectArgs...)
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

// GetAlertHistoryTotalsByOrgIDs returns the total count of alert history records
// scoped to the given list of organization IDs. An empty slice returns 0 (the
// caller has no orgs in scope, so by definition there are no records to count).
// The caller is expected to have validated hierarchy access for every ID.
func (r *LocalAlertHistoryRepository) GetAlertHistoryTotalsByOrgIDs(orgIDs []string) (int, error) {
	if len(orgIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(orgIDs))
	args := make([]interface{}, len(orgIDs))
	for i, id := range orgIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`SELECT COUNT(*) FROM alert_history WHERE organization_id IN (%s)`, strings.Join(placeholders, ","))
	var total int
	if err := r.db.QueryRow(query, args...).Scan(&total); err != nil {
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

// AlertStatsQuery captures the inputs for GetAlertStats: scope, optional date
// range, and the top-N cap for grouped breakdowns.
type AlertStatsQuery struct {
	OrgIDs []string
	From   *time.Time
	To     *time.Time
	TopN   int
}

// AlertNameCount is a (alertname, count) pair used in top-N stats.
type AlertNameCount struct {
	Alertname string `json:"alertname"`
	Count     int    `json:"count"`
}

// SystemKeyCount is a (system_key, count) pair used in top-N stats.
type SystemKeyCount struct {
	SystemKey string `json:"system_key"`
	Count     int    `json:"count"`
}

// AlertStats is the response shape returned by GetAlertStats. mttr_seconds
// and mtbf_seconds are pointers so they can be omitted from the JSON when
// they cannot be computed (no resolved alerts / fewer than 2 events).
type AlertStats struct {
	Total         int              `json:"total"`
	BySeverity    map[string]int   `json:"by_severity"`
	TopAlertnames []AlertNameCount `json:"top_alertnames"`
	TopSystems    []SystemKeyCount `json:"top_systems"`
	MTTRSeconds   *int64           `json:"mttr_seconds,omitempty"`
	MTBFSeconds   *int64           `json:"mtbf_seconds,omitempty"`
}

// GetAlertStats returns aggregate statistics over alert_history scoped to the
// given organization IDs and (optionally) a date range. Issues five queries
// in sequence (count + 4 GROUP BYs) — they run on the same table with the
// same predicate set, so PostgreSQL's plan cache makes the marginal cost
// after the first one negligible. Empty OrgIDs short-circuits.
//
// MTBF approximation: when from/to are both provided, MTBF = (to - from) /
// total. Otherwise (max(starts_at) - min(starts_at)) / (total - 1). Returns
// nil when the result is undefined (total < 2 in the second case).
func (r *LocalAlertHistoryRepository) GetAlertStats(q AlertStatsQuery) (*AlertStats, error) {
	stats := &AlertStats{
		BySeverity:    map[string]int{},
		TopAlertnames: []AlertNameCount{},
		TopSystems:    []SystemKeyCount{},
	}
	if len(q.OrgIDs) == 0 {
		return stats, nil
	}
	if q.TopN <= 0 {
		q.TopN = 10
	}

	// Build the shared WHERE clause + args once, reuse across queries.
	conds := make([]string, 0, 4)
	args := make([]interface{}, 0, 6)
	idx := 1
	{
		ph := make([]string, len(q.OrgIDs))
		for i, id := range q.OrgIDs {
			ph[i] = fmt.Sprintf("$%d", idx)
			args = append(args, id)
			idx++
		}
		conds = append(conds, fmt.Sprintf("organization_id IN (%s)", strings.Join(ph, ",")))
	}
	if q.From != nil {
		conds = append(conds, fmt.Sprintf("created_at >= $%d", idx))
		args = append(args, *q.From)
		idx++
	}
	if q.To != nil {
		conds = append(conds, fmt.Sprintf("created_at < $%d", idx))
		args = append(args, *q.To)
		idx++
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	// 1. Total + severity buckets in a single GROUP BY (count rows w/ NULL severity too).
	{
		query := `SELECT COALESCE(severity, '') AS severity, COUNT(*) FROM alert_history ` + where + ` GROUP BY severity`
		rows, err := r.db.Query(query, args...)
		if err != nil {
			return nil, fmt.Errorf("severity stats: %w", err)
		}
		for rows.Next() {
			var sev string
			var cnt int
			if err := rows.Scan(&sev, &cnt); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan severity: %w", err)
			}
			stats.Total += cnt
			if sev != "" {
				stats.BySeverity[sev] = cnt
			}
		}
		_ = rows.Close()
	}

	if stats.Total == 0 {
		return stats, nil
	}

	// 2. Top-N alertnames.
	{
		query := fmt.Sprintf(`SELECT alertname, COUNT(*) AS c FROM alert_history %s GROUP BY alertname ORDER BY c DESC LIMIT $%d`, where, idx)
		rows, err := r.db.Query(query, append(args, q.TopN)...)
		if err != nil {
			return nil, fmt.Errorf("alertname stats: %w", err)
		}
		for rows.Next() {
			var nc AlertNameCount
			if err := rows.Scan(&nc.Alertname, &nc.Count); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan alertname: %w", err)
			}
			stats.TopAlertnames = append(stats.TopAlertnames, nc)
		}
		_ = rows.Close()
	}

	// 3. Top-N system_keys.
	{
		query := fmt.Sprintf(`SELECT system_key, COUNT(*) AS c FROM alert_history %s GROUP BY system_key ORDER BY c DESC LIMIT $%d`, where, idx)
		rows, err := r.db.Query(query, append(args, q.TopN)...)
		if err != nil {
			return nil, fmt.Errorf("system_key stats: %w", err)
		}
		for rows.Next() {
			var sc SystemKeyCount
			if err := rows.Scan(&sc.SystemKey, &sc.Count); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan system_key: %w", err)
			}
			stats.TopSystems = append(stats.TopSystems, sc)
		}
		_ = rows.Close()
	}

	// 4. MTTR (mean time to resolve, on rows with ends_at populated).
	{
		query := `SELECT AVG(EXTRACT(EPOCH FROM (ends_at - starts_at)))::bigint FROM alert_history ` + where + ` AND ends_at IS NOT NULL`
		var mttr sql.NullInt64
		if err := r.db.QueryRow(query, args...).Scan(&mttr); err != nil {
			return nil, fmt.Errorf("mttr: %w", err)
		}
		if mttr.Valid && mttr.Int64 >= 0 {
			v := mttr.Int64
			stats.MTTRSeconds = &v
		}
	}

	// 5. MTBF approximation.
	if q.From != nil && q.To != nil {
		span := int64(q.To.Sub(*q.From).Seconds())
		if span > 0 && stats.Total > 0 {
			v := span / int64(stats.Total)
			stats.MTBFSeconds = &v
		}
	} else if stats.Total >= 2 {
		var minT, maxT sql.NullTime
		query := `SELECT MIN(starts_at), MAX(starts_at) FROM alert_history ` + where
		if err := r.db.QueryRow(query, args...).Scan(&minT, &maxT); err != nil {
			return nil, fmt.Errorf("mtbf range: %w", err)
		}
		if minT.Valid && maxT.Valid {
			span := int64(maxT.Time.Sub(minT.Time).Seconds())
			if span > 0 {
				v := span / int64(stats.Total-1)
				stats.MTBFSeconds = &v
			}
		}
	}

	return stats, nil
}

// GetAlertHistoryTrendByOrgIDs returns trend data scoped to the given list of
// organization IDs. An empty slice returns a zero-filled response (no rows in
// scope). Used by /api/alerts/trend to aggregate over the caller's hierarchy
// (or a sub-tree drill-down) in a single SQL roundtrip via WHERE ... IN (...).
// The caller is expected to have validated hierarchy access for every ID.
func (r *LocalAlertHistoryRepository) GetAlertHistoryTrendByOrgIDs(period int, orgIDs []string) (*models.TrendResponse, error) {
	now := time.Now().UTC()
	currentStart := now.AddDate(0, 0, -period)
	previousStart := currentStart.AddDate(0, 0, -period)

	periodLabels := map[int]string{7: "7 days", 30: "30 days", 180: "6 months", 365: "1 year"}
	label := periodLabels[period]
	if label == "" {
		label = fmt.Sprintf("%d days", period)
	}

	zeroDataPoints := func() []models.TrendDataPoint {
		dps := make([]models.TrendDataPoint, 0, period)
		for i := 0; i < period; i++ {
			dps = append(dps, models.TrendDataPoint{Date: currentStart.AddDate(0, 0, i+1).Format("2006-01-02"), Count: 0})
		}
		return dps
	}

	if len(orgIDs) == 0 {
		return &models.TrendResponse{
			Period: period, PeriodLabel: label,
			CurrentTotal: 0, PreviousTotal: 0, Delta: 0,
			DeltaPercentage: 0, Trend: "stable",
			DataPoints: zeroDataPoints(),
		}, nil
	}

	// Build IN clause with positional placeholders. Reuse for the three queries
	// below; placeholder index is offset based on the leading time params.
	buildIn := func(startIdx int) (string, []interface{}) {
		ph := make([]string, len(orgIDs))
		args := make([]interface{}, len(orgIDs))
		for i, id := range orgIDs {
			ph[i] = fmt.Sprintf("$%d", startIdx+i)
			args[i] = id
		}
		return strings.Join(ph, ","), args
	}

	var currentTotal int
	{
		ph, args := buildIn(2) // $1 = currentStart
		query := fmt.Sprintf(`SELECT COUNT(*) FROM alert_history WHERE created_at >= $1 AND organization_id IN (%s)`, ph)
		if err := r.db.QueryRow(query, append([]interface{}{currentStart}, args...)...).Scan(&currentTotal); err != nil {
			return nil, fmt.Errorf("failed to count current period: %w", err)
		}
	}

	var previousTotal int
	{
		ph, args := buildIn(3) // $1 = previousStart, $2 = currentStart
		query := fmt.Sprintf(`SELECT COUNT(*) FROM alert_history WHERE created_at >= $1 AND created_at < $2 AND organization_id IN (%s)`, ph)
		if err := r.db.QueryRow(query, append([]interface{}{previousStart, currentStart}, args...)...).Scan(&previousTotal); err != nil {
			return nil, fmt.Errorf("failed to count previous period: %w", err)
		}
	}

	ph, args := buildIn(2) // $1 = currentStart
	dpQuery := fmt.Sprintf(`SELECT DATE(created_at) AS day, COUNT(*) AS count FROM alert_history WHERE created_at >= $1 AND organization_id IN (%s) GROUP BY day ORDER BY day`, ph)
	rows, err := r.db.Query(dpQuery, append([]interface{}{currentStart}, args...)...)
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

	dataPoints := make([]models.TrendDataPoint, 0, period)
	for i := 0; i < period; i++ {
		date := currentStart.AddDate(0, 0, i+1).Format("2006-01-02")
		dataPoints = append(dataPoints, models.TrendDataPoint{Date: date, Count: pointMap[date]})
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

	return &models.TrendResponse{
		Period: period, PeriodLabel: label,
		CurrentTotal: currentTotal, PreviousTotal: previousTotal,
		Delta: delta, DeltaPercentage: deltaPercentage, Trend: trend,
		DataPoints: dataPoints,
	}, nil
}

// ReassignSystemAlertHistory rewrites the denormalized organization_id on every
// alert_history row that belongs to the given system_key and was previously
// scoped to fromOrgID. Used by the org-reassignment flow so the new owner can
// see the system's full history while the donor org loses access along with
// the system. Returns the number of rows updated; idempotent (a second call
// matches no rows because organization_id has already moved).
func (r *LocalAlertHistoryRepository) ReassignSystemAlertHistory(systemKey, fromOrgID, toOrgID string) (int64, error) {
	if systemKey == "" || fromOrgID == "" || toOrgID == "" {
		return 0, fmt.Errorf("systemKey, fromOrgID and toOrgID are required")
	}
	if fromOrgID == toOrgID {
		return 0, nil
	}
	result, err := r.db.Exec(
		`UPDATE alert_history
		 SET organization_id = $1
		 WHERE system_key = $2 AND organization_id = $3`,
		toOrgID, systemKey, fromOrgID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to reassign alert history: %w", err)
	}
	return result.RowsAffected()
}
